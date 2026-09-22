package reviewrun

import (
	"context"
	"fmt"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// maxBackupSize is the largest SQL text prior backup can handle, the same
// bound the task executor enforces.
const maxBackupSize = common.MaxSheetCheckSize

// RuleExecutor evaluates the standard rules against every (spec, target)
// unit of the issue's plan through omni's SQL Review V2 contract: one Review
// call per (spec, engine), the caller resolving every input that needs the
// store and the engine deriving the findings.
type RuleExecutor struct {
	store       *store.Store
	reviewFuncs map[storepb.Engine]review.ReviewFunc
}

// NewRuleExecutor creates the standard-rule review executor.
func NewRuleExecutor(s *store.Store) *RuleExecutor {
	return &RuleExecutor{store: s, reviewFuncs: reviewFuncs}
}

// RunOnce implements Executor. Collect-all, no fail-fast: every unit is
// attempted, failures aggregate into one message, and the findings are
// returned only when every unit was evaluated.
func (e *RuleExecutor) RunOnce(ctx context.Context, projectID string, issueUID int64) ([]*store.IssueCommentMessage, error) {
	issue, err := e.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, UID: &issueUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue")
	}
	if issue == nil {
		return nil, errors.Errorf("issue %d not found in project %s", issueUID, projectID)
	}
	if issue.PlanUID == nil {
		return nil, errors.Errorf("issue %d has no plan", issueUID)
	}
	plan, err := e.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: issue.PlanUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return nil, errors.Errorf("plan %d not found in project %s", *issue.PlanUID, projectID)
	}
	project, err := e.store.GetProjectByResourceID(ctx, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return nil, errors.Errorf("project %s not found", projectID)
	}
	policy, err := e.store.GetEffectiveReviewRulePolicy(ctx, project.Workspace, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get review rule policy")
	}

	databaseGroup, err := plancheck.GetDatabaseGroupForPlan(ctx, e.store, plan, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group for plan")
	}
	// DeriveReviewTargets, not DeriveCheckTargets: review must evaluate every
	// (spec, target) unit, so the CI sampling limit does not apply.
	checkTargets, err := plancheck.DeriveReviewTargets(ctx, e.store, project, plan, databaseGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to derive review targets")
	}

	var unitErrs []error
	var targets []*reviewTarget
	for _, checkTarget := range checkTargets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, err := e.resolveReviewTarget(ctx, project, checkTarget)
		if err != nil {
			unitErrs = append(unitErrs, errors.Wrapf(err, "%s", checkTarget.Target))
			continue
		}
		targets = append(targets, target)
	}

	var comments []*store.IssueCommentMessage
	sheets := make(map[string]string)
	for _, unit := range groupReviewUnits(reviewRules(policy), targets) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reviewFunc, ok := e.reviewFuncs[unit.Engine]
		if !ok {
			unitErrs = append(unitErrs, unit.failures(errors.Errorf("engine %s has no standard rule reviewer", unit.Engine))...)
			continue
		}
		sql, ok := sheets[unit.SheetSha256]
		if !ok {
			sheet, err := e.store.GetSheetFull(ctx, unit.SheetSha256)
			if err != nil {
				unitErrs = append(unitErrs, unit.failures(errors.Wrapf(err, "failed to get sheet"))...)
				continue
			}
			if sheet == nil {
				unitErrs = append(unitErrs, unit.failures(errors.Errorf("sheet %s not found", unit.SheetSha256))...)
				continue
			}
			sql = sheet.Statement
			sheets[unit.SheetSha256] = sql
		}
		inputs := make([]review.Target, 0, len(unit.Targets))
		for _, target := range unit.Targets {
			inputs = append(inputs, target.Input)
		}
		result, err := reviewFunc(ctx, sql, unit.Options, inputs)
		if err != nil {
			unitErrs = append(unitErrs, unit.failures(err)...)
			continue
		}
		for _, failure := range result.Failures {
			if failure.Target < 0 || failure.Target >= len(unit.Targets) {
				unitErrs = append(unitErrs, errors.Wrapf(failure.Err, "target %d of spec %s", failure.Target, unit.SpecID))
				continue
			}
			unitErrs = append(unitErrs, errors.Wrapf(failure.Err, "%s", unit.Targets[failure.Target].Check.Target))
		}
		comments = append(comments, reviewResultComments(projectID, issueUID, unit, sql, result)...)
	}
	if err := aggregateUnitErrors(len(checkTargets), unitErrs); err != nil {
		return nil, err
	}
	return comments, nil
}

// failures reports one error per target of the unit, so the aggregate counts
// every database the failure kept from being reviewed.
func (u *reviewUnit) failures(err error) []error {
	errs := make([]error, 0, len(u.Targets))
	for _, target := range u.Targets {
		errs = append(errs, errors.Wrapf(err, "%s", target.Check.Target))
	}
	return errs
}

// resolveReviewTarget gathers the inputs the engine cannot derive from the
// SQL: the synced schema, and the instance facts the review contract lists.
// A database whose schema is not synced cannot be reviewed.
func (e *RuleExecutor) resolveReviewTarget(ctx context.Context, project *store.ProjectMessage, checkTarget *plancheck.CheckTarget) (*reviewTarget, error) {
	instance, database, err := plancheck.ResolveDatabaseTarget(ctx, e.store, checkTarget.Target)
	if err != nil {
		return nil, err
	}
	engine := instance.Metadata.GetEngine()
	dbSchema, err := e.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    instance.Workspace,
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database schema")
	}
	if dbSchema == nil || dbSchema.GetProto() == nil {
		return nil, errors.New("metadata not synced")
	}
	schema := dbSchema.GetProto()

	backupDatabaseExists, err := e.backupDatabaseExists(ctx, instance, schema)
	if err != nil {
		return nil, err
	}

	return &reviewTarget{
		Check:  checkTarget,
		Engine: engine,
		Input: review.Target{
			Schema:               schema,
			BackupDatabaseExists: backupDatabaseExists,
			SessionUser:          sessionUser(project, instance, schema.GetOwner()),
			LowerCaseTableNames:  int(instance.Metadata.GetMysqlLowerCaseTableNames()),
		},
	}, nil
}

// backupDatabaseExists reports whether the engine's backup location is in
// place, as the schema syncer decides it: PostgreSQL keeps the archive as a
// schema inside the database, the other engines with prior backup as a
// database on the instance.
func (e *RuleExecutor) backupDatabaseExists(ctx context.Context, instance *store.InstanceMessage, schema *metadatapb.DatabaseSchemaMetadata) (bool, error) {
	engine := instance.Metadata.GetEngine()
	if !common.EngineSupportPriorBackup(engine) {
		return false, nil
	}
	name := common.BackupDatabaseNameOfEngine(engine)
	if engine == storepb.Engine_POSTGRES {
		for _, s := range schema.GetSchemas() {
			if s.GetName() == name {
				return true, nil
			}
		}
		return false, nil
	}
	backupDatabase, err := e.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    instance.Workspace,
		InstanceID:   &instance.ResourceID,
		DatabaseName: &name,
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to look up backup database %q", name)
	}
	return backupDatabase != nil, nil
}

// sessionUser is the role the change runs as: the database owner when the
// project's PostgreSQL tenant mode switches to it, else the admin data
// source's login user. Empty opts out of the engine's ownership checks.
func sessionUser(project *store.ProjectMessage, instance *store.InstanceMessage, owner string) string {
	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
		return ""
	}
	if project.Setting.GetPostgresDatabaseTenantMode() {
		return owner
	}
	return utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN).GetUsername()
}

// aggregateUnitErrors folds per-unit failures into one message, e.g.
// "2 of 5 review units failed: instances/prod/databases/db1: ...;
// instances/prod/databases/db2: ... (+1 more)".
func aggregateUnitErrors(total int, unitErrs []error) error {
	if len(unitErrs) == 0 {
		return nil
	}
	const maxDetailed = 3
	details := make([]string, 0, maxDetailed)
	for i, err := range unitErrs {
		if i >= maxDetailed {
			break
		}
		details = append(details, err.Error())
	}
	suffix := ""
	if len(unitErrs) > maxDetailed {
		suffix = fmt.Sprintf(" (+%d more)", len(unitErrs)-maxDetailed)
	}
	return errors.Errorf("%d of %d review units failed: %s%s", len(unitErrs), total, strings.Join(details, "; "), suffix)
}

package reviewrun

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// RuleExecutor evaluates the standard rules against every (spec, target)
// unit of the issue's plan.
//
// Findings are not persisted yet: review results become issue comments, and
// the comment integration lands with the comment-thread design. The executor
// still evaluates every unit for real so the run lifecycle, fencing, and
// failure aggregation exercise the true path; computed advices are
// discarded.
type RuleExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// NewRuleExecutor creates the standard-rule review executor.
func NewRuleExecutor(s *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory) *RuleExecutor {
	return &RuleExecutor{
		store:        s,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// RunOnce implements Executor.
func (e *RuleExecutor) RunOnce(ctx context.Context, projectID string, issueUID int64) error {
	issue, err := e.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, UID: &issueUID})
	if err != nil {
		return errors.Wrapf(err, "failed to get issue")
	}
	if issue == nil {
		return errors.Errorf("issue %d not found in project %s", issueUID, projectID)
	}
	if issue.PlanUID == nil {
		return errors.Errorf("issue %d has no plan", issueUID)
	}
	plan, err := e.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: projectID, UID: issue.PlanUID})
	if err != nil {
		return errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return errors.Errorf("plan %d not found in project %s", *issue.PlanUID, projectID)
	}
	project, err := e.store.GetProjectByResourceID(ctx, projectID)
	if err != nil {
		return errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %s not found", projectID)
	}

	databaseGroup, err := plancheck.GetDatabaseGroupForPlan(ctx, e.store, plan, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to get database group for plan")
	}
	// DeriveReviewTargets, not DeriveCheckTargets: review must evaluate every
	// (spec, target) unit, so the CI sampling limit does not apply.
	targets, err := plancheck.DeriveReviewTargets(ctx, e.store, project, plan, databaseGroup)
	if err != nil {
		return errors.Wrapf(err, "failed to derive review targets")
	}

	adviseExecutor := plancheck.NewStatementAdviseExecutor(e.store, e.sheetManager, e.dbFactory)
	var unitErrs []error
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Collect-all, no fail-fast: every unit is attempted, and failures
		// aggregate into one message.
		if _, err := adviseExecutor.RunForTarget(ctx, target); err != nil {
			unitErrs = append(unitErrs, errors.Wrapf(err, "%s", target.Target))
		}
	}
	return aggregateUnitErrors(len(targets), unitErrs)
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

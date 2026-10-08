package reviewrun

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// AIReviewExecutor is the AI review: a model judges each spec's sheet against
// the workspace and project AI review policy, once per database, with that
// database's facts in the prompt. The reviews of one run share the model and
// run concurrently under aiReviewConcurrency.
type AIReviewExecutor struct {
	store *store.Store
}

// NewAIReviewExecutor creates the AI review executor.
func NewAIReviewExecutor(s *store.Store) *AIReviewExecutor {
	return &AIReviewExecutor{store: s}
}

// RunOnce implements Executor. Collect-all, no fail-fast: every database is
// attempted, failures aggregate into one message, and the findings are
// returned only when every database was reviewed.
func (e *AIReviewExecutor) RunOnce(ctx context.Context, projectID string, issueUID int64) ([]*store.IssueCommentMessage, error) {
	plan, project, err := loadReviewPlan(ctx, e.store, projectID, issueUID)
	if err != nil {
		return nil, err
	}
	aiSetting, err := e.store.GetAISetting(ctx, project.Workspace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get AI setting")
	}
	if !aiSetting.GetEnabled() {
		return nil, errors.New("AI is not enabled in the workspace setting")
	}
	policy, err := e.store.GetEffectiveReviewAIPolicy(ctx, project.Workspace, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get AI review policy")
	}

	databaseGroup, err := plancheck.GetDatabaseGroupForPlan(ctx, e.store, plan, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group for plan")
	}
	// DeriveReviewTargets, not DeriveCheckTargets: every database is reviewed
	// with its own facts, so the CI sampling limit does not apply.
	checkTargets, err := plancheck.DeriveReviewTargets(ctx, e.store, project, plan, databaseGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to derive review targets")
	}
	checkTargets = uniqueCheckTargets(checkTargets)

	var unitErrs []error
	var units []*aiReviewUnit
	sheets := make(map[string]string)
	for _, checkTarget := range checkTargets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		statement, err := e.sheetStatement(ctx, sheets, checkTarget.SheetSha256)
		if err != nil {
			unitErrs = append(unitErrs, errors.Wrapf(err, "%s", checkTarget.Target))
			continue
		}
		target, err := e.resolveAIReviewTarget(ctx, checkTarget)
		if err != nil {
			unitErrs = append(unitErrs, errors.Wrapf(err, "%s", checkTarget.Target))
			continue
		}
		units = append(units, &aiReviewUnit{Check: checkTarget, Target: target, Statement: statement})
	}

	reviewer := aireview.NewReviewer(aireview.NewModel(aiSetting))
	results, reviewErrs := reviewUnits(ctx, units, func(ctx context.Context, unit *aiReviewUnit) (*aireview.Result, error) {
		result, err := reviewer.Review(ctx, &aireview.Request{
			WorkspacePolicy: policy.Workspace,
			ProjectPolicy:   policy.Project,
			Target:          unit.Target,
			Statement:       unit.Statement,
		}, aireview.NoTools{})
		if err != nil {
			return nil, err
		}
		// The notes name the facts the model could not get. They are for
		// the operator, never a finding.
		slog.Info("AI review completed",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.String("spec", unit.Check.SpecID),
			slog.String("target", unit.Check.Target),
			slog.Int("model_calls", result.Calls),
			slog.Int("tokens", result.TotalTokens),
			slog.Int("findings", len(result.Findings)),
			slog.Any("notes", result.Notes))
		return result, nil
	})
	// A canceled context is shutdown: the scheduler leaves the run to the
	// reaper instead of failing it.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, err := range reviewErrs {
		if err != nil {
			unitErrs = append(unitErrs, err)
		}
	}
	if err := aggregateUnitErrors(len(checkTargets), unitErrs); err != nil {
		return nil, err
	}
	return aiReviewComments(projectID, issueUID, results), nil
}

// sheetStatement returns the sheet text, loaded once per run, and refuses a
// sheet over maxAIReviewSheetBytes.
func (e *AIReviewExecutor) sheetStatement(ctx context.Context, sheets map[string]string, sha256 string) (string, error) {
	if statement, ok := sheets[sha256]; ok {
		return statement, nil
	}
	sheet, err := e.store.GetSheetFull(ctx, sha256)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get sheet")
	}
	if sheet == nil {
		return "", errors.Errorf("sheet %s not found", sha256)
	}
	if err := checkSheetSize(sheet.Statement); err != nil {
		return "", err
	}
	sheets[sha256] = sheet.Statement
	return sheet.Statement, nil
}

// resolveAIReviewTarget gathers the facts the prompt states about a database.
// A database whose schema is not synced cannot be reviewed.
func (e *AIReviewExecutor) resolveAIReviewTarget(ctx context.Context, checkTarget *plancheck.CheckTarget) (aireview.Target, error) {
	instance, database, err := plancheck.ResolveDatabaseTarget(ctx, e.store, checkTarget.Target)
	if err != nil {
		return aireview.Target{}, err
	}
	dbSchema, err := e.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    instance.Workspace,
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return aireview.Target{}, errors.Wrapf(err, "failed to get database schema")
	}
	if dbSchema == nil || dbSchema.GetProto() == nil {
		return aireview.Target{}, errors.New("metadata not synced")
	}
	return aiReviewTarget(instance, database, dbSchema.GetProto()), nil
}

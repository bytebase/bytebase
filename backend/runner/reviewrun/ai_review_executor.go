package reviewrun

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// AIReviewExecutor is the AI review: a model judges each spec's sheet against
// the workspace and project AI review policy, once per database of the
// project's CI sample, with that database's facts in the prompt and its synced
// schema behind the tools. The reviews of one run share the model and run
// concurrently under aiReviewRunConcurrency.
type AIReviewExecutor struct {
	store    *store.Store
	newModel func(*storepb.AISetting) aireview.Model
	// slots bounds the reviews in flight across every run of the executor.
	slots reviewSlots
}

// NewAIReviewExecutor creates the AI review executor. The server creates one,
// so aiReviewServerConcurrency bounds the reviews of every run on the server.
func NewAIReviewExecutor(s *store.Store) *AIReviewExecutor {
	return &AIReviewExecutor{
		store:    s,
		newModel: aireview.NewModel,
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
}

// RunOnce implements Executor. Collect-all, no fail-fast: every database of
// the sample is attempted, failures aggregate into one message, and the
// findings are returned only when every one of them was reviewed.
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
	// DeriveCheckTargets, not DeriveReviewTargets: every database costs model
	// calls, so the AI review covers the databases plan checks sample when the
	// project sets ci_sampling_size, and every database otherwise.
	checkTargets, err := plancheck.DeriveCheckTargets(ctx, e.store, project, plan, databaseGroup)
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
		units = append(units, &aiReviewUnit{Check: checkTarget, Statement: statement})
	}

	reviewer := aireview.NewReviewer(e.newModel(aiSetting))
	results, reviewErrs := reviewUnits(ctx, e.slots, units, func(ctx context.Context, unit *aiReviewUnit) (*aireview.Result, error) {
		// The schema is loaded once the unit holds its slot and dropped when
		// the review ends, so the runs together hold at most one schema per
		// slot however many databases they review. The store's schema cache
		// keeps its own bounded copies.
		request := &aireview.Request{
			WorkspacePolicy: policy.Workspace,
			ProjectPolicy:   policy.Project,
			Statement:       unit.Statement,
		}
		tools, err := e.resolveAIReviewTarget(ctx, unit.Check, request)
		if err != nil {
			return nil, err
		}
		result, err := reviewer.Review(ctx, request, tools)
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

// resolveAIReviewTarget gathers what one review needs of a database: it sets
// the facts the prompt states as the request's target and returns the tools
// over the synced schema, sized to the request's prompt. A database whose
// schema is not synced cannot be reviewed.
func (e *AIReviewExecutor) resolveAIReviewTarget(ctx context.Context, checkTarget *plancheck.CheckTarget, request *aireview.Request) (aireview.Tools, error) {
	instance, database, err := plancheck.ResolveDatabaseTarget(ctx, e.store, checkTarget.Target)
	if err != nil {
		return nil, err
	}
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
	request.Target = aiReviewTarget(instance, database, schema)
	return aireview.NewCatalogTools(instance.Metadata.GetEngine(), schema, aireview.WithPromptOf(request)), nil
}

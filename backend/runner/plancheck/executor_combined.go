package plancheck

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// CombinedExecutor processes all plan check types.
type CombinedExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
	// ruleReviewCovers reports whether SQL Review V2's standard rule review
	// covers an engine; nil where SQL Review V2 is off.
	ruleReviewCovers func(storepb.Engine) bool
}

// NewCombinedExecutor creates a combined executor.
func NewCombinedExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	ruleReviewCovers func(storepb.Engine) bool,
) *CombinedExecutor {
	return &CombinedExecutor{
		store:            store,
		sheetManager:     sheetManager,
		dbFactory:        dbFactory,
		ruleReviewCovers: ruleReviewCovers,
	}
}

// RunForTarget runs all checks for the given target.
func (e *CombinedExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, checkType := range target.Types {
		if e.supersededBySQLReviewV2(ctx, target, checkType) {
			continue
		}
		results, err := e.runCheck(ctx, target, checkType)
		if err != nil {
			// Add error result for this target/type, continue to next
			allResults = append(allResults, &storepb.PlanCheckRunResult_Result{
				Status:      storepb.Advice_ERROR,
				Target:      target.Target,
				Type:        checkType,
				SheetSha256: target.SheetSha256,
				Title:       "Check failed",
				Content:     err.Error(),
				Code:        common.Internal.Int32(),
			})
			continue
		}
		// Tag results with target info
		for _, r := range results {
			r.Target = target.Target
			r.Type = checkType
			r.SheetSha256 = target.SheetSha256
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// supersededBySQLReviewV2 reports whether the standard rule review replaces
// the check where SQL Review V2 is on: only the SQL review advice, and only on
// an engine the rule review covers; V1 keeps advising the others. The gh-ost
// check stays, since it dry-runs the migration against the instance, which
// the ONLINE_MIGRATION rule, reading the SQL alone, cannot.
func (e *CombinedExecutor) supersededBySQLReviewV2(ctx context.Context, target *CheckTarget, checkType storepb.PlanCheckType) bool {
	if e.ruleReviewCovers == nil || checkType != storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE {
		return false
	}
	instance, _, err := ResolveDatabaseTarget(ctx, e.store, target.Target)
	if err != nil {
		// The advice runs, and reports the target it cannot resolve.
		return false
	}
	return e.ruleReviewCovers(instance.Metadata.GetEngine())
}

func (e *CombinedExecutor) runCheck(ctx context.Context, target *CheckTarget, checkType storepb.PlanCheckType) ([]*storepb.PlanCheckRunResult_Result, error) {
	switch checkType {
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE:
		return e.runStatementAdvise(ctx, target)
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT:
		return e.runStatementReport(ctx, target)
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC:
		return e.runGhostSync(ctx, target)
	default:
		return nil, nil
	}
}

func (e *CombinedExecutor) runStatementAdvise(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &StatementAdviseExecutor{
		store:        e.store,
		sheetManager: e.sheetManager,
		dbFactory:    e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}

func (e *CombinedExecutor) runStatementReport(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &StatementReportExecutor{
		store:        e.store,
		sheetManager: e.sheetManager,
		dbFactory:    e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}

func (e *CombinedExecutor) runGhostSync(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &GhostSyncExecutor{
		store:     e.store,
		dbFactory: e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}

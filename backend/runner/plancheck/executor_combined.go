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
}

// NewCombinedExecutor creates a combined executor.
func NewCombinedExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
) *CombinedExecutor {
	return &CombinedExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// RunForTarget runs all checks for the given target.
func (e *CombinedExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, checkType := range target.Types {
		results, err := e.runCheck(ctx, target, checkType)
		if err != nil {
			// Add error result for this target/type, continue to next
			allResults = append(allResults, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.Advice_ERROR,
				Target:  target.Target,
				Type:    checkType,
				Title:   "Check failed",
				Content: err.Error(),
				Code:    common.Internal.Int32(),
			})
			continue
		}
		// Tag results with target info
		for _, r := range results {
			r.Target = target.Target
			r.Type = checkType
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
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

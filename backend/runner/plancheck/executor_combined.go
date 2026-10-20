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
	sqlReviewV2  bool
}

// NewCombinedExecutor creates a combined executor.
func NewCombinedExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	sqlReviewV2 bool,
) *CombinedExecutor {
	return &CombinedExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
		sqlReviewV2:  sqlReviewV2,
	}
}

// RunForTarget runs all checks for the given target.
func (e *CombinedExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, checkType := range target.Types {
		if e.sqlReviewV2 && supersededBySQLReviewV2(checkType) {
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
// the check where SQL Review V2 is on: the SQL review advice, and gh-ost's
// eligibility, which the ONLINE_MIGRATION rule covers.
func supersededBySQLReviewV2(checkType storepb.PlanCheckType) bool {
	return checkType == storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE ||
		checkType == storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC
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

package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCombinedExecutorLeavesSQLReviewToV2(t *testing.T) {
	t.Parallel()
	// Without a store, running the advice would panic.
	executor := NewCombinedExecutor(nil, nil, nil, true)
	results, err := executor.RunForTarget(context.Background(), &CheckTarget{
		Target: "instances/prod/databases/app",
		Types:  []storepb.PlanCheckType{storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE},
	})
	require.NoError(t, err)
	require.Empty(t, results)

	require.False(t, supersededBySQLReviewV2(storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC), "no rule dry-runs gh-ost against the instance")
	require.False(t, supersededBySQLReviewV2(storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT))
}

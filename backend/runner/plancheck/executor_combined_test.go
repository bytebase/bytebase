package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCombinedExecutorLeavesSQLReviewToV2(t *testing.T) {
	t.Parallel()
	// Without a store or database factory, running either check would panic.
	executor := NewCombinedExecutor(nil, nil, nil, true)
	results, err := executor.RunForTarget(context.Background(), &CheckTarget{
		Target: "instances/prod/databases/app",
		Types: []storepb.PlanCheckType{
			storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
			storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC,
		},
	})
	require.NoError(t, err)
	require.Empty(t, results)
}

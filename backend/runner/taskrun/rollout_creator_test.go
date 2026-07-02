package taskrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestPlanCheckRunBlocksRolloutIgnoresStaleVersion(t *testing.T) {
	plan := &store.PlanMessage{
		Config: &storepb.PlanConfig{ApprovalInputVersion: 2},
	}

	require.False(t, planCheckRunBlocksRollout(plan, nil))
	require.False(t, planCheckRunBlocksRollout(plan, &store.PlanCheckRunMessage{
		Status: store.PlanCheckRunStatusRunning,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	}))
	require.True(t, planCheckRunBlocksRollout(plan, &store.PlanCheckRunMessage{
		Status: store.PlanCheckRunStatusRunning,
		Result: &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	}))
}

package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestPlanWebhookDeliveryClaims pins the state machine behind PIPELINE_COMPLETED
// and PIPELINE_FAILED: one row per plan, so the first claim wins and the second
// event of either kind stays silent until a retry resets the row.
func TestPlanWebhookDeliveryClaims(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('c', 'c@example.com', 'x');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p1', 'default', 'P1'), ('p2', 'default', 'P2');
	`)
	require.NoError(t, err)

	newPlan := func(projectID string) int64 {
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: projectID, Name: "plan", Config: &storepb.PlanConfig{},
		}, "c@example.com")
		require.NoError(t, err)
		return plan.UID
	}
	planA, planB := newPlan("p1"), newPlan("p2")

	claimed, err := stores.ClaimPipelineCompletionNotification(ctx, "p1", planA)
	require.NoError(t, err)
	require.True(t, claimed, "the first completion claims the plan")

	claimed, err = stores.ClaimPipelineCompletionNotification(ctx, "p1", planA)
	require.NoError(t, err)
	require.False(t, claimed, "a second completion for the same plan sends nothing")

	// Both events share the row, so a completed plan cannot then report failure.
	claimed, err = stores.ClaimPipelineFailureNotification(ctx, "p1", planA)
	require.NoError(t, err)
	require.False(t, claimed)

	// A retry resets the row, which is what re-arms both events.
	require.NoError(t, stores.ResetPlanWebhookDelivery(ctx, "p1", planA))
	claimed, err = stores.ClaimPipelineFailureNotification(ctx, "p1", planA)
	require.NoError(t, err)
	require.True(t, claimed, "the reset re-arms the plan")

	// Plan UIDs restart per project, so a neighbor's claim must not be read as
	// this plan's.
	require.Equal(t, planA, planB, "plan UIDs must collide across projects for this to prove anything")
	claimed, err = stores.ClaimPipelineCompletionNotification(ctx, "p2", planB)
	require.NoError(t, err)
	require.True(t, claimed, "project p2 claims its own plan")

	var rows int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM plan_webhook_delivery WHERE project = $1 AND plan_id = $2 AND event_type = 'PIPELINE_FAILED'`,
		"p1", planA).Scan(&rows))
	require.Equal(t, 1, rows)
}

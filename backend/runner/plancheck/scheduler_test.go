package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/bus"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMarkPlanCheckRunDoneSkipsDraftIssue(t *testing.T) {
	ctx := context.Background()
	stores := setupPlancheckStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)
	claimed, err := stores.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	scheduler := NewScheduler(stores, b, nil, nil)
	scheduler.markPlanCheckRunDone(ctx, "project-a", claimed[0].UID, plan.UID, 1, nil)

	run, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.Equal(t, store.PlanCheckRunStatusDone, run.Status)
	require.Empty(t, b.ApprovalCheckChan)
}

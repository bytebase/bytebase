package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestApplyApprovalTemplateFailsWhenProjectLifecycleGateHeld(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupWorkflowStore(ctx, t)
	plan, issue := newReviewLifecycleFixture(ctx, t, stores)
	unlock := lockReviewProjectLifecycleGate(ctx, t, stores, issue.ProjectID)
	defer unlock()

	evaluator := &ApprovalEvaluator{
		workflow: NewWorkflow(stores),
		evaluateApproval: func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
			issue.Payload.Approval = &storepb.IssuePayloadApproval{ApprovalFindingDone: true, ApprovalInputVersion: plan.Config.GetApprovalInputVersion()}
			return nil
		},
	}
	_, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
		Workspace: "default", ProjectID: issue.ProjectID, IssueUID: issue.UID,
	})
	requireReviewLifecycleBusy(t, err)

	current, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{issue.ProjectID}, UID: &issue.UID})
	require.NoError(t, err)
	require.False(t, current.Payload.GetApproval().GetApprovalFindingDone())
}

func TestCreateDraftIssueFailsWhenProjectLifecycleGateHeld(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupWorkflowStore(ctx, t)
	plan, _ := newReviewLifecycleFixture(ctx, t, stores)
	unlock := lockReviewProjectLifecycleGate(ctx, t, stores, plan.ProjectID)
	defer unlock()

	_, err := NewWorkflow(stores).CreateDraftIssue(ctx, CreateDraftIssueInput{
		Workspace: "default",
		Issue: &store.IssueMessage{
			ProjectID: plan.ProjectID, CreatorEmail: "creator@example.com", PlanUID: &plan.UID,
			Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{Draft: true},
		},
	})
	requireReviewLifecycleBusy(t, err)

	issues, err := stores.ListIssues(ctx, &store.FindIssueMessage{ProjectIDs: []string{plan.ProjectID}, PlanUID: &plan.UID})
	require.NoError(t, err)
	require.Len(t, issues, 1)
}

func newReviewLifecycleFixture(ctx context.Context, t *testing.T, stores *store.Store) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a", Name: "review lifecycle", Config: &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID: "project-a", CreatorEmail: "creator@example.com", Title: "review lifecycle", Type: storepb.Issue_DATABASE_CHANGE, PlanUID: &plan.UID,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{ApprovalInputVersion: 2}},
	})
	require.NoError(t, err)
	return plan, issue
}

func lockReviewProjectLifecycleGate(ctx context.Context, t *testing.T, stores *store.Store, projectID string) func() {
	t.Helper()
	conn, err := stores.GetDB().Conn(ctx)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1, hashtext($2))", int64(store.AdvisoryLockKeyProjectLifecycle), projectID)
	require.NoError(t, err)
	return func() {
		_, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1, hashtext($2))", int64(store.AdvisoryLockKeyProjectLifecycle), projectID)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}
}

func requireReviewLifecycleBusy(t *testing.T, err error) {
	t.Helper()
	require.ErrorIs(t, err, store.ErrLifecycleBusy)
	require.EqualError(t, err, store.ErrLifecycleBusy.Error())
}

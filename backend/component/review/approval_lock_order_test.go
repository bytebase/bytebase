package review

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type approvalLockOrderApplyOutcome struct {
	result *ApplyApprovalTemplateResult
	err    error
}

type approvalLockOrderRefreshOutcome struct {
	created bool
	err     error
}

// TestApplyApprovalTemplateAndCreatePlanCheckRunDoNotDeadlock proves both
// acquisition directions for the shared Plan review advisory lock. Approval
// takes the advisory lock before locking the Issue and Plan Check Run and reads
// the Plan without a row lock. Refresh takes the advisory lock before locking
// the Plan Check Run and Plan. Each first transaction is deliberately blocked
// after acquiring the advisory lock so the test can prove the competitor waits
// at the common coordination point.
func TestApplyApprovalTemplateAndCreatePlanCheckRunDoNotDeadlock(t *testing.T) {
	t.Run("approval-first", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		stores := setupWorkflowStore(ctx, t)
		plan, issue, initialGeneration := newApprovalLockOrderFixture(ctx, t, stores, 0)

		blockerTx, blockerPID := lockApprovalIssueForTest(ctx, t, stores.GetDB(), issue)
		defer blockerTx.Rollback()

		applyReady := make(chan struct{})
		applyResult := startApprovalLockOrderApply(ctx, stores, issue, applyReady)
		<-applyReady
		applyPID := waitForApprovalLockOrderBlock(ctx, t, stores.GetDB(), blockerPID,
			"approval should acquire the advisory lock, then wait for the Issue row")

		refreshResult := startApprovalLockOrderRefresh(ctx, stores, plan)
		waitForApprovalLockOrderBlock(ctx, t, stores.GetDB(), applyPID,
			"Plan check refresh should wait for approval's advisory lock")

		require.NoError(t, blockerTx.Commit())
		apply := <-applyResult
		require.NoError(t, apply.err)
		require.NotNil(t, apply.result)
		require.True(t, apply.result.Applied)
		refresh := <-refreshResult
		require.NoError(t, refresh.err)
		require.True(t, refresh.created)
		assertApprovalLockOrderTerminalState(ctx, t, stores, plan, issue, initialGeneration, true)
	})

	t.Run("refresh-first", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		stores := setupWorkflowStore(ctx, t)
		plan, issue, initialGeneration := newApprovalLockOrderFixture(ctx, t, stores, 0)

		blockerTx, blockerPID := lockPlanCheckRunForTest(ctx, t, stores.GetDB(), plan)
		defer blockerTx.Rollback()

		refreshResult := startApprovalLockOrderRefresh(ctx, stores, plan)
		refreshPID := waitForApprovalLockOrderBlock(ctx, t, stores.GetDB(), blockerPID,
			"Plan check refresh should acquire the advisory lock, then wait for the Plan Check Run row")

		applyReady := make(chan struct{})
		applyResult := startApprovalLockOrderApply(ctx, stores, issue, applyReady)
		<-applyReady
		waitForApprovalLockOrderBlock(ctx, t, stores.GetDB(), refreshPID,
			"approval should wait for Plan check refresh's advisory lock")

		require.NoError(t, blockerTx.Commit())
		refresh := <-refreshResult
		require.NoError(t, refresh.err)
		require.True(t, refresh.created)
		apply := <-applyResult
		require.Nil(t, apply.result)
		var workflowErr *Error
		require.ErrorAs(t, apply.err, &workflowErr)
		require.Equal(t, ErrorConflict, workflowErr.Code)
		assertApprovalLockOrderTerminalState(ctx, t, stores, plan, issue, initialGeneration, false)
	})
}

func newApprovalLockOrderFixture(ctx context.Context, t *testing.T, stores *store.Store, sequence int) (*store.PlanMessage, *store.IssueMessage, int64) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      fmt.Sprintf("approval lock order %d", sequence),
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        fmt.Sprintf("approval lock order %d", sequence),
		Type:         storepb.Issue_DATABASE_CHANGE,
		PlanUID:      &plan.UID,
		Payload: &storepb.Issue{Approval: &storepb.IssuePayloadApproval{
			ApprovalInputVersion: 2,
		}},
	})
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)
	planCheckRun, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NoError(t, stores.UpdatePlanCheckRun(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 2,
	}, planCheckRun.UID))
	planCheckRun, err = stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	return plan, issue, planCheckRun.Generation
}

func newApprovalLockOrderEvaluator(stores *store.Store) *ApprovalEvaluator {
	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	return evaluator
}

func startApprovalLockOrderApply(ctx context.Context, stores *store.Store, issue *store.IssueMessage, ready chan<- struct{}) <-chan approvalLockOrderApplyOutcome {
	evaluator := newApprovalLockOrderEvaluator(stores)
	evaluator.beforeCommit = func() { close(ready) }
	result := make(chan approvalLockOrderApplyOutcome, 1)
	go func() {
		applied, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: issue.ProjectID,
			IssueUID:  issue.UID,
		})
		result <- approvalLockOrderApplyOutcome{result: applied, err: err}
	}()
	return result
}

func startApprovalLockOrderRefresh(ctx context.Context, stores *store.Store, plan *store.PlanMessage) <-chan approvalLockOrderRefreshOutcome {
	result := make(chan approvalLockOrderRefreshOutcome, 1)
	go func() {
		created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: plan.ProjectID,
			PlanUID:   plan.UID,
			Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
		})
		result <- approvalLockOrderRefreshOutcome{created: created, err: err}
	}()
	return result
}

func lockApprovalIssueForTest(ctx context.Context, t *testing.T, db *sql.DB, issue *store.IssueMessage) (*sql.Tx, int) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	var backendPID int
	var issueUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT pg_backend_pid(), id
		FROM issue
		WHERE project = $1 AND id = $2
		FOR UPDATE
	`, issue.ProjectID, issue.UID).Scan(&backendPID, &issueUID))
	require.Equal(t, issue.UID, issueUID)
	return tx, backendPID
}

func lockPlanCheckRunForTest(ctx context.Context, t *testing.T, db *sql.DB, plan *store.PlanMessage) (*sql.Tx, int) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	var backendPID int
	var planCheckRunUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT pg_backend_pid(), id
		FROM plan_check_run
		WHERE project = $1 AND plan_id = $2
		FOR UPDATE
	`, plan.ProjectID, plan.UID).Scan(&backendPID, &planCheckRunUID))
	require.NotZero(t, planCheckRunUID)
	return tx, backendPID
}

func waitForApprovalLockOrderBlock(ctx context.Context, t *testing.T, db *sql.DB, blockerPID int, message string) int {
	t.Helper()
	var blockedPID int
	require.Eventually(t, func() bool {
		return db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT pid
				FROM pg_stat_activity
				WHERE $1 = ANY(pg_blocking_pids(pid))
				ORDER BY pid
				LIMIT 1
			), 0)
		`, blockerPID).Scan(&blockedPID) == nil && blockedPID != 0
	}, 10*time.Second, 10*time.Millisecond, message)
	return blockedPID
}

func assertApprovalLockOrderTerminalState(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage, issue *store.IssueMessage, initialGeneration int64, approvalApplied bool) {
	t.Helper()
	gotIssue, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  "default",
		ProjectIDs: []string{issue.ProjectID},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.NotNil(t, gotIssue)
	require.Equal(t, approvalApplied, gotIssue.Payload.GetApproval().GetApprovalFindingDone())
	require.EqualValues(t, 2, gotIssue.Payload.GetApproval().GetApprovalInputVersion())

	planCheckRun, err := stores.GetPlanCheckRun(ctx, plan.ProjectID, plan.UID)
	require.NoError(t, err)
	require.NotNil(t, planCheckRun)
	require.Equal(t, store.PlanCheckRunStatusAvailable, planCheckRun.Status)
	require.EqualValues(t, 2, planCheckRun.Result.GetApprovalInputVersion())
	require.NotEqual(t, initialGeneration, planCheckRun.Generation)
}

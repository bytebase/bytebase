package review

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const approvalLifecycleAdvisoryLockID = 990138

type approvalLifecycleApplyOutcome struct {
	result *ApplyApprovalTemplateResult
	err    error
}

func TestApplyApprovalTemplateAndProjectDeletionPurgeFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupWorkflowStore(ctx, t)
	plan, issue := newApprovalLifecycleFixture(ctx, t, stores)
	softDeleteApprovalLifecycleProject(ctx, t, stores)

	db := stores.GetDB()
	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", approvalLifecycleAdvisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", approvalLifecycleAdvisoryLockID)
		}
	}()
	_, err = db.ExecContext(ctx, `
		CREATE FUNCTION block_approval_lifecycle_issue_delete() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(990138);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_approval_lifecycle_issue_delete
		AFTER DELETE ON issue
		FOR EACH ROW EXECUTE FUNCTION block_approval_lifecycle_issue_delete();
	`)
	require.NoError(t, err)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- stores.DeleteProject(ctx, "default", "project-a")
	}()
	deleteBackendPID := waitForAdvisoryLockWaiter(ctx, t, db, approvalLifecycleAdvisoryLockID)

	applyReady := make(chan struct{})
	applyResult := startApprovalLifecycleApply(ctx, stores, issue, applyReady)
	<-applyReady
	waitForBackendBlockedBy(ctx, t, db, deleteBackendPID,
		"approval should wait for the Issue row being purged")

	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", approvalLifecycleAdvisoryLockID)
	require.NoError(t, err)
	lockReleased = true
	require.NoError(t, <-deleteResult)
	assertApprovalLifecycleRejected(t, <-applyResult)
	assertApprovalLifecyclePurged(ctx, t, stores, plan, issue)
}

func TestApplyApprovalTemplateAndProjectDeletionApprovalFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stores := setupWorkflowStore(ctx, t)
	plan, issue := newApprovalLifecycleFixture(ctx, t, stores)
	softDeleteApprovalLifecycleProject(ctx, t, stores)

	planCheckRunLock, planCheckRunLockPID := lockApprovalLifecyclePlanCheckRun(ctx, t, stores.GetDB(), plan)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_ = planCheckRunLock.Rollback()
		}
	}()

	applyReady := make(chan struct{})
	applyResult := startApprovalLifecycleApply(ctx, stores, issue, applyReady)
	<-applyReady
	applyBackendPID := waitForBackendBlockedBy(ctx, t, stores.GetDB(), planCheckRunLockPID,
		"approval should lock the Issue, then wait for the Plan Check Run row")

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- stores.DeleteProject(ctx, "default", "project-a")
	}()
	waitForBackendBlockedBy(ctx, t, stores.GetDB(), applyBackendPID,
		"project deletion should wait for approval's Issue row lock")

	require.NoError(t, planCheckRunLock.Commit())
	lockReleased = true
	apply := <-applyResult
	require.NoError(t, apply.err)
	require.NotNil(t, apply.result)
	require.True(t, apply.result.Applied)
	require.NoError(t, <-deleteResult)
	assertApprovalLifecyclePurged(ctx, t, stores, plan, issue)
}

func softDeleteApprovalLifecycleProject(ctx context.Context, t *testing.T, stores *store.Store) {
	t.Helper()
	deleted := true
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		Workspace:  "default",
		ResourceID: "project-a",
		Delete:     &deleted,
	}))
}

func newApprovalLifecycleFixture(ctx context.Context, t *testing.T, stores *store.Store) (*store.PlanMessage, *store.IssueMessage) {
	t.Helper()
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "approval lifecycle",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "approval lifecycle",
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
	return plan, issue
}

func startApprovalLifecycleApply(ctx context.Context, stores *store.Store, issue *store.IssueMessage, ready chan<- struct{}) <-chan approvalLifecycleApplyOutcome {
	evaluator := &ApprovalEvaluator{workflow: NewWorkflow(stores)}
	evaluator.evaluateApproval = func(_ context.Context, issue *store.IssueMessage, _ *store.ProjectMessage, _ *storepb.WorkspaceApprovalSetting) error {
		issue.Payload.Approval = &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		}
		return nil
	}
	evaluator.beforeCommit = func() { close(ready) }
	result := make(chan approvalLifecycleApplyOutcome, 1)
	go func() {
		applied, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
			Workspace: "default",
			ProjectID: issue.ProjectID,
			IssueUID:  issue.UID,
		})
		result <- approvalLifecycleApplyOutcome{result: applied, err: err}
	}()
	return result
}

func lockApprovalLifecyclePlanCheckRun(ctx context.Context, t *testing.T, db *sql.DB, plan *store.PlanMessage) (*sql.Tx, int) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	var backendPID int
	var planCheckRunUID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT pg_backend_pid(), id
		FROM plan_check_run
		WHERE project = $1
		  AND plan_id = $2
		FOR UPDATE
	`, plan.ProjectID, plan.UID).Scan(&backendPID, &planCheckRunUID))
	require.NotZero(t, planCheckRunUID)
	return tx, backendPID
}

func assertApprovalLifecycleRejected(t *testing.T, outcome approvalLifecycleApplyOutcome) {
	t.Helper()
	require.Nil(t, outcome.result)
	var workflowErr *Error
	require.Error(t, outcome.err)
	require.True(t, errors.As(outcome.err, &workflowErr))
	require.Equal(t, ErrorNotFound, workflowErr.Code)
	require.NotContains(t, strings.ToLower(outcome.err.Error()), "foreign key")
	require.NotContains(t, outcome.err.Error(), "23503")
	require.NotContains(t, outcome.err.Error(), "40P01")
}

func assertApprovalLifecyclePurged(ctx context.Context, t *testing.T, stores *store.Store, plan *store.PlanMessage, issue *store.IssueMessage) {
	t.Helper()
	projectID := "project-a"
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  "default",
		ResourceID: &projectID,
	})
	require.NoError(t, err)
	require.Nil(t, project)

	gotIssue, err := stores.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  "default",
		ProjectIDs: []string{projectID},
		UID:        &issue.UID,
	})
	require.NoError(t, err)
	require.Nil(t, gotIssue)

	gotPlan, err := stores.GetPlan(ctx, &store.FindPlanMessage{
		Workspace: "default",
		ProjectID: projectID,
		UID:       &plan.UID,
	})
	require.NoError(t, err)
	require.Nil(t, gotPlan)

	planCheckRun, err := stores.GetPlanCheckRun(ctx, projectID, plan.UID)
	require.NoError(t, err)
	require.Nil(t, planCheckRun)
}

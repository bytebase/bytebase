package review

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const draftIssueDeletionAdvisoryLockID = 990137

type draftIssueDeletionFixture struct {
	ctx     context.Context
	stores  *store.Store
	planUID int64
}

type createDraftIssueOutcome struct {
	result *CreateDraftIssueResult
	err    error
}

func TestCreateDraftIssueAndProjectDeletionPurgeFirst(t *testing.T) {
	fixture := newDraftIssueDeletionFixture(t)
	ctx, db, stores := fixture.ctx, fixture.stores.GetDB(), fixture.stores

	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", draftIssueDeletionAdvisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", draftIssueDeletionAdvisoryLockID)
		}
	}()
	_, err = db.ExecContext(ctx, `
		CREATE FUNCTION block_draft_issue_plan_delete() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(990137);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_draft_issue_plan_delete
		AFTER DELETE ON plan
		FOR EACH ROW EXECUTE FUNCTION block_draft_issue_plan_delete();
	`)
	require.NoError(t, err)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- stores.DeleteProject(ctx, "default", "project-a")
	}()
	deleteBackendPID := waitForAdvisoryLockWaiter(ctx, t, db, draftIssueDeletionAdvisoryLockID)

	createResult := make(chan createDraftIssueOutcome, 1)
	go func() {
		result, err := createDraftIssue(fixture)
		createResult <- createDraftIssueOutcome{result: result, err: err}
	}()
	waitForBackendBlockedBy(ctx, t, db, deleteBackendPID, "draft Issue creation should wait for the Plan being purged")

	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", draftIssueDeletionAdvisoryLockID)
	require.NoError(t, err)
	lockReleased = true
	require.NoError(t, <-deleteResult)
	assertDraftIssueCreationNotFound(t, <-createResult)
	assertDraftIssueProjectPurged(t, fixture)
}

func TestCreateDraftIssueAndProjectDeletionCreationFirst(t *testing.T) {
	fixture := newDraftIssueDeletionFixture(t)
	ctx, db, stores := fixture.ctx, fixture.stores.GetDB(), fixture.stores

	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	lockTx, err := lockConn.BeginTx(ctx, nil)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_ = lockTx.Rollback()
		}
	}()
	var lockBackendPID int
	require.NoError(t, lockTx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&lockBackendPID))
	var projectID string
	require.NoError(t, lockTx.QueryRowContext(ctx, `
		SELECT resource_id
		FROM project
		WHERE workspace = 'default' AND resource_id = 'project-a'
		FOR UPDATE
	`).Scan(&projectID))
	require.Equal(t, "project-a", projectID)

	createResult := make(chan createDraftIssueOutcome, 1)
	go func() {
		result, err := createDraftIssue(fixture)
		createResult <- createDraftIssueOutcome{result: result, err: err}
	}()
	createBackendPID := waitForBackendBlockedBy(ctx, t, db, lockBackendPID, "draft Issue creation should lock the Plan, then wait for the project")

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- stores.DeleteProject(ctx, "default", "project-a")
	}()
	waitForBackendBlockedBy(ctx, t, db, createBackendPID, "project deletion should pass Issue cleanup, then wait for the locked Plan")

	require.NoError(t, lockTx.Commit())
	lockReleased = true
	assertDraftIssueCreationNotFound(t, <-createResult)
	require.NoError(t, <-deleteResult)
	assertDraftIssueProjectPurged(t, fixture)
}

func newDraftIssueDeletionFixture(t *testing.T) *draftIssueDeletionFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	stores := setupWorkflowStore(ctx, t)
	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft Plan",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	deleted := true
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		Workspace: "default", ResourceID: "project-a", Delete: &deleted,
	}))
	return &draftIssueDeletionFixture{ctx: ctx, stores: stores, planUID: plan.UID}
}

func createDraftIssue(fixture *draftIssueDeletionFixture) (*CreateDraftIssueResult, error) {
	return NewWorkflow(fixture.stores).CreateDraftIssue(fixture.ctx, CreateDraftIssueInput{
		Workspace: "default",
		Issue: &store.IssueMessage{
			ProjectID: "project-a", CreatorEmail: "creator@example.com", PlanUID: &fixture.planUID,
			Type: storepb.Issue_DATABASE_CHANGE,
			Payload: &storepb.Issue{
				Draft: true, Approval: &storepb.IssuePayloadApproval{},
			},
		},
	})
}

func waitForAdvisoryLockWaiter(ctx context.Context, t *testing.T, db *sql.DB, lockID int) int {
	t.Helper()
	var backendPID int
	require.Eventually(t, func() bool {
		return db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT pid
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
				ORDER BY pid
				LIMIT 1
			), 0)
		`, lockID).Scan(&backendPID) == nil && backendPID != 0
	}, 10*time.Second, 10*time.Millisecond, "project deletion should reach the blocked Plan delete")
	return backendPID
}

func waitForBackendBlockedBy(ctx context.Context, t *testing.T, db *sql.DB, blockerPID int, message string) int {
	t.Helper()
	var backendPID int
	require.Eventually(t, func() bool {
		return db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT pid
				FROM pg_stat_activity
				WHERE $1 = ANY(pg_blocking_pids(pid))
				ORDER BY pid
				LIMIT 1
			), 0)
		`, blockerPID).Scan(&backendPID) == nil && backendPID != 0
	}, 10*time.Second, 10*time.Millisecond, message)
	return backendPID
}

func assertDraftIssueCreationNotFound(t *testing.T, outcome createDraftIssueOutcome) {
	t.Helper()
	require.Nil(t, outcome.result)
	var workflowErr *Error
	require.True(t, errors.As(outcome.err, &workflowErr))
	require.Equal(t, ErrorNotFound, workflowErr.Code)
}

func assertDraftIssueProjectPurged(t *testing.T, fixture *draftIssueDeletionFixture) {
	t.Helper()
	projectID := "project-a"
	project, err := fixture.stores.GetProject(fixture.ctx, &store.FindProjectMessage{
		Workspace: "default", ResourceID: &projectID,
	})
	require.NoError(t, err)
	require.Nil(t, project)
	issue, err := fixture.stores.GetIssue(fixture.ctx, &store.FindIssueMessage{
		Workspace: "default", ProjectIDs: []string{projectID}, PlanUID: &fixture.planUID,
	})
	require.NoError(t, err)
	require.Nil(t, issue)
}

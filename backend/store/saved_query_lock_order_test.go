package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func savedQueryStarCount(t *testing.T, fixture *projectDeletionLockOrderFixture) int {
	t.Helper()
	var count int
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM saved_query_star WHERE saved_query = 'saved-query-a'").Scan(&count))
	return count
}

// Existing saved-query and star rows still use child-before-parent locking;
// these paths are independent of lifecycle-gate contention.
func TestSavedQueryDeleteAndStarToggleLockOrder(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'default', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES ('saved-query-a', 'user@example.com');
	`)
	const barrierID = 9935
	barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
	installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER DELETE ON saved_query FOR EACH ROW")

	deleteResult := make(chan error, 1)
	go func() {
		_, err := fixture.store.DeleteSavedQuery(fixture.ctx, "default", "saved-query-a")
		deleteResult <- err
	}()
	waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
	deletePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

	starResult := make(chan error, 1)
	starApplied := make(chan bool, 1)
	go func() {
		applied, err := fixture.store.SetSavedQueryStar(fixture.ctx, "default", "saved-query-a", "other@example.com", true)
		starApplied <- applied
		starResult <- err
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, deletePID)
	barrier.release(t)
	requireMaintenanceResult(t, deleteResult)
	select {
	case err := <-starResult:
		require.NoError(t, err)
		require.False(t, <-starApplied)
	case <-time.After(maintenanceLockWait):
		t.Fatal("timed out waiting for the star toggle")
	}
	require.Zero(t, savedQueryStarCount(t, fixture))
}

func TestSavedQueryFolderMovesLockOrder(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO saved_query (resource_id, creator, project, name, statement, folder) VALUES
			('saved-query-a', 'user@example.com', 'default', 'A', '', 'start'),
			('saved-query-b', 'user@example.com', 'default', 'B', '', 'start/sub');
	`)
	const barrierID = 9939
	barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
	installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER UPDATE OF folder ON saved_query FOR EACH ROW")

	firstResult := make(chan error, 1)
	go func() {
		_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "start", "one")
		firstResult <- err
	}()
	waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
	firstPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)
	secondResult := make(chan error, 1)
	go func() {
		_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "start", "two")
		secondResult <- err
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, firstPID)
	barrier.release(t)
	requireMaintenanceResult(t, firstResult)
	requireMaintenanceResult(t, secondResult)
}

func TestCreateSavedQueryRejectsInactiveProject(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, "")
	_, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "project-a", Creator: "user@example.com", Title: "archived", Statement: "SELECT 1",
	})
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	created, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "default", Creator: "user@example.com", Title: "active", Statement: "SELECT 1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ResourceID)
}

func TestSavedQueryWritersProjectScoped(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'project-a', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES ('saved-query-a', 'user@example.com');
	`)
	_, err := fixture.db.ExecContext(fixture.ctx, "UPDATE saved_query SET project = 'default' WHERE resource_id = 'saved-query-a'")
	require.NoError(t, err)
	title := "raced"
	require.NoError(t, fixture.store.PatchSavedQuery(fixture.ctx, &store.PatchSavedQueryMessage{
		ResourceID: "saved-query-a", ProjectID: "project-a", Title: &title,
	}))
	deleted, err := fixture.store.DeleteSavedQuery(fixture.ctx, "project-a", "saved-query-a")
	require.NoError(t, err)
	require.False(t, deleted)
	applied, err := fixture.store.SetSavedQueryStar(fixture.ctx, "project-a", "saved-query-a", "other@example.com", true)
	require.NoError(t, err)
	require.False(t, applied)

	bindings := []*storepb.SavedQueryBinding{{Level: storepb.SavedQueryBinding_VIEWER, Members: []string{"user:viewer@example.com"}}}
	etag, err := store.SavedQueryPolicyEtag(nil)
	require.NoError(t, err)
	applied, err = fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", bindings, etag)
	require.NoError(t, err)
	require.False(t, applied)
}

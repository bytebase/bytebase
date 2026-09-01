package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

func savedQueryStarCount(t *testing.T, fixture *storePostgresFixture) int {
	t.Helper()
	var count int
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM saved_query_star WHERE saved_query = 'saved-query-a'").Scan(&count))
	return count
}

func TestSavedQueryDeleteAndStarToggleLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'default', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES ('saved-query-a', 'user@example.com');
	`

	t.Run("delete first", func(t *testing.T) {
		fixture := newStorePostgresFixture(t, seedSQL)
		const barrierID = 9935
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.DeleteSavedQuery(fixture.ctx, "default", "saved-query-a")
			deleteResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		deletePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		starResult := make(chan error, 1)
		var starApplied bool
		go func() {
			applied, err := fixture.store.SetSavedQueryStar(fixture.ctx, "default", "saved-query-a", "other@example.com", true)
			starApplied = applied
			starResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, deletePID)
		barrier.release(t)

		requireMaintenanceResult(t, deleteResult)
		select {
		case err := <-starResult:
			require.NoError(t, err)
			require.False(t, starApplied)
		case <-time.After(maintenanceLockWait):
			t.Fatal("timed out waiting for the star toggle")
		}
		require.Zero(t, savedQueryStarCount(t, fixture))
	})

	t.Run("toggle first", func(t *testing.T) {
		fixture := newStorePostgresFixture(t, seedSQL)
		const barrierID = 9936
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query_star FOR EACH ROW")

		unstarResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryStar(fixture.ctx, "default", "saved-query-a", "user@example.com", false)
			unstarResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		unstarPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		deleteResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.DeleteSavedQuery(fixture.ctx, "default", "saved-query-a")
			deleteResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, unstarPID)
		barrier.release(t)

		requireMaintenanceResult(t, unstarResult)
		requireMaintenanceResult(t, deleteResult)
		var queryCount int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT COUNT(*) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&queryCount))
		require.Zero(t, queryCount)
	})
}

func TestSavedQueryBatchFolderMovesLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement, folder) VALUES
			('saved-query-a', 'user@example.com', 'default', 'A', '', 'start'),
			('saved-query-b', 'user@example.com', 'default', 'B', '', 'start/sub');
	`

	fixture := newStorePostgresFixture(t, seedSQL)
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

	secondMoved := make(chan int, 1)
	secondResult := make(chan error, 1)
	go func() {
		moved, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "start", "two")
		secondMoved <- moved
		secondResult <- err
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, firstPID)
	barrier.release(t)
	requireMaintenanceResult(t, firstResult)
	requireMaintenanceResult(t, secondResult)
	require.Equal(t, 0, <-secondMoved)

	var folders []string
	rows, err := fixture.db.QueryContext(fixture.ctx,
		"SELECT folder FROM saved_query ORDER BY resource_id")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var folder string
		require.NoError(t, rows.Scan(&folder))
		folders = append(folders, folder)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"one", "one/sub"}, folders)
}

func TestCreateSavedQueryRejectsInactiveProject(t *testing.T) {
	fixture := newStorePostgresFixture(t, "")

	_, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "project-a",
		Creator:   "user@example.com",
		Title:     "into an archived project",
		Statement: "SELECT 1",
	})
	require.Error(t, err)
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	_, err = fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "no-such-project",
		Creator:   "user@example.com",
		Title:     "into a purged project",
		Statement: "SELECT 1",
	})
	require.Error(t, err)
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	created, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "default",
		Creator:   "user@example.com",
		Title:     "into an active project",
		Statement: "SELECT 1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ResourceID)
}

func TestSavedQueryFolderPrefixMovesLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement, folder) VALUES
			('saved-query-a', 'user@example.com', 'default', 'A', '', 'a'),
			('saved-query-b', 'user@example.com', 'default', 'B', '', 'a/child');
	`

	folders := func(t *testing.T, fixture *storePostgresFixture) map[string]string {
		t.Helper()
		rows, err := fixture.db.QueryContext(fixture.ctx,
			"SELECT resource_id, folder FROM saved_query ORDER BY resource_id")
		require.NoError(t, err)
		defer rows.Close()
		got := map[string]string{}
		for rows.Next() {
			var id, folder string
			require.NoError(t, rows.Scan(&id, &folder))
			got[id] = folder
		}
		require.NoError(t, rows.Err())
		return got
	}

	t.Run("parent first", func(t *testing.T) {
		fixture := newStorePostgresFixture(t, seedSQL)
		const barrierID = 9961
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF folder ON saved_query FOR EACH ROW")

		parentResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "a", "x")
			parentResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		parentPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		childResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "a/child", "y")
			childResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, parentPID)
		barrier.release(t)

		requireMaintenanceResult(t, parentResult)
		requireMaintenanceResult(t, childResult)
		require.Equal(t, map[string]string{
			"saved-query-a": "x",
			"saved-query-b": "x/child",
		}, folders(t, fixture))
	})

	t.Run("child first", func(t *testing.T) {
		fixture := newStorePostgresFixture(t, seedSQL)
		const barrierID = 9962
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF folder ON saved_query FOR EACH ROW")

		childResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "a/child", "y")
			childResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		childPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		parentResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.MoveSavedQueryFolder(fixture.ctx, "default", "user@example.com", "a", "x")
			parentResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, childPID)
		barrier.release(t)

		requireMaintenanceResult(t, childResult)
		requireMaintenanceResult(t, parentResult)
		require.Equal(t, map[string]string{
			"saved-query-a": "x",
			"saved-query-b": "y",
		}, folders(t, fixture))
	})
}

func TestSavedQueryWritersProjectScoped(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement, folder)
			VALUES ('saved-query-a', 'user@example.com', 'project-a', 'Saved Query A', '', 'kept');
		INSERT INTO saved_query_star (saved_query, principal) VALUES ('saved-query-a', 'user@example.com');
	`
	fixture := newStorePostgresFixture(t, seedSQL)

	_, err := fixture.db.ExecContext(fixture.ctx,
		"UPDATE saved_query SET project = 'default' WHERE resource_id = 'saved-query-a'")
	require.NoError(t, err)

	title := "raced"
	require.NoError(t, fixture.store.PatchSavedQuery(fixture.ctx, &store.PatchSavedQueryMessage{
		ResourceID: "saved-query-a",
		ProjectID:  "project-a",
		Title:      &title,
	}))
	deleted, err := fixture.store.DeleteSavedQuery(fixture.ctx, "project-a", "saved-query-a")
	require.NoError(t, err)
	require.False(t, deleted, "a delete authorized in the old project must not land")
	applied, err := fixture.store.SetSavedQueryStar(fixture.ctx, "project-a", "saved-query-a", "other@example.com", true)
	require.NoError(t, err)
	require.False(t, applied, "first star must take the parent fence in the authorized project")

	var name, folder string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT name, folder FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&name, &folder))
	require.Equal(t, "Saved Query A", name, "a patch authorized in the old project must not land")
	require.Equal(t, "kept", folder)
	require.Equal(t, 1, savedQueryStarCount(t, fixture))
}

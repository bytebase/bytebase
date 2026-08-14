package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Deterministic regression locks for the saved-query lock-order invariants:
// child-before-parent for existing rows, parent fence only for a new child
// insert, and full primary-key order for every multi-row lock. Each pair is
// exercised in both acquisition directions, asserting terminal outcomes —
// never a deadlock (40P01) or an FK failure.

func savedQueryStarCount(t *testing.T, fixture *projectDeletionLockOrderFixture) int {
	t.Helper()
	var count int
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM saved_query_star WHERE saved_query = 'saved-query-a'").Scan(&count))
	return count
}

func TestSavedQueryCreateAndDeleteProjectLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'project-a', 'Saved Query A', '');
	`

	t.Run("create rejects a deleted project while its purge is parked", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9931
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF project ON saved_query FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		// The create fence locks the project row (which the parked purge does
		// not yet hold — the purge locks parents last), sees the deleted flag,
		// and rejects cleanly instead of racing to an FK failure.
		createResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
				ProjectID: "project-a",
				Creator:   "user@example.com",
				Title:     "raced",
				Statement: "SELECT 1;",
			})
			createResult <- err
		}()
		select {
		case err := <-createResult:
			require.Error(t, err)
			require.ErrorContains(t, err, "deleted")
		case <-time.After(maintenanceLockWait):
			t.Fatal("saved query creation should reject a deleted project before purge completion")
		}
		barrier.release(t)
		requireMaintenanceResult(t, deleteResult)
	})

	t.Run("create succeeds on an unrelated project while a purge is parked", func(t *testing.T) {
		// project-b is untouched by project-a's purge. The default project is
		// deliberately not the target here: the purge re-parents saved queries
		// into it, and that FK write holds a KEY SHARE lock on the default
		// project row for the rest of the purge transaction, so a create fence
		// there legitimately waits for the (short-lived) purge to commit.
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL+`
			INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B');
		`)
		const barrierID = 9932
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF project ON saved_query FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		created, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
			ProjectID: "project-b",
			Creator:   "user@example.com",
			Title:     "independent",
			Statement: "SELECT 1;",
		})
		require.NoError(t, err)
		require.NotEmpty(t, created.ResourceID)

		barrier.release(t)
		requireMaintenanceResult(t, deleteResult)
	})
}

func TestSavedQueryFirstStarAndDeleteProjectLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'project-a', 'Saved Query A', '');
	`

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9933
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF project ON saved_query FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		purgePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		// The first star takes the parent fence on the saved_query row the
		// parked purge holds via its re-parent update.
		starResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryStar(fixture.ctx, "saved-query-a", "user@example.com", true)
			starResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, purgePID)
		barrier.release(t)

		requireMaintenanceResult(t, deleteResult)
		requireMaintenanceResult(t, starResult)
		// The row survived the purge by re-parenting, so the star landed.
		require.Equal(t, 1, savedQueryStarCount(t, fixture))
	})

	t.Run("star first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9934
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER INSERT ON saved_query_star FOR EACH ROW")

		starResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryStar(fixture.ctx, "saved-query-a", "user@example.com", true)
			starResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		starPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, starPID)
		barrier.release(t)

		requireMaintenanceResult(t, starResult)
		requireMaintenanceResult(t, deleteResult)
		require.Equal(t, 1, savedQueryStarCount(t, fixture))
	})
}

func TestSavedQueryDeleteAndStarToggleLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'user@example.com', 'default', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES ('saved-query-a', 'user@example.com');
	`

	t.Run("delete first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9935
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		// Park the delete after it has removed the star rows and taken the
		// parent row itself — that is the point where a first star from another
		// principal must be fenced out.
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteSavedQuery(fixture.ctx, "saved-query-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		deletePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		// A second principal's first star waits on the parent fence held by
		// the parked delete, then reports the parent gone once it lands.
		starResult := make(chan error, 1)
		var starApplied bool
		go func() {
			applied, err := fixture.store.SetSavedQueryStar(fixture.ctx, "saved-query-a", "other@example.com", true)
			starApplied = applied
			starResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, deletePID)
		barrier.release(t)

		requireMaintenanceResult(t, deleteResult)
		select {
		case err := <-starResult:
			// The parent is gone, so there is nothing to star: reported as
			// not applied rather than as a failure.
			require.NoError(t, err)
			require.False(t, starApplied)
		case <-time.After(maintenanceLockWait):
			t.Fatal("timed out waiting for the star toggle")
		}
		require.Zero(t, savedQueryStarCount(t, fixture))
	})

	t.Run("toggle first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9936
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query_star FOR EACH ROW")

		// Unstar locks and deletes the existing child row, parking at the
		// barrier with the child lock held.
		unstarResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryStar(fixture.ctx, "saved-query-a", "user@example.com", false)
			unstarResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		unstarPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteSavedQuery(fixture.ctx, "saved-query-a") }()
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

func TestSavedQueryDeleteAndDeleteProjectLockOrder(t *testing.T) {
	// The saved query is owned by a project service account, so the purge
	// deletes it (stars first) rather than re-parenting it — the contended
	// pair is the star child rows, locked in primary-key order on both paths.
	const seedSQL = `
		INSERT INTO service_account (name, email, workspace, service_key_hash, project)
			VALUES ('service account', 'sa@example.com', 'default', 'unused', 'project-a');
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'sa@example.com', 'project-a', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES
			('saved-query-a', 'user-1@example.com'),
			('saved-query-a', 'user-2@example.com');
	`

	requireGone := func(t *testing.T, fixture *projectDeletionLockOrderFixture) {
		t.Helper()
		var queryCount int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT COUNT(*) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&queryCount))
		require.Zero(t, queryCount)
		require.Zero(t, savedQueryStarCount(t, fixture))
	}

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9937
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query_star FOR EACH ROW")

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		purgePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteSavedQuery(fixture.ctx, "saved-query-a") }()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, purgePID)
		barrier.release(t)

		requireMaintenanceResult(t, purgeResult)
		requireMaintenanceResult(t, deleteResult)
		requireGone(t, fixture)
	})

	t.Run("delete first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9938
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query_star FOR EACH ROW")

		deleteResult := make(chan error, 1)
		go func() { deleteResult <- fixture.store.DeleteSavedQuery(fixture.ctx, "saved-query-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		deletePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, deletePID)
		barrier.release(t)

		requireMaintenanceResult(t, deleteResult)
		requireMaintenanceResult(t, purgeResult)
		requireGone(t, fixture)
	})
}

func TestSavedQueryBatchFolderMovesLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO saved_query (resource_id, creator, project, name, statement, folder) VALUES
			('saved-query-a', 'user@example.com', 'default', 'A', '', 'start'),
			('saved-query-b', 'user@example.com', 'default', 'B', '', 'start');
	`

	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
	const barrierID = 9939
	barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
	installMaintenanceLockBarrier(t, fixture.db, barrierID,
		"AFTER UPDATE OF folder ON saved_query FOR EACH ROW")

	// Two overlapping batches with intersecting, reverse-ordered targets:
	// both lock in full primary-key order, so the second serializes behind
	// the first instead of deadlocking.
	firstResult := make(chan error, 1)
	go func() {
		_, err := fixture.store.MoveSavedQueries(fixture.ctx, "default", "user@example.com",
			[]string{"saved-query-a", "saved-query-b"}, "one")
		firstResult <- err
	}()
	waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
	firstPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

	secondResult := make(chan error, 1)
	go func() {
		_, err := fixture.store.MoveSavedQueries(fixture.ctx, "default", "user@example.com",
			[]string{"saved-query-b", "saved-query-a"}, "two")
		secondResult <- err
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, firstPID)
	barrier.release(t)

	requireMaintenanceResult(t, firstResult)
	requireMaintenanceResult(t, secondResult)

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
	require.Equal(t, []string{"two", "two"}, folders)
}

// TestCreateSavedQueryRejectsInactiveProject covers the shared fence's outcome
// on this path: the API layer checks the parent before calling, so the fence
// only fires when the project is archived or purged in between, and the caller
// has to be able to tell that from a genuine failure.
func TestCreateSavedQueryRejectsInactiveProject(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, "")

	// project-a is seeded soft-deleted.
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

	// An active parent is unaffected.
	created, err := fixture.store.CreateSavedQuery(fixture.ctx, &store.SavedQueryMessage{
		ProjectID: "default",
		Creator:   "user@example.com",
		Title:     "into an active project",
		Statement: "SELECT 1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ResourceID)
}

// SetSavedQueryBindings updates one existing row and touches no child table, so
// it cannot form a wait-for cycle with the purge by itself. What it can do is
// race project deletion, and both orders must reach a terminal outcome: either
// the policy lands and the purge then removes the row, or the purge wins and the
// write reports the saved query as gone rather than failing on the FK.
func TestSavedQuerySetBindingsAndDeleteProjectLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO service_account (name, email, workspace, service_key_hash, project)
			VALUES ('service account', 'sa@example.com', 'default', 'unused', 'project-a');
		INSERT INTO saved_query (resource_id, creator, project, name, statement)
			VALUES ('saved-query-a', 'sa@example.com', 'project-a', 'Saved Query A', '');
		INSERT INTO saved_query_star (saved_query, principal) VALUES
			('saved-query-a', 'user-1@example.com');
	`

	bindings := []*storepb.SavedQueryBinding{{
		Level:   storepb.SavedQueryBinding_VIEWER,
		Members: []string{"user:grantee@example.com"},
	}}

	requireGone := func(t *testing.T, fixture *projectDeletionLockOrderFixture) {
		t.Helper()
		var count int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT COUNT(*) FROM saved_query WHERE resource_id = 'saved-query-a'").Scan(&count))
		require.Zero(t, count, "the purge owns the terminal state in both orders")
		require.Zero(t, savedQueryStarCount(t, fixture))
	}

	emptyEtag, err := store.SavedQueryPolicyEtag(nil)
	require.NoError(t, err)

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9951
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		// The barrier has to fire where the purge actually holds the
		// saved_query row: pausing it after the star delete leaves that row
		// unlocked, so the policy write would sail past and prove nothing.
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON saved_query FOR EACH ROW")

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		purgePID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		setResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", bindings, emptyEtag)
			setResult <- err
		}()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, purgePID)
		barrier.release(t)

		requireMaintenanceResult(t, purgeResult)
		// The row is gone by the time the lock is granted, which the store
		// reports as "not applied" (a nil error) rather than an FK failure.
		requireMaintenanceResult(t, setResult)
		requireGone(t, fixture)
	})

	t.Run("policy write first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9952
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF bindings ON saved_query FOR EACH ROW")

		setResult := make(chan error, 1)
		go func() {
			_, err := fixture.store.SetSavedQueryBindings(fixture.ctx, "project-a", "saved-query-a", bindings, emptyEtag)
			setResult <- err
		}()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		setPID := maintenanceBarrierWaitingPID(fixture.ctx, t, fixture.db, barrierID)

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, setPID)
		barrier.release(t)

		requireMaintenanceResult(t, setResult)
		requireMaintenanceResult(t, purgeResult)
		requireGone(t, fixture)
	})
}

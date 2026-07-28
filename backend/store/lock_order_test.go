package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

type projectDeletionLockOrderFixture struct {
	ctx   context.Context
	db    *sql.DB
	store *store.Store
}

func newProjectDeletionLockOrderFixture(t *testing.T, seedSQL string) *projectDeletionLockOrderFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES ('project-a', 'default', 'Project A', TRUE);
	`+seedSQL)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	return &projectDeletionLockOrderFixture{ctx: ctx, db: db, store: s}
}

func runWithConcurrentProjectDeletion(
	t *testing.T,
	seedSQL string,
	blockedTable string,
	advisoryLockID int,
	operation func(context.Context, *store.Store) error,
) error {
	t.Helper()
	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
	ctx, db, s := fixture.ctx, fixture.db, fixture.store

	lockConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer lockConn.Close()
	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased := false
	defer func() {
		if !lockReleased {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID)
		}
	}()
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		CREATE FUNCTION block_%[1]s_delete() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(%[2]d);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_%[1]s_delete
		AFTER DELETE ON %[1]s
		FOR EACH ROW EXECUTE FUNCTION block_%[1]s_delete();
	`, blockedTable, advisoryLockID))
	require.NoError(t, err)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- s.DeleteProject(ctx, "default", "project-a")
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "project deletion should reach the blocked %s delete", blockedTable)

	operationResult := make(chan error, 1)
	go func() {
		operationResult <- operation(ctx, s)
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			WITH delete_backend AS (
				SELECT pid
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
				LIMIT 1
			)
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity, delete_backend
				WHERE delete_backend.pid = ANY(pg_blocking_pids(activity.pid))
			)
		`, advisoryLockID).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "the competing operation should wait behind project deletion")

	_, err = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	require.NoError(t, err)
	lockReleased = true
	deleteErr := <-deleteResult
	operationErr := <-operationResult
	require.NoError(t, deleteErr)
	return operationErr
}

func runWithCreationBeforeProjectDeletion(
	t *testing.T,
	seedSQL string,
	lockedChildTable string,
	operation func(context.Context, *store.Store) error,
) (error, error) {
	t.Helper()
	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
	ctx, db, s := fixture.ctx, fixture.db, fixture.store

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
	var lockedProjectID string
	require.NoError(t, lockTx.QueryRowContext(ctx, `
		SELECT resource_id
		FROM project
		WHERE resource_id = 'project-a'
		FOR UPDATE
	`).Scan(&lockedProjectID))
	require.Equal(t, "project-a", lockedProjectID)

	operationResult := make(chan error, 1)
	go func() {
		operationResult <- operation(ctx, s)
	}()

	var operationBackendPID int
	require.Eventually(t, func() bool {
		return db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT pid
				FROM pg_stat_activity
				WHERE $1 = ANY(pg_blocking_pids(pid))
				ORDER BY pid
				LIMIT 1
			), 0)
		`, lockBackendPID).Scan(&operationBackendPID) == nil && operationBackendPID != 0
	}, 10*time.Second, 10*time.Millisecond, "creation should lock %s, then wait for the project row", lockedChildTable)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- s.DeleteProject(ctx, "default", "project-a")
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity
				WHERE $1 = ANY(pg_blocking_pids(pid))
			)
		`, operationBackendPID).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "project deletion should pass empty child cleanup, then wait for the locked %s", lockedChildTable)

	require.NoError(t, lockTx.Commit())
	lockReleased = true
	return <-operationResult, <-deleteResult
}

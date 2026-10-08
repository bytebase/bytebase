package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

type storePostgresFixture struct {
	ctx   context.Context
	db    *sql.DB
	store *store.Store
	// pgURL lets a test open a second store against the same database, e.g. one
	// with the caches enabled.
	pgURL string
}

func newStorePostgresFixture(t *testing.T, seedSQL string) *storePostgresFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, s, url := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('default', 'default', 'Default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES ('project-a', 'default', 'Project A', TRUE);
	`+seedSQL)
	require.NoError(t, err)

	return &storePostgresFixture{ctx: ctx, db: db, store: s, pgURL: url}
}

const maintenanceLockWait = 10 * time.Second

type maintenanceLockBarrier struct {
	ctx  context.Context
	conn *sql.Conn
	id   int
}

func newMaintenanceLockBarrier(ctx context.Context, t *testing.T, db *sql.DB, id int) *maintenanceLockBarrier {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", id)
	require.NoError(t, err)
	barrier := &maintenanceLockBarrier{ctx: ctx, conn: conn, id: id}
	t.Cleanup(func() { barrier.release(t) })
	return barrier
}

func (b *maintenanceLockBarrier) release(t *testing.T) {
	t.Helper()
	if b.conn == nil {
		return
	}
	_, err := b.conn.ExecContext(b.ctx, "SELECT pg_advisory_unlock($1)", b.id)
	require.NoError(t, err)
	require.NoError(t, b.conn.Close())
	b.conn = nil
}

func installMaintenanceLockBarrier(t *testing.T, db *sql.DB, id int, trigger string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf(`
		CREATE FUNCTION maintenance_lock_barrier_%[1]d() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(%[1]d);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER maintenance_lock_barrier_%[1]d
		%[2]s EXECUTE FUNCTION maintenance_lock_barrier_%[1]d();
	`, id, trigger))
	require.NoError(t, err)
}

func waitForMaintenanceBarrier(ctx context.Context, t *testing.T, db *sql.DB, id int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND objid = $1 AND NOT granted
		)`, id).Scan(&waiting)
		return err == nil && waiting
	}, maintenanceLockWait, 10*time.Millisecond)
}

func maintenanceBarrierWaitingPID(ctx context.Context, t *testing.T, db *sql.DB, id int) int {
	t.Helper()
	var pid int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT pid FROM pg_locks WHERE locktype = 'advisory' AND objid = $1 AND NOT granted LIMIT 1
	`, id).Scan(&pid))
	return pid
}

func waitForBackendBlockedByPID(ctx context.Context, t *testing.T, db *sql.DB, pid int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid))
		)`, pid).Scan(&waiting)
		return err == nil && waiting
	}, maintenanceLockWait, 10*time.Millisecond)
}

func requireMaintenanceResult(t *testing.T, result <-chan error) {
	t.Helper()
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(maintenanceLockWait):
		t.Fatal("timed out waiting for maintenance operation")
	}
}

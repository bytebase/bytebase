package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func waitForBackendBlockedBy(ctx context.Context, t *testing.T, db *sql.DB, blockingPID, waitingPID int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked bool
		err := db.QueryRowContext(ctx, `
			SELECT $1 = ANY(pg_blocking_pids($2))
		`, blockingPID, waitingPID).Scan(&blocked)
		return err == nil && blocked
	}, maintenanceLockWait, 10*time.Millisecond, "backend %d should wait for backend %d", waitingPID, blockingPID)
}

func TestDeleteInstanceDatabaseLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, deleted) VALUES ('instance-a', 'default', TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO policy (workspace, resource_type, resource, type)
			VALUES ('default', 'INSTANCE', 'instances/instance-a', 'TAG');
	`
	testPurgeDatabaseLockOrder(t, seedSQL, func(fixture *projectDeletionLockOrderFixture) error {
		return fixture.store.DeleteInstance(fixture.ctx, "default", "instance-a")
	})
}

func TestDeleteProjectDatabaseLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project, deleted)
			VALUES ('instance-a', 'default', 'project-a', TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO policy (workspace, resource_type, resource, type)
			VALUES ('default', 'INSTANCE', 'projects/project-a/instances/instance-a', 'TAG');
	`
	testPurgeDatabaseLockOrder(t, seedSQL, func(fixture *projectDeletionLockOrderFixture) error {
		return fixture.store.DeleteProject(fixture.ctx, "default", "project-a")
	})
}

func testPurgeDatabaseLockOrder(
	t *testing.T,
	seedSQL string,
	purge func(*projectDeletionLockOrderFixture) error,
) {
	t.Run("policy writer first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		writerTx, err := fixture.db.BeginTx(fixture.ctx, nil)
		require.NoError(t, err)
		defer writerTx.Rollback()

		var writerPID int
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&writerPID))
		var policyResource string
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, `
			SELECT resource FROM policy
			WHERE workspace = 'default' AND type = 'TAG'
			FOR UPDATE
		`).Scan(&policyResource))

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- purge(fixture) }()
		require.Eventually(t, func() bool {
			var purgePID int
			err := fixture.db.QueryRowContext(fixture.ctx, `
				SELECT COALESCE((
					SELECT pid FROM pg_stat_activity
					WHERE $1 = ANY(pg_blocking_pids(pid))
					ORDER BY pid LIMIT 1
				), 0)
			`, writerPID).Scan(&purgePID)
			return err == nil && purgePID != 0
		}, maintenanceLockWait, 10*time.Millisecond, "purge should wait for the policy writer")

		var name string
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, `
			SELECT name FROM db
			WHERE instance = 'instance-a' AND name = 'db-a'
			FOR UPDATE
		`).Scan(&name))

		var instanceID string
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, `
			SELECT resource_id FROM instance WHERE resource_id = 'instance-a' FOR UPDATE
		`).Scan(&instanceID))
		var projectID string
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, `
			SELECT resource_id FROM project WHERE resource_id = 'project-a' FOR UPDATE
		`).Scan(&projectID))
		require.NoError(t, writerTx.Commit())
		requireMaintenanceResult(t, purgeResult)

		var count int
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx, "SELECT COUNT(*) FROM instance").Scan(&count))
		require.Zero(t, count)
	})

	t.Run("purge locks database before instance", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		instanceTx, err := fixture.db.BeginTx(fixture.ctx, nil)
		require.NoError(t, err)
		defer instanceTx.Rollback()

		var instanceLockPID int
		require.NoError(t, instanceTx.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&instanceLockPID))
		var instanceID string
		require.NoError(t, instanceTx.QueryRowContext(fixture.ctx, `
			SELECT resource_id FROM instance WHERE resource_id = 'instance-a' FOR UPDATE
		`).Scan(&instanceID))

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- purge(fixture) }()
		var purgePID int
		require.Eventually(t, func() bool {
			return fixture.db.QueryRowContext(fixture.ctx, `
				SELECT COALESCE((
					SELECT pid FROM pg_stat_activity
					WHERE $1 = ANY(pg_blocking_pids(pid))
					ORDER BY pid LIMIT 1
				), 0)
			`, instanceLockPID).Scan(&purgePID) == nil && purgePID != 0
		}, maintenanceLockWait, 10*time.Millisecond, "purge should wait for the instance lock")

		writerConn, err := fixture.db.Conn(fixture.ctx)
		require.NoError(t, err)
		defer writerConn.Close()
		writerTx, err := writerConn.BeginTx(fixture.ctx, nil)
		require.NoError(t, err)
		defer writerTx.Rollback()
		var writerPID int
		require.NoError(t, writerTx.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&writerPID))
		writerResult := make(chan error, 1)
		go func() {
			var name string
			writerResult <- writerTx.QueryRowContext(fixture.ctx, `
				SELECT name FROM db
				WHERE instance = 'instance-a' AND name = 'db-a'
				FOR UPDATE
			`).Scan(&name)
		}()
		waitForBackendBlockedBy(fixture.ctx, t, fixture.db, purgePID, writerPID)

		require.NoError(t, instanceTx.Commit())
		requireMaintenanceResult(t, purgeResult)
		require.ErrorIs(t, <-writerResult, sql.ErrNoRows)
	})

	t.Run("purge locks policy before database", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		databaseTx, err := fixture.db.BeginTx(fixture.ctx, nil)
		require.NoError(t, err)
		defer databaseTx.Rollback()

		var databasePID int
		require.NoError(t, databaseTx.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&databasePID))
		var databaseName string
		require.NoError(t, databaseTx.QueryRowContext(fixture.ctx, `
			SELECT name FROM db
			WHERE instance = 'instance-a' AND name = 'db-a'
			FOR UPDATE
		`).Scan(&databaseName))

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- purge(fixture) }()
		var purgePID int
		require.Eventually(t, func() bool {
			return fixture.db.QueryRowContext(fixture.ctx, `
				SELECT COALESCE((
					SELECT pid FROM pg_stat_activity
					WHERE $1 = ANY(pg_blocking_pids(pid))
					ORDER BY pid LIMIT 1
				), 0)
			`, databasePID).Scan(&purgePID) == nil && purgePID != 0
		}, maintenanceLockWait, 10*time.Millisecond, "purge should wait for the database lock")

		policyConn, err := fixture.db.Conn(fixture.ctx)
		require.NoError(t, err)
		defer policyConn.Close()
		policyTx, err := policyConn.BeginTx(fixture.ctx, nil)
		require.NoError(t, err)
		defer policyTx.Rollback()
		var policyPID int
		require.NoError(t, policyTx.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&policyPID))
		policyResult := make(chan error, 1)
		go func() {
			var resource string
			policyResult <- policyTx.QueryRowContext(fixture.ctx, `
				SELECT resource FROM policy
				WHERE workspace = 'default' AND type = 'TAG'
				FOR UPDATE
			`).Scan(&resource)
		}()
		waitForBackendBlockedBy(fixture.ctx, t, fixture.db, purgePID, policyPID)

		require.NoError(t, databaseTx.Commit())
		requireMaintenanceResult(t, purgeResult)
		require.ErrorIs(t, <-policyResult, sql.ErrNoRows)
	})
}

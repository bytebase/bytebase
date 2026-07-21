package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
	t.Cleanup(func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", id)
		require.NoError(t, conn.Close())
	})
	return barrier
}

func (b *maintenanceLockBarrier) release(t *testing.T) {
	t.Helper()
	_, err := b.conn.ExecContext(b.ctx, "SELECT pg_advisory_unlock($1)", b.id)
	require.NoError(t, err)
}

func installMaintenanceLockBarrier(t *testing.T, db *sql.DB, id int, trigger string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), fmt.Sprintf(`
		CREATE FUNCTION maintenance_lock_barrier_%[1]d() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(%[1]d);
			RETURN NULL;
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
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
			)
		`, id).Scan(&waiting)
		return err == nil && waiting
	}, maintenanceLockWait, 10*time.Millisecond, "operation should reach the advisory barrier")
}

func waitForOperationBehindMaintenanceBarrier(ctx context.Context, t *testing.T, db *sql.DB, id int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRowContext(ctx, `
			WITH barrier_backend AS (
				SELECT pid
				FROM pg_locks
				WHERE locktype = 'advisory'
					AND objid = $1
					AND NOT granted
				LIMIT 1
			)
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity, barrier_backend
				WHERE barrier_backend.pid = ANY(pg_blocking_pids(activity.pid))
			)
		`, id).Scan(&waiting)
		return err == nil && waiting
	}, maintenanceLockWait, 10*time.Millisecond, "competing operation should wait behind the barrier owner")
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

func requireProjectInstanceDeletionState(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"project", "instance", "db", "revision", "service_account"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count))
		if table == "project" {
			require.Equal(t, 1, count, "only the default project should remain")
		} else {
			require.Zero(t, count, "%s should be removed", table)
		}
	}

	var project string
	var instance, database sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT project, instance, db_name
		FROM worksheet
		WHERE resource_id = 'worksheet-a'
	`).Scan(&project, &instance, &database))
	require.Equal(t, "default", project)
	require.False(t, instance.Valid)
	require.False(t, database.Valid)
}

func requireProjectUserDeletionState(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	var projectCount, planCount, queryHistoryCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM project").Scan(&projectCount))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM plan").Scan(&planCount))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM query_history").Scan(&queryHistoryCount))
	require.Equal(t, 1, projectCount, "only the default project should remain")
	require.Zero(t, planCount)
	require.Zero(t, queryHistoryCount)

	var email string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT email FROM principal WHERE id = 101").Scan(&email))
	require.Equal(t, "renamed@example.com", email)
}

func requireUserInstanceDeletionState(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"instance", "db", "query_history", "task_run", "task"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count))
		require.Zero(t, count, "%s should be removed", table)
	}

	var email string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT email FROM principal WHERE id = 101").Scan(&email))
	require.Equal(t, "renamed@example.com", email)
}

func TestDeleteProjectAndDeleteInstanceLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, deleted) VALUES ('instance-a', 'default', TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO service_account (name, email, workspace, service_key_hash, project)
			VALUES ('service account', 'service-account@example.com', 'default', 'unused', 'project-a');
		INSERT INTO revision (resource_id, instance, db_name, deleter, version)
			VALUES ('revision-a', 'instance-a', 'db-a', 'service-account@example.com', 'v1');
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'user@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO worksheet (resource_id, creator, project, instance, db_name, name, statement, visibility)
			VALUES ('worksheet-a', 'user@example.com', 'project-a', 'instance-a', 'db-a', 'Worksheet A', '', 'PROJECT_READ');
	`

	t.Run("delete project first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9921
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER UPDATE OF project ON worksheet FOR EACH ROW")

		projectResult := make(chan error, 1)
		go func() { projectResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		instanceResult := make(chan error, 1)
		go func() { instanceResult <- fixture.store.DeleteInstance(fixture.ctx, "default", "instance-a") }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, projectResult)
		requireMaintenanceResult(t, instanceResult)
		requireProjectInstanceDeletionState(fixture.ctx, t, fixture.db)
	})

	t.Run("delete instance first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9922
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID,
			"AFTER DELETE ON task FOR EACH ROW")

		instanceResult := make(chan error, 1)
		go func() { instanceResult <- fixture.store.DeleteInstance(fixture.ctx, "default", "instance-a") }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		projectResult := make(chan error, 1)
		go func() { projectResult <- fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, instanceResult)
		requireMaintenanceResult(t, projectResult)
		requireProjectInstanceDeletionState(fixture.ctx, t, fixture.db)
	})
}

func TestUpdateUserEmailAndDeleteProjectLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO principal (name, email, password_hash) VALUES ('User A', 'user@example.com', 'unused');
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'user@example.com', 'project-a', 'Plan A', '');
		INSERT INTO query_history (resource_id, creator, project, database, statement, type)
			VALUES ('query-history-a', 'user@example.com', 'project-a', 'instances/instance-a/databases/db-a', '', 'QUERY');
	`

	run := func(t *testing.T, first string) {
		t.Helper()
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		user, err := fixture.store.GetUserByEmail(fixture.ctx, "user@example.com")
		require.NoError(t, err)
		require.NotNil(t, user)

		barrierID := 9923
		trigger := "AFTER DELETE ON query_history FOR EACH ROW"
		if first == "user" {
			barrierID = 9924
			trigger = "AFTER UPDATE OF creator ON plan FOR EACH ROW"
		}
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, trigger)

		updateUser := func() error {
			_, err := fixture.store.UpdateUserEmail(fixture.ctx, user, "renamed@example.com")
			return err
		}
		deleteProject := func() error { return fixture.store.DeleteProject(fixture.ctx, "default", "project-a") }

		var firstResult, secondResult chan error
		if first == "project" {
			firstResult, secondResult = make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- deleteProject() }()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
			go func() { secondResult <- updateUser() }()
		} else {
			firstResult, secondResult = make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- updateUser() }()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
			go func() { secondResult <- deleteProject() }()
		}
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, firstResult)
		requireMaintenanceResult(t, secondResult)
		requireProjectUserDeletionState(fixture.ctx, t, fixture.db)
	}

	t.Run("delete project first", func(t *testing.T) { run(t, "project") })
	t.Run("update user first", func(t *testing.T) { run(t, "user") })
}

func TestUpdateUserEmailAndDeleteInstanceLockOrder(t *testing.T) {
	const seedSQL = `
		INSERT INTO principal (name, email, password_hash) VALUES ('User A', 'user@example.com', 'unused');
		INSERT INTO instance (resource_id, workspace, deleted) VALUES ('instance-a', 'default', TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'project-a');
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'user@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO task_run (id, creator, project, task_id, attempt, status)
			VALUES (101, 'user@example.com', 'project-a', 101, 0, 'DONE');
		INSERT INTO query_history (resource_id, creator, project, database, statement, type)
			VALUES ('query-history-a', 'user@example.com', 'project-a', 'instances/instance-a/databases/db-a', '', 'QUERY');
	`

	run := func(t *testing.T, first string) {
		t.Helper()
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		user, err := fixture.store.GetUserByEmail(fixture.ctx, "user@example.com")
		require.NoError(t, err)
		require.NotNil(t, user)

		barrierID := 9925
		trigger := "AFTER DELETE ON query_history FOR EACH ROW"
		if first == "user" {
			barrierID = 9926
			trigger = "AFTER UPDATE OF creator ON task_run FOR EACH ROW"
		}
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, trigger)

		updateUser := func() error {
			_, err := fixture.store.UpdateUserEmail(fixture.ctx, user, "renamed@example.com")
			return err
		}
		deleteInstance := func() error { return fixture.store.DeleteInstance(fixture.ctx, "default", "instance-a") }

		var firstResult, secondResult chan error
		if first == "instance" {
			firstResult, secondResult = make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- deleteInstance() }()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
			go func() { secondResult <- updateUser() }()
		} else {
			firstResult, secondResult = make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- updateUser() }()
			waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
			go func() { secondResult <- deleteInstance() }()
		}
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, firstResult)
		requireMaintenanceResult(t, secondResult)
		requireUserInstanceDeletionState(fixture.ctx, t, fixture.db)
	}

	t.Run("delete instance first", func(t *testing.T) { run(t, "instance") })
	t.Run("update user first", func(t *testing.T) { run(t, "user") })
}

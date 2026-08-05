package store_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func batchMoveDatabaseIntoProjectA(fixture *projectDeletionLockOrderFixture) error {
	projectID := "project-a"
	return fixture.store.BatchUpdateDatabases(fixture.ctx, []*store.DatabaseMessage{{
		InstanceID:   "instance-a",
		DatabaseName: "db-a",
	}}, &store.BatchUpdateDatabases{
		Workspace: "default",
		ProjectID: &projectID,
	})
}

func requireProjectPurgeTerminalState(t *testing.T, fixture *projectDeletionLockOrderFixture) {
	t.Helper()
	var projectCount, orphanCount, databaseCount int
	var project string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM project WHERE resource_id = 'project-a'").Scan(&projectCount))
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM db WHERE project = 'project-a'").Scan(&orphanCount))
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT COUNT(*) FROM db WHERE instance = 'instance-a' AND name = 'db-a'").Scan(&databaseCount))
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT project FROM db WHERE instance = 'instance-a' AND name = 'db-a'").Scan(&project))
	require.Zero(t, projectCount, "project purge must remove the archived project")
	require.Zero(t, orphanCount, "no database row may reference the purged project")
	require.Equal(t, 1, databaseCount, "the workspace-instance database must survive")
	require.Equal(t, "default", project, "the workspace-instance database must be reassigned to the default project")
}

func TestBatchUpdateDatabasesAndDeleteProjectSerialize(t *testing.T) {
	// The batch moves a workspace-instance database into the archived project
	// while that project is being purged. Without purge fencing the terminal
	// project deletion fails on db.project's foreign key.
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('instance-a', 'default', NULL);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'default');
	`
	purgeProjectA := func(fixture *projectDeletionLockOrderFixture) error {
		return fixture.store.DeleteProject(fixture.ctx, "default", "project-a")
	}

	t.Run("purge first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9941
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		installMaintenanceLockBarrier(t, fixture.db, barrierID, "AFTER DELETE ON project FOR EACH ROW")

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- purgeProjectA(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		batchResult := make(chan error, 1)
		go func() { batchResult <- batchMoveDatabaseIntoProjectA(fixture) }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, purgeResult)
		err := <-batchResult
		require.Error(t, err)
		require.NotContains(t, strings.ToLower(err.Error()), "foreign key",
			"the batch must reject the purged destination cleanly, not with an FK failure")
		require.ErrorContains(t, err, "project-a not found",
			"the batch must fail on the purged destination project")
		requireProjectPurgeTerminalState(t, fixture)
	})

	t.Run("batch first", func(t *testing.T) {
		fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
		const barrierID = 9942
		barrier := newMaintenanceLockBarrier(fixture.ctx, t, fixture.db, barrierID)
		_, err := fixture.db.ExecContext(fixture.ctx, `
			CREATE FUNCTION block_db_project_update_9942() RETURNS trigger AS $$
			BEGIN
				PERFORM pg_advisory_xact_lock(9942);
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			CREATE TRIGGER block_db_project_update_9942
			BEFORE UPDATE OF project ON db FOR EACH ROW
			WHEN (OLD.project IS DISTINCT FROM NEW.project)
			EXECUTE FUNCTION block_db_project_update_9942();
		`)
		require.NoError(t, err)

		batchResult := make(chan error, 1)
		go func() { batchResult <- batchMoveDatabaseIntoProjectA(fixture) }()
		waitForMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)

		purgeResult := make(chan error, 1)
		go func() { purgeResult <- purgeProjectA(fixture) }()
		waitForOperationBehindMaintenanceBarrier(fixture.ctx, t, fixture.db, barrierID)
		barrier.release(t)

		requireMaintenanceResult(t, batchResult)
		requireMaintenanceResult(t, purgeResult)
		requireProjectPurgeTerminalState(t, fixture)
	})
}

func TestBatchUpdateDatabasesRejectsArchivedInstance(t *testing.T) {
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project, deleted)
			VALUES ('instance-a', 'default', NULL, TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'default');
	`
	fixture := newProjectDeletionLockOrderFixture(t, seedSQL)
	err := batchMoveDatabaseIntoProjectA(fixture)
	require.Error(t, err)
	require.ErrorContains(t, err, "instance instance-a is archived")

	var projectID string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT project FROM db WHERE instance = 'instance-a' AND name = 'db-a'").Scan(&projectID))
	require.Equal(t, "default", projectID)
}

func TestBatchUpdateDatabasesClearsEnvironmentOnArchivedInstance(t *testing.T) {
	environmentID, unset := "env-a", ""
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace, project, deleted)
			VALUES ('instance-a', 'default', NULL, TRUE);
		INSERT INTO db (instance, name, project, environment)
			VALUES ('instance-a', 'db-a', 'default', 'env-a');
	`)

	err := fixture.store.BatchUpdateDatabases(fixture.ctx, nil, &store.BatchUpdateDatabases{
		Workspace:           "default",
		FindByEnvironmentID: &environmentID,
		EnvironmentID:       &unset,
	})
	require.NoError(t, err, "environment cleanup must remain possible after an instance is archived")

	var environment sql.NullString
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT environment FROM db WHERE instance = 'instance-a' AND name = 'db-a'").Scan(&environment))
	require.False(t, environment.Valid, "the archived instance must not retain a deleted environment reference")
}

func TestBatchUpdateDatabasesByEnvironment(t *testing.T) {
	envA, unset := "env-a", ""
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO workspace (resource_id) VALUES ('other');
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('instance-a', 'default', NULL),
			('instance-b', 'other', NULL);
		INSERT INTO db (instance, name, project, environment) VALUES
			('instance-a', 'db-a', 'default', 'env-a'),
			('instance-a', 'db-b', 'default', 'env-b'),
			('instance-b', 'db-c', 'default', 'env-a');
	`)

	err := fixture.store.BatchUpdateDatabases(fixture.ctx, nil, &store.BatchUpdateDatabases{
		Workspace:           "default",
		FindByEnvironmentID: &envA,
		EnvironmentID:       &unset,
	})
	require.NoError(t, err)

	requireDatabaseEnvironment := func(instanceID, databaseName string, want string) {
		t.Helper()
		var environment sql.NullString
		require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
			"SELECT environment FROM db WHERE instance = $1 AND name = $2", instanceID, databaseName).Scan(&environment))
		if want == "" {
			require.False(t, environment.Valid, "database %s/%s environment should be unset", instanceID, databaseName)
		} else {
			require.True(t, environment.Valid)
			require.Equal(t, want, environment.String)
		}
	}
	requireDatabaseEnvironment("instance-a", "db-a", "")
	requireDatabaseEnvironment("instance-a", "db-b", "env-b")
	requireDatabaseEnvironment("instance-b", "db-c", "env-a")
}

func TestBatchUpdateDatabasesByEnvironmentRevalidatesDynamicTargets(t *testing.T) {
	environmentID, unset := "env-a", ""
	fixture := newProjectDeletionLockOrderFixture(t, `
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('instance-a', 'default', NULL);
		INSERT INTO db (instance, name, project, environment) VALUES
			('instance-a', 'db-a', 'default', 'env-a'),
			('instance-a', 'db-b', 'default', 'env-b');
	`)

	// Block the batch on its default-project purge fence. It reaches that fence
	// only after the outside-transaction target read has completed.
	conn, err := fixture.db.Conn(fixture.ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	_, err = conn.ExecContext(fixture.ctx, "SELECT pg_advisory_lock($1, hashtext($2))", int64(store.AdvisoryLockKeyProjectPurge), "default")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := conn.ExecContext(fixture.ctx, "SELECT pg_advisory_unlock($1, hashtext($2))", int64(store.AdvisoryLockKeyProjectPurge), "default")
		require.NoError(t, err)
	})
	var barrierPID int
	require.NoError(t, conn.QueryRowContext(fixture.ctx, "SELECT pg_backend_pid()").Scan(&barrierPID))

	result := make(chan error, 1)
	go func() {
		result <- fixture.store.BatchUpdateDatabases(fixture.ctx, nil, &store.BatchUpdateDatabases{
			Workspace:           "default",
			FindByEnvironmentID: &environmentID,
			EnvironmentID:       &unset,
		})
	}()
	waitForBackendBlockedByPID(fixture.ctx, t, fixture.db, barrierPID)

	// db-b was not in the pre-read target set, but becomes a match before the
	// fenced transaction scans. The batch must either include it or explicitly
	// request a retry; it must not report success and silently leave it behind.
	_, err = fixture.db.ExecContext(fixture.ctx,
		"UPDATE db SET environment = 'env-a' WHERE instance = 'instance-a' AND name = 'db-b'")
	require.NoError(t, err)
	_, err = conn.ExecContext(fixture.ctx, "SELECT pg_advisory_unlock($1, hashtext($2))", int64(store.AdvisoryLockKeyProjectPurge), "default")
	require.NoError(t, err)

	err = <-result
	if err != nil {
		require.NotContains(t, strings.ToLower(err.Error()), "foreign key")
		require.Contains(t, strings.ToLower(err.Error()), "retry")
		return
	}

	var environment sql.NullString
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT environment FROM db WHERE instance = 'instance-a' AND name = 'db-b'").Scan(&environment))
	require.False(t, environment.Valid, "a matching database added during the fence wait must not be skipped")
}

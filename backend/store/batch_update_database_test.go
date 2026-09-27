package store_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func batchMoveDatabaseIntoProjectA(fixture *storePostgresFixture) error {
	projectID := "project-a"
	return fixture.store.BatchUpdateDatabases(fixture.ctx, []*store.DatabaseMessage{{
		InstanceID:   "instance-a",
		DatabaseName: "db-a",
	}}, &store.BatchUpdateDatabases{
		Workspace: "default",
		ProjectID: &projectID,
	})
}

func TestBatchUpdateDatabasesRejectsArchivedInstance(t *testing.T) {
	t.Parallel()
	const seedSQL = `
		INSERT INTO instance (resource_id, workspace, project, deleted)
			VALUES ('instance-a', 'default', NULL, TRUE);
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db-a', 'default');
	`
	fixture := newStorePostgresFixture(t, seedSQL)
	err := batchMoveDatabaseIntoProjectA(fixture)
	require.Error(t, err)
	require.ErrorContains(t, err, "instance instance-a is archived")

	var projectID string
	require.NoError(t, fixture.db.QueryRowContext(fixture.ctx,
		"SELECT project FROM db WHERE instance = 'instance-a' AND name = 'db-a'").Scan(&projectID))
	require.Equal(t, "default", projectID)
}

func TestBatchUpdateDatabasesClearsEnvironmentOnArchivedInstance(t *testing.T) {
	t.Parallel()
	environmentID, unset := "env-a", ""
	fixture := newStorePostgresFixture(t, `
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
	t.Parallel()
	envA, unset := "env-a", ""
	fixture := newStorePostgresFixture(t, `
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

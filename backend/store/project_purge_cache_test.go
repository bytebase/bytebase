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

// newProjectPurgeCacheFixture boots a real PostgreSQL with caches ENABLED and
// seeds a workspace, the default project, a deleted purgeable project, and an
// active project for ID-reuse scenarios.
func newProjectPurgeCacheFixture(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES
			('default', 'default', 'Default', FALSE),
			('project-a', 'default', 'Project A', TRUE),
			('project-b', 'default', 'Project B', FALSE);
	`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return ctx, db, s
}

// seedProjectPurgeFixture creates a project instance and a workspace instance
// owned by project-a, a database on each, and schema rows for both databases.
func seedProjectPurgeFixture(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-a'),
			('workspace-instance', 'default', NULL);
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'project-db', 'project-a'),
			('workspace-instance', 'workspace-db', 'project-a');
		INSERT INTO db_schema (instance, db_name) VALUES
			('project-instance', 'project-db'),
			('workspace-instance', 'workspace-db');
	`)
	require.NoError(t, err)
}

// warmProjectPurgeCaches populates the instance, scoped and unscoped database,
// and schema cache entries for every seeded descendant.
func warmProjectPurgeCaches(ctx context.Context, t *testing.T, s *store.Store) {
	t.Helper()
	projectInstanceID := "project-instance"
	projectDBName := "project-db"
	workspaceInstanceID := "workspace-instance"
	workspaceDBName := "workspace-db"
	for _, find := range []*store.FindInstanceMessage{
		{Workspace: "default", ResourceID: &projectInstanceID},
		{Workspace: "default", ResourceID: &workspaceInstanceID},
	} {
		instance, err := s.GetInstance(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, instance)
	}
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
		{Workspace: "", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
		{Workspace: "default", InstanceID: &workspaceInstanceID, DatabaseName: &workspaceDBName},
		{Workspace: "", InstanceID: &workspaceInstanceID, DatabaseName: &workspaceDBName},
	} {
		database, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, database)
	}
	for _, schema := range []struct {
		instanceID   string
		databaseName string
	}{
		{instanceID: projectInstanceID, databaseName: projectDBName},
		{instanceID: workspaceInstanceID, databaseName: workspaceDBName},
	} {
		metadata, err := s.GetDBSchemaSnapshot(ctx, "default", schema.instanceID, schema.databaseName)
		require.NoError(t, err)
		require.NotNil(t, metadata)
	}
}

// TestDeleteProjectPurgeInvalidatesDescendantCaches warms every descendant
// cache entry, purges the project, and requires the purged descendants to be
// immediately unavailable through the cached getters while surviving rows
// still resolve correctly.
func TestDeleteProjectPurgeInvalidatesDescendantCaches(t *testing.T) {
	ctx, db, s := newProjectPurgeCacheFixture(t)
	seedProjectPurgeFixture(ctx, t, db)
	warmProjectPurgeCaches(ctx, t, s)

	projectInstanceID := "project-instance"
	projectDBName := "project-db"
	workspaceInstanceID := "workspace-instance"
	workspaceDBName := "workspace-db"

	require.NoError(t, s.DeleteProject(ctx, "default", "project-a"))

	// Observable seam: purged descendants are immediately unavailable through
	// the cached getters.
	instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{Workspace: "default", ResourceID: &projectInstanceID})
	require.NoError(t, err)
	require.Nil(t, instance)
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
		{Workspace: "", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
	} {
		database, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.Nil(t, database)
	}
	schema, err := s.GetDBSchemaSnapshot(ctx, "default", projectInstanceID, projectDBName)
	require.NoError(t, err)
	require.Nil(t, schema)

	// Surviving rows still resolve: the workspace instance was not flushed and
	// its schema snapshot is still readable.
	workspaceInstance, err := s.GetInstance(ctx, &store.FindInstanceMessage{Workspace: "default", ResourceID: &workspaceInstanceID})
	require.NoError(t, err)
	require.NotNil(t, workspaceInstance)
	workspaceSchema, err := s.GetDBSchemaSnapshot(ctx, "default", workspaceInstanceID, workspaceDBName)
	require.NoError(t, err)
	require.NotNil(t, workspaceSchema)

	// The workspace-instance database moved to the default project is re-read
	// from the database instead of being served from a stale cached entry.
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &workspaceInstanceID, DatabaseName: &workspaceDBName},
		{Workspace: "", InstanceID: &workspaceInstanceID, DatabaseName: &workspaceDBName},
	} {
		movedDB, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, movedDB)
		require.Equal(t, "default", movedDB.ProjectID)
	}
}

// TestDeleteProjectPurgeSupportsDescendantIDReuse proves that reusing a
// purged instance resource ID and database name cannot expose stale cached
// data. Reused rows are inserted directly so a stale cache entry would still
// be observable through the getters.
func TestDeleteProjectPurgeSupportsDescendantIDReuse(t *testing.T) {
	ctx, db, s := newProjectPurgeCacheFixture(t)
	seedProjectPurgeFixture(ctx, t, db)
	warmProjectPurgeCaches(ctx, t, s)

	require.NoError(t, s.DeleteProject(ctx, "default", "project-a"))

	projectInstanceID := "project-instance"
	projectDBName := "project-db"
	_, err := db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-b');
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'project-db', 'project-b');
	`)
	require.NoError(t, err)

	reusedInstance, err := s.GetInstance(ctx, &store.FindInstanceMessage{Workspace: "default", ResourceID: &projectInstanceID})
	require.NoError(t, err)
	require.NotNil(t, reusedInstance)
	require.Equal(t, "project-b", *reusedInstance.ProjectID)
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
		{Workspace: "", InstanceID: &projectInstanceID, DatabaseName: &projectDBName},
	} {
		reusedDB, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, reusedDB)
		require.Equal(t, "project-b", reusedDB.ProjectID)
	}
	// The reused database has no schema row; a stale schema cache entry for the
	// purged database must not leak.
	reusedSchema, err := s.GetDBSchemaSnapshot(ctx, "default", projectInstanceID, projectDBName)
	require.NoError(t, err)
	require.Nil(t, reusedSchema)
}

// TestDeleteProjectFailedTransactionKeepsDescendantCaches pins that a purge
// transaction that deletes descendants but fails before commit publishes no
// invalidation. After the failed purge the underlying rows are removed
// directly, so any surviving getter result can only come from the still-warm
// cache entries.
func TestDeleteProjectFailedTransactionKeepsDescendantCaches(t *testing.T) {
	ctx, db, s := newProjectPurgeCacheFixture(t)
	seedProjectPurgeFixture(ctx, t, db)
	warmProjectPurgeCaches(ctx, t, s)

	projectInstanceID := "project-instance"
	projectDBName := "project-db"
	// project-a is not marked deleted, so DeleteProject removes all descendant
	// rows inside the transaction and then fails on the deleted guard, rolling
	// the whole purge back.
	_, err := db.ExecContext(ctx, `UPDATE project SET deleted = FALSE WHERE resource_id = 'project-a'`)
	require.NoError(t, err)
	err = s.DeleteProject(ctx, "default", "project-a")
	require.Error(t, err)

	// Remove the descendant rows behind the store's back so that only a
	// surviving cache entry can satisfy the getters.
	_, err = db.ExecContext(ctx, `
		DELETE FROM db_schema WHERE instance = 'project-instance' AND db_name = 'project-db';
		DELETE FROM db WHERE instance = 'project-instance' AND name = 'project-db';
		DELETE FROM instance WHERE resource_id = 'project-instance';
	`)
	require.NoError(t, err)

	instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{Workspace: "default", ResourceID: &projectInstanceID})
	require.NoError(t, err)
	require.NotNil(t, instance)
	require.Equal(t, "project-a", *instance.ProjectID)
	database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{Workspace: "default", InstanceID: &projectInstanceID, DatabaseName: &projectDBName})
	require.NoError(t, err)
	require.NotNil(t, database)
	require.Equal(t, "project-a", database.ProjectID)
	schema, err := s.GetDBSchemaSnapshot(ctx, "default", projectInstanceID, projectDBName)
	require.NoError(t, err)
	require.NotNil(t, schema)
}

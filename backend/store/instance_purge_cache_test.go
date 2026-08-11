package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestDeleteInstancePurgeInvalidatesDescendantCaches(t *testing.T) {
	ctx, db, s := newProjectPurgeCacheFixture(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, project, deleted)
			VALUES ('project-instance', 'default', 'project-a', TRUE);
		INSERT INTO db (instance, name, project)
			VALUES ('project-instance', 'project-db', 'project-a');
		INSERT INTO db_schema (instance, db_name)
			VALUES ('project-instance', 'project-db');
	`)
	require.NoError(t, err)

	instanceID := "project-instance"
	databaseName := "project-db"
	instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  "default",
		ResourceID: &instanceID,
	})
	require.NoError(t, err)
	require.NotNil(t, instance)
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &instanceID, DatabaseName: &databaseName, ShowDeleted: true},
		{Workspace: "", InstanceID: &instanceID, DatabaseName: &databaseName, ShowDeleted: true},
	} {
		database, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, database)
	}
	schema, err := s.GetDBSchemaSnapshot(ctx, "default", instanceID, databaseName)
	require.NoError(t, err)
	require.NotNil(t, schema)

	require.NoError(t, s.DeleteInstance(ctx, "default", instanceID))

	instance, err = s.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  "default",
		ResourceID: &instanceID,
	})
	require.NoError(t, err)
	require.Nil(t, instance)
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &instanceID, DatabaseName: &databaseName, ShowDeleted: true},
		{Workspace: "", InstanceID: &instanceID, DatabaseName: &databaseName, ShowDeleted: true},
	} {
		database, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.Nil(t, database)
	}
	schema, err = s.GetDBSchemaSnapshot(ctx, "default", instanceID, databaseName)
	require.NoError(t, err)
	require.Nil(t, schema)

	_, err = db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('project-instance', 'default', 'project-b');
		INSERT INTO db (instance, name, project)
			VALUES ('project-instance', 'project-db', 'project-b');
	`)
	require.NoError(t, err)
	for _, find := range []*store.FindDatabaseMessage{
		{Workspace: "default", InstanceID: &instanceID, DatabaseName: &databaseName},
		{Workspace: "", InstanceID: &instanceID, DatabaseName: &databaseName},
	} {
		database, err := s.GetDatabase(ctx, find)
		require.NoError(t, err)
		require.NotNil(t, database)
		require.Equal(t, "project-b", database.ProjectID)
	}
	schema, err = s.GetDBSchemaSnapshot(ctx, "default", instanceID, databaseName)
	require.NoError(t, err)
	require.Nil(t, schema)
}

package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestDatabaseWritersRespectProjectInstanceOwnership(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-b', 'default', 'Project B'),
			('project-c', 'default', 'Project C');
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-b'),
			('workspace-instance', 'default', NULL);
	`)
	ctx, s := fixture.ctx, fixture.store

	projectB, projectC := "project-b", "project-c"

	// Discovery always inherits the project-instance owner, even though schema
	// sync supplies the default project as its legacy assignment.
	database, err := s.CreateDatabaseDefault(ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
		ProjectID:    "default",
	})
	require.NoError(t, err)
	require.Equal(t, projectB, database.ProjectID)

	database, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "upsert-discovered",
		ProjectID:    projectC,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, projectB, database.ProjectID)

	database, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
		ProjectID:    projectB,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, projectB, database.ProjectID)

	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
		ProjectID:    &projectB,
	})
	require.NoError(t, err)

	err = s.BatchUpdateDatabases(ctx, []*store.DatabaseMessage{{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
	}}, &store.BatchUpdateDatabases{ProjectID: &projectB})
	require.NoError(t, err)

	// Every writer rejects changing a project-instance database's assignment.
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
		ProjectID:    projectC,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.Error(t, err)
	require.Equal(t, projectB, getDatabaseProject(ctx, t, s, "project-instance", "discovered"))

	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
		ProjectID:    &projectC,
	})
	require.Error(t, err)
	require.Equal(t, projectB, getDatabaseProject(ctx, t, s, "project-instance", "discovered"))

	err = s.BatchUpdateDatabases(ctx, []*store.DatabaseMessage{{
		InstanceID:   "project-instance",
		DatabaseName: "discovered",
	}}, &store.BatchUpdateDatabases{ProjectID: &projectC})
	require.Error(t, err)
	require.Equal(t, projectB, getDatabaseProject(ctx, t, s, "project-instance", "discovered"))

	// A workspace instance retains its existing independent database assignment.
	database, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "workspace-instance",
		DatabaseName: "independent",
		ProjectID:    projectB,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, projectB, database.ProjectID)

	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "workspace-instance",
		DatabaseName: "independent",
		ProjectID:    &projectC,
	})
	require.NoError(t, err)
	require.Equal(t, projectC, getDatabaseProject(ctx, t, s, "workspace-instance", "independent"))

	err = s.BatchUpdateDatabases(ctx, []*store.DatabaseMessage{{
		InstanceID:   "workspace-instance",
		DatabaseName: "independent",
	}}, &store.BatchUpdateDatabases{ProjectID: &projectB})
	require.NoError(t, err)
	require.Equal(t, projectB, getDatabaseProject(ctx, t, s, "workspace-instance", "independent"))
}

func getDatabaseProject(ctx context.Context, t *testing.T, s *store.Store, instanceID, databaseName string) string {
	t.Helper()
	database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, database)
	return database.ProjectID
}

func TestProjectInstanceDatabaseUpsertRequiresActiveOwnerForNewDatabase(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name, deleted)
			VALUES ('project-b', 'default', 'Project B', TRUE);
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('project-instance', 'default', 'project-b');
	`)

	_, err := fixture.store.UpsertDatabase(fixture.ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "new-database",
		ProjectID:    "project-b",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.Error(t, err)
}

func TestProjectInstanceDatabaseUpdateRejectsArchivedProject(t *testing.T) {
	fixture := newProjectDeletionLockOrderFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name, deleted)
			VALUES ('project-b', 'default', 'Project B', TRUE);
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('project-instance', 'default', 'project-b');
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'existing-database', 'project-b');
	`)

	deleted := true
	_, err := fixture.store.UpdateDatabase(fixture.ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "existing-database",
		Deleted:      &deleted,
	})
	require.Error(t, err)
	require.Equal(t, "project-b", getDatabaseProject(fixture.ctx, t, fixture.store, "project-instance", "existing-database"))
}

package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func newInstanceProjectFixture(t *testing.T) (context.Context, *sql.DB, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES
			('default', 'default', 'Default', FALSE),
			('project-a', 'default', 'Project A', FALSE),
			('deleted-project', 'default', 'Deleted Project', TRUE);
	`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return ctx, db, s
}

func testInstanceMetadata() *storepb.Instance {
	return &storepb.Instance{
		DataSources: []*storepb.DataSource{{
			Id:   "admin",
			Type: storepb.DataSourceType_ADMIN,
		}},
	}
}

func TestCreateAndListProjectInstance(t *testing.T) {
	ctx, db, s := newInstanceProjectFixture(t)
	projectID := "project-a"
	instance, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "project-instance",
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata:   testInstanceMetadata(),
	})
	require.NoError(t, err)
	require.Equal(t, &projectID, instance.ProjectID)
	instanceID := "project-instance"
	updated, err := s.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instanceID,
		Workspace:  "default",
		Metadata:   testInstanceMetadata(),
	})
	require.NoError(t, err)
	require.Equal(t, &projectID, updated.ProjectID)
	_, err = s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "workspace-instance",
		Workspace:  "default",
		Metadata:   testInstanceMetadata(),
	})
	require.NoError(t, err)

	var persistedProjectID string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT project FROM instance WHERE resource_id = 'project-instance'
	`).Scan(&persistedProjectID))
	require.Equal(t, projectID, persistedProjectID)

	projectInstances, err := s.ListInstances(ctx, &store.FindInstanceMessage{
		Workspace: "default",
		ProjectID: &projectID,
	})
	require.NoError(t, err)
	require.Len(t, projectInstances, 1)
	require.Equal(t, "project-instance", projectInstances[0].ResourceID)
	wrongScope, err := s.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  "default",
		ProjectID:  new(string),
		ResourceID: &instanceID,
	})
	require.NoError(t, err)
	require.Nil(t, wrongScope)

	unfilteredInstances, err := s.ListInstances(ctx, &store.FindInstanceMessage{Workspace: "default"})
	require.NoError(t, err)
	require.Len(t, unfilteredInstances, 2)

	workspaceInstances, err := s.ListInstances(ctx, &store.FindInstanceMessage{
		Workspace: "default",
		ProjectID: new(string),
	})
	require.NoError(t, err)
	require.Len(t, workspaceInstances, 1)
	require.Equal(t, "workspace-instance", workspaceInstances[0].ResourceID)

	instanceByResourceID, err := s.GetInstanceByResourceID(ctx, "project-instance")
	require.NoError(t, err)
	require.Equal(t, &projectID, instanceByResourceID.ProjectID)

	allInstances, err := s.ListAllInstances(ctx, false)
	require.NoError(t, err)
	require.Len(t, allInstances, 2)
	require.Equal(t, &projectID, allInstances[0].ProjectID)
}

func TestCreateProjectInstanceRejectsDefaultDeletedAndMissingProject(t *testing.T) {
	ctx, _, s := newInstanceProjectFixture(t)
	for _, projectID := range []string{"default", "deleted-project", "missing-project"} {
		t.Run(projectID, func(t *testing.T) {
			_, err := s.CreateInstance(ctx, &store.InstanceMessage{
				ResourceID: projectID + "-instance",
				Workspace:  "default",
				ProjectID:  &projectID,
				Metadata:   testInstanceMetadata(),
			})
			require.Error(t, err)
		})
	}
}

func TestDeleteProjectDeletesProjectInstancesAndKeepsWorkspaceInstanceDatabases(t *testing.T) {
	ctx, db, s := newInstanceProjectFixture(t)
	_, err := db.ExecContext(ctx, `
		UPDATE project SET deleted = TRUE WHERE resource_id = 'project-a';
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-a'),
			('workspace-instance', 'default', NULL);
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'project-db', 'project-a'),
			('workspace-instance', 'workspace-db', 'project-a');
	`)
	require.NoError(t, err)

	require.NoError(t, s.DeleteProject(ctx, "default", "project-a"))

	var projectInstanceCount, projectDatabaseCount int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM instance WHERE resource_id = 'project-instance'
	`).Scan(&projectInstanceCount))
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM db WHERE instance = 'project-instance'
	`).Scan(&projectDatabaseCount))
	require.Zero(t, projectInstanceCount)
	require.Zero(t, projectDatabaseCount)

	var workspaceDatabaseProject string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT project FROM db WHERE instance = 'workspace-instance' AND name = 'workspace-db'
	`).Scan(&workspaceDatabaseProject))
	require.Equal(t, "default", workspaceDatabaseProject)
}

func TestDeleteProjectInstancePurgesHistory(t *testing.T) {
	ctx, db, s := newInstanceProjectFixture(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO instance (resource_id, workspace, project, deleted) VALUES
			('project-instance', 'default', 'project-a', TRUE);
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'project-db', 'project-a');
		INSERT INTO query_history (resource_id, creator, project, database, statement, type) VALUES
			('history-a', 'user@example.com', 'project-a',
			 'projects/project-a/instances/project-instance/databases/project-db', '', 'QUERY');
	`)
	require.NoError(t, err)

	require.NoError(t, s.DeleteInstance(ctx, "default", "project-instance"))

	for _, table := range []string{"instance", "db", "query_history"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, "%s should be removed", table)
	}
}

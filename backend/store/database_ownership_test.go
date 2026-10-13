package store_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestDatabaseWritersRespectProjectInstanceOwnership(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-b', 'default', 'Project B'),
			('project-c', 'default', 'Project C');
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-b'),
			('workspace-instance', 'default', NULL),
			('workspace-instance-2', 'default', NULL);
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

	// A create-database upsert cannot rebind a live database across projects; a
	// real move uses UpdateDatabase, allowed above.
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "workspace-instance",
		DatabaseName: "independent",
		ProjectID:    projectC,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.Error(t, err)
	require.Equal(t, common.Invalid, common.ErrorCode(err))
	require.Equal(t, projectB, getDatabaseProject(ctx, t, s, "workspace-instance", "independent"))

	// A create-database task that re-runs against its own database must not be
	// refused, so the same-project upsert still succeeds.
	database, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "workspace-instance",
		DatabaseName: "independent",
		ProjectID:    projectB,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, projectB, database.ProjectID)

	// The guard keys on (instance, name): the same name on another instance is
	// a different database, so project C may create it there.
	database, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "workspace-instance-2",
		DatabaseName: "independent",
		ProjectID:    projectC,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, projectC, database.ProjectID)
}

// workspaceInstanceSeed seeds two projects and one workspace-level instance.
const workspaceInstanceSeed = `
	ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
	INSERT INTO project (resource_id, workspace, name) VALUES
		('project-b', 'default', 'Project B'),
		('project-c', 'default', 'Project C');
	INSERT INTO instance (resource_id, workspace, project) VALUES
		('workspace-instance', 'default', NULL);
`

// TestWorkspaceInstanceUpsertDroppedNameStaysWithItsProject: a dropped name
// cannot be taken by another project, but the owning project may reuse it.
func TestWorkspaceInstanceUpsertDroppedNameStaysWithItsProject(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, workspaceInstanceSeed)
	ctx, s := fixture.ctx, fixture.store

	_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "workspace-instance", DatabaseName: "freed", ProjectID: "project-b",
		Metadata: &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	deleted := true
	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID: "workspace-instance", DatabaseName: "freed", Deleted: &deleted,
	})
	require.NoError(t, err)

	// Another project cannot take the dropped name.
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "workspace-instance", DatabaseName: "freed", ProjectID: "project-c",
		Metadata: &storepb.DatabaseMetadata{},
	})
	require.Error(t, err)
	require.Equal(t, common.Invalid, common.ErrorCode(err))
	require.Equal(t, "project-b", getDatabaseProject(ctx, t, s, "workspace-instance", "freed"))

	// The owning project may re-create it.
	database, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "workspace-instance", DatabaseName: "freed", ProjectID: "project-b",
		Metadata: &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, "project-b", database.ProjectID)
}

// TestWorkspaceInstanceUpsertRaceKeepsOneOwner pins that the guard lives in the
// write, not in a prior read: two concurrent first-creates of one name from
// different projects cannot both win. Exactly one succeeds; the other is refused.
func TestWorkspaceInstanceUpsertRaceKeepsOneOwner(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, workspaceInstanceSeed)
	ctx, s := fixture.ctx, fixture.store

	upsert := func(project string) error {
		_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
			InstanceID: "workspace-instance", DatabaseName: "racer", ProjectID: project,
			Metadata: &storepb.DatabaseMetadata{},
		})
		return err
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Go(func() { errs[0] = upsert("project-b") })
	wg.Go(func() { errs[1] = upsert("project-c") })
	wg.Wait()

	winners := 0
	for _, err := range errs {
		if err == nil {
			winners++
			continue
		}
		require.Equal(t, common.Invalid, common.ErrorCode(err))
	}
	require.Equal(t, 1, winners, "exactly one create must win; errs: %v", errs)
	require.Contains(t, []string{"project-b", "project-c"},
		getDatabaseProject(ctx, t, s, "workspace-instance", "racer"))
}

func TestListDatabasesIncludesInstanceProject(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B');
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-b'),
			('workspace-instance', 'default', NULL);
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'project-db', 'project-b'),
			('workspace-instance', 'workspace-db', 'project-b');
	`)

	databases, err := fixture.store.ListDatabases(fixture.ctx, &store.FindDatabaseMessage{ShowDeleted: true})
	require.NoError(t, err)
	require.Len(t, databases, 2)
	byInstance := make(map[string]*store.DatabaseMessage, len(databases))
	for _, database := range databases {
		byInstance[database.InstanceID] = database
	}
	require.NotNil(t, byInstance["project-instance"].InstanceProjectID)
	require.Equal(t, "project-b", *byInstance["project-instance"].InstanceProjectID)
	require.Equal(t, "projects/project-b/instances/project-instance/databases/project-db", byInstance["project-instance"].ResourceName())
	require.Nil(t, byInstance["workspace-instance"].InstanceProjectID)
	require.Equal(t, "instances/workspace-instance/databases/workspace-db", byInstance["workspace-instance"].ResourceName())
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

func TestProjectInstanceDatabaseUpsertAllowsArchivedOwner(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name, deleted)
			VALUES ('project-b', 'default', 'Project B', TRUE);
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('project-instance', 'default', 'project-b');
	`)

	database, err := fixture.store.UpsertDatabase(fixture.ctx, &store.DatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "new-database",
		ProjectID:    "project-b",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	require.Equal(t, "project-b", database.ProjectID)
}

func TestProjectInstanceDatabaseUpdateAllowsArchivedOwner(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		ALTER TABLE instance ADD COLUMN IF NOT EXISTS project TEXT REFERENCES project(resource_id);
		INSERT INTO project (resource_id, workspace, name, deleted)
			VALUES ('project-b', 'default', 'Project B', TRUE);
		INSERT INTO instance (resource_id, workspace, project)
			VALUES ('project-instance', 'default', 'project-b');
		INSERT INTO db (instance, name, project) VALUES
			('project-instance', 'existing-database', 'project-b');
	`)

	deleted := true
	database, err := fixture.store.UpdateDatabase(fixture.ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "project-instance",
		DatabaseName: "existing-database",
		Deleted:      &deleted,
	})
	require.NoError(t, err)
	require.True(t, database.Deleted)
}

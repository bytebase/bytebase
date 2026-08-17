package sampleprojectinstance

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestStoreMetadataCreatesAndLooksUpScopedSampleProjectInstance(t *testing.T) {
	ctx, db, stores := newManagerStore(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO project (resource_id, workspace, name)
		VALUES
			('project-a', 'workspace-a', 'Project A'),
			('project-b', 'workspace-a', 'Project B')
	`)
	require.NoError(t, err)
	allocation := Allocation{Database: "sample-db", Role: "sample-role", Password: "secret"}
	metadata := NewStoreMetadata(stores)
	instance, err := metadata.Create(ctx, Registration{
		WorkspaceID:   "workspace-a",
		ProjectID:     "project-a",
		EnvironmentID: "test",
		InstanceID:    "sample-instance",
		Title:         sampleProjectInstanceTitle,
		Engine:        storepb.Engine_POSTGRES,
		Allocation:    allocation,
		AdminDataSource: &storepb.DataSource{
			Id:       "admin",
			Type:     storepb.DataSourceType_ADMIN,
			Database: allocation.Database,
			Username: allocation.Role,
			Password: allocation.Password,
		},
		SyncDatabaseNames: []string{allocation.Database},
	})
	require.NoError(t, err)
	require.True(t, instance.Metadata.GetActivation())
	require.Equal(t, allocation.Password, instance.Metadata.GetDataSources()[0].GetPassword())

	state, err := metadata.Lookup(ctx, allocation, "sample-instance", "workspace-a", "project-a")
	require.NoError(t, err)
	require.True(t, state.ProjectActive)
	require.True(t, state.InstanceMatches)
	require.NotNil(t, state.Instance)
	require.Nil(t, state.Database)

	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   "sample-instance",
		DatabaseName: allocation.Database,
		ProjectID:    "project-a",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	state, err = metadata.Lookup(ctx, allocation, "sample-instance", "workspace-a", "project-a")
	require.NoError(t, err)
	require.NotNil(t, state.Database)

	instance.Metadata.Title = "Edited Sample Project Instance"
	_, err = stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Workspace:  instance.Workspace,
		Metadata:   instance.Metadata,
	})
	require.NoError(t, err)
	state, err = metadata.Lookup(ctx, allocation, "sample-instance", "workspace-a", "project-a")
	require.NoError(t, err)
	require.True(t, state.active())

	require.Error(t, metadata.Remove(ctx, allocation, "sample-instance", "workspace-a", "project-a"))
	projectID := "project-a"
	instanceID := "sample-instance"
	remaining, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  "workspace-a",
		ProjectID:  &projectID,
		ResourceID: &instanceID,
	})
	require.NoError(t, err)
	require.NotNil(t, remaining)

	instance.Metadata.Title = sampleProjectInstanceTitle
	_, err = stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Workspace:  instance.Workspace,
		Metadata:   instance.Metadata,
	})
	require.NoError(t, err)
	require.NoError(t, metadata.Remove(ctx, allocation, "sample-instance", "workspace-a", "project-a"))
	removed, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  "workspace-a",
		ProjectID:  &projectID,
		ResourceID: &instanceID,
	})
	require.NoError(t, err)
	require.Nil(t, removed)

	_, err = metadata.Create(ctx, Registration{
		WorkspaceID:   "workspace-a",
		ProjectID:     "project-a",
		EnvironmentID: "test",
		InstanceID:    "cross-project-instance",
		Title:         sampleProjectInstanceTitle,
		Engine:        storepb.Engine_POSTGRES,
		Allocation:    allocation,
		AdminDataSource: &storepb.DataSource{
			Id:       "admin",
			Type:     storepb.DataSourceType_ADMIN,
			Database: allocation.Database,
			Username: allocation.Role,
			Password: allocation.Password,
		},
		SyncDatabaseNames: []string{allocation.Database},
	})
	require.NoError(t, err)
	state, err = metadata.Lookup(ctx, allocation, "cross-project-instance", "workspace-a", "project-b")
	require.NoError(t, err)
	require.True(t, state.ProjectActive)
	require.False(t, state.InstanceMatches)
	require.NotNil(t, state.Instance)
	require.Nil(t, state.Database)
	require.Error(t, metadata.Remove(ctx, allocation, "cross-project-instance", "workspace-a", "project-b"))
}

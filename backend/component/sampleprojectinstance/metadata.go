package sampleprojectinstance

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// lookupMetadata reads the deterministic Instance across its workspace, then the
// expected project-scoped Database, including their deleted state.
func (m *Manager) lookupMetadata(
	ctx context.Context,
	allocation Allocation,
	instanceID, workspaceID, projectID string,
) (metadataState, error) {
	project, err := m.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		Workspace:   workspaceID,
		ShowDeleted: true,
	})
	if err != nil {
		return metadataState{}, errors.Wrap(err, "get project")
	}
	state := metadataState{ProjectActive: project != nil && !project.Deleted}
	instance, err := m.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   workspaceID,
		ResourceID:  &instanceID,
		ShowDeleted: true,
	})
	if err != nil {
		return metadataState{}, errors.Wrap(err, "get instance")
	}
	state.Instance = instance
	state.InstanceMatches = instance != nil && instance.ProjectID != nil && *instance.ProjectID == projectID
	database, err := m.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    workspaceID,
		ProjectID:    &projectID,
		InstanceID:   &instanceID,
		DatabaseName: &allocation.Database,
		ShowDeleted:  true,
	})
	if err != nil {
		return metadataState{}, errors.Wrap(err, "get database")
	}
	state.Database = database
	return state, nil
}

// createMetadata persists the activated, project-scoped PostgreSQL Instance through
// Store.CreateInstance so credential obfuscation remains centralized.
func (m *Manager) createMetadata(ctx context.Context, registration registration) (*store.InstanceMessage, error) {
	return m.store.CreateInstance(ctx, &store.InstanceMessage{
		Workspace:     registration.WorkspaceID,
		ProjectID:     &registration.ProjectID,
		EnvironmentID: &registration.EnvironmentID,
		ResourceID:    registration.InstanceID,
		Metadata: &storepb.Instance{
			Title:       registration.Title,
			Engine:      registration.Engine,
			Activation:  true,
			DataSources: []*storepb.DataSource{registration.AdminDataSource},
			SyncDatabases: &storepb.SyncDatabases{
				Databases: registration.SyncDatabaseNames,
			},
		},
	})
}

// removeMetadata deletes only the exact, deterministic Sample Project Instance in its
// owning project and workspace. It rejects collisions so the caller leaves
// physical resources and the reservation available for later diagnosis.
func (m *Manager) removeMetadata(
	ctx context.Context,
	allocation Allocation,
	instanceID, workspaceID, projectID string,
) error {
	instance, err := m.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   workspaceID,
		ResourceID:  &instanceID,
		ShowDeleted: true,
	})
	if err != nil {
		return errors.Wrap(err, "get instance for removal")
	}
	if instance == nil {
		return nil
	}
	if instance.ProjectID == nil || *instance.ProjectID != projectID {
		return errors.Errorf("Sample Project Instance %q belongs to a different project", instanceID)
	}
	if !matchesRegistration(instance, allocation) {
		return errors.Errorf("Sample Project Instance %q metadata does not match its allocation", instanceID)
	}
	if !instance.Deleted {
		deleted := true
		if _, err := m.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
			ResourceID: &instanceID,
			Workspace:  workspaceID,
			Deleted:    &deleted,
		}); err != nil {
			return errors.Wrap(err, "soft delete instance")
		}
	}
	if err := m.store.DeleteInstance(ctx, workspaceID, instanceID); err != nil {
		return errors.Wrap(err, "delete instance")
	}
	return nil
}

func matchesRegistration(instance *store.InstanceMessage, allocation Allocation) bool {
	if instance.Metadata == nil ||
		instance.Metadata.GetTitle() != sampleProjectInstanceTitle ||
		instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
		return false
	}
	for _, dataSource := range instance.Metadata.GetDataSources() {
		if dataSource.GetId() == "admin" &&
			dataSource.GetType() == storepb.DataSourceType_ADMIN &&
			dataSource.GetDatabase() == allocation.Database &&
			dataSource.GetUsername() == allocation.Role {
			return true
		}
	}
	return false
}

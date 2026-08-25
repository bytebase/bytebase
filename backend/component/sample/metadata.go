package sample

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Registration contains the ordinary Instance metadata for one sample.
type Registration struct {
	WorkspaceID       string
	InstanceProjectID *string
	EnvironmentID     *string
	InstanceID        string
	Title             string
	AdminDataSource   *storepb.DataSource
	SyncDatabaseNames []string
}

// MetadataLookup identifies one implementation-owned Instance and database.
type MetadataLookup struct {
	WorkspaceID       string
	InstanceProjectID *string
	DatabaseProjectID string
	InstanceID        string
	DatabaseName      string
}

// MetadataState is the current ordinary Instance/database state.
type MetadataState struct {
	ProjectActive   bool
	InstanceMatches bool
	Instance        *store.InstanceMessage
	Database        *store.DatabaseMessage
}

// Active reports whether all expected metadata is active.
func (s MetadataState) Active() bool {
	return s.ProjectActive && s.InstanceMatches && s.Instance != nil && !s.Instance.Deleted && s.Database != nil && !s.Database.Deleted
}

// LookupMetadata returns the metadata matching lookup.
func LookupMetadata(ctx context.Context, stores *store.Store, lookup MetadataLookup) (MetadataState, error) {
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{
		ResourceID:  &lookup.DatabaseProjectID,
		Workspace:   lookup.WorkspaceID,
		ShowDeleted: true,
	})
	if err != nil {
		return MetadataState{}, errors.Wrap(err, "get sample database project")
	}
	state := MetadataState{ProjectActive: project != nil && !project.Deleted}
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   lookup.WorkspaceID,
		ResourceID:  &lookup.InstanceID,
		ShowDeleted: true,
	})
	if err != nil {
		return MetadataState{}, errors.Wrap(err, "get sample instance")
	}
	state.Instance = instance
	state.InstanceMatches = instance != nil && sameOptionalString(instance.ProjectID, lookup.InstanceProjectID)
	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    lookup.WorkspaceID,
		ProjectID:    &lookup.DatabaseProjectID,
		InstanceID:   &lookup.InstanceID,
		DatabaseName: &lookup.DatabaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return MetadataState{}, errors.Wrap(err, "get sample database")
	}
	state.Database = database
	return state, nil
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// CreateMetadata registers an ordinary Bytebase Instance.
func CreateMetadata(ctx context.Context, stores *store.Store, registration Registration) (*store.InstanceMessage, error) {
	return stores.CreateInstance(ctx, &store.InstanceMessage{
		Workspace:     registration.WorkspaceID,
		ProjectID:     registration.InstanceProjectID,
		EnvironmentID: registration.EnvironmentID,
		ResourceID:    registration.InstanceID,
		Metadata: &storepb.Instance{
			Title:      registration.Title,
			Engine:     storepb.Engine_POSTGRES,
			Activation: false,
			DataSources: []*storepb.DataSource{
				registration.AdminDataSource,
			},
			SyncDatabases: &storepb.SyncDatabases{Databases: registration.SyncDatabaseNames},
		},
	})
}

// TransferDatabase transfers one discovered sample database.
func TransferDatabase(ctx context.Context, stores *store.Store, projectID, instanceID, databaseName string) error {
	_, err := stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		ProjectID:    &projectID,
	})
	return errors.Wrap(err, "transfer sample database")
}

// ArchiveMetadata soft-deletes an implementation-owned sample Instance.
func ArchiveMetadata(ctx context.Context, stores *store.Store, workspaceID string, instanceProjectID *string, instanceID string) (bool, error) {
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:   workspaceID,
		ResourceID:  &instanceID,
		ShowDeleted: true,
	})
	if err != nil {
		return false, errors.Wrap(err, "get sample instance for archive")
	}
	if instance == nil {
		return false, nil
	}
	if !sameOptionalString(instance.ProjectID, instanceProjectID) {
		return false, errors.Errorf("sample instance %q belongs to a different project", instanceID)
	}
	if !instance.Deleted {
		deleted := true
		if _, err := stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
			ResourceID: &instanceID,
			Workspace:  workspaceID,
			Deleted:    &deleted,
		}); err != nil {
			return false, errors.Wrap(err, "soft delete sample instance")
		}
	}
	return true, nil
}

// PurgePartialMetadata removes metadata created by an unactivated setup.
func PurgePartialMetadata(ctx context.Context, stores *store.Store, workspaceID string, instanceProjectID *string, instanceID string) error {
	found, err := ArchiveMetadata(ctx, stores, workspaceID, instanceProjectID, instanceID)
	if err != nil || !found {
		return err
	}
	return errors.Wrap(stores.DeleteInstance(ctx, workspaceID, instanceID), "delete partial sample instance")
}

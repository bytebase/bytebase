package plancheck

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

// resolveDatabaseTarget resolves a workspace- or project-scoped database resource name.
func resolveDatabaseTarget(ctx context.Context, stores *store.Store, target string) (*store.InstanceMessage, *store.DatabaseMessage, error) {
	projectID, instanceID, databaseName, err := common.GetDatabaseResourceName(target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse target %s", target)
	}

	instance, err := stores.GetInstanceByResourceID(ctx, instanceID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance %s not found", instanceID)
	}
	if instance.Deleted {
		return nil, nil, errors.Errorf("instance %s has been deleted", instanceID)
	}
	if (projectID == nil) != (instance.ProjectID == nil) ||
		projectID != nil && *instance.ProjectID != *projectID {
		if projectID == nil {
			return nil, nil, errors.Errorf("project instance %q must be addressed through its project", instanceID)
		}
		return nil, nil, errors.Errorf("instance %q does not belong to project %q", instanceID, *projectID)
	}

	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database not found %q", databaseName)
	}
	if projectID != nil && database.ProjectID != *projectID {
		return nil, nil, errors.Errorf("database %q does not belong to project %q", target, *projectID)
	}

	return instance, database, nil
}

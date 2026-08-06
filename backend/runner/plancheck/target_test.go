package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestResolveDatabaseTarget(t *testing.T) {
	ctx := context.Background()
	stores := setupPlancheckStore(ctx, t)

	projectID := "project-a"
	for _, instance := range []*store.InstanceMessage{
		{
			ResourceID: "workspace-instance",
			Workspace:  "default",
			Metadata:   &storepb.Instance{Engine: storepb.Engine_POSTGRES, DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}}},
		},
		{
			ResourceID: "project-instance",
			Workspace:  "default",
			ProjectID:  &projectID,
			Metadata:   &storepb.Instance{Engine: storepb.Engine_POSTGRES, DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}}},
		},
	} {
		_, err := stores.CreateInstance(ctx, instance)
		require.NoError(t, err)
	}
	for _, database := range []*store.DatabaseMessage{
		{ProjectID: projectID, InstanceID: "workspace-instance", DatabaseName: "workspace-db", Metadata: &storepb.DatabaseMetadata{}},
		{ProjectID: projectID, InstanceID: "project-instance", DatabaseName: "project-db", Metadata: &storepb.DatabaseMetadata{}},
	} {
		_, err := stores.UpsertDatabase(ctx, database)
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		target     string
		wantErr    string
		instanceID string
		databaseID string
	}{
		{
			name:       "workspace canonical database",
			target:     common.FormatDatabase("workspace-instance", "workspace-db"),
			instanceID: "workspace-instance",
			databaseID: "workspace-db",
		},
		{
			name:       "project canonical database",
			target:     common.FormatProjectDatabase(projectID, "project-instance", "project-db"),
			instanceID: "project-instance",
			databaseID: "project-db",
		},
		{
			name:    "project instance rejects workspace alias",
			target:  common.FormatDatabase("project-instance", "project-db"),
			wantErr: "must be addressed through its project",
		},
		{
			name:    "nested project must own instance",
			target:  common.FormatProjectDatabase(projectID, "workspace-instance", "workspace-db"),
			wantErr: "does not belong to project",
		},
		{
			name:    "nested project must match instance owner",
			target:  common.FormatProjectDatabase("project-b", "project-instance", "project-db"),
			wantErr: "does not belong to project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, database, err := resolveDatabaseTarget(ctx, stores, tt.target)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, instance)
				require.Nil(t, database)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.instanceID, instance.ResourceID)
			require.Equal(t, tt.databaseID, database.DatabaseName)
		})
	}
}

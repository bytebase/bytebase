package common_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/common"
)

func TestGetInstanceDatabaseID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		instanceID string
		databaseID string
		wantErr    bool
	}{
		{
			name:       "workspace instance",
			target:     "instances/test/databases/hr",
			instanceID: "test",
			databaseID: "hr",
		},
		{
			name:       "project instance",
			target:     "projects/hr/instances/test/databases/hr",
			instanceID: "test",
			databaseID: "hr",
		},
		{
			name:    "missing project instance ID",
			target:  "projects/hr/instances//databases/hr",
			wantErr: true,
		},
		{
			name:    "missing database ID",
			target:  "instances/test/databases/",
			wantErr: true,
		},
		{
			name:    "project database group",
			target:  "projects/hr/databaseGroups/all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			instanceID, databaseID, err := common.GetInstanceDatabaseID(tt.target)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.instanceID, instanceID)
			require.Equal(t, tt.databaseID, databaseID)
		})
	}
}

func TestGetNameParentTokensRejectsEmptyTokens(t *testing.T) {
	t.Parallel()

	_, err := common.GetNameParentTokens("projects//databaseGroups/all", common.ProjectNamePrefix, common.DatabaseGroupNamePrefix)
	require.Error(t, err)
}

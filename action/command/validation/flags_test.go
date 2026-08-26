package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []string
		wantErr string
	}{
		{
			name: "workspace database targets",
			targets: []string{
				"instances/test/databases/hr_test",
				"instances/prod/databases/hr_prod",
			},
		},
		{
			name: "project instance database targets",
			targets: []string{
				"projects/hr/instances/test/databases/hr_test",
				"projects/hr/instances/prod/databases/hr_prod",
			},
		},
		{
			name: "mixed workspace and project instance database targets",
			targets: []string{
				"instances/test/databases/hr_test",
				"projects/hr/instances/prod/databases/hr_prod",
			},
		},
		{
			name:    "database target and database group",
			targets: []string{"projects/hr/instances/test/databases/hr_test", "projects/hr/databaseGroups/all"},
			wantErr: "either database targets or a database group target",
		},
		{
			name:    "multiple database groups",
			targets: []string{"projects/hr/databaseGroups/one", "projects/hr/databaseGroups/two"},
			wantErr: "single database group target",
		},
		{
			name:    "malformed project instance target",
			targets: []string{"projects/hr/instances//databases/hr_test"},
			wantErr: "invalid target format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTargets(tt.targets)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

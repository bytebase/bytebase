package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListProjectFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSQL:  "",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name:     "name filter",
			filter:   `name == "test-project"`,
			wantSQL:  "(project.name = $1)",
			wantArgs: []any{"test-project"},
			wantErr:  false,
		},
		{
			name:     "resource_id filter",
			filter:   `resource_id == "test-project-id"`,
			wantSQL:  "(project.resource_id = $1)",
			wantArgs: []any{"test-project-id"},
			wantErr:  false,
		},
		{
			name:     "exclude_default filter - true",
			filter:   `exclude_default == true`,
			wantSQL:  "(project.resource_id != $1)",
			wantArgs: []any{"default"},
			wantErr:  false,
		},
		{
			name:     "exclude_default filter - false",
			filter:   `exclude_default == false`,
			wantSQL:  "(TRUE)",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "state filter - ACTIVE",
			filter:   `state == "STATE_ACTIVE"`,
			wantSQL:  "(project.deleted = $1)",
			wantArgs: []any{false},
			wantErr:  false,
		},
		{
			name:     "state filter - DELETED",
			filter:   `state == "STATE_DELETED"`,
			wantSQL:  "(project.deleted = $1)",
			wantArgs: []any{true},
			wantErr:  false,
		},
		{
			name:     "name matches",
			filter:   `name.matches("test")`,
			wantSQL:  "(LOWER(project.name) LIKE $1)",
			wantArgs: []any{"%test%"},
			wantErr:  false,
		},
		{
			name:     "resource_id matches",
			filter:   `resource_id.matches("prod")`,
			wantSQL:  "(LOWER(project.resource_id) LIKE $1)",
			wantArgs: []any{"%prod%"},
			wantErr:  false,
		},
		{
			name:     "label filter - string",
			filter:   `labels.environment == "production"`,
			wantSQL:  "(project.setting->'labels'->>'environment' = $1)",
			wantArgs: []any{"production"},
			wantErr:  false,
		},
		{
			name:     "label filter - in operator",
			filter:   `labels.environment in ["production", "staging"]`,
			wantSQL:  "(project.setting->'labels'->>'environment' = ANY($1))",
			wantArgs: []any{[]any{"production", "staging"}},
			wantErr:  false,
		},
		{
			name:     "AND condition with name and resource_id",
			filter:   `name == "test-project" && resource_id == "test-id"`,
			wantSQL:  "((project.name = $1 AND project.resource_id = $2))",
			wantArgs: []any{"test-project", "test-id"},
			wantErr:  false,
		},
		{
			name:     "OR condition with name",
			filter:   `name == "project1" || name == "project2"`,
			wantSQL:  "((project.name = $1 OR project.name = $2))",
			wantArgs: []any{"project1", "project2"},
			wantErr:  false,
		},
		{
			name:     "complex nested AND/OR",
			filter:   `(name == "p1" || name == "p2") && exclude_default == true`,
			wantSQL:  "(((project.name = $1 OR project.name = $2) AND project.resource_id != $3))",
			wantArgs: []any{"p1", "p2", "default"},
			wantErr:  false,
		},
		{
			name:        "invalid filter syntax",
			filter:      `invalid syntax {{`,
			wantErr:     true,
			errContains: "failed to parse filter",
		},
		{
			name:        "unsupported variable",
			filter:      `unsupported == "value"`,
			wantErr:     true,
			errContains: "unsupport variable",
		},
		{
			name:        "invalid state value",
			filter:      `state == "INVALID_STATE"`,
			wantErr:     true,
			errContains: "invalid state filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListProjectFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.filter == "" {
				require.Nil(t, q)
				return
			}

			require.NotNil(t, q)

			sql, args, err := q.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}

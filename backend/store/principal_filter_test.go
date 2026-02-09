package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListUserFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantProject *string
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
			filter:   `name == "ed"`,
			wantSQL:  "(principal.name = $1)",
			wantArgs: []any{"ed"},
			wantErr:  false,
		},
		{
			name:     "email filter",
			filter:   `email == "test@example.com"`,
			wantSQL:  "(principal.email = $1)",
			wantArgs: []any{"test@example.com"},
			wantErr:  false,
		},
		{
			name:     "name matches",
			filter:   `name.matches("ED")`,
			wantSQL:  "(LOWER(principal.name) LIKE $1)",
			wantArgs: []any{"%ed%"},
			wantErr:  false,
		},
		{
			name:     "email matches",
			filter:   `email.matches("test")`,
			wantSQL:  "(LOWER(principal.email) LIKE $1)",
			wantArgs: []any{"%test%"},
			wantErr:  false,
		},
		{
			name:     "state filter - STATE_ACTIVE",
			filter:   `state == "STATE_ACTIVE"`,
			wantSQL:  "(principal.deleted = $1)",
			wantArgs: []any{false},
			wantErr:  false,
		},
		{
			name:     "state filter - STATE_DELETED",
			filter:   `state == "STATE_DELETED"`,
			wantSQL:  "(principal.deleted = $1)",
			wantArgs: []any{true},
			wantErr:  false,
		},
		{
			name:        "project filter",
			filter:      `project == "projects/sample-project"`,
			wantSQL:     "(TRUE)",
			wantArgs:    []any{},
			wantProject: func() *string { s := "sample-project"; return &s }(),
			wantErr:     false,
		},
		{
			name:     "AND condition",
			filter:   `name == "ed" && email == "ed@test.com"`,
			wantSQL:  "((principal.name = $1 AND principal.email = $2))",
			wantArgs: []any{"ed", "ed@test.com"},
			wantErr:  false,
		},
		{
			name:     "OR condition",
			filter:   `name == "ed" || name == "alice"`,
			wantSQL:  "((principal.name = $1 OR principal.name = $2))",
			wantArgs: []any{"ed", "alice"},
			wantErr:  false,
		},
		{
			name:     "complex nested condition",
			filter:   `(name == "ed" || name == "alice")`,
			wantSQL:  "((principal.name = $1 OR principal.name = $2))",
			wantArgs: []any{"ed", "alice"},
			wantErr:  false,
		},
		{
			name:        "unsupported variable",
			filter:      `title == "ed"`,
			wantErr:     true,
			errContains: "unsupport variable",
		},
		{
			name:        "invalid filter syntax",
			filter:      `invalid syntax {{`,
			wantErr:     true,
			errContains: "failed to parse filter",
		},
		{
			name:        "invalid user type",
			filter:      `user_type == "INVALID_TYPE"`,
			wantErr:     true,
			errContains: "invalid user type filter",
		},
		{
			name:        "invalid state",
			filter:      `state == "INVALID_STATE"`,
			wantErr:     true,
			errContains: "invalid state filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetListUserFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.wantProject != nil {
				require.NotNil(t, result.ProjectID)
				require.Equal(t, *tt.wantProject, *result.ProjectID)
			} else {
				require.Nil(t, result.ProjectID)
			}

			if query := result.Query; query != nil {
				sql, args, err := query.ToSQL()
				require.NoError(t, err)
				require.Equal(t, tt.wantSQL, sql)
				require.Equal(t, tt.wantArgs, args)
			}
		})
	}
}

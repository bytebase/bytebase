package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListGroupFilter(t *testing.T) {
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
			name:     "title filter",
			filter:   `title == "Developers"`,
			wantSQL:  "(name = $1)",
			wantArgs: []any{"Developers"},
			wantErr:  false,
		},
		{
			name:     "email filter",
			filter:   `email == "dev@example.com"`,
			wantSQL:  "(email = $1)",
			wantArgs: []any{"dev@example.com"},
			wantErr:  false,
		},
		{
			name:     "title matches",
			filter:   `title.matches("dev")`,
			wantSQL:  "(LOWER(name) LIKE $1)",
			wantArgs: []any{"%dev%"},
			wantErr:  false,
		},
		{
			name:     "email matches",
			filter:   `email.matches("example")`,
			wantSQL:  "(LOWER(email) LIKE $1)",
			wantArgs: []any{"%example%"},
			wantErr:  false,
		},
		{
			name:     "AND condition with title and email",
			filter:   `title == "Developers" && email == "dev@example.com"`,
			wantSQL:  "((name = $1 AND email = $2))",
			wantArgs: []any{"Developers", "dev@example.com"},
			wantErr:  false,
		},
		{
			name:     "OR condition with title",
			filter:   `title == "Developers" || title == "Admins"`,
			wantSQL:  "((name = $1 OR name = $2))",
			wantArgs: []any{"Developers", "Admins"},
			wantErr:  false,
		},
		{
			name:     "complex nested AND/OR",
			filter:   `(title == "Developers" || title == "Admins") && email.matches("example")`,
			wantSQL:  "(((name = $1 OR name = $2) AND LOWER(email) LIKE $3))",
			wantArgs: []any{"Developers", "Admins", "%example%"},
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
			name:        "empty matches value",
			filter:      `title.matches("")`,
			wantErr:     true,
			errContains: "empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			find := &FindGroupMessage{}
			q, err := GetListGroupFilter(find, tt.filter)

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

func TestGetListGroupFilterProject(t *testing.T) {
	// Test project filter separately since it modifies the find struct
	find := &FindGroupMessage{}
	q, err := GetListGroupFilter(find, `project == "projects/test-project"`)

	require.NoError(t, err)
	require.NotNil(t, q)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "(TRUE)", sql)
	require.Empty(t, args)
	require.NotNil(t, find.ProjectID)
	require.Equal(t, "test-project", *find.ProjectID)
}

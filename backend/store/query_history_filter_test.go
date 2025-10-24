package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListQueryHistoryFilter(t *testing.T) {
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
			name:     "project filter",
			filter:   `project == "projects/test-project"`,
			wantSQL:  "(query_history.project_id = $1)",
			wantArgs: []any{"test-project"},
			wantErr:  false,
		},
		{
			name:     "database filter",
			filter:   `database == "instances/test-instance/databases/test-db"`,
			wantSQL:  "(query_history.database = $1)",
			wantArgs: []any{"instances/test-instance/databases/test-db"},
			wantErr:  false,
		},
		{
			name:     "instance filter",
			filter:   `instance == "instances/test-instance/%"`,
			wantSQL:  "(query_history.database LIKE $1)",
			wantArgs: []any{"instances/test-instance/%"},
			wantErr:  false,
		},
		{
			name:     "type filter",
			filter:   `type == "QUERY"`,
			wantSQL:  "(query_history.type = $1)",
			wantArgs: []any{QueryHistoryType("QUERY")},
			wantErr:  false,
		},
		{
			name:     "statement exact match",
			filter:   `statement == "SELECT * FROM users"`,
			wantSQL:  "(query_history.statement LIKE $1)",
			wantArgs: []any{"SELECT * FROM users"},
			wantErr:  false,
		},
		{
			name:     "statement matches operator",
			filter:   `statement.matches("SELECT")`,
			wantSQL:  "(query_history.statement LIKE $1)",
			wantArgs: []any{"%SELECT%"},
			wantErr:  false,
		},
		{
			name:     "AND condition with project and database",
			filter:   `project == "projects/test-project" && database == "instances/test-instance/databases/test-db"`,
			wantSQL:  "((query_history.project_id = $1 AND query_history.database = $2))",
			wantArgs: []any{"test-project", "instances/test-instance/databases/test-db"},
			wantErr:  false,
		},
		{
			name:     "AND condition with three filters",
			filter:   `project == "projects/test-project" && database == "instances/test-instance/databases/test-db" && type == "QUERY"`,
			wantSQL:  "(((query_history.project_id = $1 AND query_history.database = $2) AND query_history.type = $3))",
			wantArgs: []any{"test-project", "instances/test-instance/databases/test-db", QueryHistoryType("QUERY")},
			wantErr:  false,
		},
		{
			name:     "OR condition with project and database",
			filter:   `project == "projects/project1" || project == "projects/project2"`,
			wantSQL:  "((query_history.project_id = $1 OR query_history.project_id = $2))",
			wantArgs: []any{"project1", "project2"},
			wantErr:  false,
		},
		{
			name:     "OR condition with type",
			filter:   `type == "QUERY" || type == "EXPORT"`,
			wantSQL:  "((query_history.type = $1 OR query_history.type = $2))",
			wantArgs: []any{QueryHistoryType("QUERY"), QueryHistoryType("EXPORT")},
			wantErr:  false,
		},
		{
			name:     "complex nested AND/OR",
			filter:   `(project == "projects/p1" || project == "projects/p2") && type == "QUERY"`,
			wantSQL:  "(((query_history.project_id = $1 OR query_history.project_id = $2) AND query_history.type = $3))",
			wantArgs: []any{"p1", "p2", QueryHistoryType("QUERY")},
			wantErr:  false,
		},
		{
			name:     "complex nested with statement matches",
			filter:   `project == "projects/test" && statement.matches("SELECT") && type == "QUERY"`,
			wantSQL:  "(((query_history.project_id = $1 AND query_history.statement LIKE $2) AND query_history.type = $3))",
			wantArgs: []any{"test", "%SELECT%", QueryHistoryType("QUERY")},
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
			name:        "matches on non-statement variable",
			filter:      `project.matches("test")`,
			wantErr:     true,
			errContains: `only "statement" support`,
		},
		{
			name:        "invalid project format",
			filter:      `project == "invalid-format"`,
			wantErr:     true,
			errContains: "invalid project filter",
		},
		{
			name:        "matches with non-string value",
			filter:      `statement.matches(123)`,
			wantErr:     true,
			errContains: "expect string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListQueryHistoryFilter(tt.filter)

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

func TestGetListQueryHistoryFilter_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		description string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
	}{
		{
			name:        "statement with special characters",
			filter:      `statement == "SELECT * FROM users WHERE name = 'test'"`,
			description: "statement with quotes should be handled correctly",
			wantSQL:     "(query_history.statement LIKE $1)",
			wantArgs:    []any{"SELECT * FROM users WHERE name = 'test'"},
			wantErr:     false,
		},
		{
			name:        "statement matches with wildcard pattern",
			filter:      `statement.matches("SELECT.*FROM")`,
			description: "regex pattern in matches should be wrapped with %",
			wantSQL:     "(query_history.statement LIKE $1)",
			wantArgs:    []any{"%SELECT.*FROM%"},
			wantErr:     false,
		},
		{
			name:        "multiple OR conditions",
			filter:      `project == "projects/p1" || project == "projects/p2" || project == "projects/p3"`,
			description: "multiple OR conditions should be chained",
			wantSQL:     "(((query_history.project_id = $1 OR query_history.project_id = $2) OR query_history.project_id = $3))",
			wantArgs:    []any{"p1", "p2", "p3"},
			wantErr:     false,
		},
		{
			name:        "database with instance prefix",
			filter:      `database == "instances/prod-instance/databases/users_db"`,
			description: "full database path should be preserved",
			wantSQL:     "(query_history.database = $1)",
			wantArgs:    []any{"instances/prod-instance/databases/users_db"},
			wantErr:     false,
		},
		{
			name:        "instance with wildcard",
			filter:      `instance == "instances/prod-%"`,
			description: "instance filter with wildcard for LIKE query",
			wantSQL:     "(query_history.database LIKE $1)",
			wantArgs:    []any{"instances/prod-%"},
			wantErr:     false,
		},
		{
			name:        "complex filter with all supported fields",
			filter:      `project == "projects/test" && database == "instances/i1/databases/db1" && type == "EXPORT" && statement.matches("INSERT")`,
			description: "combination of all supported filter types",
			wantSQL:     "(((query_history.project_id = $1 AND query_history.database = $2) AND (query_history.type = $3 AND query_history.statement LIKE $4)))",
			wantArgs:    []any{"test", "instances/i1/databases/db1", QueryHistoryType("EXPORT"), "%INSERT%"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListQueryHistoryFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err, tt.description)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err, tt.description)
			require.NotNil(t, q)

			sql, args, err := q.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql, tt.description)
			require.Equal(t, tt.wantArgs, args, tt.description)
		})
	}
}

// TestGetListQueryHistoryFilter_CompareWithOriginal verifies that the new implementation
// produces equivalent SQL to the original v1 implementation
func TestGetListQueryHistoryFilter_CompareWithOriginal(t *testing.T) {
	tests := []struct {
		name               string
		filter             string
		expectedConditions []string // The conditions we expect in the WHERE clause
	}{
		{
			name:               "simple project filter",
			filter:             `project == "projects/test-project"`,
			expectedConditions: []string{"query_history.project_id = $1"},
		},
		{
			name:               "database filter",
			filter:             `database == "instances/test-instance/databases/test-db"`,
			expectedConditions: []string{"query_history.database = $1"},
		},
		{
			name:               "type and project combined",
			filter:             `type == "QUERY" && project == "projects/my-project"`,
			expectedConditions: []string{"query_history.type = $1", "AND", "query_history.project_id = $2"},
		},
		{
			name:               "statement matches",
			filter:             `statement.matches("SELECT")`,
			expectedConditions: []string{"query_history.statement LIKE $1"},
		},
		{
			name:               "OR condition",
			filter:             `type == "QUERY" || type == "EXPORT"`,
			expectedConditions: []string{"query_history.type = $1", "OR", "query_history.type = $2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListQueryHistoryFilter(tt.filter)
			require.NoError(t, err)
			require.NotNil(t, q)

			sql, _, err := q.ToSQL()
			require.NoError(t, err)

			// Verify that all expected conditions are present in the SQL
			for _, condition := range tt.expectedConditions {
				require.Contains(t, sql, condition, "SQL should contain expected condition")
			}
		})
	}
}

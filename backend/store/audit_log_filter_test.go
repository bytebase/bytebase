package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return parsedTime
}

func TestGetSearchAuditLogsFilter_WithoutUserLookup(t *testing.T) {
	// Note: This test skips user filter tests since they require database access
	// User filter functionality is tested in integration tests

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
			name:     "resource filter",
			filter:   `resource == "projects/test-project"`,
			wantSQL:  "(payload->>'resource' = $1)",
			wantArgs: []any{"projects/test-project"},
			wantErr:  false,
		},
		{
			name:     "method filter",
			filter:   `method == "bytebase.v1.ProjectService.GetProject"`,
			wantSQL:  "(payload->>'method' = $1)",
			wantArgs: []any{"bytebase.v1.ProjectService.GetProject"},
			wantErr:  false,
		},
		{
			name:     "severity filter",
			filter:   `severity == "INFO"`,
			wantSQL:  "(payload->>'severity' = $1)",
			wantArgs: []any{"INFO"},
			wantErr:  false,
		},
		{
			name:     "create_time greater than or equal",
			filter:   `create_time >= "2024-01-01T00:00:00Z"`,
			wantSQL:  "(created_at >= $1)",
			wantArgs: []any{mustParseTime(t, "2024-01-01T00:00:00Z")},
			wantErr:  false,
		},
		{
			name:     "create_time less than or equal",
			filter:   `create_time <= "2024-12-31T23:59:59Z"`,
			wantSQL:  "(created_at <= $1)",
			wantArgs: []any{mustParseTime(t, "2024-12-31T23:59:59Z")},
			wantErr:  false,
		},
		{
			name:     "AND condition with resource and method",
			filter:   `resource == "projects/test-project" && method == "bytebase.v1.ProjectService.GetProject"`,
			wantSQL:  "((payload->>'resource' = $1 AND payload->>'method' = $2))",
			wantArgs: []any{"projects/test-project", "bytebase.v1.ProjectService.GetProject"},
			wantErr:  false,
		},
		{
			name:     "AND condition with three filters",
			filter:   `resource == "projects/test" && method == "bytebase.v1.ProjectService.GetProject" && severity == "INFO"`,
			wantSQL:  "(((payload->>'resource' = $1 AND payload->>'method' = $2) AND payload->>'severity' = $3))",
			wantArgs: []any{"projects/test", "bytebase.v1.ProjectService.GetProject", "INFO"},
			wantErr:  false,
		},
		{
			name:     "OR condition with resource",
			filter:   `resource == "projects/project1" || resource == "projects/project2"`,
			wantSQL:  "((payload->>'resource' = $1 OR payload->>'resource' = $2))",
			wantArgs: []any{"projects/project1", "projects/project2"},
			wantErr:  false,
		},
		{
			name:     "OR condition with severity",
			filter:   `severity == "INFO" || severity == "WARNING"`,
			wantSQL:  "((payload->>'severity' = $1 OR payload->>'severity' = $2))",
			wantArgs: []any{"INFO", "WARNING"},
			wantErr:  false,
		},
		{
			name:     "complex nested AND/OR",
			filter:   `(resource == "projects/p1" || resource == "projects/p2") && severity == "INFO"`,
			wantSQL:  "(((payload->>'resource' = $1 OR payload->>'resource' = $2) AND payload->>'severity' = $3))",
			wantArgs: []any{"projects/p1", "projects/p2", "INFO"},
			wantErr:  false,
		},
		{
			name:     "time range filter",
			filter:   `create_time >= "2024-01-01T00:00:00Z" && create_time <= "2024-12-31T23:59:59Z"`,
			wantSQL:  "((created_at >= $1 AND created_at <= $2))",
			wantArgs: []any{mustParseTime(t, "2024-01-01T00:00:00Z"), mustParseTime(t, "2024-12-31T23:59:59Z")},
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
			errContains: "unknown variable",
		},
		{
			name:        "invalid time format",
			filter:      `create_time >= "invalid-time"`,
			wantErr:     true,
			errContains: "failed to parse time",
		},
		{
			name:        "time comparison on wrong field",
			filter:      `resource >= "2024-01-01T00:00:00Z"`,
			wantErr:     true,
			errContains: `">=" and "<=" are only supported for "create_time"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetSearchAuditLogsFilter(tt.filter)

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
			if tt.wantArgs != nil {
				require.Equal(t, tt.wantArgs, args)
			}
		})
	}
}

func TestGetSearchAuditLogsFilter_EdgeCases(t *testing.T) {
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
			name:        "resource with special characters",
			filter:      `resource == "projects/test-project/databases/test_db"`,
			description: "resource with special characters should be handled correctly",
			wantSQL:     "(payload->>'resource' = $1)",
			wantArgs:    []any{"projects/test-project/databases/test_db"},
			wantErr:     false,
		},
		{
			name:        "multiple OR conditions",
			filter:      `severity == "INFO" || severity == "WARNING" || severity == "ERROR"`,
			description: "multiple OR conditions should be chained",
			wantSQL:     "(((payload->>'severity' = $1 OR payload->>'severity' = $2) OR payload->>'severity' = $3))",
			wantArgs:    []any{"INFO", "WARNING", "ERROR"},
			wantErr:     false,
		},
		{
			name:        "complex filter with all supported fields",
			filter:      `resource == "projects/test" && method == "bytebase.v1.ProjectService.GetProject" && severity == "INFO" && create_time >= "2024-01-01T00:00:00Z"`,
			description: "combination of all supported filter types",
			wantSQL:     "(((payload->>'resource' = $1 AND payload->>'method' = $2) AND (payload->>'severity' = $3 AND created_at >= $4)))",
			wantArgs:    []any{"projects/test", "bytebase.v1.ProjectService.GetProject", "INFO", mustParseTime(t, "2024-01-01T00:00:00Z")},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetSearchAuditLogsFilter(tt.filter)

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
			if tt.wantArgs != nil {
				require.Equal(t, tt.wantArgs, args, tt.description)
			}
		})
	}
}

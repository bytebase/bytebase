package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetListPlanFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
		skipTest    bool // Skip tests that require database access
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSQL:  "",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name:     "has_rollout filter - true",
			filter:   `has_rollout == true`,
			wantSQL:  "(plan.config->>'hasRollout' = $1)",
			wantArgs: []any{"true"},
			wantErr:  false,
		},
		{
			name:     "has_rollout filter - false",
			filter:   `has_rollout == false`,
			wantSQL:  "((plan.config->>'hasRollout' IS NULL OR plan.config->>'hasRollout' = $1))",
			wantArgs: []any{"false"},
			wantErr:  false,
		},
		{
			name:     "has_issue filter - true",
			filter:   `has_issue == true`,
			wantSQL:  "(issue.id IS NOT NULL)",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "has_issue filter - false",
			filter:   `has_issue == false`,
			wantSQL:  "(issue.id IS NULL)",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "title filter",
			filter:   `title == "My Plan"`,
			wantSQL:  "(plan.name = $1)",
			wantArgs: []any{"My Plan"},
			wantErr:  false,
		},
		{
			name:     "title matches",
			filter:   `title.matches("test")`,
			wantSQL:  "(LOWER(plan.name) LIKE $1)",
			wantArgs: []any{"%test%"},
			wantErr:  false,
		},
		{
			name:     "spec_type filter - create_database_config",
			filter:   `spec_type == "create_database_config"`,
			wantSQL:  "(EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'createDatabaseConfig' IS NOT NULL))",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "spec_type filter - change_database_config",
			filter:   `spec_type == "change_database_config"`,
			wantSQL:  "(EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'changeDatabaseConfig' IS NOT NULL))",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "spec_type filter - export_data_config",
			filter:   `spec_type == "export_data_config"`,
			wantSQL:  "(EXISTS (SELECT 1 FROM jsonb_array_elements(plan.config->'specs') AS spec WHERE spec->>'exportDataConfig' IS NOT NULL))",
			wantArgs: []any{},
			wantErr:  false,
		},
		{
			name:     "state filter - ACTIVE",
			filter:   `state == "STATE_ACTIVE"`,
			wantSQL:  "(plan.deleted = $1)",
			wantArgs: []any{false},
			wantErr:  false,
		},
		{
			name:     "state filter - DELETED",
			filter:   `state == "STATE_DELETED"`,
			wantSQL:  "(plan.deleted = $1)",
			wantArgs: []any{true},
			wantErr:  false,
		},
		{
			name:    "create_time greater than or equal",
			filter:  `create_time >= "2024-01-01T00:00:00Z"`,
			wantSQL: "(plan.created_at >= $1)",
			wantArgs: []any{func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:    "create_time less than or equal",
			filter:  `create_time <= "2024-12-31T23:59:59Z"`,
			wantSQL: "(plan.created_at <= $1)",
			wantArgs: []any{func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-12-31T23:59:59Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:     "AND condition with has_rollout and has_issue",
			filter:   `has_rollout == true && has_issue == true`,
			wantSQL:  "((plan.config->>'hasRollout' = $1 AND issue.id IS NOT NULL))",
			wantArgs: []any{"true"},
			wantErr:  false,
		},
		{
			name:     "complex AND condition",
			filter:   `title == "Test Plan" && state == "STATE_ACTIVE" && has_rollout == true`,
			wantSQL:  "(((plan.name = $1 AND plan.deleted = $2) AND plan.config->>'hasRollout' = $3))",
			wantArgs: []any{"Test Plan", false, "true"},
			wantErr:  false,
		},
		{
			name:        "creator filter requires database",
			filter:      `creator == "users/test@example.com"`,
			skipTest:    true,
			wantErr:     false, // Would work with database
			errContains: "",
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
			errContains: "unsupported variable",
		},
		{
			name:        "invalid state value",
			filter:      `state == "INVALID_STATE"`,
			wantErr:     true,
			errContains: "invalid state filter",
		},
		{
			name:        "invalid spec_type value",
			filter:      `spec_type == "invalid_type"`,
			wantErr:     true,
			errContains: "invalid spec_type value",
		},
		{
			name:        "has_rollout with non-bool value",
			filter:      `has_rollout == "true"`,
			wantErr:     true,
			errContains: `"has_rollout" should be bool`,
		},
		{
			name:        "has_issue with non-bool value",
			filter:      `has_issue == "true"`,
			wantErr:     true,
			errContains: `"has_issue" should be bool`,
		},
		{
			name:        "invalid time format",
			filter:      `create_time >= "invalid-time"`,
			wantErr:     true,
			errContains: "failed to parse time",
		},
		{
			name:        "comparison operator on unsupported field",
			filter:      `title >= "test"`,
			wantErr:     true,
			errContains: `">=" and "<=" are only supported for "create_time"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Test requires database connection")
			}

			q, err := GetListPlanFilter(tt.filter)

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

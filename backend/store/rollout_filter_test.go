package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetListRolloutFilter(t *testing.T) {
	// The subquery used for update_time filtering
	updatedAtSubquery := `COALESCE((SELECT MAX(task_run.updated_at) FROM task JOIN task_run ON task_run.task_id = task.id WHERE task.plan_id = plan.id), plan.created_at)`

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
			name:     "task_type in filter",
			filter:   `task_type in ["DATABASE_MIGRATE", "DATABASE_EXPORT"]`,
			wantSQL:  "(EXISTS (SELECT 1 FROM task WHERE task.plan_id = plan.id AND task.type = ANY($1)))",
			wantArgs: []any{[]string{"DATABASE_MIGRATE", "DATABASE_EXPORT"}},
			wantErr:  false,
		},
		{
			name:    "update_time greater than or equal",
			filter:  `update_time >= "2024-01-01T00:00:00Z"`,
			wantSQL: "(" + updatedAtSubquery + " >= $1)",
			wantArgs: []any{func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:    "update_time less than or equal",
			filter:  `update_time <= "2024-12-31T23:59:59Z"`,
			wantSQL: "(" + updatedAtSubquery + " <= $1)",
			wantArgs: []any{func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-12-31T23:59:59Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:    "update_time with ISO format including milliseconds",
			filter:  `update_time >= "2024-01-01T00:00:00.000Z"`,
			wantSQL: "(" + updatedAtSubquery + " >= $1)",
			wantArgs: []any{func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00.000Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:    "AND condition with task_type and update_time",
			filter:  `task_type in ["DATABASE_MIGRATE"] && update_time >= "2024-01-01T00:00:00Z"`,
			wantSQL: "((EXISTS (SELECT 1 FROM task WHERE task.plan_id = plan.id AND task.type = ANY($1)) AND " + updatedAtSubquery + " >= $2))",
			wantArgs: []any{[]string{"DATABASE_MIGRATE"}, func() time.Time {
				ts, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				return ts
			}()},
			wantErr: false,
		},
		{
			name:        "invalid filter syntax",
			filter:      `invalid syntax {{`,
			wantErr:     true,
			errContains: "failed to parse filter",
		},
		{
			name:        "unsupported variable",
			filter:      `unsupported in ["value"]`,
			wantErr:     true,
			errContains: "unsupported variable",
		},
		{
			name:        "invalid task_type value",
			filter:      `task_type in ["INVALID_TYPE"]`,
			wantErr:     true,
			errContains: "invalid task_type value",
		},
		{
			name:        "invalid time format",
			filter:      `update_time >= "invalid-time"`,
			wantErr:     true,
			errContains: "failed to parse time",
		},
		{
			name:        "comparison operator on unsupported field",
			filter:      `creator >= "test"`,
			wantErr:     true,
			errContains: `">=" and "<=" are only supported for "update_time"`,
		},
		{
			name:        "task_type with non-string value",
			filter:      `task_type in [123]`,
			wantErr:     true,
			errContains: "task_type value must be a string",
		},
		{
			name:        "task_type in with empty list",
			filter:      `task_type in []`,
			wantErr:     true,
			errContains: "empty list value for filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListRolloutFilter(tt.filter)

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

package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

func TestParseChangelogFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantStatus  *store.ChangelogStatus
		wantAfter   *time.Time
		wantBefore  *time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:   "empty filter",
			filter: "",
		},
		{
			name:       "status filter",
			filter:     `status == "DONE"`,
			wantStatus: ptr(store.ChangelogStatusDone),
		},
		{
			name:      "create_time greater than or equal",
			filter:    `create_time >= "2024-01-01T00:00:00Z"`,
			wantAfter: ptr(mustParseTime(t, "2024-01-01T00:00:00Z")),
		},
		{
			name:       "create_time less than or equal",
			filter:     `create_time <= "2024-12-31T23:59:59Z"`,
			wantBefore: ptr(mustParseTime(t, "2024-12-31T23:59:59Z")),
		},
		{
			name:       "create_time range",
			filter:     `create_time >= "2024-01-01T00:00:00Z" && create_time <= "2024-01-02T00:00:00Z"`,
			wantAfter:  ptr(mustParseTime(t, "2024-01-01T00:00:00Z")),
			wantBefore: ptr(mustParseTime(t, "2024-01-02T00:00:00Z")),
		},
		{
			name:       "combined status and time filter",
			filter:     `status == "DONE" && create_time >= "2024-01-01T00:00:00Z"`,
			wantStatus: ptr(store.ChangelogStatusDone),
			wantAfter:  ptr(mustParseTime(t, "2024-01-01T00:00:00Z")),
		},
		{
			name:        "invalid time format",
			filter:      `create_time >= "invalid-time"`,
			wantErr:     true,
			errContains: "failed to parse time",
		},
		{
			name:        "time comparison on wrong field",
			filter:      `status >= "2024-01-01T00:00:00Z"`,
			wantErr:     true,
			errContains: `">=" and "<=" are only supported for "create_time"`,
		},
		{
			name:        "unsupported variable",
			filter:      `unknown == "value"`,
			wantErr:     true,
			errContains: "unsupport variable",
		},
		{
			name:      "create_time with timezone offset",
			filter:    `create_time >= "2024-01-01T00:00:00+05:30"`,
			wantAfter: ptr(mustParseTime(t, "2024-01-01T00:00:00+05:30")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			find := &store.FindChangelogMessage{}
			err := parseChangelogFilter(tt.filter, find)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.wantStatus != nil {
				require.NotNil(t, find.Status)
				require.Equal(t, *tt.wantStatus, *find.Status)
			}
			if tt.wantAfter != nil {
				require.NotNil(t, find.CreatedAtAfter)
				require.Equal(t, *tt.wantAfter, *find.CreatedAtAfter)
			}
			if tt.wantBefore != nil {
				require.NotNil(t, find.CreatedAtBefore)
				require.Equal(t, *tt.wantBefore, *find.CreatedAtBefore)
			}
		})
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return parsedTime
}

func ptr[T any](v T) *T {
	return &v
}

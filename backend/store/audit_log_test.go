package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/qb"
)

func TestApplyRetentionFilter(t *testing.T) {
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	tests := []struct {
		name           string
		userFilterQ    *qb.Query
		cutoff         *time.Time
		wantSQLPattern []string // Pattern to match in SQL
		description    string
	}{
		{
			name:        "No retention cutoff (Enterprise plan)",
			userFilterQ: qb.Q().Space("payload->>'method' = ?", "/bytebase.v1.SQLService/Query"),
			cutoff:      nil,
			wantSQLPattern: []string{
				"payload->>'method' = $1",
			},
			description: "Should return user filter unchanged when no cutoff",
		},
		{
			name:        "Apply retention to empty filter (Team plan)",
			userFilterQ: nil,
			cutoff:      &sevenDaysAgo,
			wantSQLPattern: []string{
				"created_at >= $1",
			},
			description: "Should create retention filter when no user filter",
		},
		{
			name:        "Merge retention with existing filter (Team plan)",
			userFilterQ: qb.Q().Space("payload->>'method' = ?", "/bytebase.v1.SQLService/Query"),
			cutoff:      &sevenDaysAgo,
			wantSQLPattern: []string{
				"payload->>'method' = $1",
				"AND",
				"created_at >= $2",
			},
			description: "Should combine user filter with retention filter using AND",
		},
		{
			name:        "Complex filter with retention",
			userFilterQ: qb.Q().Space("payload->>'method' = ?", "/bytebase.v1.SQLService/Query").And("payload->>'severity' = ?", "ERROR"),
			cutoff:      &sevenDaysAgo,
			wantSQLPattern: []string{
				"payload->>'method' = $1",
				"AND",
				"payload->>'severity' = $2",
				"AND",
				"created_at >= $3",
			},
			description: "Should handle complex filters with multiple conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyRetentionFilter(tt.userFilterQ, tt.cutoff)

			require.NotNil(t, got, tt.description)

			sql, args, err := got.ToSQL()
			require.NoError(t, err)

			// Check that all expected patterns are in the SQL
			for _, pattern := range tt.wantSQLPattern {
				assert.Contains(t, sql, pattern, "SQL should contain expected pattern: %s", pattern)
			}

			// For time comparisons in args, we just check the type
			hasTimeArg := false
			for _, arg := range args {
				if _, ok := arg.(time.Time); ok {
					hasTimeArg = true
					break
				}
			}

			if tt.cutoff != nil {
				assert.True(t, hasTimeArg, "Should have time argument when cutoff is set")
			}
		})
	}
}

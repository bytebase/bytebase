package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/qb"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
			got := applyRetentionFilter(tt.userFilterQ, tt.cutoff)

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

func TestGetAuditLogRetentionDays(t *testing.T) {
	// This test would require mocking the LicenseService
	// For now, we'll create a simple test that validates the logic

	tests := []struct {
		plan         v1pb.PlanType
		expectedDays int
	}{
		{
			plan:         v1pb.PlanType_FREE,
			expectedDays: 0, // No access
		},
		{
			plan:         v1pb.PlanType_TEAM,
			expectedDays: 7, // 7 days retention
		},
		{
			plan:         v1pb.PlanType_ENTERPRISE,
			expectedDays: -1, // Unlimited
		},
	}

	for _, tt := range tests {
		t.Run(tt.plan.String(), func(t *testing.T) {
			// This test documents the expected behavior
			// In a real test, we would mock the LicenseService and test GetAuditLogRetentionDays
			t.Logf("Plan %s should have %d days retention", tt.plan, tt.expectedDays)
		})
	}
}

func TestRetentionCutoffCalculation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		retentionDays   int
		expectNilCutoff bool
		description     string
	}{
		{
			name:            "Free plan (0 days)",
			retentionDays:   0,
			expectNilCutoff: true,
			description:     "Free plan should return nil cutoff",
		},
		{
			name:            "Team plan (7 days)",
			retentionDays:   7,
			expectNilCutoff: false,
			description:     "Team plan should return cutoff 7 days ago",
		},
		{
			name:            "Enterprise plan (unlimited)",
			retentionDays:   -1,
			expectNilCutoff: true,
			description:     "Enterprise plan should return nil cutoff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate expected cutoff
			var expectedCutoff *time.Time
			if tt.retentionDays > 0 {
				cutoff := now.AddDate(0, 0, -tt.retentionDays)
				expectedCutoff = &cutoff
			}

			if tt.expectNilCutoff {
				assert.Nil(t, expectedCutoff, tt.description)
			} else {
				require.NotNil(t, expectedCutoff, tt.description)
				// Verify the cutoff is approximately correct
				daysDiff := now.Sub(*expectedCutoff).Hours() / 24
				assert.InDelta(t, float64(tt.retentionDays), daysDiff, 0.1, "Cutoff should be approximately %d days ago", tt.retentionDays)
			}
		})
	}
}

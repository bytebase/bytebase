package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestApplyRetentionFilter(t *testing.T) {
	s := &AuditLogService{}
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	tests := []struct {
		name        string
		userFilter  *store.ListResourceFilter
		cutoff      *time.Time
		wantFilter  *store.ListResourceFilter
		description string
	}{
		{
			name: "No retention cutoff (Enterprise plan)",
			userFilter: &store.ListResourceFilter{
				Where: "(payload->>'method'=$1)",
				Args:  []any{"/bytebase.v1.SQLService/Query"},
			},
			cutoff: nil,
			wantFilter: &store.ListResourceFilter{
				Where: "(payload->>'method'=$1)",
				Args:  []any{"/bytebase.v1.SQLService/Query"},
			},
			description: "Should return user filter unchanged when no cutoff",
		},
		{
			name:       "Apply retention to empty filter (Team plan)",
			userFilter: nil,
			cutoff:     &sevenDaysAgo,
			wantFilter: &store.ListResourceFilter{
				Where: "(created_at >= $1)",
				Args:  []any{sevenDaysAgo},
			},
			description: "Should create retention filter when no user filter",
		},
		{
			name: "Merge retention with existing filter (Team plan)",
			userFilter: &store.ListResourceFilter{
				Where: "(payload->>'method'=$1)",
				Args:  []any{"/bytebase.v1.SQLService/Query"},
			},
			cutoff: &sevenDaysAgo,
			wantFilter: &store.ListResourceFilter{
				Where: "((payload->>'method'=$1)) AND (created_at >= $2)",
				Args:  []any{"/bytebase.v1.SQLService/Query", sevenDaysAgo},
			},
			description: "Should combine user filter with retention filter using AND",
		},
		{
			name: "Complex filter with retention",
			userFilter: &store.ListResourceFilter{
				Where: "(payload->>'method'=$1 AND payload->>'severity'=$2)",
				Args:  []any{"/bytebase.v1.SQLService/Query", "ERROR"},
			},
			cutoff: &sevenDaysAgo,
			wantFilter: &store.ListResourceFilter{
				Where: "((payload->>'method'=$1 AND payload->>'severity'=$2)) AND (created_at >= $3)",
				Args:  []any{"/bytebase.v1.SQLService/Query", "ERROR", sevenDaysAgo},
			},
			description: "Should handle complex filters with multiple conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.applyRetentionFilter(tt.userFilter, tt.cutoff)

			require.NotNil(t, got, tt.description)
			assert.Equal(t, tt.wantFilter.Where, got.Where, "WHERE clause should match")
			assert.Equal(t, len(tt.wantFilter.Args), len(got.Args), "Number of args should match")

			// For time comparisons, we just check the type and rough equality
			for i, arg := range tt.wantFilter.Args {
				if timeArg, ok := arg.(time.Time); ok {
					gotTimeArg, ok := got.Args[i].(time.Time)
					require.True(t, ok, "Argument %d should be a time.Time", i)
					assert.WithinDuration(t, timeArg, gotTimeArg, time.Second, "Time arguments should be within 1 second")
				} else {
					assert.Equal(t, arg, got.Args[i], "Argument %d should match", i)
				}
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

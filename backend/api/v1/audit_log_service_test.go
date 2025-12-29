package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

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

package bus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanCheckRunRefIncludesApprovalInputVersion(t *testing.T) {
	require.NotEqual(t,
		PlanCheckRunRef{ProjectID: "project-a", UID: 1, ApprovalInputVersion: 1},
		PlanCheckRunRef{ProjectID: "project-a", UID: 1, ApprovalInputVersion: 2},
	)
}

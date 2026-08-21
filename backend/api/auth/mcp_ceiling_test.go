package auth

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestClassifyMCPCeiling pins the one split three doors depend on.
//
// The line that matters is between a stored value and a failed read. A value
// nobody can interpret never succeeds on retry, so telling a client to retry is
// a lie; a read that failed is an outage, so blaming an admin is a lie the
// other way. Everything this build cannot act on refuses either way, which is
// what makes the distinction about the message and the audit row rather than
// about access.
func TestClassifyMCPCeiling(t *testing.T) {
	unreadable := errors.Wrapf(store.ErrMCPCapabilityUnreadable, "READ_ONLYY is not a value this build understands")

	tests := []struct {
		name       string
		capability storepb.MCPSetting_Capability
		err        error
		want       MCPCeilingVerdict
		policy     bool
	}{
		{"read-write serves", storepb.MCPSetting_READ_WRITE, nil, MCPCeilingServes, false},
		{"read-only serves", storepb.MCPSetting_READ_ONLY, nil, MCPCeilingServes, false},
		{"disabled", storepb.MCPSetting_DISABLED, nil, MCPCeilingDisabled, true},
		{"the reserved number", storepb.MCPSetting_Capability(2), nil, MCPCeilingUnserved, true},
		{"a value from a newer build", storepb.MCPSetting_Capability(99), nil, MCPCeilingUnserved, true},
		// The store resolves an absent setting to READ_WRITE and an explicit
		// unset to the unreadable error, so a zero value arriving here was
		// resolved by nobody.
		{"nobody resolved it", storepb.MCPSetting_CAPABILITY_UNSPECIFIED, nil, MCPCeilingUnserved, true},
		{"a stored value nobody can read", storepb.MCPSetting_CAPABILITY_UNSPECIFIED, unreadable, MCPCeilingUnreadable, true},
		// The error wins over the value, whatever the value happens to be.
		{"an unreadable value that parsed to something", storepb.MCPSetting_READ_WRITE, unreadable, MCPCeilingUnreadable, true},
		{"the read failed", storepb.MCPSetting_CAPABILITY_UNSPECIFIED, errors.New("connection refused"), MCPCeilingUnavailable, false},
		{"the read failed on a permissive workspace", storepb.MCPSetting_READ_WRITE, errors.New("connection refused"), MCPCeilingUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyMCPCeiling(tt.capability, tt.err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.policy, got.IsPolicy())
		})
	}
}

// TestMCPCeilingVerdictAdmissionMatchesTheServesPredicate holds the classifier
// against the predicate the serving table already pins: exactly one verdict may
// admit work, and it must be the one MCPCeilingServesAnything agrees with over
// the whole enum. A mode that starts or stops serving a class cannot leave one
// of them stale.
func TestMCPCeilingVerdictAdmissionMatchesTheServesPredicate(t *testing.T) {
	values := storepb.MCPSetting_Capability(0).Descriptor().Values()
	for i := range values.Len() {
		capability := storepb.MCPSetting_Capability(values.Get(i).Number())
		admits := ClassifyMCPCeiling(capability, nil) == MCPCeilingServes
		require.Equal(t, MCPCeilingServesAnything(capability), admits,
			"%v must admit work in exactly one of the two", capability)
	}
}

// TestPolicyVerdictsAreExactlyTheAuditedOnes holds the list every door renders
// wording from against the predicate that decides whether a refusal is
// recorded. A verdict in one and not the other would either be refused with no
// sentence or audited as a decision nobody made.
func TestPolicyVerdictsAreExactlyTheAuditedOnes(t *testing.T) {
	policy := map[MCPCeilingVerdict]bool{}
	for _, v := range PolicyMCPCeilingVerdicts() {
		require.True(t, v.IsPolicy(), "%v is listed as policy but does not report as one", v)
		policy[v] = true
	}
	for _, v := range []MCPCeilingVerdict{
		MCPCeilingServes, MCPCeilingDisabled, MCPCeilingUnreadable, MCPCeilingUnserved, MCPCeilingUnavailable,
	} {
		require.Equal(t, v.IsPolicy(), policy[v], "%v disagrees between the list and the predicate", v)
	}
}

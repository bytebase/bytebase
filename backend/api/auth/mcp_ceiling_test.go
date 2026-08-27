package auth

import (
	"strings"
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
		name     string
		settings *storepb.MCPSetting
		err      error
		want     MCPCeilingVerdict
		policy   bool
	}{
		{"read-write serves", &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE}, nil, MCPCeilingServes, false},
		{"read-only serves", &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_ONLY}, nil, MCPCeilingServes, false},
		{"disabled", &storepb.MCPSetting{Capability: storepb.MCPSetting_DISABLED}, nil, MCPCeilingDisabled, true},
		{"the reserved number", &storepb.MCPSetting{Capability: storepb.MCPSetting_Capability(2)}, nil, MCPCeilingUnserved, true},
		{"a value from a newer build", &storepb.MCPSetting{Capability: storepb.MCPSetting_Capability(99)}, nil, MCPCeilingUnserved, true},
		{"nobody resolved it", nil, nil, MCPCeilingUnavailable, false},
		{"a stored value nobody can read", nil, unreadable, MCPCeilingUnreadable, true},
		// The error wins over the value, whatever the value happens to be.
		{"an unreadable value that parsed to something", &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE}, unreadable, MCPCeilingUnreadable, true},
		{"the read failed", nil, errors.New("connection refused"), MCPCeilingUnavailable, false},
		{"the read failed on a permissive workspace", &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_WRITE}, errors.New("connection refused"), MCPCeilingUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyMCPCeiling(tt.settings, tt.err)
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
		admits := ClassifyMCPCeiling(&storepb.MCPSetting{Capability: capability}, nil) == MCPCeilingServes
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

// TestEveryVerdictThatRefusesHasWording holds the one wording table against the
// verdicts that reach a caller. It replaces a per-door version of this check
// that each door carried: while the sentences were per-door, coverage was all a
// lint could hold them to — the wording itself was free to drift, and it did.
//
// The list spans all five values, not PolicyMCPCeilingVerdicts, because an
// outage is refused too and its sentence is the one nobody would otherwise
// exercise. Only Serves is silent.
func TestEveryVerdictThatRefusesHasWording(t *testing.T) {
	for _, v := range []MCPCeilingVerdict{
		MCPCeilingDisabled, MCPCeilingUnreadable, MCPCeilingUnserved, MCPCeilingUnavailable,
	} {
		refusal := v.Refusal()
		require.NotEmpty(t, refusal, "%v reaches a caller and must say what is wrong", v)

		// Composed into a larger error at every door but the consent page, so
		// it starts lowercase and ends unterminated.
		require.Equal(t, strings.ToLower(refusal[:1]), refusal[:1],
			"%v: a door prefixes this, so it must not start a sentence", v)
		require.NotEqual(t, ".", refusal[len(refusal)-1:],
			"%v: the consent page terminates it; the others compose it", v)

		// Every refusal names the remedy, not only the fault. A denial an
		// operator cannot act on is the failure this series exists to fix.
		require.Contains(t, refusal, "workspace settings", "%v must name where the fix is", v)
	}

	require.Empty(t, MCPCeilingServes.Refusal(), "serving refuses nothing")
}

// TestRefusalsDistinguishTheThreeStoredStates pins that the three policy
// verdicts do not collapse into one message. They have different fixes — turn
// MCP back on, rewrite a value nobody can read, choose a value this build
// serves — and a door that said "disabled" for all three would send two of them
// to the wrong control.
func TestRefusalsDistinguishTheThreeStoredStates(t *testing.T) {
	seen := map[string]MCPCeilingVerdict{}
	for _, v := range PolicyMCPCeilingVerdicts() {
		refusal := v.Refusal()
		if other, dup := seen[refusal]; dup {
			require.Failf(t, "verdicts share wording", "%v and %v say the same thing", other, v)
		}
		seen[refusal] = v
	}
	require.Contains(t, MCPCeilingDisabled.Refusal(), "turned MCP access off")
	require.Contains(t, MCPCeilingUnreadable.Refusal(), "not one this build understands")
	require.Contains(t, MCPCeilingUnserved.Refusal(), "not one this build serves")
}

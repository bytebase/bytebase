package auth

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
		{"unspecified", &storepb.MCPSetting{Capability: storepb.MCPSetting_CAPABILITY_UNSPECIFIED}, nil, MCPCeilingUnserved, true},
		{"the reserved number", &storepb.MCPSetting{Capability: storepb.MCPSetting_Capability(2)}, nil, MCPCeilingUnserved, true},
		{"a value from a newer build", &storepb.MCPSetting{Capability: storepb.MCPSetting_Capability(99)}, nil, MCPCeilingUnserved, true},
		{"nobody resolved it", nil, nil, MCPCeilingUnavailable, false},
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

// TestEveryVerdictThatRefusesHasWording holds the one wording table against the
// verdicts that reach a caller. It replaces a per-door version of this check
// that each door carried: while the sentences were per-door, coverage was all a
// lint could hold them to — the wording itself was free to drift, and it did.
//
// The list spans every refusing value, including an outage. Only Serves is
// silent.
func TestEveryVerdictThatRefusesHasWording(t *testing.T) {
	for _, v := range []MCPCeilingVerdict{
		MCPCeilingDisabled, MCPCeilingUnserved, MCPCeilingUnavailable,
	} {
		refusal := v.Refusal()
		require.NotEmpty(t, refusal, "%v reaches a caller and must say what is wrong", v)
		require.NotEmpty(t, v.Heading(), "%v reaches a caller and must have a heading", v)

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
	require.Empty(t, MCPCeilingServes.Heading(), "serving has no refusal heading")
}

// TestRefusalsDistinguishTheStoredStates pins that the policy verdicts do not
// collapse into one message.
func TestRefusalsDistinguishTheStoredStates(t *testing.T) {
	seen := map[string]MCPCeilingVerdict{}
	for _, v := range []MCPCeilingVerdict{MCPCeilingDisabled, MCPCeilingUnserved} {
		refusal := v.Refusal()
		if other, dup := seen[refusal]; dup {
			require.Failf(t, "verdicts share wording", "%v and %v say the same thing", other, v)
		}
		seen[refusal] = v
	}
	require.Contains(t, MCPCeilingDisabled.Refusal(), "turned MCP access off")
	require.Contains(t, MCPCeilingUnserved.Refusal(), "not one this build serves")
}

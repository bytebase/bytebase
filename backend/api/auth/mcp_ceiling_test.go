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

		// Shown on its own by the /mcp gate, the token endpoint and the consent
		// redirect, so it is complete sentences.
		require.Equal(t, strings.ToUpper(refusal[:1]), refusal[:1], "%v: it must start a sentence", v)
		require.True(t, strings.HasSuffix(refusal, "."), "%v: it must end a sentence", v)
		for _, r := range refusal {
			require.True(t, r >= 0x20 && r <= 0x7e && r != '"' && r != '\\',
				"%v: an OAuth error_description allows printable ASCII only, without quote or backslash (RFC 6749 section 5.2), got %q", v, r)
		}

		// Every refusal names the remedy, not only the fault, where an admin
		// finds it. A denial an operator cannot act on is the failure this
		// series exists to fix.
		require.Contains(t, refusal, MCPAccessPolicyLocation, "%v must name where the fix is", v)
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
	require.Contains(t, MCPCeilingDisabled.Refusal(), "turned off MCP access")
	require.Contains(t, MCPCeilingUnserved.Refusal(), "does not support")
}

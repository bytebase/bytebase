package auth

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// MCPCeilingVerdict is what a read of a workspace's MCP capability ceiling
// means to a caller deciding whether to proceed. Every reader of the setting
// decides from it — the consent, the token endpoint, the /mcp connection gate,
// the per-request gate, and the MCP info read — so no two can disagree about a
// workspace. Every door but the info read says so with Refusal; that one
// answers the policy verdicts in its response body and keeps Refusal for the
// outage (BOT-106).
type MCPCeilingVerdict int

// Each value names what an admin would have to do about it, which is what the
// doors turn into a sentence.
const (
	// The ceiling admits work; which work is decided per method and per statement.
	MCPCeilingServes MCPCeilingVerdict = iota
	// An admin turned MCP off. Raising the ceiling fixes it.
	MCPCeilingDisabled
	// No mode this build has serves the parsed value: the reserved 2 or
	// UNSPECIFIED. An admin has to choose a known value.
	MCPCeilingUnserved
	// The read itself failed. The only one of these a retry may fix.
	MCPCeilingUnavailable
)

// IsPolicy reports whether the verdict is a decision about the workspace, which
// is what makes a refusal an audited outcome.
func (v MCPCeilingVerdict) IsPolicy() bool {
	switch v {
	case MCPCeilingDisabled, MCPCeilingUnserved:
		return true
	default:
		return false
	}
}

// ClassifyMCPCeiling turns the result of store.GetMCPSettingsUncached into the
// verdict its caller acts on. Everything this build cannot act on refuses; what
// the split decides is the message and the audit row. A value nobody can
// interpret never succeeds on retry; an outage may.
func ClassifyMCPCeiling(settings *storepb.MCPSetting, err error) MCPCeilingVerdict {
	if err != nil {
		return MCPCeilingUnavailable
	}
	if settings == nil {
		return MCPCeilingUnavailable
	}
	switch settings.Capability {
	case storepb.MCPSetting_DISABLED:
		return MCPCeilingDisabled
	case storepb.MCPSetting_READ_ONLY, storepb.MCPSetting_READ_WRITE:
		return MCPCeilingServes
	default:
		return MCPCeilingUnserved
	}
}

// MCPAccessPolicyLocation is where an admin changes the ceiling, named the way
// the console's navigation names it.
const MCPAccessPolicyLocation = "Integration > MCP > Access policy"

// Refusal is what a door tells the caller it is refusing: what is wrong with
// this workspace's ceiling, and what an admin does about it. Empty for
// MCPCeilingServes, which refuses nothing.
//
// Door-neutral on purpose. Each door used to keep its own copy ending in "so
// the connection fails closed" / "so authorization fails closed", and the
// copies drifted — the token endpoint borrowed the consent's clause, so a
// refused refresh told the client that no client could be authorized. Which
// door refused is already carried by the status, the page, and the audit row's
// method, so the sentence does not say it again.
//
// Complete sentences, because the /mcp connection gate, the token endpoint and
// the consent redirect show it on its own. ASCII only: the last two carry it in
// an OAuth error_description, which RFC 6749 limits to printable ASCII.
func (v MCPCeilingVerdict) Refusal() string {
	switch v {
	case MCPCeilingDisabled:
		return "A workspace admin has turned off MCP access for this workspace. " +
			"Ask a workspace admin to choose Read-only or Read-write under " + MCPAccessPolicyLocation + "."
	case MCPCeilingUnserved:
		return "This workspace's MCP access policy is set to a value this version of Bytebase does not support. " +
			"Ask a workspace admin to choose a policy under " + MCPAccessPolicyLocation + "."
	case MCPCeilingUnavailable:
		return "Bytebase could not read this workspace's MCP access policy. " +
			"Retry shortly; if it keeps failing, ask a workspace admin to check " + MCPAccessPolicyLocation + "."
	default:
		return ""
	}
}

// Heading returns the short title for a refusal. Empty for MCPCeilingServes,
// which refuses nothing.
func (v MCPCeilingVerdict) Heading() string {
	switch v {
	case MCPCeilingDisabled:
		return "MCP access is turned off"
	case MCPCeilingUnserved:
		return "This workspace's MCP access policy isn't supported by this version of Bytebase"
	case MCPCeilingUnavailable:
		return "The MCP access policy is temporarily unavailable"
	default:
		return ""
	}
}

package auth

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// MCPCeilingVerdict is what a read of a workspace's MCP capability ceiling
// means to a caller deciding whether to proceed. Every reader of the setting
// decides from it and says so with Refusal — the consent, the token endpoint,
// the /mcp connection gate, the per-request gate, and the mode-contents read —
// so no two can disagree about a workspace or describe it two ways.
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
// Lowercase and unterminated: every door but the consent page composes this
// into a larger error, and that page ends the sentence itself.
func (v MCPCeilingVerdict) Refusal() string {
	switch v {
	case MCPCeilingDisabled:
		return "a workspace admin has turned MCP access off for this workspace. " +
			"Ask them to raise the MCP ceiling in the workspace settings"
	case MCPCeilingUnserved:
		return "this workspace's stored MCP capability ceiling is not one this build serves. " +
			"Ask a workspace admin to set the MCP ceiling to a supported value in the workspace settings"
	case MCPCeilingUnavailable:
		return "this workspace's MCP capability ceiling could not be read. " +
			"Retry shortly; if it persists, ask a workspace admin to check the workspace settings"
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
		return "This workspace's MCP setting is not one this version supports"
	case MCPCeilingUnavailable:
		return "MCP settings are temporarily unavailable"
	default:
		return ""
	}
}

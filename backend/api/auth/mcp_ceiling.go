package auth

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// MCPCeilingVerdict is what a read of a workspace's MCP capability ceiling
// means to a caller deciding whether to proceed. Every reader of the setting
// decides from it — the consent, the token endpoint, the /mcp connection gate,
// the per-request gate, and the mode-contents read — so no two can disagree
// about a workspace. Every door but the mode-contents read says so with
// Refusal; that one answers the policy verdicts in its response body and keeps
// Refusal for the outage (BOT-106).
type MCPCeilingVerdict int

// Each value names what an admin would have to do about it, which is what the
// doors turn into a sentence.
const (
	// The ceiling admits work; which work is decided per method and per statement.
	MCPCeilingServes MCPCeilingVerdict = iota
	// An admin turned MCP off. Raising the ceiling fixes it.
	MCPCeilingDisabled
	// The stored value is one this build cannot interpret — a mistyped enum
	// name, a wrong-typed row. An admin has to write the setting again.
	MCPCeilingUnreadable
	// The value parsed but no mode this build has serves it: the reserved 2, or
	// a ceiling a newer release wrote. An admin has to choose a known value.
	MCPCeilingUnserved
	// The read itself failed. The only one of these a retry may fix.
	MCPCeilingUnavailable
)

// IsPolicy reports whether the verdict is a decision about the workspace, which
// is what makes a refusal an audited outcome.
func (v MCPCeilingVerdict) IsPolicy() bool {
	switch v {
	case MCPCeilingDisabled, MCPCeilingUnreadable, MCPCeilingUnserved:
		return true
	default:
		return false
	}
}

// ClassifyMCPCeiling turns the result of store.GetMCPSettingsUncached into the
// verdict its caller acts on. Everything this build cannot act on refuses; what
// the split decides is the message and the audit row. A value nobody can
// interpret never succeeds on retry; an outage may.
func ClassifyMCPCeiling(capability storepb.MCPSetting_Capability, err error) MCPCeilingVerdict {
	if err != nil {
		if errors.Is(err, store.ErrMCPCapabilityUnreadable) {
			return MCPCeilingUnreadable
		}
		return MCPCeilingUnavailable
	}
	if capability == storepb.MCPSetting_DISABLED {
		return MCPCeilingDisabled
	}
	if !MCPCeilingServesAnything(capability) {
		return MCPCeilingUnserved
	}
	return MCPCeilingServes
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
	case MCPCeilingUnreadable:
		return "this workspace's stored MCP capability ceiling is not one this build understands. " +
			"Ask a workspace admin to set the MCP ceiling again in the workspace settings"
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

// PolicyMCPCeilingVerdicts is every verdict that is a decision about the
// workspace rather than an outage, which is what decides whether a refusal is
// recorded. Refusal covers the outage too; this list does not.
func PolicyMCPCeilingVerdicts() []MCPCeilingVerdict {
	return []MCPCeilingVerdict{MCPCeilingDisabled, MCPCeilingUnreadable, MCPCeilingUnserved}
}

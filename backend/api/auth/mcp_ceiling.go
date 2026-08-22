package auth

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// MCPCeilingVerdict is what a read of a workspace's MCP capability ceiling
// means to a caller deciding whether to proceed. Four doors decide from it —
// the consent, the token endpoint, the /mcp gate, the per-request gate.
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

// PolicyMCPCeilingVerdicts is every verdict a door must have wording for. Each
// door pins its own table against this list.
func PolicyMCPCeilingVerdicts() []MCPCeilingVerdict {
	return []MCPCeilingVerdict{MCPCeilingDisabled, MCPCeilingUnreadable, MCPCeilingUnserved}
}

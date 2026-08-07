package aiaccess

import (
	"slices"
	"strings"
)

// Capability is one element of the AI access vocabulary.
//
// A capability is either an existing IAM permission string ("bb.databases.get")
// or a minted operation ID ("ai.op.rollouts.create"). Permissions cover the
// methods that carry a permission annotation; operation IDs cover the methods
// that carry none — custom-authorization methods, credential-free methods, and
// the handful of methods whose annotation does not by itself describe what the
// operation can do.
//
// The two live in one namespace because a ceiling set holds both, so an admin
// reading a set sees one list. The "ai." prefix keeps operation IDs distinct
// from "bb." IAM permissions inside that list, and it is surface-neutral: these
// are Bytebase operations that any AI surface gates identically, not MCP ones.
type Capability string

// OperationPrefix marks a capability as a minted operation ID rather than an
// IAM permission.
const OperationPrefix = "ai.op."

// IsOperation reports whether the capability is a minted operation ID.
func (c Capability) IsOperation() bool {
	return strings.HasPrefix(string(c), OperationPrefix)
}

// Requirement is what one request demands of a ceiling.
//
// The evaluator is a single containment check: a request is admitted when
// Requirement.Capabilities ⊆ the ceiling's set, and Forbidden is false. Nothing
// else — no per-method mode, no second classification axis.
type Requirement struct {
	// Forbidden marks a method no ceiling may ever admit, custom sets included.
	// It is not "requires everything": it is unconditional denial.
	Forbidden bool
	// Capabilities is the set the ceiling must contain, sorted and deduplicated.
	// Empty when Forbidden is true.
	Capabilities []Capability
	// Reason records why a method resolves the way it does. It is carried for
	// the generated inventory and for denial messages, never for the decision.
	Reason string
}

// sortedCapabilities returns a sorted, deduplicated copy.
func sortedCapabilities(capabilities []Capability) []Capability {
	out := slices.Clone(capabilities)
	slices.Sort(out)
	return slices.Compact(out)
}

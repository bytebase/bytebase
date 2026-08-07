package aiaccess

import (
	"github.com/pkg/errors"
)

// Universe returns every capability the vocabulary knows, sorted.
//
// It is derived, not listed: the capability of every v1 RPC plus the
// statement-level permissions. A capability outside it can never be required by
// anything, so it must never appear in a set — that is the second lint clause,
// and it is what 1b-5's authorable universe is bounded by.
func Universe() ([]Capability, error) {
	var capabilities []Capability
	for _, procedure := range V1Procedures() {
		requirement, err := Required(procedure, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve %q", procedure)
		}
		capabilities = append(capabilities, requirement.Capabilities...)
	}
	capabilities = append(capabilities, statementCapabilities...)
	return sortedCapabilities(capabilities), nil
}

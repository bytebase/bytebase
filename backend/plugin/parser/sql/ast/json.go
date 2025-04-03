package ast

import (
	"strings"
)

var (
	_ DataType = (*JSON)(nil)
)

// Text is the struct for the character.
type JSON struct {
	dataType
	JSONB bool
}

// EquivalentType implements the DataType interface.
func (j *JSON) EquivalentType(tp string) bool {
	if j.JSONB {
		return strings.EqualFold(tp, "jsonb")
	}
	return strings.EqualFold(tp, "json")
}

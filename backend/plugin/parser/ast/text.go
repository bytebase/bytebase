package ast

import (
	"strings"
)

var (
	_ DataType = (*Text)(nil)
)

// Text is the struct for the character.
type Text struct {
	characterType
}

// EquivalentType implements the DataType interface.
func (*Text) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	return tp == "text"
}

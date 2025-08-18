package ast

import (
	"fmt"
	"strings"
)

var (
	_ DataType = (*Character)(nil)
)

// Character is the struct for the character.
type Character struct {
	characterType

	// the storage size(characters, not bytes)
	Size int
}

// EquivalentType implements the DataType interface.
func (c *Character) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	return tp == fmt.Sprintf("char(%d)", c.Size) || tp == fmt.Sprintf("character(%d)", c.Size)
}

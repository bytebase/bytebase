package ast

import (
	"fmt"
	"strings"
)

var (
	_ DataType = (*CharacterVarying)(nil)
)

// CharacterVarying is the struct for the character varying(varchar).
type CharacterVarying struct {
	characterType

	// the storage size(characters, not bytes)
	Size int
}

// EquivalentType implements the DataType interface.
func (c *CharacterVarying) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	return tp == fmt.Sprintf("varchar(%d)", c.Size) || tp == fmt.Sprintf("character varying(%d)", c.Size)
}

package ast

import (
	"fmt"
	"strings"
)

var (
	_ DataType = (*Varchar)(nil)
)

// Varchar is the struct for the varchar.
type Varchar struct {
	characterType

	// the storage size(characters, not bytes)
	Size int
}

// EquivalentType implements the DataType interface.
func (c *Varchar) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	return tp == fmt.Sprintf("varchar(%d)", c.Size) || tp == fmt.Sprintf("character varying(%d)", c.Size)
}

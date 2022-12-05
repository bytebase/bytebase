package ast

import "strings"

// Integer is the struct for integer.
type Integer struct {
	numericType

	// the storage size(bytes)
	Size int
}

// EquivalentType implements the DataType interface.
func (i *Integer) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	switch i.Size {
	case 2:
		return tp == "smallint" || tp == "int2"
	case 4:
		return tp == "int" || tp == "int4" || tp == "integer"
	case 8:
		return tp == "bigint" || tp == "int8"
	}
	return false
}

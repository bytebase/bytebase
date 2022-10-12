package ast

import "strings"

// Float is the struct for floating-point.
type Float struct {
	numericType

	// the storage size(bytes)
	Size int
}

// EqualTypeName implements the DataType interface.
func (f *Float) EqualTypeName(tp string) bool {
	tp = strings.ToLower(tp)
	switch f.Size {
	case 4:
		return tp == "float4" || tp == "real"
	case 8:
		return tp == "float" || tp == "float8" || tp == "double precision"
	}
	return false
}

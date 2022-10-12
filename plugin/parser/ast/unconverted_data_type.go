package ast

import "strings"

// UnconvertedDataType is the struct for unconverted data type.
// TODO(rebelice): remove it.
// We define this because we cannot convert all data types now.
type UnconvertedDataType struct {
	dataType

	Name []string
}

// EqualTypeName implements the DataType interface.
func (u *UnconvertedDataType) EqualTypeName(tp string) bool {
	tp = strings.ToLower(tp)
	return strings.ToLower(strings.Join(u.Name, ".")) == tp
}

package ast

// UnconvertedDataType is the struct for unconverted data type.
// TODO(rebelice): remove it.
// We define this because we cannot convert all data types now.
type UnconvertedDataType struct {
	dataType

	Name []string
}

// EquivalentType implements the DataType interface.
func (u *UnconvertedDataType) EquivalentType(tp string) bool {
	return u.Text() == tp
}

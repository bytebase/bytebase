package ast

var (
	_ DataType    = (*dataType)(nil)
	_ NumericType = (*numericType)(nil)
)

// DataType is the interface for data type.
type DataType interface {
	Node

	EqualTypeName(string) bool
	dataTypeInterface()
}

type dataType struct {
	node
}

func (*dataType) dataTypeInterface() {}

func (*dataType) EqualTypeName(_ string) bool {
	return false
}

// NumericType is the interface for numeric type.
type NumericType interface {
	DataType

	numericTypeInterface()
}

type numericType struct {
	dataType
}

func (*numericType) numericTypeInterface() {}

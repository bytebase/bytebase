package ast

var (
	_ DataType    = (*dataType)(nil)
	_ NumericType = (*numericType)(nil)
)

// DataType is the interface for data type.
type DataType interface {
	Node

	dataTypeInterface()
}

type dataType struct {
	node
}

func (*dataType) dataTypeInterface() {}

// NumericType is the interface for numeric type.
type NumericType interface {
	DataType

	numericTypeInterface()
}

type numericType struct {
	dataType
}

func (*numericType) numericTypeInterface() {}

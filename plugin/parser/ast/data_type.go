package ast

var (
	_ DataType        = (*dataType)(nil)
	_ NumericType     = (*numericType)(nil)
	_ UserDefinedType = (*userDefinedType)(nil)
)

// DataType is the interface for data type.
type DataType interface {
	Node

	EquivalentType(string) bool
	dataTypeInterface()
}

type dataType struct {
	node
}

func (*dataType) dataTypeInterface() {}

func (*dataType) EquivalentType(_ string) bool {
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

// CharacterType is the interface for character type.
type CharacterType interface {
	DataType

	characterTypeInterface()
}

type characterType struct {
	dataType
}

func (*characterType) characterTypeInterface() {}

// UserDefinedType is the interface for user defined types.
type UserDefinedType interface {
	DataType

	TypeName() *TypeNameDef
	userDefinedTypeInterface()
}

type userDefinedType struct {
	dataType
}

func (*userDefinedType) userDefinedTypeInterface() {}
func (*userDefinedType) TypeName() *TypeNameDef    { return &TypeNameDef{} }

// TypeNameDef is the struct for user defined type names.
type TypeNameDef struct {
	node

	Schema string
	Name   string
}

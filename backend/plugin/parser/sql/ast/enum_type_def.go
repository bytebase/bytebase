package ast

// EnumTypeDef is the struct for enum types.
type EnumTypeDef struct {
	userDefinedType

	Name      *TypeNameDef
	LabelList []string
}

// EquivalentType implements the DataType interface.
func (e *EnumTypeDef) EquivalentType(tp string) bool {
	return tp == e.Name.Name
}

// TypeName implements the UserDefinedType interface.
func (e EnumTypeDef) TypeName() *TypeNameDef {
	return e.Name
}

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

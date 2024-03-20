package base

type StringsManipulatorActionType int

const (
	StringsManipulatorActionTypeNone StringsManipulatorActionType = iota
	StringsManipulatorActionTypeDropTable
	StringsManipulatorActionTypeAddTable
	StringsManipulatorActionTypeDropColumn
	StringsManipulatorActionTypeAddColumn
	StringsManipulatorActionTypeModifyColumnType
	StringsManipulatorActionTypeDropColumnOption
	StringsManipulatorActionTypeAddColumnOption
	StringsManipulatorActionTypeModifyColumnOption
	StringsManipulatorActionTypeDropTableConstraint
	StringsManipulatorActionTypeModifyTableConstraint
	StringsManipulatorActionTypeAddTableConstraint
	StringsManipulatorActionTypeDropTableOption
	StringsManipulatorActionTypeModifyTableOption
	StringsManipulatorActionTypeAddTableOption
	StringsManipulatorActionTypeDropIndex
	StringsManipulatorActionTypeAddIndex
	StringsManipulatorActionTypeModifyIndex
)

type StringsManipulatorAction interface {
	GetType() StringsManipulatorActionType
	GetSchemaName() string
	GetTopLevelNaming() string
	GetSecondLevelNaming() string
}

type StringsManipulatorActionBase struct {
	Type       StringsManipulatorActionType
	SchemaName string
}

func (s *StringsManipulatorActionBase) GetType() StringsManipulatorActionType {
	return s.Type
}

func (s *StringsManipulatorActionBase) GetSchemaName() string {
	return s.SchemaName
}

type ColumnOptionType int

const (
	ColumnOptionTypeNone ColumnOptionType = iota
	ColumnOptionTypeNotNull
	ColumnOptionTypeDefault
)

type TableConstraintType int

const (
	TableConstraintTypeNone TableConstraintType = iota
	TableConstraintTypePrimaryKey
	TableConstraintTypeUnique
)

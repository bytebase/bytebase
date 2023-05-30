package ast

// ColumnNameDef is the struct for column name definition.
type ColumnNameDef struct {
	expression

	Table      *TableDef
	ColumnName string
}

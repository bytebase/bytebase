package ast

// AlterColumnTypeStmt is the struct for alter column type statement.
type AlterColumnTypeStmt struct {
	node

	Table      *TableDef
	ColumnName string
	Type       DataType
}

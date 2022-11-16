package ast

// DropDefaultStmt is the struct for drop default statement.
type DropDefaultStmt struct {
	node

	Table      *TableDef
	ColumnName string
}

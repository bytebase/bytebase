package ast

// DropNotNullStmt is the struct for set not null statement.
type DropNotNullStmt struct {
	node

	Table      *TableDef
	ColumnName string
}

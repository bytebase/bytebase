package ast

// SetNotNullStmt is the struct for set not null statement.
type SetNotNullStmt struct {
	node

	Table      *TableDef
	ColumnName string
}

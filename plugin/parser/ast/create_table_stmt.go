package ast

// CreateTableStmt is the strcut for create table stmt.
type CreateTableStmt struct {
	node

	IfNotExists bool
	Name        *TableDef
	ColumnList  []*ColumnDef
}

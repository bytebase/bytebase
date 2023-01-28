package ast

// AddColumnListStmt is the struct for add columns statement.
// For PostgreSQL dialect is ALTER TABLE ADD COLUMN.
type AddColumnListStmt struct {
	node

	Table       *TableDef
	ColumnList  []*ColumnDef
	IfNotExists bool
}

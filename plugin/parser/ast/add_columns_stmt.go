package ast

// AddColumnsStmt is the struct for add columns statement.
// For PostgreSQL dialect is ALTER TABLE ADD COLUMN.
type AddColumnsStmt struct {
	node

	Table   *TableName
	Columns []*ColumnDef
}

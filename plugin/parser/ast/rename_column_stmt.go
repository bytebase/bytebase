package ast

// RenameColumnStmt is the rename column statement.
// For PostgreSQL dialect is the ALTER TABLE RENAME COLUMN.
type RenameColumnStmt struct {
	node

	Table   *TableName
	Column  *ColumnDef
	NewName string
}

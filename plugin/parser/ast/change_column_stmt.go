package ast

// ChangeColumnStmt is the strcut for change column statement.
type ChangeColumnStmt struct {
	node

	Table         *TableName
	OldColumnName string
	Column        *ColumnDef
}

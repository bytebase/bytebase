package ast

// ChangeColumnStmt is the strcut for change column statement.
type ChangeColumnStmt struct {
	node

	Table         *TableDef
	OldColumnName string
	Column        *ColumnDef
}

package ast

// SetDefaultStmt is the struct for set default statement.
type SetDefaultStmt struct {
	node

	Table      *TableDef
	ColumnName string
	Expression ExpressionNode
}

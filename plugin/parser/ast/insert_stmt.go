package ast

// InsertStmt is the struct for insert statement.
type InsertStmt struct {
	dml

	Table     *TableDef
	ValueList [][]ExpressionNode
	Select    *SelectStmt
}

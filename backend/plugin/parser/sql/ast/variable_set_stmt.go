package ast

type VariableSetStmt struct {
	node

	Name    string
	Args    []ExpressionNode
	IsLocal bool
}

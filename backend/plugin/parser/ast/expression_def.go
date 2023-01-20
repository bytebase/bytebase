package ast

var (
	_ ExpressionNode = (*expression)(nil)
)

// ExpressionNode is the interface for expressions.
type ExpressionNode interface {
	Node

	expressionNode()
}

// expression is the base struct for expression nodes.
type expression struct {
	node
}

func (*expression) expressionNode() {}

package ast

// PatternLikeDef is the struct for LIKE expression definition, e.g. a LIKE '%abc'.
type PatternLikeDef struct {
	expression

	Not        bool
	Expression ExpressionNode
	Pattern    ExpressionNode
}

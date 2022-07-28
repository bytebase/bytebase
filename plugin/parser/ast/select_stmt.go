package ast

// SetOperationType is the type for set operations.
type SetOperationType int

const (
	// SetOperationTypeNone is the type for no set operation, which means a SELECT stmt.
	SetOperationTypeNone SetOperationType = iota
	// SetOperationTypeUnion is the type for UNION.
	SetOperationTypeUnion
	// SetOperationTypeIntersect is the type for INTERSECT.
	SetOperationTypeIntersect
	// SetOperationTypeExcept is the type for EXCEPT.
	SetOperationTypeExcept
)

// SelectStmt is the struct for select statement.
type SelectStmt struct {
	node

	// if SetOperation is SetOperatironTypeNode:
	//   LQuery and RQuery is nil and SELECT fields is useful
	// otherwise:
	//   only use LQuery and RQuery
	SetOperation SetOperationType
	LQuery       *SelectStmt
	RQuery       *SelectStmt

	// SELECT fields
	TargetList  []ExpressionNode
	WhereClause ExpressionNode

	// TODO(rebelice): support all expression and remove them.
	// We define them because we cannot convert all expression now.
	// And we only need to check the following nodes currently.
	//
	// PatternLikeList is the list of the patternLike nodes.
	PatternLikeList []*PatternLikeDef
	// SubqueryList is the list of the subquery nodes.
	SubqueryList []*SubqueryDef
}

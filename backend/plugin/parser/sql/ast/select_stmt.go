package ast

import (
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

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

	// if SetOperation is SetOperationTypeNode:
	//   LQuery and RQuery is nil and SELECT fields is useful
	// otherwise:
	//   only use LQuery and RQuery
	SetOperation SetOperationType
	// LQuery and RQuery are used for set operation, such as UNION.
	// In this case, we define the SQL as:
	//   LQuery UNION RQuery;
	// Easy to know, LQuery and RQuery are SELECT statement, also.
	// Details at https://www.postgresql.org/docs/current/sql-select.html
	LQuery *SelectStmt
	RQuery *SelectStmt

	// SELECT fields
	FieldList     []ExpressionNode
	WhereClause   ExpressionNode
	OrderByClause []*ByItemDef
	Limit         *int32

	// TODO(rebelice): support all expression and remove them.
	// We define them because we cannot convert all expression now.
	// And we only need to check the following nodes currently.
	//
	// PatternLikeList is the list of the patternLike nodes.
	PatternLikeList []*PatternLikeDef
	// SubqueryList is the list of the subquery nodes.
	SubqueryList []*SubqueryDef
	// fromClause is the raw pgquery from clause nodes.
	fromClause []*pgquery.Node

	originalNode *pgquery.SelectStmt
}

func (ss *SelectStmt) SetOriginalNode(node *pgquery.SelectStmt) {
	ss.originalNode = node
}

func (ss *SelectStmt) GetOriginalNode() *pgquery.SelectStmt {
	return ss.originalNode
}

// SetFromClause sets the from clause for select statement.
func (ss *SelectStmt) SetFromClause(fromClause []*pgquery.Node) {
	ss.fromClause = fromClause
}

// GetFromClause gets the from clause for select statement.
func (ss *SelectStmt) GetFromClause() []*pgquery.Node {
	return ss.fromClause
}

// ByItemDef is the struct for item in order by or group by.
type ByItemDef struct {
	node

	Expression ExpressionNode
}

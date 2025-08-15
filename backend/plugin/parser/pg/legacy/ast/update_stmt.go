package ast

import pgquery "github.com/pganalyze/pg_query_go/v6"

// UpdateStmt is the struct for update statement.
type UpdateStmt struct {
	dml

	Table       *TableDef
	WhereClause ExpressionNode

	// TODO(rebelice): support all expression and remove them.
	// We define them because we cannot convert all expression now.
	// And we only need to check the following nodes currently.
	//
	// PatternLikeList is the list of the patternLike nodes.
	PatternLikeList []*PatternLikeDef
	// SubqueryList is the list of the subquery nodes.
	SubqueryList []*SubqueryDef

	originalNode *pgquery.Node_UpdateStmt
}

func (us *UpdateStmt) SetOriginalNode(node *pgquery.Node_UpdateStmt) {
	us.originalNode = node
}

func (us *UpdateStmt) GetOriginalNode() *pgquery.Node_UpdateStmt {
	return us.originalNode
}

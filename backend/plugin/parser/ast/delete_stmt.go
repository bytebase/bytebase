package ast

// DeleteStmt is the struct for delete statement.
type DeleteStmt struct {
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
}

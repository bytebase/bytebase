package ast

import pgquery "github.com/pganalyze/pg_query_go/v6"

type CreateViewStmt struct {
	ddl

	Name    *TableDef
	Aliases []string
	Select  *SelectStmt
	Replace bool

	originalNode *pgquery.Node_ViewStmt
}

func (cv *CreateViewStmt) SetOriginalNode(node *pgquery.Node_ViewStmt) {
	cv.originalNode = node
}

func (cv *CreateViewStmt) GetOriginalNode() *pgquery.Node_ViewStmt {
	return cv.originalNode
}

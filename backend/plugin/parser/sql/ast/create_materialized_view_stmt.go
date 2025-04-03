package ast

import pgquery "github.com/pganalyze/pg_query_go/v6"

type CreateMaterializedViewStmt struct {
	ddl

	Name *TableDef

	originalNode *pgquery.Node_CreateTableAsStmt
}

func (cv *CreateMaterializedViewStmt) SetOriginalNode(node *pgquery.Node_CreateTableAsStmt) {
	cv.originalNode = node
}

func (cv *CreateMaterializedViewStmt) GetOriginalNode() *pgquery.Node_CreateTableAsStmt {
	return cv.originalNode
}

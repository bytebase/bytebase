package ast

import (
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// CreateFunctionStmt is the struct for create function statement.
type CreateFunctionStmt struct {
	ddl

	Function *FunctionDef

	originalNode *pgquery.Node_CreateFunctionStmt
}

func (cv *CreateFunctionStmt) SetOriginalNode(node *pgquery.Node_CreateFunctionStmt) {
	cv.originalNode = node
}

func (cv *CreateFunctionStmt) GetOriginalNode() *pgquery.Node_CreateFunctionStmt {
	return cv.originalNode
}

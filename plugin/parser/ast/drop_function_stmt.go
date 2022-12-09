package ast

// DropFunctionStmt is the struct for drop function statement.
type DropFunctionStmt struct {
	ddl

	// Here use FunctionDef because the drop function statement may need the schema name for PostgreSQL.
	FunctionList []*FunctionDef
	IfExists     bool
	Behavior     DropBehavior
}

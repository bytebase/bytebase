package ast

// DropFunctionStmt is the struct for drop function statement.
type DropFunctionStmt struct {
	ddl

	// Here use FunctionDef because the drop function statement needs:
	// 1. the schema name
	// 2. the function name
	// 3. the IN/INOUT/VARIADIC parameter type
	FunctionList []*FunctionDef
	IfExists     bool
	Behavior     DropBehavior
}

package ast

// CreateFunctionStmt is the struct for create function statement.
type CreateFunctionStmt struct {
	ddl

	Function *FunctionDef
}

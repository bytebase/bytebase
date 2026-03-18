package pg

import (
	"github.com/bytebase/omni/pg/ast"
)

// IsPlSQLBlock checks if the given statement is a PL/pgSQL anonymous code block (DO statement).
// Returns true if the statement is a DO block only, false otherwise.
func IsPlSQLBlock(stmt string) bool {
	stmts, err := ParsePg(stmt)
	if err != nil {
		return false
	}

	// Must be exactly one statement
	if len(stmts) != 1 {
		return false
	}

	_, ok := stmts[0].AST.(*ast.DoStmt)
	return ok
}

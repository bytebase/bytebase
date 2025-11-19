package pg

import (
	parser "github.com/bytebase/parser/postgresql"
)

// IsPlSQLBlock checks if the given statement is a PL/pgSQL anonymous code block (DO statement).
// Returns true if the statement is a DO block only, false otherwise.
func IsPlSQLBlock(stmt string) bool {
	// Parse using the existing ANTLR-based parser
	results, err := ParsePostgreSQL(stmt)
	if err != nil {
		return false
	}

	// Must be exactly one statement
	if len(results) != 1 {
		return false
	}

	// Check if the parsed tree is a single DO statement
	root, ok := results[0].Tree.(*parser.RootContext)
	if !ok || root == nil {
		return false
	}

	stmtblock := root.Stmtblock()
	if stmtblock == nil {
		return false
	}

	stmtmulti := stmtblock.Stmtmulti()
	if stmtmulti == nil {
		return false
	}

	stmts := stmtmulti.AllStmt()
	// Must be exactly one statement
	if len(stmts) != 1 {
		return false
	}

	// Check if the single statement is a DO statement
	return stmts[0].Dostmt() != nil
}

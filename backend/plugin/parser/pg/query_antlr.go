package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// validateQueryANTLR validates the SQL statement for SQL editor using omni parser.
// Returns (isReadOnly, allQueriesReturnData, error)
//
// Validation rules:
//  1. Allow: SELECT statements
//  2. Allow: EXPLAIN statements (but not EXPLAIN ANALYZE for non-SELECT)
//  3. Allow: SHOW/SET statements (SET is considered executable)
//  4. Allow: CTEs whose every term is a read (SELECT/VALUES/TABLE/set-op/RECURSIVE);
//     reject a SELECT with a data-modifying CTE term — a fail-safe whitelist.
//  5. Reject: SELECT ... INTO (creates a table) and all other statements (DDL, DML).
func validateQueryANTLR(statement string) (bool, bool, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return false, false, syntaxErr
		}
		return false, false, err
	}

	var hasExecute bool

	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}

		switch n := stmt.AST.(type) {
		case *ast.SelectStmt:
			// SELECT is read-only unless it writes: SELECT ... INTO creates a table,
			// or a data-modifying CTE smuggles a write into the read path.
			if isWriteSelect(n) {
				return false, false, nil
			}

		case *ast.ExplainStmt:
			if isExplainAnalyze(n) {
				// EXPLAIN ANALYZE executes the query, so it must be a read-only SELECT.
				sel, ok := n.Query.(*ast.SelectStmt)
				if !ok || isWriteSelect(sel) {
					return false, false, nil
				}
			}

		case *ast.VariableSetStmt:
			hasExecute = true

		case *ast.VariableShowStmt:
			// SHOW is allowed

		default:
			// All other statements (DDL, DML) are not allowed
			return false, false, nil
		}
	}

	return true, !hasExecute, nil
}

// isExplainAnalyze checks if an ExplainStmt has the ANALYZE option.
func isExplainAnalyze(n *ast.ExplainStmt) bool {
	if n.Options == nil {
		return false
	}
	for _, item := range n.Options.Items {
		if de, ok := item.(*ast.DefElem); ok {
			if strings.EqualFold(de.Defname, "analyze") {
				return true
			}
		}
	}
	return false
}

// isWriteSelect reports whether a SelectStmt actually writes — and so must not take
// the read-only editor path. A SELECT writes if it has an INTO target (SELECT ...
// INTO creates a table) or any of its CTE terms is data-modifying.
//
// This is the unified write-detection primitive for the read-only gate;
// classifyQueryType (query_type.go) keeps reporting the root statement type for its
// own consumers — only the detection is shared, not the classifiers' outputs.
func isWriteSelect(n *ast.SelectStmt) bool {
	if n == nil {
		return false
	}
	if hasOmniIntoClause(n) {
		return true
	}
	return containsWriteCTE(n)
}

// containsWriteCTE reports whether any common table expression in the tree is
// data-modifying. Fail-safe by construction — a whitelist, not a denylist: a CTE
// term is read-only ONLY if it is an *ast.SelectStmt, which is how pg models
// SELECT, VALUES, TABLE, set operations and WITH RECURSIVE. Anything else —
// INSERT/UPDATE/DELETE/MERGE today, or any write node added to the grammar
// tomorrow — counts as a write, so the next forgotten write type fails closed.
func containsWriteCTE(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if cte, ok := n.(*ast.CommonTableExpr); ok {
			if _, isSelect := cte.Ctequery.(*ast.SelectStmt); !isSelect {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

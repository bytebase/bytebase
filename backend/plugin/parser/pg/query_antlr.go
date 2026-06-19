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
// 1. Allow: SELECT statements
// 2. Allow: EXPLAIN statements (but not EXPLAIN ANALYZE for non-SELECT)
// 3. Allow: SHOW/SET statements (SET is considered executable)
// 4. Allow: CTEs with only SELECT (reject CTEs with INSERT/UPDATE/DELETE/MERGE)
// 5. Reject: All other statements (DDL, DML except SELECT)
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
			// SELECT is allowed. Check for DML in CTEs.
			if n.WithClause != nil && hasDMLInTree(n) {
				return false, false, nil
			}

		case *ast.ExplainStmt:
			if isExplainAnalyze(n) {
				// EXPLAIN ANALYZE is only valid for SELECT
				if _, ok := n.Query.(*ast.SelectStmt); !ok {
					return false, false, nil
				}
				// Check for DML in CTEs within the explained SELECT
				if hasDMLInTree(n.Query) {
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

// hasDMLInTree walks the AST tree and returns true if any INSERT/UPDATE/DELETE/MERGE
// is found. MERGE is included so a data-modifying MERGE smuggled into a CTE
// (e.g. WITH x AS (MERGE ... RETURNING col) SELECT col FROM x) is classified as a
// write: this keeps the set aligned with classifyQueryType (query_type.go) and
// stops the editor from taking the row-returning read path for it, which would
// otherwise return masked columns unmasked.
func hasDMLInTree(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		switch n.(type) {
		case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
			found = true
			return false
		}
		return true
	})
	return found
}

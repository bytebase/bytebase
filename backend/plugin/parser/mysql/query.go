package mysql

import (
	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MYSQL, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_MARIADB, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Only SELECT, EXPLAIN, SHOW, SET, and DESCRIBE are allowed in read-only mode.
// EXPLAIN ANALYZE is treated as non-read-only since it actually executes the query.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := ParseMySQLOmni(statement)
	if err != nil {
		return false, false, convertOmniError(err, base.Statement{Text: statement})
	}

	hasExecute := false
	readOnly := true
	for _, node := range stmts.Items {
		switch stmt := node.(type) {
		case *ast.SelectStmt:
			// INTO OUTFILE and INTO DUMPFILE write a file on the server, so
			// they are writes (pg refuses its own SELECT ... INTO likewise).
			// INTO <variable> returns no rows: the SET case, not a write.
			writesFile, assignsVariables := selectIntoTargets(stmt)
			if writesFile {
				return false, false, nil
			}
			if assignsVariables {
				hasExecute = true
			}
		case *ast.ExplainStmt:
			if stmt.Analyze {
				readOnly = false
			}
		case *ast.ShowStmt:
			// SHOW is always allowed.
		case *ast.SetStmt, *ast.SetPasswordStmt, *ast.SetDefaultRoleStmt, *ast.SetRoleStmt, *ast.SetResourceGroupStmt, *ast.SetTransactionStmt:
			hasExecute = true
		default:
			return false, false, nil
		}
	}
	return readOnly, !hasExecute, nil
}

// selectIntoTargets reports what a SELECT's INTO clause targets. It searches
// the whole statement rather than its root because the parser attaches INTO to
// an arm of a set operation or to a parenthesized query, not necessarily to the
// node the caller holds — the same reason pg's omniIntoClause walks its arms.
//
// A subquery cannot carry INTO in MySQL, so reaching into the tree cannot
// over-match a legal statement; if the grammar ever allowed it, the answer here
// would be conservative rather than permissive.
func selectIntoTargets(n *ast.SelectStmt) (writesFile, assignsVariables bool) {
	ast.Inspect(n, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectStmt)
		if !ok || sel.Into == nil {
			return true
		}
		// The clause form decides, not the filename it carries: those fields
		// hold the decoded path, so "INTO OUTFILE ''" leaves both empty while
		// still being a file write.
		if len(sel.Into.Vars) > 0 {
			assignsVariables = true
		} else {
			writesFile = true
		}
		return true
	})
	return writesFile, assignsVariables
}

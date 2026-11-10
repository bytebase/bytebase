package tidb

import (
	"github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Only SELECT, EXPLAIN, SHOW, SET and DESCRIBE are allowed in read-only mode.
// EXPLAIN ANALYZE executes what it wraps, so it is allowed only over a read.
func validateQuery(statement string) (bool, bool, error) {
	stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, stmt := range stmts {
		node, ok := GetOmniNode(stmt.AST)
		if !ok {
			// An empty or comment-only statement carries no AST.
			continue
		}
		switch n := node.(type) {
		case *ast.SelectStmt:
			// omni folds UNION/INTERSECT/EXCEPT into SelectStmt, so this covers
			// a set operation too. INTO OUTFILE writes a file on the database
			// server, so the statement is not a read whatever the query itself
			// does — pg refuses its own SELECT ... INTO on the same grounds.
			if hasSelectInto(n) {
				return false, false, nil
			}
		case *ast.SetStmt, *ast.SetTransactionStmt:
			// omni splits SET TRANSACTION into its own node; pingcap folded it
			// into SetStmt, which the editor allowed as a session setting that
			// requires execution. getQueryType classifies both the same way.
			hasExecute = true
		case *ast.ShowStmt:
		case *ast.ExplainStmt:
			if !explainReads(n) {
				readOnly = false
			}
		default:
			return false, false, nil
		}
	}
	return readOnly, !hasExecute, nil
}

// explainReads reports whether an EXPLAIN is a read. The two EXPLAIN forms are
// decided here together so the export gate below is the only one either goes
// through: a second gate on the ANALYZE path could only drift from this one.
//
// A statement-less EXPLAIN (FOR CONNECTION / by digest) has no wrapped
// statement; the nil guard refuses it, matching the validator's default.
func explainReads(explain *ast.ExplainStmt) bool {
	if explain.Stmt == nil {
		return false
	}
	// An export is refused under either form. A plain EXPLAIN does not run it,
	// but the editor must not label as a read what it refuses when asked
	// directly.
	if hasSelectInto(explain.Stmt) {
		return false
	}
	if !explain.Analyze {
		// A plain EXPLAIN only plans, so it reads whatever it wraps, DML
		// included — the editor has always allowed that.
		return true
	}
	// EXPLAIN ANALYZE runs the statement to report actual row counts and
	// timings, so only the SELECT family is a read. Postgres gates its own
	// EXPLAIN ANALYZE the same way. The refusal matters most for DML: EXPLAIN
	// ANALYZE DELETE really deletes, and a plain EXPLAIN does not. omni folds a
	// set operation into SelectStmt, so this one case covers both.
	if _, ok := explain.Stmt.(*ast.SelectStmt); ok {
		return true
	}
	return false
}

// hasSelectInto reports whether any SELECT in node carries an INTO clause.
//
// The whole tree is searched rather than the node the caller holds, because the
// parser attaches INTO to an arm of a set operation and not to the set
// operation itself: `SELECT 1 UNION SELECT a FROM t INTO OUTFILE '/x'` carries
// the export on its second arm. The parser also accepts INTO inside a subquery
// or a CTE, so the search reaches statements whose root is an ordinary SELECT;
// refusing those is the conservative answer for a clause that writes a file on
// the server, and it is what the MySQL validator already does.
//
// Every INTO form counts, not just OUTFILE: a form a later parser accepts is
// refused by default.
func hasSelectInto(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if sel, ok := n.(*ast.SelectStmt); ok && sel.Into != nil {
			found = true
			return false
		}
		return true
	})
	return found
}

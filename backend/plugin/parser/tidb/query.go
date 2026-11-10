package tidb

import (
	"github.com/pingcap/tidb/pkg/parser/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Only SELECT, EXPLAIN, SHOW, SET and DESCRIBE are allowed in read-only mode.
// EXPLAIN ANALYZE executes what it wraps, so it is allowed only over a read.
//
// TODO(tidb-omni): this validator still parses with the native pingcap parser.
// Migrate it to omni (ParseTiDBOmni + omni/tidb/ast) as a follow-up, after the
// query-span extractor cutover.
func validateQuery(statement string) (bool, bool, error) {
	stmtList, err := ParseTiDB(statement, "", "")
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *ast.SelectStmt, *ast.SetOprStmt:
			// INTO OUTFILE writes a file on the database server, so the
			// statement is not a read whatever the query itself does — pg
			// refuses its own SELECT ... INTO on the same grounds.
			if hasSelectInto(stmt) {
				return false, false, nil
			}
		case *ast.SetStmt:
			hasExecute = true
		case *ast.ShowStmt:
		case *ast.ExplainStmt:
			if !explainReads(stmt) {
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
// The parser gives every EXPLAIN form this handles a statement to wrap, DESCRIBE
// included; statement-less `EXPLAIN FOR CONNECTION` parses to ExplainForStmt,
// which the validator refuses by default. So the nil guard is for safety here,
// not a policy.
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
	// ANALYZE DELETE really deletes, and a plain EXPLAIN does not.
	switch explain.Stmt.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt:
		return true
	}
	return false
}

// hasSelectInto reports whether any SELECT in node carries an INTO clause.
//
// The whole tree is searched rather than the node the caller holds, because the
// parser attaches INTO to an arm of a set operation and not to the set
// operation itself: `SELECT 1 UNION SELECT a FROM t INTO OUTFILE '/x'` is a
// SetOprStmt whose second arm carries the export. The parser also accepts INTO
// inside a subquery or a CTE, so the search reaches statements whose root is an
// ordinary SELECT; refusing those is the conservative answer for a clause that
// writes a file on the server, and it is what the MySQL validator already does.
//
// Every INTO form counts, not just OUTFILE: the parser rejects DUMPFILE and
// INTO <variable> outright, so OUTFILE is the only form that reaches here
// today and any form a later parser accepts is refused by default.
func hasSelectInto(node ast.Node) bool {
	v := &selectIntoVisitor{}
	node.Accept(v)
	return v.found
}

// selectIntoVisitor finds a SELECT carrying an INTO clause.
type selectIntoVisitor struct {
	found bool
}

func (v *selectIntoVisitor) Enter(node ast.Node) (ast.Node, bool) {
	if sel, ok := node.(*ast.SelectStmt); ok && sel.SelectIntoOpt != nil {
		v.found = true
	}
	return node, v.found
}

func (*selectIntoVisitor) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}

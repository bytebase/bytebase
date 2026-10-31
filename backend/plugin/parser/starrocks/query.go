package starrocks

import (
	"github.com/bytebase/omni/starrocks/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
}

// validateQuery reports whether the given statement is a read-only query
// suitable for the SQL editor / data-query path.
//
// Decision is AST-based: each parsed top-level statement must be a SELECT,
// SHOW, DESCRIBE, EXPLAIN, HELP, or a session SET. Relying on leading-keyword classification
// alone would mis-accept CTE-prefixed DML such as `WITH x AS (...) UPDATE ...`,
// which Classify would tag as SELECT because `WITH` is its first token.
//
// Syntax errors are surfaced up; that lets validateQueryRequest reject
// malformed read-only SQL before execution.
//
// The (bool, bool, error) return shape matches the bytebase QueryValidator
// contract: (isReadOnly, isExplicitReadOnly, syntaxError). isExplicitReadOnly is
// false once a statement needs executing rather than reading, which is how the
// driver decides between QueryContext and ExecContext.
func validateQuery(statement string) (bool, bool, error) {
	parsed, err := parseStarRocksSQL(statement)
	if err != nil {
		return false, false, err
	}
	var hasExecute bool
	for _, p := range parsed {
		switch {
		case isReadOnlyAST(p.Node()):
		case needsExecute(p.Node()):
			hasExecute = true
		default:
			return false, false, nil
		}
	}
	return true, !hasExecute, nil
}

// needsExecute reports whether a statement is allowed in the editor but is not a
// read. A session variable assignment returns no rows, so the driver has to run it
// with ExecContext rather than read a result set; validateQuery signals that through
// its second return value. MySQL and Postgres gate SET the same way.
//
// Only variable assignment qualifies. SET PASSWORD and SET DEFAULT STORAGE VAULT are
// separate nodes and stay refused, as they are for Postgres: they administer the
// server rather than prepare a read.
func needsExecute(node ast.Node) bool {
	switch node.(type) {
	case *ast.SetStmt, *ast.UnsetStmt:
		return true
	}
	return false
}

// isReadOnlyAST returns true when the given top-level AST node represents a
// read-only Doris statement (SELECT family, SHOW, DESCRIBE, EXPLAIN, HELP).
//
// Omni's parser currently has stub-shaped acceptance for some shapes — bare
// `SHOW`, `DESCRIBE`, `EXPLAIN` all produce a corresponding AST node with
// empty content rather than a parse error. We reject those here so the
// previous ANTLR helper's behaviour (rejecting incomplete forms) is
// preserved; the database would only reject them at execution time
// otherwise.
//
// Nil nodes are conservatively rejected — they indicate a parse path that
// produced no concrete statement, which shouldn't happen for valid read-only
// SQL after parseStarRocksSQL succeeds.
func isReadOnlyAST(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt, *ast.SetOpStmt, *ast.ParenSelect:
		return isReadOnlyQuery(node)
	case *ast.ShowStmt:
		// Reject bare `SHOW` (Type is empty when the parser took the stub path
		// without seeing a recognised variant keyword).
		return n.Type != ""
	case *ast.ShowRoutineLoadStmt, *ast.ShowRoutineLoadTaskStmt,
		*ast.ShowJobStmt, *ast.ShowJobTaskStmt,
		*ast.ShowConstraintsStmt, *ast.ShowAnalyzeStmt, *ast.ShowStatsStmt:
		return true
	case *ast.DescribeStmt:
		// Reject bare `DESCRIBE` / `DESC` (Target nil means no table named).
		return n.Target != nil
	case *ast.ExplainStmt:
		// EXPLAIN requires an inner query; the inner query must itself be a
		// shape EXPLAIN is meaningful for (SELECT family or DML). This blocks
		// inputs like `EXPLAIN`, `EXPLAIN DROP TABLE t`, etc.
		if n.Query == nil || !isExplainableInner(n.Query) {
			return false
		}
		// EXPLAIN ANALYZE runs the statement rather than only planning it, so
		// over DML it writes. StarRocks aborts the INSERT transaction by
		// default, but the run still creates any partitions the rows need and
		// those outlive the abort. The plan-only modifiers never execute, so
		// they stay readable over DML.
		if n.Type == "ANALYZE" {
			return isReadOnlyQuery(n.Query)
		}
		return true
	case *ast.HelpStmt:
		return true
	default:
		return false
	}
}

// isExplainableInner reports whether `node` is a shape that EXPLAIN can
// legitimately wrap (SELECT family or DML). DDL inside EXPLAIN is rejected:
// StarRocks only supports EXPLAIN on query and DML statements.
//
// An export stays refused under EXPLAIN, because the read gate it goes
// through is the same one the bare statement goes through: the editor must not
// accept by way of EXPLAIN what it refuses when asked directly.
func isExplainableInner(node ast.Node) bool {
	if isReadOnlyQuery(node) {
		return true
	}
	switch node.(type) {
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt,
		*ast.MergeStmt, *ast.TruncateTableStmt:
		return true
	}
	return false
}

// isReadOnlyQuery reports whether `node` is a SELECT-family query that reads
// and never writes. INTO OUTFILE exports to S3 or HDFS, past masking and past
// the export gate, so a query carrying one is not a read.
//
// This is the single export gate for the SELECT family. Both the top level
// (isReadOnlyAST) and the EXPLAIN path (isExplainableInner) go through it, so
// neither can drift from the other: a type-only second copy on the EXPLAIN
// path let every EXPLAIN shape launder an export past this gate.
func isReadOnlyQuery(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.Into == nil
	case *ast.SetOpStmt:
		// The parser attaches INTO to an arm rather than the set operation.
		return !setOpWritesOutfile(n)
	case *ast.ParenSelect:
		// (SELECT ...) / ((SELECT ...)) — classify by the wrapped query so the
		// gate still applies inside the parens. ParenSelect only ever wraps a
		// SELECT, SetOpStmt or another ParenSelect, so no DML hides here.
		return isReadOnlyQuery(n.Sel)
	}
	return false
}

// setOpWritesOutfile walks the arms: the parser attaches INTO to an arm rather
// than to the set operation, and arms nest.
func setOpWritesOutfile(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.Into != nil
	case *ast.SetOpStmt:
		return setOpWritesOutfile(n.Left) || setOpWritesOutfile(n.Right)
	case *ast.ParenSelect:
		// A parenthesized arm — SELECT 1 UNION (SELECT ... INTO OUTFILE ...) —
		// must not slip the export gate.
		return setOpWritesOutfile(n.Sel)
	default:
		return false
	}
}

package doris

import (
	"github.com/bytebase/omni/doris/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

// validateQuery reports whether the given statement is a read-only query
// suitable for the SQL editor / data-query path.
//
// Decision is AST-based: each parsed top-level statement must be a SELECT,
// SHOW, DESCRIBE, EXPLAIN, HELP, or a session SET. Relying on leading-keyword classification
// alone would mis-accept CTE-prefixed DML such as `WITH x AS (...) UPDATE ...`,
// which Classify would tag as SELECT because `WITH` is its first token.
//
// A query that carries INTO OUTFILE is refused: it writes its result set to
// S3 or HDFS, so the data leaves the database past masking and past the
// export gate. That is read off the tree (isReadOnlyAST), which it could not
// be until omni's Doris grammar grew the clause — before then this function
// matched an OUTFILE keyword token in the statement text instead (BOT-92).
//
// Syntax errors are surfaced up; that lets validateQueryRequest reject
// malformed read-only SQL before execution.
//
// The (bool, bool, error) return shape matches the bytebase QueryValidator
// contract: (isReadOnly, isExplicitReadOnly, syntaxError). isExplicitReadOnly is
// false once a statement needs executing rather than reading, which is how the
// driver decides between QueryContext and ExecContext.
func validateQuery(statement string) (bool, bool, error) {
	parsed, err := parseDorisSQL(statement)
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
// SQL after parseDorisSQL succeeds.
func isReadOnlyAST(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt:
		// INTO OUTFILE exports to S3 or HDFS, past masking and the export gate.
		return n.Into == nil
	case *ast.SetOpStmt:
		return n.Into == nil
	case *ast.GroupedQuery:
		// A repeated trailing clause group or a parenthesized WITH group —
		// (SELECT 1 LIMIT 1) LIMIT 2 — wraps a query; classify by the query.
		// The export gate applies to the group as well, since the clause
		// attaches to whichever node is the statement's outermost query.
		return n.Into == nil && isReadOnlyAST(n.Query)
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
		// shape EXPLAIN is meaningful for (SELECT/DML/Show). This blocks
		// inputs like `EXPLAIN`, `EXPLAIN DROP TABLE t`, etc.
		return n.Query != nil && isExplainableInner(n.Query)
	case *ast.HelpStmt:
		return true
	default:
		return false
	}
}

// isExplainableInner reports whether `node` is a shape that EXPLAIN can
// legitimately wrap (SELECT family or DML). DDL inside EXPLAIN is rejected:
// Doris only supports EXPLAIN on query and DML statements.
//
// An export stays refused under EXPLAIN. The engine does not treat
// `EXPLAIN SELECT ... INTO OUTFILE ...` as a plain plan: it fails with the
// same "Please specify BROKER properties or check your local file path" as
// the bare export, where `EXPLAIN SELECT 1` returns a plan (container-
// verified). So the clause is not inert here, and the gate holds.
func isExplainableInner(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.Into == nil
	case *ast.SetOpStmt:
		return n.Into == nil
	case *ast.GroupedQuery:
		// EXPLAIN (SELECT 1 LIMIT 1) LIMIT 2 — engine-verified accept;
		// classify by the wrapped query.
		return n.Into == nil && isExplainableInner(n.Query)
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt,
		*ast.MergeStmt, *ast.TruncateTableStmt:
		return true
	}
	return false
}

package doris

import (
	"regexp"

	"github.com/bytebase/omni/doris/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_DORIS, validateQuery)
}

// validateQuery reports whether the given statement is a read-only query
// suitable for the SQL editor / data-query path.
//
// Decision is AST-based: each parsed top-level statement must be a SELECT,
// SHOW, DESCRIBE, EXPLAIN, or HELP. Relying on leading-keyword classification
// alone would mis-accept CTE-prefixed DML such as `WITH x AS (...) UPDATE ...`,
// which Classify would tag as SELECT because `WITH` is its first token.
//
// Syntax errors are surfaced up; that lets validateQueryRequest reject
// malformed read-only SQL before execution.
//
// The (bool, bool, error) return shape matches the bytebase QueryValidator
// contract: (isReadOnly, isExplicitReadOnly, syntaxError).
func validateQuery(statement string) (bool, bool, error) {
	// INTO OUTFILE exports the result straight out of the database, to S3 or
	// HDFS, so it is a write and it leaves through the engine before Bytebase
	// can mask a returned row.
	//
	// It is matched on the text rather than on the AST because omni's Doris
	// grammar does not represent the clause at all: "SELECT 1 INTO OUTFILE
	// '/tmp/x'" parses to a bare *ast.SelectStmt indistinguishable from
	// "SELECT 1" (BOT-92). StarRocks carries an ast.IntoOutfileClause and is
	// checked there instead. Delete this once the Doris grammar has the
	// clause, and check n.Into the way starrocks/query.go does.
	if statementMentionsOutfile(statement) {
		return false, false, nil
	}
	parsed, err := parseDorisSQL(statement)
	if err != nil {
		return false, false, err
	}
	for _, p := range parsed {
		if !isReadOnlyAST(p.Node()) {
			return false, false, nil
		}
	}
	return true, true, nil
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
	case *ast.SelectStmt, *ast.SetOpStmt:
		return true
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
func isExplainableInner(node ast.Node) bool {
	switch node.(type) {
	case *ast.SelectStmt, *ast.SetOpStmt:
		return true
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt,
		*ast.MergeStmt, *ast.TruncateTableStmt:
		return true
	}
	return false
}

// statementMentionsOutfile reports whether an INTO OUTFILE or INTO DUMPFILE
// clause appears outside quoted text
// and comments. Only Doris uses it; see validateQuery for why the AST cannot
// answer this there.
func statementMentionsOutfile(statement string) bool {
	stripped, err := tokenizer.StandardRemoveQuotedTextAndComment(statement)
	if err != nil {
		// A statement this tokenizer cannot read is one whose OUTFILE could be
		// hiding anywhere in it.
		return true
	}
	return outfilePattern.MatchString(stripped)
}

var outfilePattern = regexp.MustCompile(`(?i)\bINTO\s+(OUTFILE|DUMPFILE)\b`)

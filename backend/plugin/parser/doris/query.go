package doris

import (
	"github.com/bytebase/omni/doris/ast"
	"github.com/bytebase/omni/doris/parser"

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
	// INTO OUTFILE exports to S3 or HDFS, past masking and the export gate.
	// Matched on text because omni's Doris grammar has no INTO node: it parses
	// to a bare SelectStmt identical to one without the clause (BOT-92).
	// Replace with an AST check, as starrocks/query.go does, once it has one.
	//
	if mentionsFileExport(statement) {
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

// mentionsFileExport reports whether the statement carries an OUTFILE keyword
// token. Doris only; see validateQuery for why the AST cannot answer this.
//
// The lexer decides, not the text. It closes Doris's nested block comments,
// which the shared comment stripper does not, so a keyword buried after
// "/* a /* b */ c */" is still reached; and it yields no keyword for OUTFILE
// inside a string literal, a backtick-quoted identifier or a comment, so
// reading an audit log for attempted exports stays a servable query.
//
// A statement that will not lex fails closed: an unterminated string or
// comment is exactly where a keyword could be hiding.
func mentionsFileExport(statement string) bool {
	tokens, lexErrors := parser.Tokenize(statement)
	if len(lexErrors) > 0 {
		return true
	}
	for _, token := range tokens {
		if token.Kind == outfileTokenKind {
			return true
		}
	}
	return false
}

// outfileTokenKind is resolved from the keyword table rather than hardcoded.
// DUMPFILE is deliberately absent: it is MySQL syntax, not a Doris keyword.
var outfileTokenKind, _ = parser.KeywordToken("outfile")

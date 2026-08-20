package doris

import (
	"regexp"

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
	// The raw statement, deliberately, with nothing stripped first. Doris
	// nests block comments and the shared stripper stops at the first "*/",
	// so "INTO /* a /* b */ c */ OUTFILE" survives stripping with text
	// between INTO and OUTFILE and slips a keyword-pair match. A comment is
	// whitespace to the lexer, so the keyword itself cannot be split by one:
	// matching it where it must literally appear is the part no comment
	// structure can defeat. The cost is that the keyword inside a string
	// literal refuses too, which fails closed.
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

// mentionsFileExport reports whether the export keyword appears anywhere in the
// statement. Doris only; see validateQuery for why not the AST, and why not the
// stripped text.
func mentionsFileExport(statement string) bool {
	return fileExportPattern.MatchString(statement)
}

var fileExportPattern = regexp.MustCompile(`(?i)\b(OUTFILE|DUMPFILE)\b`)

package snowflake

import (
	"strings"

	"github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType classifies a single parsed Snowflake statement into a
// base.QueryType. It mirrors the legacy ANTLR listener's
// getQueryTypeForBatch/getQueryTypeForDmlCommand/getQueryTypeForOtherCommand
// (which switched on the sql_command / dml_command / other_command grammar
// branches) onto a type-switch over omni's AST nodes.
//
// Mapping (legacy branch → omni node → base.QueryType):
//   - dml_command query_statement (SELECT / set-op)  → *ast.SelectStmt,
//     *ast.SetOperationStmt                            → base.Select
//   - dml_command (INSERT/UPDATE/DELETE/MERGE)        → *ast.InsertStmt,
//     *ast.InsertMultiStmt, *ast.UpdateStmt,
//     *ast.DeleteStmt, *ast.MergeStmt                  → base.DML
//   - show_command                                    → *ast.ShowStmt
//     describe_command                                → *ast.DescribeStmt   → base.SelectInfoSchema
//   - use_command                                     → *ast.UseStmt        → base.Select
//   - other_command set                               → *ast.SetStmt        → base.Select
//   - other_command copy_into_table                   → *ast.CopyIntoTableStmt → base.DML
//   - other_command comment                           → *ast.CommentStmt    → base.DDL
//   - ddl_command (CREATE/ALTER/DROP/UNDROP/...)       → the *ast.*Stmt set  → base.DDL
//
// Everything else (the remaining other_command forms the legacy listener left
// as base.QueryTypeUnknown — TRUNCATE, GRANT/REVOKE, COMMIT/ROLLBACK, PUT/GET,
// LIST/REMOVE, UNSET, CALL, ...) defaults to base.DDL: the safe upper bound for
// access-control purposes, matching the Trino migration's "everything else is
// DDL" default. The only current consumer is validateQuery, which only
// distinguishes read-only (Select/SelectInfoSchema) from everything else, so
// the upper-bound default never relaxes the read-only gate.
//
// A nil node yields base.QueryTypeUnknown. omni does not yet emit a node for
// EXPLAIN or CALL (Parse returns an "... not yet supported" error for those);
// EXPLAIN is handled at the statement-text level in validateQuery.
func getQueryType(node ast.Node) base.QueryType {
	if node == nil {
		return base.QueryTypeUnknown
	}
	switch node.(type) {
	// SELECT / set operations.
	case *ast.SelectStmt, *ast.SetOperationStmt:
		return base.Select

	// DML: INSERT (single + multi-table), UPDATE, DELETE, MERGE.
	case *ast.InsertStmt, *ast.InsertMultiStmt,
		*ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
		return base.DML

	// SHOW / DESCRIBE read system metadata.
	case *ast.ShowStmt, *ast.DescribeStmt:
		return base.SelectInfoSchema

	// USE is a session-scoped no-op for read purposes; the legacy listener
	// classified use_command as base.Select.
	case *ast.UseStmt:
		return base.Select

	// SET (session variable) was classified base.Select by the legacy listener.
	case *ast.SetStmt:
		return base.Select

	// COPY INTO <table> loads data, so the legacy listener classified it DML.
	case *ast.CopyIntoTableStmt:
		return base.DML

	// COMMENT ON ... is DDL.
	case *ast.CommentStmt:
		return base.DDL
	}

	// Everything else (CREATE/ALTER/DROP/UNDROP/TRUNCATE/GRANT/REVOKE/... and any
	// other recognized statement) is DDL — the safe upper bound for ACL.
	return base.DDL
}

// isExplainStatement reports whether the given single-statement text is an
// EXPLAIN statement. omni's Snowflake parser does not yet support EXPLAIN
// (parser.Parse returns "EXPLAIN statement parsing is not yet supported"), so
// EXPLAIN is detected lexically here, exactly mirroring the legacy listener,
// which treated any other_command.explain as a read-only, data-returning
// statement WITHOUT inspecting the inner statement.
//
// The check skips leading whitespace and a leading line/block comment, then
// matches a leading EXPLAIN keyword on a word boundary, case-insensitively.
func isExplainStatement(text string) bool {
	return strings.EqualFold(leadingKeyword(text), "EXPLAIN")
}

// leadingKeyword returns the first identifier-like token of the statement text
// (uppercased callers handle case), after skipping leading whitespace and a
// single leading -- line comment or /* */ block comment. It is intentionally
// minimal: it only needs to recognize the leading EXPLAIN keyword.
func leadingKeyword(text string) string {
	s := strings.TrimLeft(text, " \t\r\n\f\v")
	for {
		switch {
		case strings.HasPrefix(s, "--"):
			if idx := strings.IndexByte(s, '\n'); idx >= 0 {
				s = strings.TrimLeft(s[idx+1:], " \t\r\n\f\v")
				continue
			}
			return ""
		case strings.HasPrefix(s, "/*"):
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = strings.TrimLeft(s[idx+2:], " \t\r\n\f\v")
				continue
			}
			return ""
		}
		break
	}
	end := 0
	for end < len(s) {
		c := s[end]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			end++
			continue
		}
		break
	}
	return s[:end]
}

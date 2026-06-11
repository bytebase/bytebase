package snowflake

import (
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
// CALL / EXECUTE IMMEDIATE / EXECUTE TASK classify base.DML and EXPLAIN
// classifies base.Explain, matching the legacy other_command branches.
//
// Everything else (the remaining other_command forms the legacy listener left
// as base.QueryTypeUnknown — TRUNCATE, GRANT/REVOKE, COMMIT/ROLLBACK, PUT/GET,
// LIST/REMOVE, UNSET, ...) defaults to base.DDL: the safe upper bound for
// access-control purposes, matching the Trino migration's "everything else is
// DDL" default. validateQuery only distinguishes read-only
// (Select/SelectInfoSchema/Explain) from everything else, so the upper-bound
// default never relaxes the read-only gate.
//
// A nil node yields base.QueryTypeUnknown.
func getQueryType(node ast.Node) base.QueryType {
	if node == nil {
		return base.QueryTypeUnknown
	}
	switch n := node.(type) {
	// SELECT / set operations.
	case *ast.SelectStmt, *ast.SetOperationStmt:
		return base.Select

	// DML: INSERT (single + multi-table), UPDATE, DELETE, MERGE. The legacy
	// listener also classified CALL / EXECUTE IMMEDIATE / EXECUTE TASK as DML
	// (other_command branches) — a stored procedure can mutate data, so DML is
	// the right ACL bucket, not the DDL fallback.
	case *ast.InsertStmt, *ast.InsertMultiStmt,
		*ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt,
		*ast.CallStmt, *ast.ExecuteImmediateStmt, *ast.ExecuteTaskStmt:
		return base.DML

	// SHOW / DESCRIBE read system metadata.
	case *ast.ResultScanStmt:
		// stmt ->> query: the result shape is the trailing query's (typically a
		// SELECT over $1). Read-only-ness of the SOURCE is enforced separately
		// in classifyForEditor.
		return getQueryType(n.Query)
	case *ast.ExplainStmt:
		// EXPLAIN is read-only and data-returning regardless of the inner
		// statement (legacy other_command.explain never recursed into it).
		return base.Explain
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

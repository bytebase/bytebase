package trino

import (
	"strings"

	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType classifies a single parsed Trino statement.
//
// It returns the base.QueryType plus a bool that is true only for
// EXPLAIN ANALYZE. EXPLAIN ANALYZE actually executes the inner query, so the
// legacy plugin reported it as base.Select (not base.Explain) with the flag
// set; we preserve that. A plain EXPLAIN is base.Explain and never executes.
//
// allSystems, when true, promotes a user SELECT to base.SelectInfoSchema. The
// query-span extractor passes this in based on whether every accessed table is
// a system table; callers that only have the AST pass false and rely on the
// statement-text system-schema scan (containsSystemSchema) instead.
//
// node is an omni AST node (one of the *parser.*Stmt types). A nil node yields
// (QueryTypeUnknown, false).
func getQueryType(node ast.Node) (base.QueryType, bool) {
	if node == nil {
		return base.QueryTypeUnknown, false
	}
	switch n := node.(type) {
	case *parser.QueryStmt:
		return base.Select, false
	case *parser.ExplainStmt:
		if n.Analyze {
			// EXPLAIN ANALYZE EXECUTES the inner statement (oracle-confirmed:
			// Trino 481 runs it), so its read-only-ness is the inner statement's,
			// not unconditionally read-only. Report the inner type and flag
			// isAnalyze so validateQuery rejects EXPLAIN ANALYZE of a non-SELECT
			// (e.g. EXPLAIN ANALYZE UPDATE, which would otherwise run a write
			// through the read-only SQL-editor gate).
			innerType, _ := getQueryType(n.Statement)
			return innerType, true
		}
		return base.Explain, false
	case *parser.ShowStmt,
		*parser.DescribeInputStmt, *parser.DescribeOutputStmt:
		return base.SelectInfoSchema, false
	case *parser.SetSessionStmt, *parser.ResetSessionStmt,
		*parser.SetSessionAuthorizationStmt, *parser.ResetSessionAuthorizationStmt,
		*parser.SetPathStmt, *parser.SetRoleStmt, *parser.SetTimeZoneStmt,
		*parser.UseStmt:
		// Session-state statements were classified read-only (base.Select) by the
		// legacy listener (EnterSetSession / EnterResetSession). USE is also a
		// session-scoped no-op for read purposes. Keep them read-only.
		return base.Select, false
	case *parser.InsertStmt, *parser.UpdateStmt, *parser.DeleteStmt,
		*parser.MergeStmt, *parser.TruncateStmt, *parser.CallStmt:
		return base.DML, false
	}
	// Everything else (CREATE/ALTER/DROP/COMMENT/GRANT/REVOKE/ANALYZE/
	// transaction control/prepared-statement admin/...) is DDL. This is the safe
	// upper bound for ACL purposes and matches the legacy listener's
	// "everything else is DDL" default.
	return base.DDL, false
}

// queryTypeFromText parses statement and classifies its single top-level
// statement, applying the legacy statement-text system-schema promotion: a
// base.Select that references a Trino system/metadata schema becomes
// base.SelectInfoSchema. EXPLAIN over such a query is likewise promoted.
//
// On parse failure or empty/multi input it returns base.QueryTypeUnknown.
func queryTypeFromText(statement string) base.QueryType {
	file, errs := parser.Parse(statement)
	if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
		return base.QueryTypeUnknown
	}
	qt, _ := getQueryType(file.Stmts[0])
	switch qt {
	case base.Select, base.Explain:
		if containsSystemSchema(statement) {
			return base.SelectInfoSchema
		}
	default:
	}
	return qt
}

// IsReadOnlyStatement reports whether the given omni AST node is a read-only
// Trino statement (SELECT family, EXPLAIN, SHOW/DESCRIBE, session-state).
func IsReadOnlyStatement(node ast.Node) bool {
	queryType, _ := getQueryType(node)
	return queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema
}

// IsDataChangingStatement reports whether the node is a DML statement.
func IsDataChangingStatement(node ast.Node) bool {
	queryType, _ := getQueryType(node)
	return queryType == base.DML
}

// IsSchemaChangingStatement reports whether the node is a DDL statement.
func IsSchemaChangingStatement(node ast.Node) bool {
	queryType, _ := getQueryType(node)
	return queryType == base.DDL
}

// containsSystemSchema reports whether a query references a Trino system/metadata
// schema, via a case-insensitive substring scan. This mirrors the legacy
// plugin's heuristic (and omni analysis.Classify's refineSelect) so the SELECT →
// SelectInfoSchema promotion is identical across the legacy and omni stacks.
func containsSystemSchema(sql string) bool {
	lowerSQL := strings.ToLower(sql)
	systemPrefixes := []string{
		"system.",
		"information_schema.",
		"$system.",
		"catalog.",
		"metadata.",
	}
	for _, prefix := range systemPrefixes {
		if strings.Contains(lowerSQL, prefix) {
			return true
		}
	}
	return false
}

package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"
	"github.com/bytebase/omni/starrocks/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType classifies a single statement.
//
// Uses AST inspection where possible — keyword-based Classify alone would
// mislabel CTE-prefixed DML (`WITH ... UPDATE ...`) as Select because `WITH`
// is its first token, which would propagate into ACL checks. AST inspection
// surfaces the real operation. If parsing fails or produces no statements
// (e.g. comment-only input), we fall back to the keyword-based classifier so
// callers still receive a best-effort classification.
//
// allSystems is the flag computed by the query-span extractor that indicates
// whether every accessed table belongs to a system database. When true and
// the resolved type is a user SELECT, the result is promoted to
// SelectInfoSchema to match the legacy ANTLR listener behaviour.
//
// EXPLAIN-prefixed statements defer to the inner statement's type — an
// `EXPLAIN DROP TABLE t` classifies as DDL, not Select, so type-based ACL
// checks see the inner operation rather than a downgrade-to-read-only.
func getQueryType(statement string, allSystems bool) base.QueryType {
	qt := classifyByAST(statement)

	switch qt {
	case analysis.QueryTypeSelect:
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	case analysis.QueryTypeSelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.QueryTypeDML:
		return base.DML
	case analysis.QueryTypeDDL:
		return base.DDL
	default:
		return base.QueryTypeUnknown
	}
}

// classifyByAST parses the statement and inspects the first top-level AST
// node. On parse failure or empty input it falls back to the keyword-based
// Classify.
func classifyByAST(statement string) analysis.QueryType {
	file, errs := parser.Parse(statement)
	if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
		return analysis.Classify(statement)
	}
	if qt, ok := astQueryType(file.Stmts[0]); ok {
		return qt
	}
	return analysis.Classify(statement)
}

// astQueryType maps a top-level AST node to its QueryType. The bool return
// is false only when the node was nil-shaped (e.g. EXPLAIN with no inner
// Query); callers should fall back to Classify in that case.
//
// EXPLAIN recurses into its inner Query so the classification reflects the
// real operation (e.g. `EXPLAIN DROP TABLE t` → DDL).
//
// Anything we don't otherwise enumerate is treated as DDL — that's a safe
// upper bound for ACL purposes (it won't accidentally label a write as
// read-only) and matches the legacy listener's "everything else is DDL"
// behaviour.
func astQueryType(node ast.Node) (analysis.QueryType, bool) {
	if node == nil {
		return 0, false
	}
	switch n := node.(type) {
	case *ast.SelectStmt, *ast.SetOpStmt:
		return analysis.QueryTypeSelect, true
	case *ast.ShowStmt,
		*ast.ShowRoutineLoadStmt, *ast.ShowRoutineLoadTaskStmt,
		*ast.ShowJobStmt, *ast.ShowJobTaskStmt,
		*ast.ShowConstraintsStmt, *ast.ShowAnalyzeStmt, *ast.ShowStatsStmt:
		return analysis.QueryTypeSelectInfoSchema, true
	case *ast.DescribeStmt, *ast.HelpStmt:
		return analysis.QueryTypeSelectInfoSchema, true
	case *ast.ExplainStmt:
		if n.Query == nil {
			return 0, false
		}
		return astQueryType(n.Query)
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt,
		*ast.MergeStmt, *ast.TruncateTableStmt:
		return analysis.QueryTypeDML, true
	case *ast.UseStmt:
		// USE was rejected as Unknown by the legacy listener — its ACL flow
		// treats Unknown as a hard deny while DDL is permitted via
		// bb.sql.ddl. Keep the classification at Unknown so users with DDL
		// rights cannot run USE.
		return analysis.QueryTypeUnknown, true
	}
	return analysis.QueryTypeDDL, true
}

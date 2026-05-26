package doris

import (
	"github.com/bytebase/omni/doris/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_STARROCKS, validateQuery)
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
// Nil nodes are conservatively rejected — they indicate a parse path that
// produced no concrete statement, which shouldn't happen for valid read-only
// SQL after parseDorisSQL succeeds.
func isReadOnlyAST(node ast.Node) bool {
	switch node.(type) {
	case *ast.SelectStmt, *ast.SetOpStmt:
		return true
	case *ast.ShowStmt,
		*ast.ShowRoutineLoadStmt, *ast.ShowRoutineLoadTaskStmt,
		*ast.ShowJobStmt, *ast.ShowJobTaskStmt,
		*ast.ShowConstraintsStmt, *ast.ShowAnalyzeStmt, *ast.ShowStatsStmt:
		return true
	case *ast.DescribeStmt, *ast.ExplainStmt, *ast.HelpStmt:
		return true
	default:
		return false
	}
}

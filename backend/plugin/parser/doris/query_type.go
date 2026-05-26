package doris

import (
	"github.com/bytebase/omni/doris/analysis"
	"github.com/bytebase/omni/doris/ast"
	"github.com/bytebase/omni/doris/parser"

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
// EXPLAIN-prefixed statements are mapped to base.Select to match the legacy
// listener (which promoted EXPLAIN-on-DML to Select).
func getQueryType(statement string, allSystems bool) base.QueryType {
	if isExplainStatement(statement) {
		return base.Select
	}

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
// is false when the node's type is not in our table — callers should fall
// back to Classify in that case.
func astQueryType(node ast.Node) (analysis.QueryType, bool) {
	switch node.(type) {
	case *ast.SelectStmt, *ast.SetOpStmt:
		return analysis.QueryTypeSelect, true
	case *ast.ShowStmt,
		*ast.ShowRoutineLoadStmt, *ast.ShowRoutineLoadTaskStmt,
		*ast.ShowJobStmt, *ast.ShowJobTaskStmt,
		*ast.ShowConstraintsStmt, *ast.ShowAnalyzeStmt, *ast.ShowStatsStmt:
		return analysis.QueryTypeSelectInfoSchema, true
	case *ast.DescribeStmt, *ast.ExplainStmt, *ast.HelpStmt:
		return analysis.QueryTypeSelectInfoSchema, true
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt,
		*ast.MergeStmt, *ast.TruncateTableStmt:
		return analysis.QueryTypeDML, true
	}
	return 0, false
}

// isExplainStatement reports whether the first meaningful keyword of the
// statement is EXPLAIN. Used to override the SelectInfoSchema classification
// with base.Select for EXPLAIN-prefixed queries, matching the legacy ANTLR
// listener.
func isExplainStatement(statement string) bool {
	// Walk over leading whitespace and -- / /* */ comments before sniffing the
	// first identifier-like token. We deliberately don't bring in the full
	// lexer here — a cheap prefix scan suffices because the only token we care
	// about is the literal word EXPLAIN.
	i := 0
	for i < len(statement) {
		// Skip whitespace.
		for i < len(statement) {
			c := statement[i]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v' {
				i++
				continue
			}
			break
		}
		if i >= len(statement) {
			return false
		}
		// Skip -- line comments.
		if i+1 < len(statement) && statement[i] == '-' && statement[i+1] == '-' {
			i += 2
			for i < len(statement) && statement[i] != '\n' {
				i++
			}
			continue
		}
		// Skip # line comments (Doris/MySQL-style).
		if statement[i] == '#' {
			i++
			for i < len(statement) && statement[i] != '\n' {
				i++
			}
			continue
		}
		// Skip /* block */ comments.
		if i+1 < len(statement) && statement[i] == '/' && statement[i+1] == '*' {
			i += 2
			for i+1 < len(statement) {
				if statement[i] == '*' && statement[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		break
	}
	const explain = "EXPLAIN"
	if i+len(explain) > len(statement) {
		return false
	}
	for j := 0; j < len(explain); j++ {
		c := statement[i+j]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		if c != explain[j] {
			return false
		}
	}
	// Ensure the following char is a non-identifier (whitespace, comment, EOF).
	next := i + len(explain)
	if next >= len(statement) {
		return true
	}
	c := statement[next]
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
		return false
	}
	return true
}

package doris

import (
	"github.com/bytebase/omni/doris/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType classifies a single statement by delegating to omni's
// analysis.Classify and mapping the result to base.QueryType.
//
// allSystems is the flag computed by the query-span extractor that indicates
// whether every accessed table belongs to a system database. When true and
// the omni classification is a user SELECT, the result is promoted to
// SelectInfoSchema to match the legacy ANTLR listener's behaviour.
//
// The omni classifier treats anything starting with EXPLAIN as
// SelectInfoSchema; that includes EXPLAIN-on-DML, which the legacy listener
// promoted to base.Select. We mirror that promotion: if a statement starts
// with EXPLAIN, the result is base.Select.
func getQueryType(statement string, allSystems bool) base.QueryType {
	omniType := analysis.Classify(statement)

	if isExplainStatement(statement) {
		return base.Select
	}

	switch omniType {
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

// isExplainStatement reports whether the first meaningful keyword of the
// statement is EXPLAIN. Used to override omni's SelectInfoSchema
// classification with base.Select for EXPLAIN-prefixed queries, matching
// the legacy ANTLR listener.
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

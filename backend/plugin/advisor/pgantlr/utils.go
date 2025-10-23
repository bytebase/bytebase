package pgantlr

import (
	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"
)

// isTopLevel checks if the context is at the top level of the parse tree.
// Top level contexts are: RootContext, StmtblockContext, StmtmultiContext, or StmtContext.
func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
}

// splitIdentifier splits a qualified identifier by dots, handling quoted parts.
// For example:
//   - "public.table" -> ["public", "table"]
//   - "\"public\".\"table\"" -> ["public", "table"]
//   - "table" -> ["table"]
func splitIdentifier(s string) []string {
	var parts []string
	var current string
	inQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == '.' && !inQuote {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

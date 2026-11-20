package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// parsedNodes returns the statements that carry an omni node and their nodes,
// index for index.
func parsedNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}

// contentStartLine returns the 1-based line number of the first non-whitespace
// character in text. Returns 1 if the text is empty or has no leading newlines.
func contentStartLine(text string) int32 {
	idx := strings.IndexFunc(text, func(c rune) bool {
		return c != ' ' && c != '\t' && c != '\n' && c != '\r'
	})
	if idx <= 0 {
		return 1
	}
	return pgparser.ByteOffsetToRunePosition(text, idx).Line
}

// contentEndLine returns the 1-based line number of the last non-whitespace
// character in text.
func contentEndLine(text string) int32 {
	if text == "" {
		return 1
	}
	idx := -1
	for i := len(text) - 1; i >= 0; i-- {
		c := text[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != ';' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return 1
	}
	return pgparser.ByteOffsetToRunePosition(text, idx).Line
}

// findLineByName searches for an identifier name in text and returns its
// 1-based line number. Falls back to contentStartLine(text) if not found.
func findLineByName(text, name string) int32 {
	if name == "" || text == "" {
		return contentStartLine(text)
	}
	idx := strings.Index(text, name)
	if idx < 0 {
		return contentStartLine(text)
	}
	return pgparser.ByteOffsetToRunePosition(text, idx).Line
}

// locToLine converts an omni Loc byte offset to a 1-based line number
// relative to the statement text.
func locToLine(text string, loc ast.Loc) int32 {
	if loc.Start < 0 || text == "" {
		return contentStartLine(text)
	}
	return pgparser.ByteOffsetToRunePosition(text, loc.Start).Line
}

// trimmedStmtText returns text with leading/trailing whitespace and trailing
// semicolons removed.
func trimmedStmtText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";")
}

// findingLine returns the absolute line a V1 advisor reported a finding of
// stmt at: the line of the first occurrence of the finding's Search text in
// the statement, or the line the statement's content starts on.
func findingLine(stmt base.ParsedStatement, f v1review.Finding) int32 {
	line := contentStartLine(stmt.Text)
	if f.Search != "" {
		line = findLineByName(stmt.Text, f.Search)
	}
	return line + int32(stmt.BaseLine())
}

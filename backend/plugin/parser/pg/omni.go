package pg

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/bytebase/omni/pg/ast"
	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateStmt).
	Node ast.Node
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParsePg parses SQL using omni's parser and returns omni Statement objects directly.
// This is the recommended entry point for new code that needs omni AST nodes.
func ParsePg(sql string) ([]omnipg.Statement, error) {
	return omnipg.Parse(sql)
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an OmniAST, nil and false otherwise.
func GetOmniNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok {
		return nil, false
	}
	return omniAST.Node, true
}

// byteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column
// where column is measured in Unicode code points (runes), matching storepb.Position semantics.
func byteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0) // 0-based rune count on current line
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			runeCol = 0
		} else {
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1, // convert to 1-based
	}
}

// convertOmniError converts omni's ParseError to base.SyntaxError.
func convertOmniError(sql string, err error) error {
	var parseErr *parser.ParseError
	if errors.As(err, &parseErr) {
		pos := byteOffsetToRunePosition(sql, parseErr.Position)
		return &base.SyntaxError{
			Position: pos,
			Message: fmt.Sprintf("Syntax error at line %d:%d \nrelated text: %s",
				pos.Line, pos.Column, extractErrorContext(sql, parseErr.Position)),
			RawMessage: parseErr.Message,
		}
	}
	return err
}

// extractErrorContext extracts a snippet of SQL around the error position for context.
func extractErrorContext(sql string, bytePos int) string {
	// Find the start: search backward for newline or beginning.
	start := bytePos
	for start > 0 && sql[start-1] != '\n' {
		start--
	}
	end := bytePos + 20
	if end > len(sql) {
		end = len(sql)
	}
	return sql[start:end]
}

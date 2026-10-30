package oceanbase

import (
	"unicode/utf8"

	"github.com/bytebase/omni/oceanbase/ast"
	omniparser "github.com/bytebase/omni/oceanbase/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTSource returns the statement's own text. Consumers shared with MySQL, such as
// backend/plugin/schema/mysql's walk-through, read it to re-derive a MySQL AST node
// for an OceanBase statement.
func (a *OmniAST) ASTSource() string {
	return a.Text
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseOceanBase parses SQL using omni's parser and returns an ast.List.
// This is the recommended entry point for new code that needs omni AST nodes.
func ParseOceanBase(sql string) (*ast.List, error) {
	return omniparser.Parse(sql)
}

// IsKeyword reports whether s is a registered MySQL keyword.
func IsKeyword(s string) bool {
	return omniparser.IsKeyword(s)
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

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column
// where column is measured in Unicode code points (runes), matching storepb.Position semantics.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
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

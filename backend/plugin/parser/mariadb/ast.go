package mariadb

import (
	"unicode/utf8"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni MariaDB AST node and implements the base.AST interface.
type AST struct {
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTSource returns the statement's own text, which the schema walk-through reads
// when it needs the statement as written.
func (a *AST) ASTSource() string {
	return a.Text
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseMariaDB parses SQL using omni's MariaDB parser and returns an ast.List.
func ParseMariaDB(sql string) (*ast.List, error) {
	return mariadbparser.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an AST, nil and false otherwise.
func GetNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	node, ok := a.(*AST)
	if !ok {
		return nil, false
	}
	return node.Node, true
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

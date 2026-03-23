package pg

import (
	"unicode/utf8"

	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
// During migration, it also carries an optional ANTLR AST for backward compatibility
// with advisors that still use ANTLR parse trees.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	// antlrAST is the legacy ANTLR parse tree, populated during migration for
	// backward compatibility with advisors. Will be removed once all advisors migrate to omni.
	antlrAST *base.ANTLRAST
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AsANTLRAST implements base.AntlrASTProvider, returning the legacy ANTLR AST if available.
func (a *OmniAST) AsANTLRAST() (*base.ANTLRAST, bool) {
	if a.antlrAST == nil {
		return nil, false
	}
	return a.antlrAST, true
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

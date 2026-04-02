package mysql

import (
	"unicode/utf8"

	"github.com/bytebase/omni/mysql/ast"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
// During migration, it also implements AntlrASTProvider so that callers
// still using GetANTLRAST() can fall back to the ANTLR tree.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position

	// antlrAST is lazily populated when AsANTLRAST() is called.
	// This field will be removed once all callers are migrated to omni.
	antlrAST *base.ANTLRAST
	// antlrParsed tracks whether we've attempted the ANTLR parse.
	antlrParsed bool
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// AsANTLRAST implements base.AntlrASTProvider for backward compatibility.
// It lazily parses the SQL text with the ANTLR parser and caches the result.
// This will be removed once all MySQL modules are migrated to omni.
func (a *OmniAST) AsANTLRAST() (*base.ANTLRAST, bool) {
	if a.antlrParsed {
		return a.antlrAST, a.antlrAST != nil
	}
	a.antlrParsed = true

	// Use lenient ANTLR parsing (no error listeners) since omni already
	// validated the SQL. The ANTLR tree is only needed for backward-compatible
	// tree walking by callers that haven't migrated to omni yet.
	tree, tokens := parseSingleStatementLenient(a.Text)
	a.antlrAST = &base.ANTLRAST{
		StartPosition: a.StartPosition,
		Tree:          tree,
		Tokens:        tokens,
	}
	return a.antlrAST, true
}

// ParseMySQLOmni parses SQL using omni's parser and returns an ast.List.
// This is the recommended entry point for new code that needs omni AST nodes.
func ParseMySQLOmni(sql string) (*ast.List, error) {
	return mysqlparser.Parse(sql)
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

package plsql

import (
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	antlrparser "github.com/bytebase/parser/plsql"

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
	// This field will be removed once all Oracle modules are migrated to omni.
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
func (a *OmniAST) AsANTLRAST() (*base.ANTLRAST, bool) {
	if a.antlrParsed {
		return a.antlrAST, a.antlrAST != nil
	}
	a.antlrParsed = true

	tree, tokens := parseSinglePLSQLLenient(a.Text)
	a.antlrAST = &base.ANTLRAST{
		StartPosition: a.StartPosition,
		Tree:          tree,
		Tokens:        tokens,
	}
	return a.antlrAST, true
}

// parseSinglePLSQLLenient parses a single Oracle statement without error listeners.
// The omni parser has already validated the SQL; this tree only exists for
// backward-compatible ANTLR consumers during the migration window.
func parseSinglePLSQLLenient(statement string) (antlr.Tree, *antlr.CommonTokenStream) {
	inputStream := antlr.NewInputStream(addSemicolonIfNeeded(statement))
	lexer := antlrparser.NewPlSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := antlrparser.NewPlSqlParser(stream)
	p.SetVersion12(true)

	lexer.RemoveErrorListeners()
	p.RemoveErrorListeners()
	p.BuildParseTrees = true

	tree := p.Sql_script()
	return tree, stream
}

// ParsePLSQLOmni parses SQL using omni's parser and returns an ast.List.
// This is the recommended entry point for new Oracle code that needs omni AST nodes.
func ParsePLSQLOmni(sql string) (*ast.List, error) {
	return oracleparser.Parse(sql)
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

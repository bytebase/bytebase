package plsql

import (
	"testing"

	parser "github.com/bytebase/parser/plsql"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/oracle/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParsePLSQLOmni(t *testing.T) {
	list, err := ParsePLSQLOmni("SELECT * FROM T; INSERT INTO T VALUES (1);")
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 2)

	first, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, first.Stmt)

	second, ok := list.Items[1].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.InsertStmt{}, second.Stmt)
}

func TestParsePLSQLOmniReturnsParseError(t *testing.T) {
	_, err := ParsePLSQLOmni("SELECT * FROM")
	require.Error(t, err)
}

func TestOracleOmniASTWrapper(t *testing.T) {
	node := &ast.SelectStmt{}
	start := &storepb.Position{Line: 2, Column: 3}
	omniAST := &OmniAST{
		Node:          node,
		Text:          "SELECT 1 FROM DUAL",
		StartPosition: start,
	}

	var parsed base.AST = omniAST
	require.Equal(t, start, parsed.ASTStartPosition())

	got, ok := GetOmniNode(parsed)
	require.True(t, ok)
	require.Same(t, node, got)

	got, ok = GetOmniNode(nil)
	require.False(t, ok)
	require.Nil(t, got)
}

func TestOracleOmniASTAsANTLRAST(t *testing.T) {
	start := &storepb.Position{Line: 4, Column: 1}
	omniAST := &OmniAST{
		Node:          &ast.SelectStmt{},
		Text:          "SELECT * FROM T",
		StartPosition: start,
	}

	antlrAST, ok := base.GetANTLRAST(omniAST)
	require.True(t, ok)
	require.Equal(t, start, antlrAST.StartPosition)
	require.IsType(t, &parser.Sql_scriptContext{}, antlrAST.Tree)
	require.NotNil(t, antlrAST.Tokens)

	antlrASTAgain, ok := base.GetANTLRAST(omniAST)
	require.True(t, ok)
	require.Same(t, antlrAST, antlrASTAgain)
}

func TestOracleByteOffsetToRunePosition(t *testing.T) {
	position := ByteOffsetToRunePosition("SELECT\n日本 FROM DUAL", len("SELECT\n日"))
	require.Equal(t, int32(2), position.Line)
	require.Equal(t, int32(2), position.Column)
}

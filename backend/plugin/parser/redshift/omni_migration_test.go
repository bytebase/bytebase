package redshift

import (
	"errors"
	"testing"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseStatementsUsesOmniAST(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, "SELECT 1;")
	require.NoError(t, err)
	require.Len(t, stmts, 1)

	_, ok := stmts[0].AST.(*OmniAST)
	require.True(t, ok)
}

func TestParseStatementsUsesOmniASTForMultipleStatements(t *testing.T) {
	statement := "SELECT 1;\nINSERT INTO t VALUES (1);"

	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	firstNode, ok := GetOmniNode(stmts[0].AST)
	require.True(t, ok)
	require.IsType(t, &redshiftast.SelectStmt{}, firstNode)
	require.Equal(t, "SELECT 1;", stmts[0].Text)
	require.Equal(t, int32(1), stmts[0].Start.Line)
	require.Equal(t, int32(1), stmts[0].Start.Column)

	secondNode, ok := GetOmniNode(stmts[1].AST)
	require.True(t, ok)
	require.IsType(t, &redshiftast.InsertStmt{}, secondNode)
	require.Equal(t, "\nINSERT INTO t VALUES (1);", stmts[1].Text)
	require.Equal(t, int32(1), stmts[1].Start.Line)
	require.Equal(t, int32(10), stmts[1].Start.Column)
}

func TestParseStatementsOmniErrorUsesOriginalColumn(t *testing.T) {
	_, err := base.ParseStatements(storepb.Engine_REDSHIFT, "SELECT 1; SELECT * FROM")
	require.Error(t, err)

	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr))
	require.Equal(t, int32(1), syntaxErr.Position.Line)
	require.Equal(t, int32(24), syntaxErr.Position.Column)
}

func TestRedshiftOmniASTWrapper(t *testing.T) {
	node := &redshiftast.SelectStmt{}
	start := &storepb.Position{Line: 2, Column: 3}
	omniAST := &OmniAST{
		Node:          node,
		Text:          "SELECT 1",
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

func TestRedshiftByteOffsetToRunePosition(t *testing.T) {
	position := ByteOffsetToRunePosition("SELECT\n日本 FROM t", len("SELECT\n日"))
	require.Equal(t, int32(2), position.Line)
	require.Equal(t, int32(2), position.Column)
}

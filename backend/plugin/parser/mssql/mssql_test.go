package mssql

import (
	"testing"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type fakeAST struct{}

func (*fakeAST) ASTStartPosition() *storepb.Position { return nil }

func TestParseStatementsSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, "SELECT 1;\nGO\nSELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	node, ok := GetNode(stmts[0].AST)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, node)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].AST.ASTStartPosition())

	node, ok = GetNode(stmts[1].AST)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, node)
	require.Equal(t, int32(2), stmts[1].AST.ASTStartPosition().Line)

	_, ok = GetNode(nil)
	require.False(t, ok)
	_, ok = GetNode(&fakeAST{})
	require.False(t, ok)
}

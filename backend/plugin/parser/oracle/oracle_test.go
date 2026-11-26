package oracle

import (
	"testing"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type fakeAST struct{}

func (*fakeAST) ASTStartPosition() *storepb.Position { return nil }

func TestParseStatementsSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, "SELECT 1 FROM DUAL;\n;\nSELECT 2 FROM DUAL;")
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

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("Oracle Database 19c Enterprise Edition Release 19.3.0.0.0 - Production")
	require.NoError(t, err)
	require.Equal(t, &Version{First: 19, Second: 3}, v)

	v, err = ParseVersion("Oracle Database 11g Express Edition Release 11.2.0.2.0")
	require.NoError(t, err)
	require.Equal(t, &Version{First: 11, Second: 2}, v)

	_, err = ParseVersion("unknown banner")
	require.Error(t, err)
}

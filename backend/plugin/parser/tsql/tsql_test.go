package tsql

import (
	"errors"
	"testing"

	"github.com/bytebase/omni/mssql/ast"
	mssqlparser "github.com/bytebase/omni/mssql/parser"
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

func TestConvertError(t *testing.T) {
	sql := "SELECT '你好' FROM t"

	parseErr := &mssqlparser.ParseError{Position: len("SELECT '你好' "), Message: "unexpected token"}
	err := convertError(parseErr, base.Statement{
		Text:  sql,
		Start: &storepb.Position{Line: 4, Column: 1},
	})
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, &storepb.Position{Line: 4, Column: 13}, syntaxErr.Position)
	require.Equal(t, "Syntax error at line 4: unexpected token", syntaxErr.Message)
	require.Equal(t, "unexpected token", syntaxErr.RawMessage)

	sentinel := errors.New("boom")
	require.ErrorIs(t, convertError(sentinel, base.Statement{}), sentinel)
}

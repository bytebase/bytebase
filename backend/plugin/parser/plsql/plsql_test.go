package plsql

import (
	"errors"
	"testing"

	"github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"
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

func TestConvertErrorAndPositions(t *testing.T) {
	// Error on line 1 shifts both line and column by stmt.Start.
	parseErr := &oracleparser.ParseError{Position: 4, Message: "unexpected token"}
	err := convertError(parseErr, base.Statement{
		Text:  "SELECT FROM",
		Start: &storepb.Position{Line: 3, Column: 5},
	})
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, &storepb.Position{Line: 3, Column: 9}, syntaxErr.Position)
	require.Equal(t, "Syntax error at line 3:9: unexpected token", syntaxErr.Message)
	require.Equal(t, "unexpected token", syntaxErr.RawMessage)

	// Error on a later line shifts only the line number.
	parseErr = &oracleparser.ParseError{Position: len("SELECT 1\nFROM "), Message: "bad table"}
	err = convertError(parseErr, base.Statement{
		Text:  "SELECT 1\nFROM ;",
		Start: &storepb.Position{Line: 3, Column: 5},
	})
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, &storepb.Position{Line: 4, Column: 6}, syntaxErr.Position)

	// Non-ParseError errors pass through unchanged.
	sentinel := errors.New("boom")
	require.ErrorIs(t, convertError(sentinel, base.Statement{}), sentinel)
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

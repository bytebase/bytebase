package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseSnowflakeStatements(t *testing.T) {
	stmts, err := parseSnowflakeStatements("SELECT 1;\n\nSELECT 2 FROM t;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	for i, stmt := range stmts {
		require.False(t, stmt.Empty, i)
		require.NotNil(t, stmt.AST, i)
		node, ok := GetOmniNode(stmt.AST)
		require.True(t, ok, i)
		require.NotNil(t, node, i)
		// ASTStartPosition keeps the legacy shape: BaseLine()+1, no column.
		require.Equal(t, int32(stmt.BaseLine())+1, stmt.AST.ASTStartPosition().Line, i)
	}
}

func TestParseSnowflakeStatementsSyntaxError(t *testing.T) {
	testCases := []struct {
		sql string
		// 1-based line the syntax error is reported on, relative to the whole
		// multi-statement input.
		line int32
	}{
		{
			sql:  `SELEC 5;`,
			line: 1,
		},
		{
			// The error position is offset by the failing statement's start
			// line in the original input.
			sql:  "SELECT 1;\n   SELEC 5;\nSELECT 6;",
			line: 2,
		},
	}

	for _, tc := range testCases {
		_, err := parseSnowflakeStatements(tc.sql)
		require.Error(t, err, tc.sql)
		syntaxErr, ok := err.(*base.SyntaxError)
		require.True(t, ok, "expected *base.SyntaxError, got %T: %v", err, err)
		require.NotNil(t, syntaxErr.Position, tc.sql)
		require.Equal(t, tc.line, syntaxErr.Position.Line, tc.sql)
		require.Contains(t, syntaxErr.Message, "Syntax error at line", tc.sql)
	}
}

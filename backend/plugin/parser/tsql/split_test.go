package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	script := "\nSELECT '你好';\nGO\nSELECT 2;"
	stmts, err := base.SplitMultiSQL(storepb.Engine_MSSQL, script)
	require.NoError(t, err)
	require.Len(t, stmts, 3)

	// Byte contiguity and position adjacency invariants enforced by toBaseStatements.
	require.Equal(t, int32(0), stmts[0].Range.Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	for i := 1; i < len(stmts); i++ {
		require.Equal(t, stmts[i-1].Range.End, stmts[i].Range.Start)
		require.Equal(t, stmts[i-1].End, stmts[i].Start)
	}
	require.Equal(t, int32(len(script)), stmts[len(stmts)-1].Range.End)

	// Multi-byte rune columns on line 2: "\nSELECT '你好';" -> 12 runes on line 2 => End column 13.
	require.False(t, stmts[0].Empty)
	require.Equal(t, &storepb.Position{Line: 2, Column: 13}, stmts[0].End)

	// GO separator is marked Empty=true.
	require.True(t, stmts[1].Empty)
	require.Contains(t, stmts[1].Text, "GO")

	require.False(t, stmts[2].Empty)
	require.Contains(t, stmts[2].Text, "SELECT 2;")

	// Syntax error falls back to semicolon-based tokenizer splitter.
	fallback, err := base.SplitMultiSQL(storepb.Engine_MSSQL, "SELECT FROM ; SELECT 2;")
	require.NoError(t, err)
	require.Len(t, fallback, 2)
	require.Contains(t, fallback[0].Text, "SELECT FROM ;")
	require.Contains(t, fallback[1].Text, "SELECT 2;")
}

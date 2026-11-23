package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	script := "SELECT '你好' FROM DUAL;\nSELECT 2 FROM DUAL;"
	stmts, err := base.SplitMultiSQL(storepb.Engine_ORACLE, script)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	require.Equal(t, "SELECT '你好' FROM DUAL", stmts[0].Text)
	require.False(t, stmts[0].Empty)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 22}, stmts[0].End)
	require.Equal(t, &storepb.Range{Start: 0, End: int32(len("SELECT '你好' FROM DUAL"))}, stmts[0].Range)

	require.Contains(t, stmts[1].Text, "SELECT 2 FROM DUAL")
	require.Equal(t, int32(1), stmts[1].Start.Line)
	require.Equal(t, int32(2), stmts[1].End.Line)
}

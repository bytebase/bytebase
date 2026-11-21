package partiql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	script := "DELETE FROM Music WHERE Artist = 'Pink Floyd';\n\tINSERT INTO Music VALUE {'Artist': 'Pink Floyd'};"
	stmts, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, script)
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "DELETE FROM Music WHERE Artist = 'Pink Floyd';", stmts[0].Text)
	require.Equal(t, int32(1), stmts[0].Start.Line)
	require.Equal(t, int32(1), stmts[0].Start.Column)
	require.Equal(t, "\n\tINSERT INTO Music VALUE {'Artist': 'Pink Floyd'};", stmts[1].Text)
	require.Equal(t, int32(2), stmts[1].End.Line)

	// Leading whitespace preserves 1-based start/end coordinates via base.BuildStatements.
	posStmts, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, "SELECT 1;\n\n  SELECT 2;")
	require.NoError(t, err)
	require.Len(t, posStmts, 2)
	require.Equal(t, &storepb.Range{Start: 0, End: 9}, posStmts[0].Range)
	require.Equal(t, int32(3), posStmts[1].End.Line)
	require.Equal(t, int32(12), posStmts[1].End.Column)
}

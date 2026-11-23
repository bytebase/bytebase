package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	empty, err := base.SplitMultiSQL(storepb.Engine_COSMOSDB, "   \n  ")
	require.NoError(t, err)
	require.Nil(t, empty)

	// Leading whitespace and newlines verify 1-based Start/End and byte Range mapping.
	stmts, err := base.SplitMultiSQL(storepb.Engine_COSMOSDB, "\n  \n  SELECT 1")
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.Equal(t, "SELECT 1", stmts[0].Text)
	require.Equal(t, int32(3), stmts[0].Start.Line)
	require.Equal(t, int32(3), stmts[0].Start.Column)
	require.Equal(t, int32(3), stmts[0].End.Line)
	require.Equal(t, int32(11), stmts[0].End.Column)
	require.Equal(t, &storepb.Range{Start: 6, End: 14}, stmts[0].Range)
	require.False(t, stmts[0].Empty)
}

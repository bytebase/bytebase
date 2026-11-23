package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestBigQuerySplitSQLSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_BIGQUERY, "SELECT 1;\n\n  SELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	require.Equal(t, "SELECT 1;", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[0].End)
	require.Equal(t, &storepb.Range{Start: 0, End: 9}, stmts[0].Range)
	require.False(t, stmts[0].Empty)

	require.Equal(t, "\n\n  SELECT 2;", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 3, Column: 12}, stmts[1].End)
	require.Equal(t, &storepb.Range{Start: 9, End: 22}, stmts[1].Range)
	require.False(t, stmts[1].Empty)
}

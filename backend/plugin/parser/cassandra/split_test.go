package cassandra

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestCassandraSplitSQLSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_CASSANDRA, "SELECT 1;\n\n  SELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "SELECT 1", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 9}, stmts[0].End)
	require.Equal(t, &storepb.Range{Start: 0, End: 8}, stmts[0].Range)

	require.Equal(t, "SELECT 2", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 3, Column: 3}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 3, Column: 11}, stmts[1].End)
	require.Equal(t, &storepb.Range{Start: 13, End: 21}, stmts[1].Range)

	stmts, err = base.SplitMultiSQL(storepb.Engine_CASSANDRA, "SELECT 1;;\n-- c\n/* d */;\n-- comment\nSELECT 2;\n// done")
	require.NoError(t, err)
	var empty []bool
	for _, stmt := range stmts {
		empty = append(empty, stmt.Empty)
	}
	require.Equal(t, []bool{false, true, true, false, true}, empty)
}

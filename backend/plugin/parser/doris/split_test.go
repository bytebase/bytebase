package doris

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDorisSplitSQLSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_DORIS, "SELECT 1;\n\n  SELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "SELECT 1;", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[0].End)
	require.Equal(t, "\n\n  SELECT 2;", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 3, Column: 12}, stmts[1].End)
}

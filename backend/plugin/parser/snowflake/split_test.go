package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSnowflakeSplitSQLSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_SNOWFLAKE, "SELECT 1;\n\n  SELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "SELECT 1;", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[0].End)
	require.Equal(t, "\n\n  SELECT 2;", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 10}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 3, Column: 12}, stmts[1].End)

	// The lone ';' between them is kept as an empty segment, as in pg and mysql.
	stmts, err = base.SplitMultiSQL(storepb.Engine_SNOWFLAKE, "SELECT 1;;SELECT 2;")
	require.NoError(t, err)
	require.Len(t, stmts, 3)
	require.False(t, stmts[0].Empty)
	require.True(t, stmts[1].Empty)
	require.Equal(t, ";", stmts[1].Text)
	require.False(t, stmts[2].Empty)
}

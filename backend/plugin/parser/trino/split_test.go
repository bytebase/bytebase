package trino

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTrinoSplitSQLSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_TRINO, "SELECT * FROM 表名; INSERT INTO 表 VALUES (1);")
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	require.Equal(t, "SELECT * FROM 表名;", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 18}, stmts[0].End)
	require.Equal(t, &storepb.Range{Start: 0, End: 21}, stmts[0].Range)

	require.Equal(t, " INSERT INTO 表 VALUES (1);", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 18}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 44}, stmts[1].End)
	require.Equal(t, &storepb.Range{Start: 21, End: 49}, stmts[1].Range)
}

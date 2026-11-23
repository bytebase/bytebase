package standard

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	script := "SELECT '中';\nSELECT '文';"

	for _, engine := range []storepb.Engine{
		storepb.Engine_CLICKHOUSE,
		storepb.Engine_HIVE,
		storepb.Engine_DATABRICKS,
	} {
		t.Run(engine.String(), func(t *testing.T) {
			stmts, err := base.SplitMultiSQL(engine, script)
			require.NoError(t, err)
			require.Len(t, stmts, 2)

			require.Equal(t, "SELECT '中';", stmts[0].Text)
			require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
			require.Equal(t, &storepb.Position{Line: 1, Column: 12}, stmts[0].End)
			require.Equal(t, &storepb.Range{Start: 0, End: 13}, stmts[0].Range)
			require.False(t, stmts[0].Empty)

			require.Equal(t, "\nSELECT '文';", stmts[1].Text)
			require.Equal(t, &storepb.Position{Line: 1, Column: 12}, stmts[1].Start)
			require.Equal(t, &storepb.Position{Line: 2, Column: 12}, stmts[1].End)
			require.Equal(t, &storepb.Range{Start: 13, End: 27}, stmts[1].Range)
			require.False(t, stmts[1].Empty)
		})
	}
}

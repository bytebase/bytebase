package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	const script = "SELECT '你好';\nSELECT 2;\n  "
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		t.Run(engine.String(), func(t *testing.T) {
			stmts, err := base.SplitMultiSQL(engine, script)
			require.NoError(t, err)
			require.Len(t, stmts, 3)

			require.Equal(t, "SELECT '你好';", stmts[0].Text)
			require.False(t, stmts[0].Empty)
			require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
			require.Equal(t, &storepb.Position{Line: 1, Column: 13}, stmts[0].End)
			require.Equal(t, &storepb.Range{Start: 0, End: int32(len("SELECT '你好';"))}, stmts[0].Range)

			require.Equal(t, "\nSELECT 2;", stmts[1].Text)
			require.False(t, stmts[1].Empty)
			require.Equal(t, int32(2), stmts[1].End.Line)

			require.True(t, stmts[2].Empty)
		})
	}
}

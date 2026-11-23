package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSpannerSplitSQLSeam(t *testing.T) {
	// ContiguousSplitText attaches inter-statement whitespace to the following statement.
	stmts, err := base.SplitMultiSQL(storepb.Engine_SPANNER, "SELECT * FROM users;\n\tSELECT * FROM orders;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "SELECT * FROM users;", stmts[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, stmts[0].Start)
	require.Equal(t, &storepb.Position{Line: 1, Column: 21}, stmts[0].End)
	require.Equal(t, "\n\tSELECT * FROM orders;", stmts[1].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 21}, stmts[1].Start)
	require.Equal(t, &storepb.Position{Line: 2, Column: 23}, stmts[1].End)

	// CASE ... END blocks stay whole across statements.
	stmts, err = SplitSQL("SELECT CASE WHEN a = 1 THEN 1 ELSE 0 END FROM users;\nSELECT * FROM orders;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.False(t, stmts[0].Empty)
	require.False(t, stmts[1].Empty)
}

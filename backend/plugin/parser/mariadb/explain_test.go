package mariadb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_MARIADB, "SELECT 1", base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	got, err = base.ExplainStatement(storepb.Engine_MARIADB, "SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN FORMAT=JSON SELECT 1", got)

	got, err = base.ExplainStatement(storepb.Engine_MARIADB, "SELECT 1", base.ExplainFormatText)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	_, err = base.ExplainStatement(storepb.Engine_MARIADB, "ANALYZE SELECT 1", base.ExplainFormatDefault)
	require.ErrorIs(t, err, base.ErrExplainAnalyzeExecutes)

	_, err = base.ExplainStatement(storepb.Engine_MARIADB, "SELECT * FORM t", base.ExplainFormatDefault)
	require.ErrorAs(t, err, new(*base.SyntaxError))

	format, executed, isExplain := DescribeExplain("EXPLAIN FORMAT=JSON SELECT 1")
	require.True(t, isExplain)
	require.Equal(t, "json", format)
	require.False(t, executed)
}

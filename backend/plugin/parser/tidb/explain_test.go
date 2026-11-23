package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_TIDB, "SELECT 1", base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	got, err = base.ExplainStatement(storepb.Engine_TIDB, "SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN FORMAT=tidb_json SELECT 1", got)

	got, err = base.ExplainStatement(storepb.Engine_TIDB, "SELECT 1", base.ExplainFormatText)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	_, err = base.ExplainStatement(storepb.Engine_TIDB, "EXPLAIN ANALYZE SELECT 1", base.ExplainFormatDefault)
	require.ErrorIs(t, err, base.ErrExplainAnalyzeExecutes)

	_, err = base.ExplainStatement(storepb.Engine_TIDB, "SELECT * FORM t", base.ExplainFormatDefault)
	require.ErrorAs(t, err, new(*base.SyntaxError))

	format, executed, isExplain := DescribeExplain("EXPLAIN ANALYZE SELECT 1")
	require.True(t, isExplain)
	require.Equal(t, "text", format)
	require.True(t, executed)
}

package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	for _, tc := range []struct {
		engine storepb.Engine
		format base.ExplainFormat
		want   string
	}{
		{storepb.Engine_POSTGRES, base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{storepb.Engine_POSTGRES, base.ExplainFormatText, "EXPLAIN SELECT 1"},
		{storepb.Engine_POSTGRES, base.ExplainFormatJSON, "EXPLAIN (FORMAT JSON) SELECT 1"},
		{storepb.Engine_POSTGRES, base.ExplainFormatXML, "EXPLAIN (FORMAT XML) SELECT 1"},
		{storepb.Engine_POSTGRES, base.ExplainFormatYAML, "EXPLAIN (FORMAT YAML) SELECT 1"},
		{storepb.Engine_COCKROACHDB, base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{storepb.Engine_COCKROACHDB, base.ExplainFormatJSON, "EXPLAIN (DISTSQL) SELECT 1"},
	} {
		got, err := base.ExplainStatement(tc.engine, "SELECT 1", tc.format)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}

	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		_, err := base.ExplainStatement(engine, "EXPLAIN ANALYZE SELECT 1", base.ExplainFormatJSON)
		require.ErrorIs(t, err, base.ErrExplainAnalyzeExecutes)
	}

	format, executed, ok := DescribeExplain("EXPLAIN (ANALYZE, FORMAT JSON) SELECT 1")
	require.True(t, ok)
	require.True(t, executed)
	require.Equal(t, "json", format)
}

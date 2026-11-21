package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_SNOWFLAKE, "SELECT 1", base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	got, err = base.ExplainStatement(storepb.Engine_SNOWFLAKE, "SELECT 1", base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN USING JSON SELECT 1", got)

	format, executed, isExplain := DescribeExplain("EXPLAIN USING JSON SELECT 1")
	require.Equal(t, "json", format)
	require.False(t, executed)
	require.True(t, isExplain)

	_, err = base.ExplainStatement(storepb.Engine_SNOWFLAKE, "SELECT 1; SELECT 2", base.ExplainFormatDefault)
	require.Error(t, err)
}

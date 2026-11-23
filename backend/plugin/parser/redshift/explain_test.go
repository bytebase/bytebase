package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_REDSHIFT, "SELECT 1", base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	_, err = base.ExplainStatement(storepb.Engine_REDSHIFT, "EXPLAIN ANALYZE DELETE FROM t", base.ExplainFormatDefault)
	require.ErrorIs(t, err, base.ErrExplainAnalyzeExecutes)
}

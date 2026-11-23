package doris

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_DORIS, "SELECT 1", base.ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", got)

	_, err = base.ExplainStatement(storepb.Engine_DORIS, "SELECT 1; SELECT 2", base.ExplainFormatDefault)
	require.Error(t, err)

	_, err = base.ExplainStatement(storepb.Engine_DORIS, "SELECT FROM", base.ExplainFormatDefault)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}

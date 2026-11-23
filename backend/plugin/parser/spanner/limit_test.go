package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	require.Equal(t, "SELECT * FROM t LIMIT 100", base.StatementWithResultLimit(storepb.Engine_SPANNER, "SELECT * FROM t", 100, ""))
	require.Equal(t, "SELECT x FROM t |> WHERE x > 1 |> LIMIT 100", base.StatementWithResultLimit(storepb.Engine_SPANNER, "SELECT x FROM t |> WHERE x > 1", 100, ""))
	require.Equal(t, "SELECT x FROM t LIMIT 10", base.StatementWithResultLimit(storepb.Engine_SPANNER, "SELECT x FROM t LIMIT 10", 100, ""))
}

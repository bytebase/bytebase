package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	got := base.StatementWithResultLimit(storepb.Engine_MSSQL, "SELECT * FROM dbo.t;", 10, "")
	require.Equal(t, "SELECT TOP 10 * FROM dbo.t;", got)

	// On omni rewrite failure, falls back to CTE wrapper.
	got = base.StatementWithResultLimit(storepb.Engine_MSSQL, "SELECT FROM ;", 10, "")
	require.Equal(t, "WITH result AS (SELECT FROM) SELECT TOP 10 * FROM result;", got)
}

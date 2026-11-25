package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	engine := storepb.Engine_POSTGRES
	require.Equal(t, "SELECT * FROM t LIMIT 10",
		base.StatementWithResultLimit(engine, "SELECT * FROM t LIMIT 100", 10, ""))
	require.Equal(t, "SELECT * FROM t LIMIT 0",
		base.StatementWithResultLimit(engine, "SELECT * FROM t LIMIT 0", 10, ""))
	require.Equal(t, "DELETE FROM t",
		base.StatementWithResultLimit(engine, "DELETE FROM t", 10, ""))

	// When omni cannot rewrite inline (e.g. LIMIT ALL / LIMIT NULL), Bytebase wraps in a CTE.
	require.Equal(t, "WITH result AS (\nSELECT * FROM t LIMIT ALL OFFSET 2\n) SELECT * FROM result LIMIT 10;",
		base.StatementWithResultLimit(engine, "SELECT * FROM t LIMIT ALL OFFSET 2;", 10, ""))
}

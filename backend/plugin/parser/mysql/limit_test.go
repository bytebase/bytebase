package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	require.Equal(t, "SELECT * FROM t LIMIT 10;", base.StatementWithResultLimit(storepb.Engine_MYSQL, "SELECT * FROM t;", 10, ""))
	require.Equal(t, "SELECT * FROM t LIMIT 5;", base.StatementWithResultLimit(storepb.Engine_MYSQL, "SELECT * FROM t LIMIT 10;", 5, ""))

	// Statements omni cannot rewrite fall back to subquery wrapping.
	require.Equal(t, "SELECT * FROM (SELECT * FROM t LIMIT @row_count) result LIMIT 5;", base.StatementWithResultLimit(storepb.Engine_MYSQL, "SELECT * FROM t LIMIT @row_count", 5, ""))
}

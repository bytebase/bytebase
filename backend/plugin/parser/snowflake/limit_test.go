package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	require.Equal(t, "SELECT * FROM CUSTOMER LIMIT 10\n;", base.StatementWithResultLimit(storepb.Engine_SNOWFLAKE, "SELECT * FROM CUSTOMER LIMIT 15;", 10, ""))
	// Fallback wrapper when omni cannot rewrite.
	require.Equal(t, "SELECT * FROM (SELECT * FROM T OFFSET 2) LIMIT 10", base.StatementWithResultLimit(storepb.Engine_SNOWFLAKE, "SELECT * FROM T OFFSET 2;", 10, ""))
}

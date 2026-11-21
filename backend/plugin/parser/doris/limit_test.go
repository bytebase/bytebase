package doris

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	require.Equal(t, "SELECT * FROM t LIMIT 10;", base.StatementWithResultLimit(storepb.Engine_DORIS, "SELECT * FROM t;", 10, ""))
	require.Equal(t, "SHOW DATA", base.StatementWithResultLimit(storepb.Engine_DORIS, "SHOW DATA", 10, ""))
}

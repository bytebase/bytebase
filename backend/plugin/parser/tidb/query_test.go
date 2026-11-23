package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditorSeam(t *testing.T) {
	valid, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_TIDB, "SELECT * FROM t1;")
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, allQuery)

	valid, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_TIDB, "SET max_execution_time = 1000; SELECT * FROM t;")
	require.NoError(t, err)
	require.True(t, valid)
	require.False(t, allQuery)

	valid, _, err = base.ValidateSQLForEditor(storepb.Engine_TIDB, "EXPLAIN ANALYZE DELETE FROM t;")
	require.NoError(t, err)
	require.False(t, valid)

	_, _, err = base.ValidateSQLForEditor(storepb.Engine_TIDB, "SELECT * FORM invalid_table;")
	require.ErrorAs(t, err, new(*base.SyntaxError))
}

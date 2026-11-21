package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditorSeam(t *testing.T) {
	valid, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_ORACLE, "SELECT 1 FROM DUAL;")
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, allQuery)

	valid, _, err = base.ValidateSQLForEditor(storepb.Engine_ORACLE, "DELETE FROM t;")
	require.NoError(t, err)
	require.False(t, valid)

	_, _, err = base.ValidateSQLForEditor(storepb.Engine_ORACLE, "SELECT FROM ;")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(1), syntaxErr.Position.Line)
}

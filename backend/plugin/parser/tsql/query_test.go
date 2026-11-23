package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditorSeam(t *testing.T) {
	valid, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_MSSQL, "SELECT 1;")
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, allQuery)

	valid, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_MSSQL, "SET NOCOUNT ON; SELECT 1;")
	require.NoError(t, err)
	require.True(t, valid)
	require.False(t, allQuery)

	valid, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_MSSQL, "DELETE FROM dbo.t;")
	require.NoError(t, err)
	require.False(t, valid)
	require.False(t, allQuery)

	_, _, err = base.ValidateSQLForEditor(storepb.Engine_MSSQL, "SELECT FROM ;")
	require.Error(t, err)
}

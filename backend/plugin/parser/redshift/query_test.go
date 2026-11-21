package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditorSeam(t *testing.T) {
	readOnly, returnsData, err := base.ValidateSQLForEditor(storepb.Engine_REDSHIFT, "SELECT * FROM users")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_REDSHIFT, "SET search_path TO public")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.False(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_REDSHIFT, "DELETE FROM users WHERE id = 1")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, returnsData)

	_, _, err = base.ValidateSQLForEditor(storepb.Engine_REDSHIFT, "SELEC * FORM users")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.NotNil(t, syntaxErr.Position)
	require.Equal(t, int32(1), syntaxErr.Position.Line)
}

package trino

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateQuerySeam(t *testing.T) {
	readOnly, returnsData, err := base.ValidateSQLForEditor(storepb.Engine_TRINO, "SELECT * FROM users")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_TRINO, "DELETE FROM users WHERE id = 1")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, returnsData)

	_, _, err = base.ValidateSQLForEditor(storepb.Engine_TRINO, "SELECT * FFROM users")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}

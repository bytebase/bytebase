package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseAndValidateQuerySeam(t *testing.T) {
	// parseCosmosDB populates Node, Text, and 1-based StartPosition.
	res, err := parseCosmosDB("\n  SELECT c.id FROM users c WHERE c.age > 18")
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NotNil(t, res[0].Node)
	require.Equal(t, int32(2), res[0].StartPosition.Line)
	require.Equal(t, int32(3), res[0].StartPosition.Column)

	empty, err := parseCosmosDB("   \n\t  ")
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = parseCosmosDB("SELCT * FRM c")
	require.Error(t, err)

	// Registry dispatch for ValidateSQLForEditor.
	readOnly, returnsData, err := base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, "SELECT * FROM c")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, returnsData)

	readOnly, returnsData, err = base.ValidateSQLForEditor(storepb.Engine_COSMOSDB, "SELECT * FROM c WHERE c.name = 'unclosed")
	require.Error(t, err)
	require.False(t, readOnly)
	require.False(t, returnsData)
}

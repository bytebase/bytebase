//nolint:revive
package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInstanceDatabaseID(t *testing.T) {
	instanceID, err := GetInstanceID("instances/i2")
	require.NoError(t, err)
	require.Equal(t, "i2", instanceID)

	_, err = GetInstanceID("instances/i2/databases/d3")
	require.Error(t, err)
}

func TestGetSchemaTableName(t *testing.T) {
	schema, table, err := GetSchemaTableName("schemas/a/tables/b")
	require.NoError(t, err)
	require.Equal(t, "a", schema)
	require.Equal(t, "b", table)

	schema, table, err = GetSchemaTableName("schemas//tables/b")
	require.NoError(t, err)
	require.Equal(t, "", schema)
	require.Equal(t, "b", table)

	schema, table, err = GetSchemaTableName("schemas/-/tables/b")
	require.NoError(t, err)
	require.Equal(t, "", schema)
	require.Equal(t, "b", table)
}

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

func TestGetProjectIDQueryHistoryID(t *testing.T) {
	projectID, historyID, err := GetProjectIDQueryHistoryID("projects/p1/queryHistories/550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	require.Equal(t, "p1", projectID)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", historyID)

	_, _, err = GetProjectIDQueryHistoryID("projects/p1/queryHistories")
	require.Error(t, err)

	_, _, err = GetProjectIDQueryHistoryID("queryHistories/h1")
	require.Error(t, err)
}

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

func TestGetProjectAndBranchID(t *testing.T) {
	projectID, branchID, err := GetProjectAndBranchID("projects/aaa/branches/feat/hello")
	require.NoError(t, err)
	require.Equal(t, "aaa", projectID)
	require.Equal(t, "feat/hello", branchID)

	projectID, branchID, err = GetProjectAndBranchID("projects/new-project/branches/vv")
	require.NoError(t, err)
	require.Equal(t, "new-project", projectID)
	require.Equal(t, "vv", branchID)
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
}

func TestGetRiskID(t *testing.T) {
	tests := []struct {
		name string
		want int64
	}{
		{
			name: "risks/1234",
			want: 1234,
		},
		{
			name: "risks/12345678901",
			want: 12345678901,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got, err := GetRiskID(test.name)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

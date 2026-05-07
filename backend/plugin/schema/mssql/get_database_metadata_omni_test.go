package mssql

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetDatabaseMetadataOmniParityWithANTLR(t *testing.T) {
	tests := []getDatabaseMetadataCase{}
	yamlFile, err := os.Open("test-data/test_get_database_metadata.yaml")
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(byteValue, &tests))

	for _, tc := range tests {
		antlrMeta, err := getDatabaseMetadataANTLR(tc.Input)
		require.NoError(t, err, tc.Input)
		omniMeta, err := getDatabaseMetadataOmni(tc.Input)
		require.NoError(t, err, tc.Input)

		antlrJSON, err := json.MarshalIndent(antlrMeta, "", "  ")
		require.NoError(t, err)
		omniJSON, err := json.MarshalIndent(omniMeta, "", "  ")
		require.NoError(t, err)
		require.Equal(t, string(antlrJSON), string(omniJSON), tc.Input)
	}
}

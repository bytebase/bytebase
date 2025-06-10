package pg

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type getDatabaseMetadataCase struct {
	Input          string `yaml:"input"`
	MetadataResult string `yaml:"metadata_result"`
}

func TestGetDatabaseMetadata(t *testing.T) {
	tests := []getDatabaseMetadataCase{}
	const (
		record = false
	)
	var (
		filepath = "testdata/get_database_metadata.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, tc := range tests {
		// Get metadata from SQL
		meta, err := GetDatabaseMetadata(tc.Input)
		a.NoError(err, "Test case %d: Failed to get metadata", i)

		// Convert metadata to JSON for comparison
		jsonBytes, err := json.MarshalIndent(meta, "", "  ")
		a.NoError(err)
		metadataResult := string(jsonBytes)

		if record {
			tests[i].MetadataResult = metadataResult
		} else {
			a.Equal(tc.MetadataResult, metadataResult, "Test case %d metadata mismatch: %s", i, tc.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

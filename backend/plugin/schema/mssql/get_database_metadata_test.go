package mssql

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type getDatabaseMetadataCase struct {
	Input  string
	Result string
}

func TestGetDatabaseMetadata(t *testing.T) {
	tests := []getDatabaseMetadataCase{}
	const (
		record = false
	)
	var (
		filepath = "test-data/test_get_database_metadata.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		meta, err := GetDatabaseMetadata(t.Input)
		a.NoError(err)

		jsonBytes, err := json.MarshalIndent(meta, "", "  ")
		a.NoError(err)
		result := string(jsonBytes)

		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Input)
		}
	}
	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

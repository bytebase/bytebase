package mysql

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
)

type getDatabaseMetadataCase struct {
	Description string `yaml:"description"`
	Schema      string `yaml:"schema"`
	Metadata    string `yaml:"metadata"`
}

// TestGetDatabaseMetadata pins what the parser extracts from each schema text.
// The goldens describe this package's own output, not MySQL's catalog: whether
// the parser agrees with a live server is engine conformance and belongs in omni.
func TestGetDatabaseMetadata(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_metadata.yaml"
	)

	var tests []getDatabaseMetadataCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			metadata, err := GetDatabaseMetadata(tc.Schema)
			require.NoError(t, err)

			encoded, err := json.MarshalIndent(metadata, "", "  ")
			require.NoError(t, err)
			result := string(encoded) + "\n"

			if record {
				tests[i].Metadata = result
				return
			}
			require.Equal(t, tc.Metadata, result)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

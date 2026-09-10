package tidb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

type getDatabaseDefinitionCase struct {
	Description string `yaml:"description"`
	Metadata    string `yaml:"metadata"`
	Expected    string `yaml:"expected"`
}

// TestGetDatabaseDefinition pins the DDL generated for each metadata input, and
// re-parses that DDL to prove it is valid TiDB syntax.
//
// The input is metadata rather than a schema text on purpose: deriving it by
// parsing would cap this test's reach at whatever GetDatabaseMetadata happens to
// extract, so any generator branch the parser cannot feed would go untested.
func TestGetDatabaseDefinition(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_definition.yaml"
	)

	var tests []getDatabaseDefinitionCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			var metadata storepb.DatabaseSchemaMetadata
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), &metadata))

			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{PrintHeader: true}, &metadata)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			_, err = GetDatabaseMetadata(definition)
			require.NoError(t, err, "generated definition should parse")

			if record {
				tests[i].Expected = definition
				return
			}
			require.Equal(t, tc.Expected, definition)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

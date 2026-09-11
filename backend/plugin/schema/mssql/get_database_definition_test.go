package mssql

import (
	"os"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
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

// TestGetDatabaseDefinition pins the DDL generated for each metadata input.
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

			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, &metadata)
			require.NoError(t, err)

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

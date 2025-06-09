package mssql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type getDatabaseDefinitionTestCase struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

func TestGetDatabaseDefinition(t *testing.T) {
	tests := []getDatabaseDefinitionTestCase{}
	const (
		record = false
	)
	var (
		filepath = "test-data/test_get_database_definition.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, tc := range tests {
		var metadata storepb.DatabaseSchemaMetadata
		err := common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Input), &metadata)
		a.NoError(err)

		result, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, &metadata)
		a.NoError(err)

		// Parse the output to ensure it's valid TSQL
		_, err = tsql.ParseTSQL(result)
		if err != nil {
			t.Logf("Test case %d SQL:\n%s", i, result)
		}
		a.NoError(err, "Test case %d: Failed to parse generated SQL", i)

		if record {
			tests[i].Output = result
		} else {
			a.Equal(tc.Output, result, "Test case %d", i)
		}
	}
	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

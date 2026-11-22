package mssql

import (
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type generateMigrationCase struct {
	Description string `yaml:"description"`
	OldSchema   string `yaml:"oldSchema"`
	NewSchema   string `yaml:"newSchema"`
	Expected    string `yaml:"expected"`
}

// TestGenerateMigration pins the DDL that takes oldSchema to newSchema.
func TestGenerateMigration(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/generate_migration.yaml"
	)

	var tests []generateMigrationCase
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		t.Run(test.Description, func(t *testing.T) {
			oldDBSchema, newDBSchema := parseMigrationSides(t, i, test.Description, test.OldSchema, test.NewSchema)

			migration, err := schema.DiffMigration(storepb.Engine_MSSQL, oldDBSchema, newDBSchema)
			require.NoErrorf(t, err, "Failed to generate migration for test case [%02d]: %s", i+1, test.Description)

			if record {
				tests[i].Expected = migration
				return
			}
			require.Equalf(t, test.Expected, migration, "Test case [%02d] failed: %s", i+1, test.Description)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// parseMigrationSides turns the fixture's schema texts into the two sides the
// migration diff takes, keeping the empty-side shape the old differ used.
func parseMigrationSides(t *testing.T, i int, description, oldSchema, newSchema string) (*model.DatabaseMetadata, *model.DatabaseMetadata) {
	t.Helper()
	var oldMetadata *metadatapb.DatabaseSchemaMetadata
	var err error
	if oldSchema != "" {
		oldMetadata, err = GetDatabaseMetadata(oldSchema)
		require.NoErrorf(t, err, "Failed to parse old schema for test case [%02d]: %s", i+1, description)
	}
	newMetadata, err := GetDatabaseMetadata(newSchema)
	require.NoErrorf(t, err, "Failed to parse new schema for test case [%02d]: %s", i+1, description)

	var oldDBSchema *model.DatabaseMetadata
	if oldMetadata != nil {
		oldDBSchema = model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_MSSQL, false)
	}
	newDBSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_MSSQL, false)
	if oldSchema == "" && oldDBSchema == nil {
		oldDBSchema = model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, nil, nil, storepb.Engine_MSSQL, false)
	}
	if newSchema == "" {
		newDBSchema = model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{{Name: "dbo", Tables: []*metadatapb.TableMetadata{}}},
		}, nil, nil, storepb.Engine_MSSQL, false)
	}
	return oldDBSchema, newDBSchema
}

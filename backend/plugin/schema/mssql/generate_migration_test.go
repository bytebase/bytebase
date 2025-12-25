package mssql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type MigrationTestData struct {
	Description string `yaml:"description"`
	OldSchema   string `yaml:"oldSchema"`
	NewSchema   string `yaml:"newSchema"`
	Expected    string `yaml:"expected"`
}

func runMigrationTest(t *testing.T, file string) {
	var tests []MigrationTestData
	filepath := filepath.Join("test-data", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		t.Run(test.Description, func(t *testing.T) {
			// Parse old schema
			var oldMetadata *storepb.DatabaseSchemaMetadata
			if test.OldSchema != "" {
				oldMetadata, err = GetDatabaseMetadata(test.OldSchema)
				require.NoErrorf(t, err, "Failed to parse old schema for test case [%02d]: %s", i+1, test.Description)
			}

			// Parse new schema
			newMetadata, err := GetDatabaseMetadata(test.NewSchema)
			require.NoErrorf(t, err, "Failed to parse new schema for test case [%02d]: %s", i+1, test.Description)

			// Convert to model.DatabaseSchema
			var oldDBSchema *model.DatabaseMetadata
			if oldMetadata != nil {
				oldDBSchema = model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}
			newDBSchema := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_MSSQL, false)

			// Get diff
			var diff *schema.MetadataDiff

			// Handle case where old schema is empty (creating from scratch)
			if test.OldSchema == "" && oldDBSchema == nil {
				// Create empty database schema for comparison
				emptyMetadata := &storepb.DatabaseSchemaMetadata{
					Name:    "",
					Schemas: []*storepb.SchemaMetadata{},
				}
				oldDBSchema = model.NewDatabaseMetadata(emptyMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}

			// Handle case where new schema is empty (dropping everything)
			if test.NewSchema == "" {
				// Create empty metadata with dbo schema to match the structure
				emptyMetadata := &storepb.DatabaseSchemaMetadata{
					Name: "",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name:   "dbo",
							Tables: []*storepb.TableMetadata{},
						},
					},
				}
				newDBSchema = model.NewDatabaseMetadata(emptyMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}

			diff, err = schema.GetDatabaseSchemaDiff(storepb.Engine_MSSQL, oldDBSchema, newDBSchema)
			require.NoErrorf(t, err, "Failed to get schema diff for test case [%02d]: %s", i+1, test.Description)

			// Generate migration
			migration, err := generateMigration(diff)
			require.NoErrorf(t, err, "Failed to generate migration for test case [%02d]: %s", i+1, test.Description)

			require.Equalf(t, test.Expected, migration, "Test case [%02d] failed: %s", i+1, test.Description)
		})
	}
}

func TestGenerateMigration_Tables(t *testing.T) {
	runMigrationTest(t, "test_migration_tables.yaml")
}

func TestGenerateMigration_Indexes(t *testing.T) {
	runMigrationTest(t, "test_migration_indexes.yaml")
}

func TestGenerateMigration_Constraints(t *testing.T) {
	runMigrationTest(t, "test_migration_constraints.yaml")
}

func TestGenerateMigration_Functions(t *testing.T) {
	runMigrationTest(t, "test_migration_functions.yaml")
}

func TestGenerateMigration_Procedures(t *testing.T) {
	runMigrationTest(t, "test_migration_procedures.yaml")
}

func TestGenerateMigration_Views(t *testing.T) {
	runMigrationTest(t, "test_migration_views.yaml")
}

func TestGenerateMigration_SafeOrder(t *testing.T) {
	runMigrationTest(t, "test_migration_safe_order.yaml")
}

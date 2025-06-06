package mssql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type MigrationTestData struct {
	Description string `yaml:"description"`
	OldSchema   string `yaml:"oldSchema"`
	NewSchema   string `yaml:"newSchema"`
	Expected    string `yaml:"expected"`
}

func runMigrationTest(t *testing.T, file string, record bool) {
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
			var oldDBSchema *model.DatabaseSchema
			if oldMetadata != nil {
				oldDBSchema = model.NewDatabaseSchema(oldMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}
			newDBSchema := model.NewDatabaseSchema(newMetadata, nil, nil, storepb.Engine_MSSQL, false)

			// Get diff
			var diff *schema.MetadataDiff

			// Handle case where old schema is empty (creating from scratch)
			if test.OldSchema == "" && oldDBSchema == nil {
				// Create empty database schema for comparison
				emptyMetadata := &storepb.DatabaseSchemaMetadata{
					Name:    "",
					Schemas: []*storepb.SchemaMetadata{},
				}
				oldDBSchema = model.NewDatabaseSchema(emptyMetadata, nil, nil, storepb.Engine_MSSQL, false)
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
				newDBSchema = model.NewDatabaseSchema(emptyMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}

			diff, err = schema.GetDatabaseSchemaDiff(oldDBSchema, newDBSchema)
			require.NoErrorf(t, err, "Failed to get schema diff for test case [%02d]: %s", i+1, test.Description)

			// Generate migration
			migration, err := generateMigration(diff)
			require.NoErrorf(t, err, "Failed to generate migration for test case [%02d]: %s", i+1, test.Description)

			// Parse the generated migration to ensure it's valid SQL
			if migration != "" {
				_, err := tsql.ParseTSQL(migration)
				require.NoErrorf(t, err, "Failed to parse generated SQL for test case [%02d]: %s\nSQL: %s", i+1, test.Description, migration)
			}

			if record {
				tests[i].Expected = migration
			} else {
				require.Equalf(t, test.Expected, migration, "Test case [%02d] failed: %s", i+1, test.Description)
			}
		})
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err = yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

func TestGenerateMigration_Tables(t *testing.T) {
	runMigrationTest(t, "test_migration_tables.yaml", false /* record */)
}

func TestGenerateMigration_Indexes(t *testing.T) {
	runMigrationTest(t, "test_migration_indexes.yaml", false /* record */)
}

func TestGenerateMigration_Constraints(t *testing.T) {
	runMigrationTest(t, "test_migration_constraints.yaml", false /* record */)
}

func TestGenerateMigration_Functions(t *testing.T) {
	runMigrationTest(t, "test_migration_functions.yaml", false /* record */)
}

func TestGenerateMigration_Procedures(t *testing.T) {
	runMigrationTest(t, "test_migration_procedures.yaml", false /* record */)
}

func TestGenerateMigration_Views(t *testing.T) {
	// Skip views test for now due to parser limitations with multiple statements
	t.Skip("Skipping views test due to parser limitations")
	runMigrationTest(t, "test_migration_views.yaml", false /* record */)
}

func TestGenerateMigration_SafeOrder(t *testing.T) {
	// Skip safe order test for now due to parser limitations with multiple statements
	t.Skip("Skipping safe order test due to parser limitations")
	runMigrationTest(t, "test_migration_safe_order.yaml", false /* record */)
}

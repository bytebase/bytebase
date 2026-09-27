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
			// Parse old schema
			var oldMetadata *metadatapb.DatabaseSchemaMetadata
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
				emptyMetadata := &metadatapb.DatabaseSchemaMetadata{
					Name:    "",
					Schemas: []*metadatapb.SchemaMetadata{},
				}
				oldDBSchema = model.NewDatabaseMetadata(emptyMetadata, nil, nil, storepb.Engine_MSSQL, false)
			}

			// Handle case where new schema is empty (dropping everything)
			if test.NewSchema == "" {
				// Create empty metadata with dbo schema to match the structure
				emptyMetadata := &metadatapb.DatabaseSchemaMetadata{
					Name: "",
					Schemas: []*metadatapb.SchemaMetadata{
						{
							Name:   "dbo",
							Tables: []*metadatapb.TableMetadata{},
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

func TestGetViewDependencies(t *testing.T) {
	cases := []struct {
		name     string
		viewDef  string
		schema   string
		expected []string
	}{
		{
			name:     "simple",
			viewDef:  "CREATE VIEW [dbo].[v] AS SELECT id FROM [dbo].[a]",
			schema:   "dbo",
			expected: []string{"dbo.a"},
		},
		{
			name: "union two tables",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"SELECT id FROM [dbo].[a] UNION SELECT id FROM [dbo].[b]",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.b"},
		},
		{
			name: "union all three tables",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"SELECT id FROM [dbo].[a] UNION ALL " +
				"SELECT id FROM [dbo].[b] UNION ALL " +
				"SELECT id FROM [dbo].[c]",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.b", "dbo.c"},
		},
		{
			// GetQuerySpan with empty mock metadata cannot distinguish CTE
			// references from real table references, so cte names appear in
			// the dependency set. This is pre-existing behavior carried over
			// from the ANTLR implementation; the test pins it down.
			name: "cte",
			viewDef: "CREATE VIEW [dbo].[v] AS " +
				"WITH cte AS (SELECT id FROM [dbo].[a]) " +
				"SELECT id FROM cte",
			schema:   "dbo",
			expected: []string{"dbo.a", "dbo.cte"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getViewDependencies(tc.viewDef, tc.schema)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, got)
		})
	}
}

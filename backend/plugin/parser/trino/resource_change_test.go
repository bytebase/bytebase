package trino

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestExtractChangedResources(t *testing.T) {
	testCases := []struct {
		name            string
		sql             string
		currentDatabase string
		currentSchema   string
		expectedTables  []string
		expectedSchemas []string
		expectedViews   []string
		expectedDML     bool
		expectedInsert  bool
	}{
		{
			name:            "CREATE TABLE",
			sql:             "CREATE TABLE users (id INT, name VARCHAR)",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     false,
		},
		{
			name:            "DROP TABLE",
			sql:             "DROP TABLE users",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     false,
		},
		{
			name:            "CREATE TABLE with schema",
			sql:             "CREATE TABLE analytics.users (id INT, name VARCHAR)",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.analytics.users"},
			expectedDML:     false,
		},
		{
			name:            "CREATE TABLE with catalog and schema",
			sql:             "CREATE TABLE catalog1.analytics.users (id INT, name VARCHAR)",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"catalog1.analytics.users"},
			expectedDML:     false,
		},
		{
			name:            "CREATE VIEW",
			sql:             "CREATE VIEW active_users AS SELECT * FROM users WHERE active = true",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedViews:   []string{"test_db.public.active_users"},
			expectedDML:     false,
		},
		{
			name:            "ALTER TABLE",
			sql:             "ALTER TABLE users ADD COLUMN email VARCHAR",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     false,
		},
		{
			name:            "CREATE SCHEMA",
			sql:             "CREATE SCHEMA analytics",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedSchemas: []string{"test_db.analytics"},
			expectedDML:     false,
		},
		{
			name:            "DROP SCHEMA",
			sql:             "DROP SCHEMA analytics",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedSchemas: []string{"test_db.analytics"},
			expectedDML:     false,
		},
		{
			name:            "INSERT",
			sql:             "INSERT INTO users (id, name) VALUES (1, 'John')",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     true,
			expectedInsert:  true,
		},
		{
			name:            "UPDATE",
			sql:             "UPDATE users SET name = 'Jane' WHERE id = 1",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     true,
		},
		{
			name:            "DELETE",
			sql:             "DELETE FROM users WHERE id = 1",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{"test_db.public.users"},
			expectedDML:     true,
		},
		{
			name:            "SELECT (no change)",
			sql:             "SELECT * FROM users",
			currentDatabase: "test_db",
			currentSchema:   "public",
			expectedTables:  []string{},
			expectedDML:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create empty database schema for test
			dbSchemaMetadata := &storepb.DatabaseSchemaMetadata{
				Name: "test_db",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
					},
				},
			}

			dbSchema := model.NewDatabaseSchema(dbSchemaMetadata, nil, nil, storepb.Engine_TRINO, false)

			// Call the function
			summary, err := extractChangedResources(tc.currentDatabase, tc.currentSchema, dbSchema, nil, tc.sql)
			require.NoError(t, err, "extractChangedResources should not error")

			// Get the ChangedResources proto to check tables, schemas, and views
			changedResourcesProto := summary.ChangedResources.Build()

			// Collect tables, schemas, and views
			var actualTables []string
			var actualSchemas []string
			var actualViews []string

			for _, db := range changedResourcesProto.Databases {
				for _, schema := range db.Schemas {
					schemaKey := fmt.Sprintf("%s.%s", db.Name, schema.Name)
					// If this is a schema-level change (has the dummy table)
					if isDummyTableSchema(schema) {
						actualSchemas = append(actualSchemas, schemaKey)
					}

					// Collect tables and views
					for _, table := range schema.Tables {
						tableKey := fmt.Sprintf("%s.%s.%s", db.Name, schema.Name, table.Name)
						if table.Name != "__schema_change__" {
							actualTables = append(actualTables, tableKey)
						}
					}

					for _, view := range schema.Views {
						viewKey := fmt.Sprintf("%s.%s.%s", db.Name, schema.Name, view.Name)
						actualViews = append(actualViews, viewKey)
					}
				}
			}

			assert.ElementsMatch(t, tc.expectedTables, actualTables, "Tables list doesn't match")
			assert.ElementsMatch(t, tc.expectedSchemas, actualSchemas, "Schemas list doesn't match")
			assert.ElementsMatch(t, tc.expectedViews, actualViews, "Views list doesn't match")

			if tc.expectedDML {
				assert.Greater(t, summary.DMLCount, 0, "Expected DML count to be greater than 0")
				if tc.expectedInsert {
					assert.Greater(t, summary.InsertCount, 0, "Expected insert count to be greater than 0")
				}
			} else {
				assert.Equal(t, 0, summary.DMLCount, "Expected DML count to be 0")
				assert.Equal(t, 0, summary.InsertCount, "Expected insert count to be 0")
			}
		})
	}
}

// isDummyTableSchema checks if the schema has only a dummy table used to signal schema-level changes
func isDummyTableSchema(schema *storepb.ChangedResourceSchema) bool {
	if len(schema.Tables) != 1 {
		return false
	}
	return schema.Tables[0].Name == "__schema_change__"
}

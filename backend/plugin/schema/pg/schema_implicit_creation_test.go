package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestGetSDLDiff_ImplicitSchemaCreation tests that when users add tables in a new schema
// without explicit CREATE SCHEMA statement, the diff should include schema creation.
// It also tests explicit CREATE SCHEMA handling and mixed scenarios.
func TestGetSDLDiff_ImplicitSchemaCreation(t *testing.T) {
	tests := []struct {
		name                  string
		currentSDLText        string
		previousUserSDLText   string
		currentSchema         *model.DatabaseMetadata
		expectedSchemaChanges int
		expectedTableChanges  int
		expectSchemaCreation  bool
		expectedSchemaName    string
		expectedSchemaNames   []string // For cases with multiple schemas
	}{
		{
			name: "add_table_in_new_schema_without_create_schema",
			// Previous SDL: only has a table in public schema
			previousUserSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY,
    name VARCHAR(100)
);
`,
			// Current SDL: user adds a table in new_schema WITHOUT CREATE SCHEMA statement
			currentSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY,
    name VARCHAR(100)
);

CREATE TABLE new_schema.t(
    id INT PRIMARY KEY,
    value VARCHAR(50)
);
`,
			// Current database state: only has public schema with existing_table
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "existing_table",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
										{
											Name:     "name",
											Type:     "character varying(100)",
											Nullable: true,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSchemaChanges: 1, // Should detect new_schema creation
			expectedTableChanges:  1, // new_schema.t creation
			expectSchemaCreation:  true,
			expectedSchemaName:    "new_schema",
		},
		{
			name: "add_multiple_tables_in_new_schema_without_create_schema",
			previousUserSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);
`,
			currentSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);

CREATE TABLE new_schema.users(
    id INT PRIMARY KEY,
    name VARCHAR(100)
);

CREATE TABLE new_schema.orders(
    id INT PRIMARY KEY,
    user_id INT REFERENCES new_schema.users(id)
);
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "existing_table",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSchemaChanges: 1, // Should detect new_schema creation
			expectedTableChanges:  2, // users and orders creation
			expectSchemaCreation:  true,
			expectedSchemaName:    "new_schema",
		},
		{
			name: "add_table_in_new_schema_with_explicit_create_schema",
			// Previous SDL: only has a table in public schema
			previousUserSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY,
    name VARCHAR(100)
);
`,
			// Current SDL: user adds EXPLICIT CREATE SCHEMA statement and table
			currentSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY,
    name VARCHAR(100)
);

CREATE SCHEMA new_schema;

CREATE TABLE new_schema.t(
    id INT PRIMARY KEY,
    value VARCHAR(50)
);
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "existing_table",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
										{
											Name:     "name",
											Type:     "character varying(100)",
											Nullable: true,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSchemaChanges: 1, // Should detect new_schema creation
			expectedTableChanges:  1, // new_schema.t creation
			expectSchemaCreation:  true,
			expectedSchemaName:    "new_schema",
		},
		{
			name: "explicit_create_schema_without_objects",
			// Test that explicit CREATE SCHEMA alone is handled correctly
			previousUserSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);
`,
			currentSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);

CREATE SCHEMA new_schema;
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "existing_table",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSchemaChanges: 1, // Should detect new_schema creation
			expectedTableChanges:  0, // No table changes
			expectSchemaCreation:  true,
			expectedSchemaName:    "new_schema",
		},
		{
			name: "mixed_explicit_and_implicit_schemas",
			// Test: one schema created explicitly, another implicitly via table reference
			previousUserSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);
`,
			currentSDLText: `
CREATE TABLE public.existing_table(
    id INT PRIMARY KEY
);

CREATE SCHEMA schema_a;

CREATE TABLE schema_a.table_a(
    id INT PRIMARY KEY
);

CREATE TABLE schema_b.table_b(
    id INT PRIMARY KEY
);
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "existing_table",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
									},
								},
							},
						},
					},
				},
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSchemaChanges: 2,                                // Should detect both schema_a (explicit) and schema_b (implicit) creation
			expectedTableChanges:  2,                                // table_a and table_b creation
			expectSchemaCreation:  true,                             // For backward compatibility
			expectedSchemaName:    "schema_a",                       // For backward compatibility
			expectedSchemaNames:   []string{"schema_a", "schema_b"}, // Both schemas should be created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get SDL diff
			diff, err := GetSDLDiff(tt.currentSDLText, tt.previousUserSDLText, tt.currentSchema, nil)
			require.NoError(t, err, "GetSDLDiff should not return error")
			require.NotNil(t, diff, "Diff should not be nil")

			// Debug: print what we got
			t.Logf("Schema changes count: %d", len(diff.SchemaChanges))
			for i, sc := range diff.SchemaChanges {
				t.Logf("  SchemaChange[%d]: Action=%v, SchemaName=%s", i, sc.Action, sc.SchemaName)
			}
			t.Logf("Table changes count: %d", len(diff.TableChanges))
			for i, tc := range diff.TableChanges {
				t.Logf("  TableChange[%d]: Action=%v, Schema=%s, Table=%s", i, tc.Action, tc.SchemaName, tc.TableName)
			}

			// Verify schema changes
			require.Len(t, diff.SchemaChanges, tt.expectedSchemaChanges,
				"Should have %d schema change(s)", tt.expectedSchemaChanges)

			if tt.expectSchemaCreation {
				// If expectedSchemaNames is provided, check all of them
				if len(tt.expectedSchemaNames) > 0 {
					for _, expectedSchema := range tt.expectedSchemaNames {
						foundSchemaCreation := false
						for _, schemaChange := range diff.SchemaChanges {
							if schemaChange.Action == schema.MetadataDiffActionCreate &&
								schemaChange.SchemaName == expectedSchema {
								foundSchemaCreation = true
								break
							}
						}
						require.True(t, foundSchemaCreation,
							"Should detect CREATE action for schema %s", expectedSchema)
					}
				} else {
					// Backward compatibility: check for single expectedSchemaName
					foundSchemaCreation := false
					for _, schemaChange := range diff.SchemaChanges {
						if schemaChange.Action == schema.MetadataDiffActionCreate &&
							schemaChange.SchemaName == tt.expectedSchemaName {
							foundSchemaCreation = true
							break
						}
					}
					require.True(t, foundSchemaCreation,
						"Should detect CREATE action for schema %s", tt.expectedSchemaName)
				}
			}

			// Verify table changes
			require.Len(t, diff.TableChanges, tt.expectedTableChanges,
				"Should have %d table change(s)", tt.expectedTableChanges)

			// Generate migration to verify it includes CREATE SCHEMA
			migration, err := generateMigration(diff)
			require.NoError(t, err, "Migration generation should not fail")

			if tt.expectSchemaCreation {
				require.Contains(t, migration, "CREATE SCHEMA",
					"Migration should include CREATE SCHEMA statement")
				require.Contains(t, migration, tt.expectedSchemaName,
					"Migration should reference schema %s", tt.expectedSchemaName)

				t.Logf("Generated migration:\n%s", migration)
			}
		})
	}
}

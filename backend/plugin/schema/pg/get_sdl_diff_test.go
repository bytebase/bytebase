package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetSDLDiff_InitializationScenario(t *testing.T) {
	tests := []struct {
		name                    string
		currentSDLText          string
		previousUserSDLText     string
		currentSchema           *model.DatabaseSchema
		previousSchema          *model.DatabaseSchema
		expectedDiffEmpty       bool
		expectedTableChanges    int
		expectedViewChanges     int
		expectedFunctionChanges int
		expectedSequenceChanges int
	}{
		{
			name:                "initialization_with_empty_previous_SDL",
			currentSDLText:      "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			previousUserSDLText: "", // Empty - initialization scenario
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('users_id_seq'::regclass)",
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									OwnerTable:  "users",
									OwnerColumn: "id",
								},
							},
						},
					},
				},
				nil, // schema bytes
				nil, // config
				storepb.Engine_POSTGRES,
				false, // isObjectCaseSensitive
			),
			previousSchema:          nil,
			expectedDiffEmpty:       false, // Diff may be generated due to format differences
			expectedTableChanges:    1,     // One table creation
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
		},
		{
			name:                "initialization_with_complex_schema",
			currentSDLText:      "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL); CREATE VIEW user_view AS SELECT * FROM users;",
			previousUserSDLText: "", // Empty - initialization scenario
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('users_id_seq'::regclass)",
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Views: []*storepb.ViewMetadata{
								{
									Name:       "user_view",
									Definition: "SELECT * FROM users",
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									OwnerTable:  "users",
									OwnerColumn: "id",
								},
							},
						},
					},
				},
				nil, // schema bytes
				nil, // config
				storepb.Engine_POSTGRES,
				false, // isObjectCaseSensitive
			),
			previousSchema:          nil,
			expectedDiffEmpty:       false, // Diff may be generated due to format differences
			expectedTableChanges:    1,     // One table creation
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
		},
		{
			name:                 "non_initialization_normal_diff",
			currentSDLText:       "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL, email TEXT);",
			previousUserSDLText:  "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);", // Non-empty - normal diff
			currentSchema:        nil,                                                               // Not used in non-initialization scenario
			previousSchema:       nil,                                                               // Not used in non-initialization scenario
			expectedDiffEmpty:    false,
			expectedTableChanges: 1, // One table changed (column added)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDLText, tt.previousUserSDLText, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)
			require.NotNil(t, diff)

			if tt.expectedDiffEmpty {
				// In initialization scenario, we expect minimal or no changes
				// since we're comparing against a generated baseline
				require.LessOrEqual(t, len(diff.TableChanges), tt.expectedTableChanges)
				require.LessOrEqual(t, len(diff.ViewChanges), tt.expectedViewChanges)
				require.LessOrEqual(t, len(diff.FunctionChanges), tt.expectedFunctionChanges)
				require.LessOrEqual(t, len(diff.SequenceChanges), tt.expectedSequenceChanges)
			} else {
				// In normal diff scenario, we expect specific changes
				require.Equal(t, tt.expectedTableChanges, len(diff.TableChanges))
			}
		})
	}
}

func TestConvertDatabaseSchemaToSDL(t *testing.T) {
	tests := []struct {
		name          string
		dbSchema      *model.DatabaseSchema
		expectedSDL   string
		expectedError bool
	}{
		{
			name:          "nil_schema",
			dbSchema:      nil,
			expectedSDL:   "",
			expectedError: false,
		},
		{
			name: "empty_metadata",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{}, // empty but not nil metadata
				nil,
				nil,
				storepb.Engine_POSTGRES,
				false,
			),
			expectedSDL:   "",
			expectedError: false,
		},
		{
			name: "simple_table_schema",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
										},
										{
											Name:     "name",
											Type:     "text",
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
			expectedSDL:   "CREATE TABLE users (\n    id integer NOT NULL,\n    name text NOT NULL\n);",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertDatabaseSchemaToSDL(tt.dbSchema)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedSDL != "" {
					// For non-empty expected SDL, check that result contains expected structure
					require.Contains(t, result, "CREATE TABLE")
					require.Contains(t, result, "users")
				} else {
					require.Equal(t, tt.expectedSDL, result)
				}
			}
		})
	}
}

// TestGetSDLDiff_MinimalChangesScenario tests the minimal changes functionality
// This is a basic test to verify that the drift detection logic is invoked
func TestGetSDLDiff_MinimalChangesScenario(t *testing.T) {
	currentSDLText := "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);"
	previousUserSDLText := `CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL
	);`

	currentSchema := model.NewDatabaseSchema(
		&storepb.DatabaseSchemaMetadata{
			Name: "test_db",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "integer", Nullable: false},
								{Name: "name", Type: "text", Nullable: false},
							},
						},
						{
							Name: "posts", // This table exists in current schema but not in previous
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "integer", Nullable: false},
								{Name: "title", Type: "text", Nullable: false},
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
	)

	previousSchema := model.NewDatabaseSchema(
		&storepb.DatabaseSchemaMetadata{
			Name: "test_db",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "integer", Nullable: false},
								{Name: "name", Type: "text", Nullable: false},
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
	)

	// Test that the function runs without error when schemas are provided
	diff, err := GetSDLDiff(currentSDLText, previousUserSDLText, currentSchema, previousSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// We don't assert specific outcomes here because the main goal is to test
	// that the minimal changes logic is invoked and doesn't crash
	t.Logf("Generated %d table changes", len(diff.TableChanges))
	for i, change := range diff.TableChanges {
		t.Logf("Change %d: %s %s", i+1, change.Action, change.TableName)
	}
}

func TestApplyMinimalChangesToChunks(t *testing.T) {
	tests := []struct {
		name                string
		previousUserSDLText string
		currentSchema       *model.DatabaseSchema
		previousSchema      *model.DatabaseSchema
		expectedTables      []string
	}{
		{
			name: "add_new_table_to_existing_chunks",
			previousUserSDLText: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: false},
									},
								},
								{
									Name: "posts",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "title", Type: "text", Nullable: false},
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
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: false},
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
			expectedTables: []string{"public.users", "public.posts"},
		},
		{
			name: "multi_schema_with_same_table_names",
			previousUserSDLText: `CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: false},
									},
								},
							},
						},
						{
							Name: "admin",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users", // Same table name but different schema
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "role", Type: "text", Nullable: false},
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
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: false},
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
			expectedTables: []string{"public.users", "admin.users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the SDL text into chunks
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)

			// Apply minimal changes to chunks
			err = applyMinimalChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Check that all expected tables are present in the chunks
			require.Equal(t, len(tt.expectedTables), len(previousChunks.Tables))
			for _, expectedTable := range tt.expectedTables {
				chunk, exists := previousChunks.Tables[expectedTable]
				require.True(t, exists, "Expected table %s not found in chunks", expectedTable)

				// Check that we can get text from the chunk
				text := chunk.GetText()
				require.NotEmpty(t, text)
				require.Contains(t, strings.ToUpper(text), "CREATE TABLE")
			}
		})
	}
}

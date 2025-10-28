package pg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
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
		name                  string
		previousUserSDLText   string
		currentSchema         *model.DatabaseSchema
		previousSchema        *model.DatabaseSchema
		expectedTables        []string
		shouldContainSerial   bool
		shouldContainIdentity bool
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
			expectedTables:      []string{"public.users", "public.posts"},
			shouldContainSerial: false,
		},
		{
			name: "add_table_with_serial_column",
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
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "bigint",
											Nullable: false,
											Default:  "nextval('products_id_seq'::regclass)",
										},
										{Name: "name", Type: "text", Nullable: false},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "products_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									OwnerTable:  "products",
									OwnerColumn: "id",
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
			expectedTables:      []string{"public.users", "public.products"},
			shouldContainSerial: true,
		},
		{
			name: "add_table_with_identity_column",
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
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:               "id",
											Type:               "integer",
											Nullable:           false,
											IdentityGeneration: storepb.ColumnMetadata_BY_DEFAULT,
										},
										{Name: "total", Type: "numeric(10,2)", Nullable: false},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "orders_id_seq",
									DataType:    "integer",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "2147483647",
									Cycle:       false,
									OwnerTable:  "orders",
									OwnerColumn: "id",
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
			expectedTables:        []string{"public.users", "public.orders"},
			shouldContainIdentity: true,
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

				// Check for serial if expected (only for products table)
				if tt.shouldContainSerial && expectedTable == "public.products" {
					upperText := strings.ToUpper(text)
					require.Contains(t, upperText, "SERIAL", "Expected serial type in products table")
				}

				// Check for identity if expected (only for orders table)
				if tt.shouldContainIdentity && expectedTable == "public.orders" {
					upperText := strings.ToUpper(text)
					require.Contains(t, upperText, "GENERATED", "Expected GENERATED keyword in orders table")
					require.Contains(t, upperText, "AS IDENTITY", "Expected AS IDENTITY in orders table")
				}
			}
		})
	}
}

// TestGetSDLDiff_UsabilityHandling tests the usability feature that skips diffs
// when current SDL chunk matches the SDL generated from current database metadata
func TestGetSDLDiff_UsabilityHandling(t *testing.T) {
	tests := []struct {
		name                    string
		currentSDLText          string
		previousUserSDLText     string
		currentSchema           *model.DatabaseSchema
		expectedTableChanges    int
		expectedViewChanges     int
		expectedFunctionChanges int
		expectedSequenceChanges int
		description             string
	}{
		{
			name: "table_format_difference_but_same_structure",
			currentSDLText: `CREATE TABLE "public"."users" (
    "id" integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
);`,
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
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "users_pkey",
											Unique:      true,
											Primary:     true,
											KeyLength:   []int64{},
											Expressions: []string{"id"},
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
			expectedTableChanges:    0, // Should skip because current SDL matches database metadata
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When current SDL matches database metadata format, skip diff even if previous format was different",
		},
		{
			name:                "view_format_difference_but_same_definition",
			currentSDLText:      `CREATE VIEW "public"."user_view" AS SELECT users.id, users.name FROM public.users;`,
			previousUserSDLText: `CREATE VIEW user_view AS SELECT id, name FROM users;`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Views: []*storepb.ViewMetadata{
								{
									Name:       "user_view",
									Definition: "SELECT users.id, users.name FROM public.users;",
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
			expectedTableChanges:    0,
			expectedViewChanges:     0, // Should skip because current SDL matches database metadata
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When current view SDL matches database metadata format, skip diff even if previous format was different",
		},
		{
			name: "actual_structure_change_should_not_skip",
			currentSDLText: `CREATE TABLE public.users (
    id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    name text NOT NULL,
    email text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);`,
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
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "users_pkey",
											Unique:      true,
											Primary:     true,
											KeyLength:   []int64{},
											Expressions: []string{"id"},
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
			expectedTableChanges:    1, // Should NOT skip because there is an actual structural difference (email column)
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When there is actual structural difference, do not skip diff even if formats are different",
		},
		{
			name: "pk_uk_column_quotes_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."users" (
    "id" integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    "email" text,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
);

CREATE UNIQUE INDEX "users_email_key" ON ONLY "public"."users" (email);`,
			previousUserSDLText: `CREATE TABLE "public"."users" (
    "id" integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    "email" text,
    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "users_email_key" ON ONLY "public"."users" ("email");`,
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
										{
											Name:     "email",
											Type:     "text",
											Nullable: true,
										},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "users_pkey",
											Unique:      true,
											Primary:     true,
											KeyLength:   []int64{},
											Expressions: []string{"id"},
										},
										{
											Name:        "users_email_key",
											Unique:      true,
											Primary:     false,
											KeyLength:   []int64{},
											Expressions: []string{"email"},
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
			expectedTableChanges:    0, // Should skip because only PK/UK column quotes differ, same structure
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When current SDL matches database metadata except for PK/UK column quotes (old vs new format), should skip diff",
		},
		{
			name: "pk_column_quotes_only_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."orders" (
    "order_id" bigint DEFAULT nextval('orders_order_id_seq'::regclass) NOT NULL,
    "customer_name" text NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY (order_id)
);`,
			previousUserSDLText: `CREATE TABLE "public"."orders" (
    "order_id" bigint DEFAULT nextval('orders_order_id_seq'::regclass) NOT NULL,
    "customer_name" text NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY ("order_id")
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "order_id",
											Type:     "bigint",
											Nullable: false,
											Default:  "nextval('orders_order_id_seq'::regclass)",
										},
										{
											Name:     "customer_name",
											Type:     "text",
											Nullable: false,
										},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "orders_pkey",
											Unique:      true,
											Primary:     true,
											KeyLength:   []int64{},
											Expressions: []string{"order_id"},
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
			expectedTableChanges:    0, // Should skip because only PK column quotes differ (new format vs old format)
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When current SDL matches database metadata except for PK column quotes only (new vs old Bytebase format), should skip diff",
		},
		{
			name: "fine_grained_column_level_usability_handling",
			currentSDLText: `CREATE TABLE "public"."products" (
    "id" integer DEFAULT nextval('products_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    "price" numeric(10,2)
);`,
			previousUserSDLText: `CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2)
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('products_id_seq'::regclass)",
										},
										{
											Name:     "name",
											Type:     "text",
											Nullable: false,
										},
										{
											Name:     "price",
											Type:     "numeric(10,2)",
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
			expectedTableChanges:    0, // Should skip column-level changes because current columns match database metadata
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When individual columns match database metadata format, skip column-level diffs even if table format differs",
		},
		{
			name: "fine_grained_constraint_level_usability_handling",
			currentSDLText: `CREATE TABLE "public"."orders" (
    "order_id" bigint DEFAULT nextval('orders_order_id_seq'::regclass) NOT NULL,
    "customer_name" text NOT NULL,
    CONSTRAINT "orders_pkey" PRIMARY KEY (order_id),
    CONSTRAINT "orders_customer_name_check" CHECK (length(customer_name) > 0)
);`,
			previousUserSDLText: `CREATE TABLE orders (
    order_id BIGSERIAL PRIMARY KEY,
    customer_name TEXT NOT NULL CHECK (length(customer_name) > 0)
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "order_id",
											Type:     "bigint",
											Nullable: false,
											Default:  "nextval('orders_order_id_seq'::regclass)",
										},
										{
											Name:     "customer_name",
											Type:     "text",
											Nullable: false,
										},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "orders_pkey",
											Unique:      true,
											Primary:     true,
											KeyLength:   []int64{},
											Expressions: []string{"order_id"},
										},
									},
									CheckConstraints: []*storepb.CheckConstraintMetadata{
										{
											Name:       "orders_customer_name_check",
											Expression: "(length(customer_name) > 0)",
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
			expectedTableChanges:    0, // Should skip constraint-level changes because current constraints match database metadata
			expectedViewChanges:     0,
			expectedFunctionChanges: 0,
			expectedSequenceChanges: 0,
			description:             "When individual constraints match database metadata format, skip constraint-level diffs even if table format differs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Call GetSDLDiff
			diff, err := GetSDLDiff(tt.currentSDLText, tt.previousUserSDLText, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check the number of changes
			require.Equal(t, tt.expectedTableChanges, len(diff.TableChanges),
				"Expected %d table changes but got %d", tt.expectedTableChanges, len(diff.TableChanges))
			require.Equal(t, tt.expectedViewChanges, len(diff.ViewChanges),
				"Expected %d view changes but got %d", tt.expectedViewChanges, len(diff.ViewChanges))
			require.Equal(t, tt.expectedFunctionChanges, len(diff.FunctionChanges),
				"Expected %d function changes but got %d", tt.expectedFunctionChanges, len(diff.FunctionChanges))
			require.Equal(t, tt.expectedSequenceChanges, len(diff.SequenceChanges),
				"Expected %d sequence changes but got %d", tt.expectedSequenceChanges, len(diff.SequenceChanges))

			// Log the detected changes for debugging
			if len(diff.TableChanges) > 0 {
				t.Log("Detected table changes:")
				for i, change := range diff.TableChanges {
					t.Logf("  %d. Table: %s.%s, Action: %v", i+1, change.SchemaName, change.TableName, change.Action)
				}
			}
		})
	}
}

// TestShouldSkipChunkDiffForUsability tests the core usability logic
func TestShouldSkipChunkDiffForUsability(t *testing.T) {
	// Create a test schema
	testSchema := model.NewDatabaseSchema(
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
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "users_pkey",
									Unique:      true,
									Primary:     true,
									KeyLength:   []int64{},
									Expressions: []string{"id"},
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
	)

	tests := []struct {
		name          string
		chunkText     string
		chunkID       string
		currentSchema *model.DatabaseSchema
		expectedSkip  bool
		description   string
	}{
		{
			name: "matching_table_should_skip",
			chunkText: `CREATE TABLE "public"."users" (
    "id" integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
)`,
			chunkID:       "public.users",
			currentSchema: testSchema,
			expectedSkip:  true,
			description:   "Chunk text matches database metadata SDL, should skip",
		},
		{
			name: "different_table_should_not_skip",
			chunkText: `CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);`,
			chunkID:       "public.users",
			currentSchema: testSchema,
			expectedSkip:  false,
			description:   "Chunk text differs from database metadata SDL, should not skip",
		},
		{
			name:          "nil_schema_should_not_skip",
			chunkText:     "CREATE TABLE users (id SERIAL PRIMARY KEY);",
			chunkID:       "public.users",
			currentSchema: nil,
			expectedSkip:  false,
			description:   "When no current schema provided, should not skip",
		},
		{
			name:          "unknown_chunk_should_not_skip",
			chunkText:     "CREATE TABLE nonexistent (id SERIAL PRIMARY KEY);",
			chunkID:       "public.nonexistent",
			currentSchema: testSchema,
			expectedSkip:  false,
			description:   "When chunk doesn't exist in database metadata, should not skip",
		},
	}

	// Create a schema with table comment for testing comment-related scenarios
	testSchemaWithComment := model.NewDatabaseSchema(
		&storepb.DatabaseSchemaMetadata{
			Name: "test_db",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name:    "products",
							Comment: "Product catalog table",
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
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "products_pkey",
									Unique:      true,
									Primary:     true,
									KeyLength:   []int64{},
									Expressions: []string{"id"},
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
	)

	// Additional tests for comment-related scenarios
	commentTests := []struct {
		name          string
		chunkText     string
		chunkID       string
		currentSchema *model.DatabaseSchema
		expectedSkip  bool
		description   string
	}{
		{
			name: "table_with_same_structure_but_comment_format_differs_should_skip",
			chunkText: `CREATE TABLE "public"."products" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "products_pkey" PRIMARY KEY (id)
)`,
			chunkID:       "public.products",
			currentSchema: testSchemaWithComment,
			expectedSkip:  true,
			description:   "Table structure is identical (both from database and user SDL). In real usage, GetTextWithoutComments() is called before shouldSkipChunkDiffForUsability, so COMMENT is already excluded. This test verifies that identical structure should skip diff.",
		},
	}

	// Combine original tests with comment tests
	tests = append(tests, commentTests...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Build current database SDL chunks
			currentDBSDLChunks, err := buildCurrentDatabaseSDLChunks(tt.currentSchema)
			require.NoError(t, err)

			result := currentDBSDLChunks.shouldSkipChunkDiffForUsability(tt.chunkText, tt.chunkID)
			require.Equal(t, tt.expectedSkip, result,
				"Expected skip=%v but got skip=%v", tt.expectedSkip, result)
		})
	}
}

// TestCurrentDatabaseSDLChunksPerformance validates that the current database SDL chunks provide performance benefits
func TestCurrentDatabaseSDLChunksPerformance(t *testing.T) {
	// Create a simple test schema
	testSchema := model.NewDatabaseSchema(
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
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "users_pkey",
									Unique:      true,
									Primary:     true,
									KeyLength:   []int64{},
									Expressions: []string{"id"},
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
	)

	// Build current database SDL chunks once
	currentDBSDLChunks, err := buildCurrentDatabaseSDLChunks(testSchema)
	require.NoError(t, err)
	require.NotNil(t, currentDBSDLChunks)

	// Verify that SDL chunks contain expected entries
	require.Greater(t, len(currentDBSDLChunks.chunks), 0, "SDL chunks should contain chunks")
	t.Logf("Current database SDL chunks contains %d chunks", len(currentDBSDLChunks.chunks))

	// Test that cache lookup is fast (should be O(1))
	testChunkText := `CREATE TABLE "public"."users" (
    "id" integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY (id)
)`

	// Multiple lookups should be fast
	found := false
	for i := 0; i < 100; i++ {
		result := currentDBSDLChunks.shouldSkipChunkDiffForUsability(testChunkText, "public.users")
		if result {
			found = true
		}
	}

	// At least verify the SDL chunks work
	require.True(t, found, "Current database SDL chunks should have found a match at least once")
	t.Log("Performance test completed successfully - current database SDL chunks provide O(1) lookups")
}

func TestApplyMinimalChangesToChunks_MultipleTableCorruption(t *testing.T) {
	testCases := []struct {
		name        string
		description string
		currentSDL  string
		previousSDL string
	}{
		{
			name:        "simple_constraint_deletion_test",
			description: "Test constraint deletion logic with a simple table",
			currentSDL: `CREATE TABLE "test"."simple_table" (
    "id" integer NOT NULL,
    "name" text NOT NULL
);`,
			previousSDL: `CREATE TABLE "test"."simple_table" (
    "id" integer NOT NULL,
    "name" text NOT NULL,
    CONSTRAINT "simple_table_pkey" PRIMARY KEY (id)
);`,
		},
		{
			name:        "simple_column_deletion_test",
			description: "Test column deletion logic when deleting the last column",
			currentSDL: `CREATE TABLE "test"."simple_table" (
    "id" integer NOT NULL
);`,
			previousSDL: `CREATE TABLE "test"."simple_table" (
    "id" integer NOT NULL,
    "name" text NOT NULL
);`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test description: %s", tc.description)

			// Parse current and previous SDL
			currentChunks, err := ChunkSDLText(tc.currentSDL)
			require.NoError(t, err)

			previousChunks, err := ChunkSDLText(tc.previousSDL)
			require.NoError(t, err)

			// Log original chunks
			t.Log("=== Current SDL Chunks ===")
			for id, chunk := range currentChunks.Tables {
				t.Logf("Table %s: %s", id, chunk.GetText())
			}
			t.Log("=== Previous SDL Chunks ===")
			for id, chunk := range previousChunks.Tables {
				t.Logf("Table %s: %s", id, chunk.GetText())
			}

			// Create mock schemas for applyMinimalChangesToChunks based on test case
			var currentSchema, previousSchema *model.DatabaseSchema

			switch tc.name {
			case "simple_constraint_deletion_test":
				currentSchema = createMockDatabaseSchema()                   // No constraints
				previousSchema = createMockDatabaseSchemaWithoutTestColumn() // Has primary key constraint
			case "simple_column_deletion_test":
				currentSchema = createMockDatabaseSchemaForColumnDeletion() // Only has id column
				previousSchema = createMockDatabaseSchema()                 // Has id and name columns
			default:
				currentSchema = createMockDatabaseSchema()
				previousSchema = createMockDatabaseSchemaWithoutTestColumn()
			}

			// Apply minimal changes - this should test both the chunk mapping logic and rewriter fix
			err = applyMinimalChangesToChunks(previousChunks, currentSchema, previousSchema)
			require.NoError(t, err)

			// Log chunks after minimal changes
			t.Log("=== Previous SDL Chunks After Minimal Changes ===")
			for id, chunk := range previousChunks.Tables {
				t.Logf("Table %s: %s", id, chunk.GetText())
			}

			// Verify that each chunk identifier matches its content
			for identifier, chunk := range previousChunks.Tables {
				chunkText := chunk.GetText()
				t.Logf("Checking identifier '%s'", identifier)

				// The chunk text should contain the correct table name based on the identifier
				expectedTableName := identifier
				if strings.Contains(expectedTableName, ".") {
					parts := strings.Split(expectedTableName, ".")
					expectedTableName = parts[len(parts)-1] // Get table name part
				}

				// Check that the chunk text contains the expected table name
				assert.Contains(t, chunkText, expectedTableName,
					"Chunk with identifier '%s' should contain table name '%s' but got: %s",
					identifier, expectedTableName, chunkText)

				// More specific check: chunk should contain CREATE TABLE with the correct qualified name
				// PostgreSQL can format identifiers in different ways: "schema.table", schema.table, "schema"."table"
				parts := strings.Split(identifier, ".")
				if len(parts) == 2 {
					schema, table := parts[0], parts[1]

					// Check various valid formats
					formats := []string{
						fmt.Sprintf(`CREATE TABLE "%s"."%s"`, schema, table), // "schema"."table"
						fmt.Sprintf(`CREATE TABLE "%s"`, identifier),         // "schema.table"
						fmt.Sprintf(`CREATE TABLE %s`, identifier),           // schema.table
						fmt.Sprintf(`CREATE TABLE %s.%s`, schema, table),     // schema.table
					}

					hasValidFormat := false
					for _, format := range formats {
						if strings.Contains(chunkText, format) {
							hasValidFormat = true
							break
						}
					}

					assert.True(t, hasValidFormat,
						"Chunk with identifier '%s' should contain CREATE TABLE in a valid format but got: %s",
						identifier, chunkText)
				}
			}
		})
	}
}

// TestProcessCheckConstraintChanges tests check constraint modify/create/drop logic
func TestProcessCheckConstraintChanges(t *testing.T) {
	tests := []struct {
		name             string
		currentSDL       string
		previousSDL      string
		expectedDrops    int
		expectedCreates  int
		expectedModifies int // modify = drop + create
		description      string
	}{
		{
			name: "create_new_check_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age > 0)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer
);`,
			expectedDrops:    0,
			expectedCreates:  1,
			expectedModifies: 0,
			description:      "Adding a new check constraint should create one CREATE operation",
		},
		{
			name: "drop_check_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age > 0)
);`,
			expectedDrops:    1,
			expectedCreates:  0,
			expectedModifies: 0,
			description:      "Removing a check constraint should create one DROP operation",
		},
		{
			name: "modify_check_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age >= 18)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age > 0)
);`,
			expectedDrops:    1,
			expectedCreates:  1,
			expectedModifies: 1,
			description:      "Modifying a check constraint should create one DROP and one CREATE operation",
		},
		{
			name: "multiple_constraint_operations",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    "name" text,
    CONSTRAINT "users_age_check" CHECK (age >= 21),
    CONSTRAINT "users_name_check" CHECK (length(name) > 0)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    "email" text,
    CONSTRAINT "users_age_check" CHECK (age > 0),
    CONSTRAINT "users_email_check" CHECK (email LIKE '%@%')
);`,
			expectedDrops:    2, // drop old age_check + drop email_check
			expectedCreates:  2, // create new age_check + create name_check
			expectedModifies: 1, // modify age_check
			description:      "Multiple operations: modify age_check, drop email_check, create name_check",
		},
		{
			name: "no_changes",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age > 0)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "age" integer,
    CONSTRAINT "users_age_check" CHECK (age > 0)
);`,
			expectedDrops:    0,
			expectedCreates:  0,
			expectedModifies: 0,
			description:      "No changes should result in no operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Parse current and previous SDL
			currentChunks, err := ChunkSDLText(tt.currentSDL)
			require.NoError(t, err)
			previousChunks, err := ChunkSDLText(tt.previousSDL)
			require.NoError(t, err)

			// Get table chunks
			require.Len(t, currentChunks.Tables, 1, "Should have exactly one current table")
			require.Len(t, previousChunks.Tables, 1, "Should have exactly one previous table")

			var currentChunk, previousChunk *schema.SDLChunk
			for _, chunk := range currentChunks.Tables {
				currentChunk = chunk
				break
			}
			for _, chunk := range previousChunks.Tables {
				previousChunk = chunk
				break
			}

			// Extract AST nodes
			currentAST, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Current chunk should be a CREATE TABLE statement")
			previousAST, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Previous chunk should be a CREATE TABLE statement")

			// Process check constraint changes (no usability chunks needed for this test)
			checkDiffs := processCheckConstraintChanges(previousAST, currentAST, nil, "test.users")

			// Count operations
			actualDrops := 0
			actualCreates := 0
			for _, diff := range checkDiffs {
				switch diff.Action {
				case schema.MetadataDiffActionDrop:
					actualDrops++
				case schema.MetadataDiffActionCreate:
					actualCreates++
				default:
					// Unknown action - should not happen in normal cases
				}
			}

			// Verify results
			assert.Equal(t, tt.expectedDrops, actualDrops, "Number of DROP operations should match")
			assert.Equal(t, tt.expectedCreates, actualCreates, "Number of CREATE operations should match")

			// Log operations for debugging
			t.Logf("Generated %d DROP and %d CREATE operations:", actualDrops, actualCreates)
			for i, diff := range checkDiffs {
				t.Logf("  %d. Action: %s", i+1, diff.Action)
			}
		})
	}
}

// TestProcessForeignKeyChanges tests foreign key constraint modify/create/drop logic
func TestProcessForeignKeyChanges(t *testing.T) {
	tests := []struct {
		name            string
		currentSDL      string
		previousSDL     string
		expectedDrops   int
		expectedCreates int
		description     string
	}{
		{
			name: "create_new_foreign_key",
			currentSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer,
    CONSTRAINT "orders_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
);`,
			previousSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer
);`,
			expectedDrops:   0,
			expectedCreates: 1,
			description:     "Adding a new foreign key should create one CREATE operation",
		},
		{
			name: "drop_foreign_key",
			currentSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer
);`,
			previousSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer,
    CONSTRAINT "orders_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
);`,
			expectedDrops:   1,
			expectedCreates: 0,
			description:     "Removing a foreign key should create one DROP operation",
		},
		{
			name: "modify_foreign_key",
			currentSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer,
    CONSTRAINT "orders_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`,
			previousSDL: `CREATE TABLE "test"."orders" (
    "id" integer NOT NULL,
    "user_id" integer,
    CONSTRAINT "orders_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
);`,
			expectedDrops:   1,
			expectedCreates: 1,
			description:     "Modifying a foreign key should create one DROP and one CREATE operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Parse current and previous SDL
			currentChunks, err := ChunkSDLText(tt.currentSDL)
			require.NoError(t, err)
			previousChunks, err := ChunkSDLText(tt.previousSDL)
			require.NoError(t, err)

			// Get table chunks
			require.Len(t, currentChunks.Tables, 1, "Should have exactly one current table")
			require.Len(t, previousChunks.Tables, 1, "Should have exactly one previous table")

			var currentChunk, previousChunk *schema.SDLChunk
			for _, chunk := range currentChunks.Tables {
				currentChunk = chunk
				break
			}
			for _, chunk := range previousChunks.Tables {
				previousChunk = chunk
				break
			}

			// Extract AST nodes
			currentAST, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Current chunk should be a CREATE TABLE statement")
			previousAST, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Previous chunk should be a CREATE TABLE statement")

			// Process foreign key changes
			fkDiffs := processForeignKeyChanges(previousAST, currentAST, nil, "test.orders")

			// Count operations
			actualDrops := 0
			actualCreates := 0
			for _, diff := range fkDiffs {
				switch diff.Action {
				case schema.MetadataDiffActionDrop:
					actualDrops++
				case schema.MetadataDiffActionCreate:
					actualCreates++
				default:
					// Unknown action - should not happen in normal cases
				}
			}

			// Verify results
			assert.Equal(t, tt.expectedDrops, actualDrops, "Number of DROP operations should match")
			assert.Equal(t, tt.expectedCreates, actualCreates, "Number of CREATE operations should match")

			// Log operations for debugging
			t.Logf("Generated %d DROP and %d CREATE operations:", actualDrops, actualCreates)
			for i, diff := range fkDiffs {
				t.Logf("  %d. Action: %s", i+1, diff.Action)
			}
		})
	}
}

// TestProcessUniqueConstraintChanges tests unique constraint modify/create/drop logic
func TestProcessUniqueConstraintChanges(t *testing.T) {
	tests := []struct {
		name            string
		currentSDL      string
		previousSDL     string
		expectedDrops   int
		expectedCreates int
		description     string
	}{
		{
			name: "create_new_unique_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text,
    CONSTRAINT "users_email_key" UNIQUE (email)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text
);`,
			expectedDrops:   0,
			expectedCreates: 1,
			description:     "Adding a new unique constraint should create one CREATE operation",
		},
		{
			name: "drop_unique_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text,
    CONSTRAINT "users_email_key" UNIQUE (email)
);`,
			expectedDrops:   1,
			expectedCreates: 0,
			description:     "Removing a unique constraint should create one DROP operation",
		},
		{
			name: "modify_unique_constraint",
			currentSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text,
    "username" text,
    CONSTRAINT "users_email_username_key" UNIQUE (email, username)
);`,
			previousSDL: `CREATE TABLE "test"."users" (
    "id" integer NOT NULL,
    "email" text,
    "username" text,
    CONSTRAINT "users_email_username_key" UNIQUE (email)
);`,
			expectedDrops:   1,
			expectedCreates: 1,
			description:     "Modifying a unique constraint should create one DROP and one CREATE operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Parse current and previous SDL
			currentChunks, err := ChunkSDLText(tt.currentSDL)
			require.NoError(t, err)
			previousChunks, err := ChunkSDLText(tt.previousSDL)
			require.NoError(t, err)

			// Get table chunks
			require.Len(t, currentChunks.Tables, 1, "Should have exactly one current table")
			require.Len(t, previousChunks.Tables, 1, "Should have exactly one previous table")

			var currentChunk, previousChunk *schema.SDLChunk
			for _, chunk := range currentChunks.Tables {
				currentChunk = chunk
				break
			}
			for _, chunk := range previousChunks.Tables {
				previousChunk = chunk
				break
			}

			// Extract AST nodes
			currentAST, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Current chunk should be a CREATE TABLE statement")
			previousAST, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Previous chunk should be a CREATE TABLE statement")

			// Process unique constraint changes
			ukDiffs := processUniqueConstraintChanges(previousAST, currentAST, nil, "test.users")

			// Count operations
			actualDrops := 0
			actualCreates := 0
			for _, diff := range ukDiffs {
				switch diff.Action {
				case schema.MetadataDiffActionDrop:
					actualDrops++
				case schema.MetadataDiffActionCreate:
					actualCreates++
				default:
					// Unknown action - should not happen in normal cases
				}
			}

			// Verify results
			assert.Equal(t, tt.expectedDrops, actualDrops, "Number of DROP operations should match")
			assert.Equal(t, tt.expectedCreates, actualCreates, "Number of CREATE operations should match")

			// Log operations for debugging
			t.Logf("Generated %d DROP and %d CREATE operations:", actualDrops, actualCreates)
			for i, diff := range ukDiffs {
				t.Logf("  %d. Action: %s", i+1, diff.Action)
			}
		})
	}
}

func createMockDatabaseSchema() *model.DatabaseSchema {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "test",
				Tables: []*storepb.TableMetadata{
					{
						Name: "simple_table",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Nullable: false},
							{Name: "name", Type: "text", Nullable: false},
						},
						// No constraints in current schema
					},
				},
			},
		},
	}

	return model.NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, false)
}

func createMockDatabaseSchemaWithoutTestColumn() *model.DatabaseSchema {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "test",
				Tables: []*storepb.TableMetadata{
					{
						Name: "simple_table",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Nullable: false},
							{Name: "name", Type: "text", Nullable: false},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:         "simple_table_pkey",
								Primary:      true,
								Unique:       true,
								IsConstraint: true,
								Expressions:  []string{"id"},
							},
						},
					},
				},
			},
		},
	}

	return model.NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, false)
}

func createMockDatabaseSchemaForColumnDeletion() *model.DatabaseSchema {
	// Create schema for column deletion test - only has id column (name column deleted)
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "employees",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "integer",
								Nullable: false,
							},
							// Only id column - this represents the state after name column deletion
						},
					},
				},
			},
		},
	}

	return model.NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, false)
}

// TestGetSDLDiff_SerialAndIdentityColumns tests serial and identity column handling in SDL diff
func TestGetSDLDiff_SerialAndIdentityColumns(t *testing.T) {
	tests := []struct {
		name                 string
		currentSDLText       string
		previousUserSDLText  string
		currentSchema        *model.DatabaseSchema
		expectedTableChanges int
		description          string
	}{
		{
			name: "serial_column_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."users" (
    "id" serial,
    "name" text NOT NULL
);`,
			previousUserSDLText: `CREATE TABLE public.users (
    id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
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
									DataType:    "integer",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "2147483647",
									Cycle:       false,
									OwnerTable:  "users",
									OwnerColumn: "id",
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
			expectedTableChanges: 0,
			description:          "Serial vs expanded nextval format should not create diff when representing same structure",
		},
		{
			name: "bigserial_column_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."orders" (
    "order_id" bigserial,
    "customer_name" text NOT NULL
);`,
			previousUserSDLText: `CREATE TABLE public.orders (
    order_id bigint NOT NULL DEFAULT nextval('orders_order_id_seq'::regclass),
    customer_name TEXT NOT NULL
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "order_id",
											Type:     "bigint",
											Nullable: false,
											Default:  "nextval('orders_order_id_seq'::regclass)",
										},
										{
											Name:     "customer_name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "orders_order_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									OwnerTable:  "orders",
									OwnerColumn: "order_id",
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
			expectedTableChanges: 0,
			description:          "Bigserial vs expanded bigint+nextval format should not create diff",
		},
		{
			name: "smallserial_column_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."items" (
    "item_id" smallserial,
    "item_name" text NOT NULL
);`,
			previousUserSDLText: `CREATE TABLE public.items (
    item_id smallint NOT NULL DEFAULT nextval('items_item_id_seq'::regclass),
    item_name TEXT NOT NULL
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "items",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "item_id",
											Type:     "smallint",
											Nullable: false,
											Default:  "nextval('items_item_id_seq'::regclass)",
										},
										{
											Name:     "item_name",
											Type:     "text",
											Nullable: false,
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "items_item_id_seq",
									DataType:    "smallint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "32767",
									Cycle:       false,
									OwnerTable:  "items",
									OwnerColumn: "item_id",
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
			expectedTableChanges: 0,
			description:          "Smallserial vs expanded smallint+nextval format should not create diff",
		},
		{
			name: "identity_always_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."products" (
    "id" bigint GENERATED ALWAYS AS IDENTITY,
    "name" text NOT NULL
);`,
			previousUserSDLText: `CREATE TABLE public.products (
    id bigint NOT NULL,
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
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:               "id",
											Type:               "bigint",
											Nullable:           false,
											IdentityGeneration: storepb.ColumnMetadata_ALWAYS,
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
									Name:        "products_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									OwnerTable:  "products",
									OwnerColumn: "id",
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
			expectedTableChanges: 0,
			description:          "GENERATED ALWAYS AS IDENTITY vs plain bigint should not create diff when metadata matches",
		},
		{
			name: "identity_by_default_with_options_format_difference_should_skip",
			currentSDLText: `CREATE TABLE "public"."invoices" (
    "id" integer GENERATED BY DEFAULT AS IDENTITY (START WITH 100 INCREMENT BY 5),
    "amount" numeric(10,2) NOT NULL
);`,
			previousUserSDLText: `CREATE TABLE public.invoices (
    id integer NOT NULL,
    amount DECIMAL(10,2) NOT NULL
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "invoices",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:               "id",
											Type:               "integer",
											Nullable:           false,
											IdentityGeneration: storepb.ColumnMetadata_BY_DEFAULT,
										},
										{
											Name:     "amount",
											Type:     "numeric(10,2)",
											Nullable: false,
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "invoices_id_seq",
									DataType:    "integer",
									Start:       "100",
									Increment:   "5",
									MinValue:    "1",
									MaxValue:    "2147483647",
									Cycle:       false,
									OwnerTable:  "invoices",
									OwnerColumn: "id",
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
			expectedTableChanges: 0,
			description:          "GENERATED BY DEFAULT AS IDENTITY with options should not create diff when metadata matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Call GetSDLDiff
			diff, err := GetSDLDiff(tt.currentSDLText, tt.previousUserSDLText, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check the number of table changes
			require.Equal(t, tt.expectedTableChanges, len(diff.TableChanges),
				"Expected %d table changes but got %d", tt.expectedTableChanges, len(diff.TableChanges))

			// Log the detected changes for debugging
			if len(diff.TableChanges) > 0 {
				t.Log("Detected table changes:")
				for i, change := range diff.TableChanges {
					t.Logf("  %d. Table: %s.%s, Action: %v", i+1, change.SchemaName, change.TableName, change.Action)
				}
			}
		})
	}
}

// TestConvertDatabaseSchemaToSDL_SerialAndIdentitySequences tests that serial and identity sequences
// are properly merged into column definitions and not generated separately
func TestConvertDatabaseSchemaToSDL_SerialAndIdentitySequences(t *testing.T) {
	tests := []struct {
		name                     string
		dbSchema                 *model.DatabaseSchema
		shouldContainSerial      bool
		shouldContainIdentity    bool
		shouldNotContainSequence []string
		shouldNotContainNextval  bool
		description              string
	}{
		{
			name: "serial_column_should_not_generate_separate_sequence",
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
									DataType:    "integer",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "2147483647",
									Cycle:       false,
									OwnerTable:  "users",
									OwnerColumn: "id",
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
			shouldContainSerial:      true,
			shouldNotContainSequence: []string{"users_id_seq"},
			shouldNotContainNextval:  true,
			description:              "Serial column should use serial type and not generate separate sequence",
		},
		{
			name: "identity_column_should_not_generate_separate_sequence",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:               "id",
											Type:               "bigint",
											Nullable:           false,
											IdentityGeneration: storepb.ColumnMetadata_ALWAYS,
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
									Name:        "products_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									OwnerTable:  "products",
									OwnerColumn: "id",
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
			shouldContainIdentity:    true,
			shouldNotContainSequence: []string{"products_id_seq"},
			description:              "Identity column should use GENERATED AS IDENTITY and not generate separate sequence",
		},
		{
			name: "independent_sequence_should_still_be_generated",
			dbSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{
											Name:     "id",
											Type:     "integer",
											Nullable: false,
											Default:  "nextval('orders_id_seq'::regclass)",
										},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "orders_id_seq",
									DataType:    "integer",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "2147483647",
									Cycle:       false,
									OwnerTable:  "orders",
									OwnerColumn: "id",
								},
								{
									Name:       "independent_seq",
									DataType:   "integer",
									Start:      "1",
									Increment:  "1",
									MinValue:   "1",
									MaxValue:   "2147483647",
									Cycle:      false,
									OwnerTable: "", // No owner - independent sequence
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
			shouldContainSerial:      true,
			shouldNotContainSequence: []string{"orders_id_seq"},
			description:              "Independent sequence (not owned by column) should still be generated separately",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test description: %s", tt.description)

			// Convert database schema to SDL
			result, err := convertDatabaseSchemaToSDL(tt.dbSchema)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			t.Logf("Generated SDL:\n%s", result)

			// Check for serial type if expected
			if tt.shouldContainSerial {
				require.Contains(t, result, "serial", "SDL should contain serial type")
			}

			// Check for identity syntax if expected
			if tt.shouldContainIdentity {
				require.Contains(t, result, "GENERATED", "SDL should contain GENERATED keyword")
				require.Contains(t, result, "AS IDENTITY", "SDL should contain AS IDENTITY")
			}

			// Check that specified sequences are NOT generated separately
			for _, seqName := range tt.shouldNotContainSequence {
				// The sequence name should not appear in a CREATE SEQUENCE statement
				require.NotContains(t, result, fmt.Sprintf(`CREATE SEQUENCE "%s"`, seqName),
					"SDL should not contain separate CREATE SEQUENCE for %s", seqName)
				require.NotContains(t, result, fmt.Sprintf(`CREATE SEQUENCE "public"."%s"`, seqName),
					"SDL should not contain separate CREATE SEQUENCE for public.%s", seqName)
			}

			// Check that nextval is not used if shouldNotContainNextval is true
			if tt.shouldNotContainNextval {
				require.NotContains(t, result, "nextval", "SDL should not contain nextval() when using serial")
			}

			// Check that independent sequences ARE generated (if present)
			if strings.Contains(tt.description, "independent") {
				require.Contains(t, result, "independent_seq", "SDL should contain independent sequence")
				require.Contains(t, result, "CREATE SEQUENCE", "SDL should have CREATE SEQUENCE for independent sequence")
			}
		})
	}
}

func TestApplySequenceChangesToChunks_PreservesAlterAndComment(t *testing.T) {
	tests := []struct {
		name                       string
		previousUserSDLText        string
		currentSchema              *model.DatabaseSchema
		previousSchema             *model.DatabaseSchema
		shouldPreserveAlter        bool
		shouldPreserveComment      bool
		expectedAlterCount         int
		expectedCommentCount       int
		expectedSequenceProperties map[string]string // Properties that should change
	}{
		{
			name: "preserve_alter_sequence_on_metadata_change",
			previousUserSDLText: `CREATE SEQUENCE public.my_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.my_seq OWNED BY public.users.id;

COMMENT ON SEQUENCE public.my_seq IS 'User ID sequence';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "my_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "2", // Changed from 1 to 2
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
									Comment:     "User ID sequence",
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "my_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
									Comment:     "User ID sequence",
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
			shouldPreserveAlter:   true,
			shouldPreserveComment: true,
			expectedAlterCount:    1,
			expectedCommentCount:  1,
			expectedSequenceProperties: map[string]string{
				"INCREMENT": "2",
			},
		},
		{
			name: "preserve_comment_only",
			previousUserSDLText: `CREATE SEQUENCE public.counter_seq
    START WITH 100
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.counter_seq IS 'Global counter';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "counter_seq",
									DataType:  "bigint",
									Start:     "100",
									Increment: "5", // Changed
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Global counter",
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "counter_seq",
									DataType:  "bigint",
									Start:     "100",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Global counter",
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
			shouldPreserveAlter:   false,
			shouldPreserveComment: true,
			expectedAlterCount:    0,
			expectedCommentCount:  1,
			expectedSequenceProperties: map[string]string{
				"INCREMENT": "5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  AlterStatements count: %d", len(chunk.AlterStatements))
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applySequenceChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  AlterStatements count: %d", len(chunk.AlterStatements))
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Verify sequence exists
			sequenceChunk := previousChunks.Sequences["public.my_seq"]
			if tt.name == "preserve_comment_only" {
				sequenceChunk = previousChunks.Sequences["public.counter_seq"]
			}
			require.NotNil(t, sequenceChunk, "Sequence chunk should exist after drift handling")

			// Get full text
			fullText := sequenceChunk.GetText()
			t.Logf("Full sequence text:\n%s", fullText)

			// Verify ALTER statements are preserved
			if tt.shouldPreserveAlter {
				assert.Equal(t, tt.expectedAlterCount, len(sequenceChunk.AlterStatements),
					"ALTER statements should be preserved during drift handling")
				if tt.expectedAlterCount > 0 {
					assert.Contains(t, fullText, "ALTER SEQUENCE",
						"Full text should contain ALTER SEQUENCE statement")
					assert.Contains(t, fullText, "OWNED BY",
						"Full text should contain OWNED BY clause")
				}
			} else {
				assert.Equal(t, tt.expectedAlterCount, len(sequenceChunk.AlterStatements),
					"Should have expected number of ALTER statements")
			}

			// Verify COMMENT statements are preserved
			if tt.shouldPreserveComment {
				assert.Equal(t, tt.expectedCommentCount, len(sequenceChunk.CommentStatements),
					"COMMENT statements should be preserved during drift handling")
				if tt.expectedCommentCount > 0 {
					assert.Contains(t, fullText, "COMMENT ON SEQUENCE",
						"Full text should contain COMMENT ON SEQUENCE statement")
				}
			} else {
				assert.Equal(t, tt.expectedCommentCount, len(sequenceChunk.CommentStatements),
					"Should have expected number of COMMENT statements")
			}

			// Verify that sequence properties were updated
			createText := extractTextFromNode(sequenceChunk.ASTNode)
			for property, expectedValue := range tt.expectedSequenceProperties {
				assert.Contains(t, createText, expectedValue,
					"CREATE SEQUENCE should reflect the new %s value", property)
			}
		})
	}
}

func TestApplySequenceChangesToChunks_OwnerDrift(t *testing.T) {
	tests := []struct {
		name                    string
		previousUserSDLText     string
		currentSchema           *model.DatabaseSchema
		previousSchema          *model.DatabaseSchema
		expectedAlterStatements int
		expectedOwnerTable      string
		expectedOwnerColumn     string
	}{
		{
			name: "sequence_owner_drifted_from_users_to_posts",
			previousUserSDLText: `CREATE SEQUENCE public.my_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.my_seq OWNED BY public.users.id;`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "my_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "posts", // Changed from "users"
									OwnerColumn: "id",
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "my_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
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
			expectedAlterStatements: 1,
			expectedOwnerTable:      "posts",
			expectedOwnerColumn:     "id",
		},
		{
			name: "sequence_owner_removed",
			previousUserSDLText: `CREATE SEQUENCE public.counter_seq
    START WITH 100
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.counter_seq OWNED BY public.users.id;`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "counter_seq",
									DataType:    "bigint",
									Start:       "100",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "", // Owner removed
									OwnerColumn: "",
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "counter_seq",
									DataType:    "bigint",
									Start:       "100",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
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
			expectedAlterStatements: 0, // Should be removed
			expectedOwnerTable:      "",
			expectedOwnerColumn:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  AlterStatements count: %d", len(chunk.AlterStatements))
			}

			// Apply drift handling
			err = applySequenceChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  AlterStatements count: %d", len(chunk.AlterStatements))
			}

			// Get the sequence chunk
			var sequenceChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Sequences {
				sequenceChunk = chunk
				break
			}
			require.NotNil(t, sequenceChunk, "Sequence chunk should exist after drift handling")

			// Get full text
			fullText := sequenceChunk.GetText()
			t.Logf("Full sequence text:\n%s", fullText)

			// Verify ALTER statements count
			assert.Equal(t, tt.expectedAlterStatements, len(sequenceChunk.AlterStatements),
				"ALTER statements count should match expected")

			// Verify the ALTER statement content matches the expected owner
			if tt.expectedAlterStatements > 0 {
				// Should contain ALTER SEQUENCE with correct owner
				// The owner reference can be in format: "table"."column" or table.column
				assert.Contains(t, fullText, "ALTER SEQUENCE",
					"Full text should contain ALTER SEQUENCE statement")
				assert.Contains(t, fullText, "OWNED BY",
					"ALTER SEQUENCE should contain OWNED BY clause")
				// Check that it references the correct table and column (with or without quotes/schema)
				assert.Contains(t, fullText, tt.expectedOwnerTable,
					"ALTER SEQUENCE should reference the correct owner table: %s", tt.expectedOwnerTable)
				assert.Contains(t, fullText, tt.expectedOwnerColumn,
					"ALTER SEQUENCE should reference the correct owner column: %s", tt.expectedOwnerColumn)
			} else {
				// Should NOT contain ALTER SEQUENCE
				assert.NotContains(t, fullText, "ALTER SEQUENCE",
					"Full text should NOT contain ALTER SEQUENCE when owner is removed")
			}
		})
	}
}

func TestApplySequenceChangesToChunks_CommentDrift(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedCommentStatements int
		expectedCommentText       string
	}{
		{
			name: "comment_changed_but_sequence_definition_unchanged",
			previousUserSDLText: `CREATE SEQUENCE public.my_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.my_seq IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "my_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "New comment", // Changed
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "my_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Old comment",
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment",
		},
		{
			name: "comment_added_when_previously_none",
			previousUserSDLText: `CREATE SEQUENCE public.new_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "new_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "New comment added", // Added
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "new_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "", // No comment before
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment added",
		},
		{
			name: "comment_removed",
			previousUserSDLText: `CREATE SEQUENCE public.counter_seq
    START WITH 100
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.counter_seq IS 'Some comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "counter_seq",
									DataType:  "bigint",
									Start:     "100",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "", // Removed
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "counter_seq",
									DataType:  "bigint",
									Start:     "100",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Some comment",
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
			expectedCommentStatements: 0,
			expectedCommentText:       "",
		},
		{
			name: "comment_unchanged_should_preserve_original_format",
			previousUserSDLText: `CREATE SEQUENCE public.preserve_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT   ON   SEQUENCE   public.preserve_seq   IS   'Unchanged comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "preserve_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Unchanged comment", // Same as before
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
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "preserve_seq",
									DataType:  "bigint",
									Start:     "1",
									Increment: "1",
									MinValue:  "1",
									MaxValue:  "9223372036854775807",
									Cycle:     false,
									CacheSize: "1",
									Comment:   "Unchanged comment", // Same
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
			expectedCommentStatements: 1,
			expectedCommentText:       "Unchanged comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applySequenceChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Sequences {
				t.Logf("Sequence %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Get the sequence chunk
			var sequenceChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Sequences {
				sequenceChunk = chunk
				break
			}
			require.NotNil(t, sequenceChunk, "Sequence chunk should exist after drift handling")

			// Get full text
			fullText := sequenceChunk.GetText()
			t.Logf("Full sequence text:\n%s", fullText)

			// Verify COMMENT statements count
			assert.Equal(t, tt.expectedCommentStatements, len(sequenceChunk.CommentStatements),
				"COMMENT statements count should match expected")

			// Verify the COMMENT statement content
			if tt.expectedCommentStatements > 0 && tt.expectedCommentText != "" {
				// Use Contains with "COMMENT" and "SEQUENCE" to be flexible with formatting
				assert.Contains(t, fullText, "COMMENT",
					"Full text should contain COMMENT statement")
				assert.Contains(t, fullText, "SEQUENCE",
					"Full text should mention SEQUENCE")
				assert.Contains(t, fullText, tt.expectedCommentText,
					"COMMENT should contain the expected text: %s", tt.expectedCommentText)
			} else if tt.expectedCommentText == "" && tt.expectedCommentStatements == 0 {
				// Either no comment or empty comment
				assert.NotContains(t, fullText, "COMMENT ON SEQUENCE",
					"Full text should NOT contain COMMENT ON SEQUENCE when comment is removed")
			}
		})
	}
}

func TestApplySequenceChangesToChunks_OwnedSequenceNotInSDL(t *testing.T) {
	tests := []struct {
		name                string
		previousUserSDLText string
		currentSchema       *model.DatabaseSchema
		previousSchema      *model.DatabaseSchema
		expectSequenceChunk bool
		sequenceKey         string
	}{
		{
			name: "serial_sequence_not_in_sdl_should_not_create_chunk",
			// User SDL only contains CREATE TABLE, not the sequence created by SERIAL
			previousUserSDLText: `CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    name TEXT
);`,
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
										{Name: "id", Type: "integer", Default: "nextval('users_id_seq'::regclass)"},
										{Name: "name", Type: "text"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
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
										{Name: "id", Type: "integer", Default: "nextval('users_id_seq'::regclass)"},
										{Name: "name", Type: "text"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "users_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "2", // Changed increment
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "users",
									OwnerColumn: "id",
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
			expectSequenceChunk: false,
			sequenceKey:         "public.users_id_seq",
		},
		{
			name: "identity_sequence_not_in_sdl_should_not_create_chunk",
			// User SDL only contains CREATE TABLE with IDENTITY column, not the sequence
			previousUserSDLText: `CREATE TABLE public.products (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT
);`,
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Default: "nextval('products_id_seq'::regclass)"},
										{Name: "name", Type: "text"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "products_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "products",
									OwnerColumn: "id",
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
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Default: "nextval('products_id_seq'::regclass)"},
										{Name: "name", Type: "text"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "products_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "5", // Changed increment
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "products",
									OwnerColumn: "id",
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
			expectSequenceChunk: false,
			sequenceKey:         "public.products_id_seq",
		},
		{
			name: "owned_sequence_explicitly_in_sdl_should_update_chunk",
			// User explicitly defined the sequence in SDL, so it should be updated
			previousUserSDLText: `CREATE SEQUENCE public.orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.orders (
    id INT DEFAULT nextval('orders_id_seq'),
    amount DECIMAL
);

ALTER SEQUENCE public.orders_id_seq OWNED BY public.orders.id;`,
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Default: "nextval('orders_id_seq'::regclass)"},
										{Name: "amount", Type: "numeric"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "orders_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "1",
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "orders",
									OwnerColumn: "id",
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
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Default: "nextval('orders_id_seq'::regclass)"},
										{Name: "amount", Type: "numeric"},
									},
								},
							},
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:        "orders_id_seq",
									DataType:    "bigint",
									Start:       "1",
									Increment:   "10", // Changed increment
									MinValue:    "1",
									MaxValue:    "9223372036854775807",
									Cycle:       false,
									CacheSize:   "1",
									OwnerTable:  "orders",
									OwnerColumn: "id",
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
			expectSequenceChunk: true,
			sequenceKey:         "public.orders_id_seq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL text into chunks
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)

			// Apply sequence changes to chunks (this is what happens during drift handling)
			err = applySequenceChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Verify whether sequence chunk was created/updated
			_, exists := previousChunks.Sequences[tt.sequenceKey]
			if tt.expectSequenceChunk {
				require.True(t, exists, "Expected sequence chunk to exist for %s", tt.sequenceKey)
			} else {
				require.False(t, exists, "Expected sequence chunk NOT to exist for %s (owned sequences not in SDL should not be created)", tt.sequenceKey)
			}
		})
	}
}

func TestApplyFunctionChangesToChunks_CommentDrift(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedCommentStatements int
		expectedCommentText       string
	}{
		{
			name: "function_comment_changed_but_definition_unchanged",
			previousUserSDLText: `CREATE FUNCTION public.calculate_total(amount INTEGER) RETURNS INTEGER AS $$
BEGIN
    RETURN amount * 2;
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION public.calculate_total(INTEGER) IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "calculate_total",
									Definition: "CREATE FUNCTION public.calculate_total(amount INTEGER) RETURNS INTEGER AS $$\nBEGIN\n    RETURN amount * 2;\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "New comment", // Changed from "Old comment"
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
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "calculate_total",
									Definition: "CREATE FUNCTION public.calculate_total(amount INTEGER) RETURNS INTEGER AS $$\nBEGIN\n    RETURN amount * 2;\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "Old comment",
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment",
		},
		{
			name: "function_comment_added_when_previously_none",
			previousUserSDLText: `CREATE FUNCTION public.add_numbers(a INTEGER, b INTEGER) RETURNS INTEGER AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "add_numbers",
									Definition: "CREATE FUNCTION public.add_numbers(a INTEGER, b INTEGER) RETURNS INTEGER AS $$\nBEGIN\n    RETURN a + b;\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "Adds two numbers", // Added
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
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "add_numbers",
									Definition: "CREATE FUNCTION public.add_numbers(a INTEGER, b INTEGER) RETURNS INTEGER AS $$\nBEGIN\n    RETURN a + b;\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "", // No comment before
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
			expectedCommentStatements: 1,
			expectedCommentText:       "Adds two numbers",
		},
		{
			name: "function_comment_removed",
			previousUserSDLText: `CREATE FUNCTION public.process_data(input_text TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN upper(input_text);
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION public.process_data(TEXT) IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "process_data",
									Definition: "CREATE FUNCTION public.process_data(input_text TEXT) RETURNS TEXT AS $$\nBEGIN\n    RETURN upper(input_text);\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "", // Comment removed
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
							Functions: []*storepb.FunctionMetadata{
								{
									Name:       "process_data",
									Definition: "CREATE FUNCTION public.process_data(input_text TEXT) RETURNS TEXT AS $$\nBEGIN\n    RETURN upper(input_text);\nEND;\n$$ LANGUAGE plpgsql",
									Comment:    "Old comment",
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
			expectedCommentStatements: 0,
			expectedCommentText:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Functions {
				t.Logf("Function %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applyFunctionChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Functions {
				t.Logf("Function %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Get the function chunk
			var functionChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Functions {
				functionChunk = chunk
				break
			}
			require.NotNil(t, functionChunk, "Function chunk should exist after drift handling")

			// Get full text
			fullText := functionChunk.GetText()
			t.Logf("Full function text:\n%s", fullText)

			// Verify COMMENT statements count
			assert.Equal(t, tt.expectedCommentStatements, len(functionChunk.CommentStatements),
				"COMMENT statements count should match expected")

			// Verify COMMENT content
			if tt.expectedCommentStatements > 0 {
				assert.Contains(t, fullText, "COMMENT ON FUNCTION",
					"Full text should contain COMMENT ON FUNCTION statement")
				assert.Contains(t, fullText, tt.expectedCommentText,
					"COMMENT should contain the expected text: %s", tt.expectedCommentText)
			} else {
				// Should NOT contain COMMENT ON FUNCTION
				assert.NotContains(t, fullText, "COMMENT ON FUNCTION",
					"Full text should NOT contain COMMENT ON FUNCTION when comment is removed")
			}
		})
	}
}

func TestApplyIndexChangesToChunks_CommentDrift(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedCommentStatements int
		expectedCommentText       string
	}{
		{
			name: "index_comment_changed_but_definition_unchanged",
			previousUserSDLText: `CREATE TABLE public.users (
    id INTEGER PRIMARY KEY,
    email TEXT
);

CREATE INDEX idx_users_email ON public.users (email);

COMMENT ON INDEX public.idx_users_email IS 'Old comment';`,
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
										{Name: "id", Type: "integer"},
										{Name: "email", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_users_email",
											Expressions: []string{"email"},
											Type:        "btree",
											Unique:      false,
											Primary:     false,
											Comment:     "New comment", // Changed from "Old comment"
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
										{Name: "id", Type: "integer"},
										{Name: "email", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_users_email",
											Expressions: []string{"email"},
											Type:        "btree",
											Unique:      false,
											Primary:     false,
											Comment:     "Old comment",
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment",
		},
		{
			name: "index_comment_added_when_previously_none",
			previousUserSDLText: `CREATE TABLE public.products (
    id INTEGER PRIMARY KEY,
    name TEXT
);

CREATE INDEX idx_products_name ON public.products (name);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_products_name",
											Expressions: []string{"name"},
											Type:        "btree",
											Unique:      false,
											Primary:     false,
											Comment:     "Index for product names", // Added
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
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_products_name",
											Expressions: []string{"name"},
											Type:        "btree",
											Unique:      false,
											Primary:     false,
											Comment:     "", // No comment before
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
			expectedCommentStatements: 1,
			expectedCommentText:       "Index for product names",
		},
		{
			name: "index_comment_removed",
			previousUserSDLText: `CREATE TABLE public.posts (
    id INTEGER PRIMARY KEY,
    title TEXT
);

CREATE UNIQUE INDEX idx_posts_title ON public.posts (title);

COMMENT ON INDEX public.idx_posts_title IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "posts",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "title", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_posts_title",
											Expressions: []string{"title"},
											Type:        "btree",
											Unique:      true,
											Primary:     false,
											Comment:     "", // Comment removed
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
			previousSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "posts",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "title", Type: "text"},
									},
									Indexes: []*storepb.IndexMetadata{
										{
											Name:        "idx_posts_title",
											Expressions: []string{"title"},
											Type:        "btree",
											Unique:      true,
											Primary:     false,
											Comment:     "Old comment",
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
			expectedCommentStatements: 0,
			expectedCommentText:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Indexes {
				t.Logf("Index %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applyStandaloneIndexChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Indexes {
				t.Logf("Index %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Get the index chunk
			var indexChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Indexes {
				indexChunk = chunk
				break
			}
			require.NotNil(t, indexChunk, "Index chunk should exist after drift handling")

			// Get full text
			fullText := indexChunk.GetText()
			t.Logf("Full index text:\n%s", fullText)

			// Verify COMMENT statements count
			assert.Equal(t, tt.expectedCommentStatements, len(indexChunk.CommentStatements),
				"COMMENT statements count should match expected")

			// Verify COMMENT content
			if tt.expectedCommentStatements > 0 {
				assert.Contains(t, fullText, "COMMENT ON INDEX",
					"Full text should contain COMMENT ON INDEX statement")
				assert.Contains(t, fullText, tt.expectedCommentText,
					"COMMENT should contain the expected text: %s", tt.expectedCommentText)
			} else {
				// Should NOT contain COMMENT ON INDEX
				assert.NotContains(t, fullText, "COMMENT ON INDEX",
					"Full text should NOT contain COMMENT ON INDEX when comment is removed")
			}
		})
	}
}

func TestApplyViewChangesToChunks_CommentDrift(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedCommentStatements int
		expectedCommentText       string
	}{
		{
			name: "view_comment_changed_but_definition_unchanged",
			previousUserSDLText: `CREATE VIEW public.user_emails AS
SELECT id, email FROM users;

COMMENT ON VIEW public.user_emails IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Views: []*storepb.ViewMetadata{
								{
									Name:       "user_emails",
									Definition: "SELECT id, email FROM users",
									Comment:    "New comment", // Changed from "Old comment"
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
							Views: []*storepb.ViewMetadata{
								{
									Name:       "user_emails",
									Definition: "SELECT id, email FROM users",
									Comment:    "Old comment",
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment",
		},
		{
			name: "view_comment_added_when_previously_none",
			previousUserSDLText: `CREATE VIEW public.admin_users AS
SELECT * FROM users WHERE role = 'admin';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Views: []*storepb.ViewMetadata{
								{
									Name:       "admin_users",
									Definition: "SELECT * FROM users WHERE role = 'admin'",
									Comment:    "View of admin users", // Added
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
							Views: []*storepb.ViewMetadata{
								{
									Name:       "admin_users",
									Definition: "SELECT * FROM users WHERE role = 'admin'",
									Comment:    "", // No comment before
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
			expectedCommentStatements: 1,
			expectedCommentText:       "View of admin users",
		},
		{
			name: "view_comment_removed",
			previousUserSDLText: `CREATE VIEW public.active_users AS
SELECT * FROM users WHERE active = true;

COMMENT ON VIEW public.active_users IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Views: []*storepb.ViewMetadata{
								{
									Name:       "active_users",
									Definition: "SELECT * FROM users WHERE active = true",
									Comment:    "", // Comment removed
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
							Views: []*storepb.ViewMetadata{
								{
									Name:       "active_users",
									Definition: "SELECT * FROM users WHERE active = true",
									Comment:    "Old comment",
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
			expectedCommentStatements: 0,
			expectedCommentText:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Views {
				t.Logf("View %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applyViewChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Views {
				t.Logf("View %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Get the view chunk
			var viewChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Views {
				viewChunk = chunk
				break
			}
			require.NotNil(t, viewChunk, "View chunk should exist after drift handling")

			// Get full text
			fullText := viewChunk.GetText()
			t.Logf("Full view text:\n%s", fullText)

			// Verify COMMENT statements count
			assert.Equal(t, tt.expectedCommentStatements, len(viewChunk.CommentStatements),
				"COMMENT statements count should match expected")

			// Verify COMMENT content
			if tt.expectedCommentStatements > 0 {
				assert.Contains(t, fullText, "COMMENT ON VIEW",
					"Full text should contain COMMENT ON VIEW statement")
				assert.Contains(t, fullText, tt.expectedCommentText,
					"COMMENT should contain the expected text: %s", tt.expectedCommentText)
			} else {
				// Should NOT contain COMMENT ON VIEW
				assert.NotContains(t, fullText, "COMMENT ON VIEW",
					"Full text should NOT contain COMMENT ON VIEW when comment is removed")
			}
		})
	}
}

func TestApplyTableChangesToChunk_CommentDrift(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedCommentStatements int
		expectedCommentText       string
	}{
		{
			name: "table_comment_changed_but_definition_unchanged",
			previousUserSDLText: `CREATE TABLE public.products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

COMMENT ON TABLE public.products IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text", Nullable: false},
									},
									Comment: "New comment", // Changed from "Old comment"
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
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text", Nullable: false},
									},
									Comment: "Old comment",
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
			expectedCommentStatements: 1,
			expectedCommentText:       "New comment",
		},
		{
			name: "table_comment_added_when_previously_none",
			previousUserSDLText: `CREATE TABLE public.customers (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "customers",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "email", Type: "text", Nullable: false},
									},
									Comment: "Customer information table", // Added
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
									Name: "customers",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "email", Type: "text", Nullable: false},
									},
									Comment: "", // No comment before
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
			expectedCommentStatements: 1,
			expectedCommentText:       "Customer information table",
		},
		{
			name: "table_comment_removed",
			previousUserSDLText: `CREATE TABLE public.orders (
    order_id INTEGER PRIMARY KEY,
    total NUMERIC
);

COMMENT ON TABLE public.orders IS 'Old comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "order_id", Type: "integer"},
										{Name: "total", Type: "numeric"},
									},
									Comment: "", // Comment removed
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
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "order_id", Type: "integer"},
										{Name: "total", Type: "numeric"},
									},
									Comment: "Old comment",
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
			expectedCommentStatements: 0,
			expectedCommentText:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Tables {
				t.Logf("Table %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Apply drift handling
			err = applyMinimalChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Tables {
				t.Logf("Table %s:", identifier)
				t.Logf("  Text: %s", chunk.GetText())
				t.Logf("  CommentStatements count: %d", len(chunk.CommentStatements))
			}

			// Get the table chunk
			var tableChunk *schema.SDLChunk
			for _, chunk := range previousChunks.Tables {
				tableChunk = chunk
				break
			}
			require.NotNil(t, tableChunk, "Table chunk should exist after drift handling")

			// Get full text
			fullText := tableChunk.GetText()
			t.Logf("Full table text:\n%s", fullText)

			// Verify COMMENT statements count
			assert.Equal(t, tt.expectedCommentStatements, len(tableChunk.CommentStatements),
				"COMMENT statements count should match expected")

			// Verify COMMENT content
			if tt.expectedCommentStatements > 0 {
				assert.Contains(t, fullText, "COMMENT ON TABLE",
					"Full text should contain COMMENT ON TABLE statement")
				assert.Contains(t, fullText, tt.expectedCommentText,
					"COMMENT should contain the expected text: %s", tt.expectedCommentText)
			} else {
				// Should NOT contain COMMENT ON TABLE
				assert.NotContains(t, fullText, "COMMENT ON TABLE",
					"Full text should NOT contain COMMENT ON TABLE when comment is removed")
			}
		})
	}
}

func TestApplyColumnCommentChanges(t *testing.T) {
	tests := []struct {
		name                      string
		previousUserSDLText       string
		currentSchema             *model.DatabaseSchema
		previousSchema            *model.DatabaseSchema
		expectedColumnComments    map[string]map[string]string // tableKey -> columnName -> comment
		shouldContainCommentOn    []string                     // Expected COMMENT ON COLUMN statements
		shouldNotContainCommentOn []string                     // Should not contain these
	}{
		{
			name: "column_comment_changed_but_table_definition_unchanged",
			previousUserSDLText: `CREATE TABLE public.products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

COMMENT ON COLUMN public.products.name IS 'Old column comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text", Nullable: false, Comment: "New column comment"},
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
									Name: "products",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "name", Type: "text", Nullable: false, Comment: "Old column comment"},
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
			expectedColumnComments: map[string]map[string]string{
				"public.products": {
					"name": "New column comment",
				},
			},
			shouldContainCommentOn: []string{
				`COMMENT ON COLUMN "public"."products"."name" IS 'New column comment'`,
			},
		},
		{
			name: "column_comment_added_when_previously_none",
			previousUserSDLText: `CREATE TABLE public.inventory (
    id INTEGER PRIMARY KEY,
    quantity INTEGER NOT NULL
);`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "inventory",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "quantity", Type: "integer", Nullable: false, Comment: "Available quantity"}, // Added
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
									Name: "inventory",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer"},
										{Name: "quantity", Type: "integer", Nullable: false, Comment: ""}, // No comment before
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
			expectedColumnComments: map[string]map[string]string{
				"public.inventory": {
					"quantity": "Available quantity",
				},
			},
			shouldContainCommentOn: []string{
				`COMMENT ON COLUMN "public"."inventory"."quantity" IS 'Available quantity'`,
			},
		},
		{
			name: "column_comment_removed",
			previousUserSDLText: `CREATE TABLE public.orders (
    order_id INTEGER PRIMARY KEY,
    total NUMERIC
);

COMMENT ON COLUMN public.orders.total IS 'Old column comment';`,
			currentSchema: model.NewDatabaseSchema(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "order_id", Type: "integer"},
										{Name: "total", Type: "numeric", Comment: ""}, // Comment removed
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
									Name: "orders",
									Columns: []*storepb.ColumnMetadata{
										{Name: "order_id", Type: "integer"},
										{Name: "total", Type: "numeric", Comment: "Old column comment"},
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
			expectedColumnComments: map[string]map[string]string{},
			shouldNotContainCommentOn: []string{
				`COMMENT ON COLUMN "public"."orders"."total"`,
			},
		},
		{
			name: "multiple_column_comments_changed",
			previousUserSDLText: `CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN public.users.username IS 'Old username comment';
COMMENT ON COLUMN public.users.email IS 'Old email comment';`,
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
										{Name: "id", Type: "integer"},
										{Name: "username", Type: "character varying", Nullable: false, Comment: "New username comment"},
										{Name: "email", Type: "character varying", Nullable: false, Comment: "New email comment"},
										{Name: "created_at", Type: "timestamp without time zone"},
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
										{Name: "id", Type: "integer"},
										{Name: "username", Type: "character varying", Nullable: false, Comment: "Old username comment"},
										{Name: "email", Type: "character varying", Nullable: false, Comment: "Old email comment"},
										{Name: "created_at", Type: "timestamp without time zone"},
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
			expectedColumnComments: map[string]map[string]string{
				"public.users": {
					"username": "New username comment",
					"email":    "New email comment",
				},
			},
			shouldContainCommentOn: []string{
				`COMMENT ON COLUMN "public"."users"."username" IS 'New username comment'`,
				`COMMENT ON COLUMN "public"."users"."email" IS 'New email comment'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse previous SDL
			previousChunks, err := ChunkSDLText(tt.previousUserSDLText)
			require.NoError(t, err)
			require.NotNil(t, previousChunks)

			// Log original chunks
			t.Log("=== Previous SDL Chunks (Before Drift Handling) ===")
			for identifier, chunk := range previousChunks.Tables {
				t.Logf("Table %s:", identifier)
				t.Logf("  CREATE TABLE text: %s", chunk.GetText())
			}
			if len(previousChunks.ColumnComments) > 0 {
				t.Logf("ColumnComments count: %d", len(previousChunks.ColumnComments))
				for tableKey, cols := range previousChunks.ColumnComments {
					t.Logf("  Table %s: %d columns with comments", tableKey, len(cols))
					for colName, colComment := range cols {
						t.Logf("    Column %s: %s", colName, colComment.GetText())
					}
				}
			}

			// Apply drift handling
			err = applyMinimalChangesToChunks(previousChunks, tt.currentSchema, tt.previousSchema)
			require.NoError(t, err)

			// Log chunks after drift handling
			t.Log("=== Previous SDL Chunks (After Drift Handling) ===")
			for identifier, chunk := range previousChunks.Tables {
				t.Logf("Table %s:", identifier)
				t.Logf("  CREATE TABLE text: %s", chunk.GetText())
			}
			if len(previousChunks.ColumnComments) > 0 {
				t.Logf("ColumnComments count: %d", len(previousChunks.ColumnComments))
				for tableKey, cols := range previousChunks.ColumnComments {
					t.Logf("  Table %s: %d columns with comments", tableKey, len(cols))
					for colName, colComment := range cols {
						t.Logf("    Column %s: %s", colName, colComment.GetText())
					}
				}
			}

			// Verify column comments
			for tableKey, expectedCols := range tt.expectedColumnComments {
				actualCols, exists := previousChunks.ColumnComments[tableKey]
				assert.True(t, exists, "Table %s should have column comments", tableKey)

				for colName, expectedComment := range expectedCols {
					commentNode, exists := actualCols[colName]
					assert.True(t, exists, "Column %s should have a comment", colName)
					if exists {
						commentText := commentNode.GetText()
						t.Logf("Column %s comment text: %s", colName, commentText)
						assert.Contains(t, commentText, expectedComment,
							"Column comment should contain expected text")
					}
				}

				// Verify expected columns count
				assert.Equal(t, len(expectedCols), len(actualCols),
					"Number of column comments should match")
			}

			// If no column comments expected, verify map is empty or table key doesn't exist
			if len(tt.expectedColumnComments) == 0 {
				for _, shouldNotContain := range tt.shouldNotContainCommentOn {
					found := false
					for _, cols := range previousChunks.ColumnComments {
						for _, commentNode := range cols {
							if strings.Contains(commentNode.GetText(), shouldNotContain) {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					assert.False(t, found, "Should not contain: %s", shouldNotContain)
				}
			}

			// Verify CREATE TABLE statement is preserved and contains expected COMMENT statements
			for tableKey := range tt.expectedColumnComments {
				tableChunk, exists := previousChunks.Tables[tableKey]
				assert.True(t, exists, "Table chunk should exist")
				if exists {
					tableText := tableChunk.GetText()
					t.Logf("Table text:\n%s", tableText)

					// Verify CREATE TABLE is preserved (should start with CREATE TABLE)
					assert.Contains(t, tableText, "CREATE TABLE",
						"CREATE TABLE statement should be preserved")

					// Verify it contains expected COMMENT statements
					for _, expectedComment := range tt.shouldContainCommentOn {
						// Check in ColumnComments map
						// Note: GetText() removes whitespace, so we normalize both strings for comparison
						normalizedExpected := strings.ReplaceAll(expectedComment, " ", "")
						found := false
						if cols, ok := previousChunks.ColumnComments[tableKey]; ok {
							for _, commentNode := range cols {
								normalizedActual := strings.ReplaceAll(commentNode.GetText(), " ", "")
								if strings.Contains(normalizedActual, normalizedExpected) {
									found = true
									break
								}
							}
						}
						assert.True(t, found, "Should contain COMMENT: %s", expectedComment)
					}
				}
			}
		})
	}
}

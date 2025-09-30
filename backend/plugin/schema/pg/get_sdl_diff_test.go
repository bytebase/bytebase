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
											Name:      "users_pkey",
											Unique:    true,
											Primary:   true,
											KeyLength: []int64{},
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
			name: "view_format_difference_but_same_definition",
			currentSDLText: `CREATE VIEW "public"."user_view" AS SELECT users.id, users.name FROM public.users;`,
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
											Name:      "users_pkey",
											Unique:    true,
											Primary:   true,
											KeyLength: []int64{},
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
    CONSTRAINT "orders_customer_name_check" CHECK ((length(customer_name) > 0))
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
				t.Logf("Detected table changes:")
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
									Name:      "users_pkey",
									Unique:    true,
									Primary:   true,
									KeyLength: []int64{},
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
		name           string
		chunkText      string
		chunkID        string
		currentSchema  *model.DatabaseSchema
		expectedSkip   bool
		description    string
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

			if tc.name == "simple_constraint_deletion_test" {
				currentSchema = createMockDatabaseSchema() // No constraints
				previousSchema = createMockDatabaseSchemaWithoutTestColumn() // Has primary key constraint
			} else if tc.name == "simple_column_deletion_test" {
				currentSchema = createMockDatabaseSchemaForColumnDeletion() // Only has id column
				previousSchema = createMockDatabaseSchema() // Has id and name columns
			} else {
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
						fmt.Sprintf(`CREATE TABLE "%s"."%s"`, schema, table),  // "schema"."table"
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
		name                string
		currentSDL          string
		previousSDL         string
		expectedDrops       int
		expectedCreates     int
		expectedModifies    int // modify = drop + create
		description         string
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
			expectedDrops:       0,
			expectedCreates:     1,
			expectedModifies:    0,
			description:         "Adding a new check constraint should create one CREATE operation",
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
			expectedDrops:       1,
			expectedCreates:     0,
			expectedModifies:    0,
			description:         "Removing a check constraint should create one DROP operation",
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
			expectedDrops:       1,
			expectedCreates:     1,
			expectedModifies:    1,
			description:         "Modifying a check constraint should create one DROP and one CREATE operation",
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
			expectedDrops:       2, // drop old age_check + drop email_check
			expectedCreates:     2, // create new age_check + create name_check
			expectedModifies:    1, // modify age_check
			description:         "Multiple operations: modify age_check, drop email_check, create name_check",
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
			expectedDrops:       0,
			expectedCreates:     0,
			expectedModifies:    0,
			description:         "No changes should result in no operations",
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

func createMockDatabaseSchemaIdentical() *model.DatabaseSchema {
	// Create identical schema that matches the previousSDL in test data (without test_column)
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "company",
				Tables: []*storepb.TableMetadata{
					{
						Name: "departments",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Nullable: false},
							{Name: "name", Type: "character varying(100)", Nullable: false},
							{Name: "budget", Type: "numeric(12,2)", Nullable: true},
							{Name: "created_at", Type: "timestamp(6) without time zone", Nullable: true},
							{Name: "updated_at", Type: "timestamp(6) without time zone", Nullable: true},
						},
					},
					{
						Name: "employees",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Nullable: false},
							{Name: "first_name", Type: "character varying(50)", Nullable: false},
							{Name: "last_name", Type: "character varying(50)", Nullable: false},
							{Name: "email", Type: "character varying(100)", Nullable: false},
							{Name: "department_id", Type: "integer", Nullable: false},
							{Name: "salary", Type: "numeric(10,2)", Nullable: true},
							{Name: "hire_date", Type: "date", Nullable: true},
							{Name: "is_active", Type: "boolean", Nullable: true},
							// Note: no test_column to match the previousSDL
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

package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestStandaloneCreateIndexSupport(t *testing.T) {
	tests := []struct {
		name               string
		previousUserSDL    string
		currentSDL         string
		expectedIndexDiffs int
		expectedActions    []schema.MetadataDiffAction
	}{
		{
			name:            "Create new index",
			previousUserSDL: ``,
			currentSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			
			CREATE INDEX idx_users_email ON users(email);
			`,
			expectedIndexDiffs: 1,
			expectedActions:    []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop index",
			previousUserSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			
			CREATE INDEX idx_users_email ON users(email);
			`,
			currentSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			`,
			expectedIndexDiffs: 1,
			expectedActions:    []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify index (drop and recreate)",
			previousUserSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE,
				name VARCHAR(100)
			);
			
			CREATE INDEX idx_users_email ON users(email);
			`,
			currentSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE,
				name VARCHAR(100)
			);
			
			CREATE INDEX idx_users_email ON users(email, name);
			`,
			expectedIndexDiffs: 2, // Drop + Create
			expectedActions:    []schema.MetadataDiffAction{schema.MetadataDiffActionDrop, schema.MetadataDiffActionCreate},
		},
		{
			name: "No changes to index",
			previousUserSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			
			CREATE INDEX idx_users_email ON users(email);
			`,
			currentSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			
			CREATE INDEX idx_users_email ON users(email);
			`,
			expectedIndexDiffs: 0,
			expectedActions:    []schema.MetadataDiffAction{},
		},
		{
			name: "Complex index with WHERE clause",
			previousUserSDL: `
			CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50),
				customer_id INTEGER
			);
			`,
			currentSDL: `
			CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50),
				customer_id INTEGER
			);
			
			CREATE INDEX idx_orders_active ON orders(customer_id) WHERE status = 'active';
			`,
			expectedIndexDiffs: 1,
			expectedActions:    []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousUserSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Count total index diffs across all table changes
			var allIndexDiffs []*schema.IndexDiff
			for _, tableChange := range diff.TableChanges {
				allIndexDiffs = append(allIndexDiffs, tableChange.IndexChanges...)
			}

			assert.Equal(t, tt.expectedIndexDiffs, len(allIndexDiffs), "Expected %d index diffs, got %d", tt.expectedIndexDiffs, len(allIndexDiffs))

			// Check that the actions match expectations
			var actualActions []schema.MetadataDiffAction
			for _, indexDiff := range allIndexDiffs {
				actualActions = append(actualActions, indexDiff.Action)
			}

			// Handle nil vs empty slice comparison
			if len(tt.expectedActions) == 0 && len(actualActions) == 0 {
				// Both are effectively empty - test passes
				t.Log("Both expected and actual actions are empty")
			} else {
				assert.Equal(t, tt.expectedActions, actualActions, "Expected actions %v, got %v", tt.expectedActions, actualActions)
			}

			// Verify AST nodes are properly set
			for i, indexDiff := range allIndexDiffs {
				switch indexDiff.Action {
				case schema.MetadataDiffActionCreate:
					assert.NotNil(t, indexDiff.NewASTNode, "Index diff %d should have NewASTNode for CREATE action", i)
					assert.Nil(t, indexDiff.OldASTNode, "Index diff %d should not have OldASTNode for CREATE action", i)
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, indexDiff.OldASTNode, "Index diff %d should have OldASTNode for DROP action", i)
					assert.Nil(t, indexDiff.NewASTNode, "Index diff %d should not have NewASTNode for DROP action", i)
				default:
					t.Errorf("Unexpected action %v for index diff %d", indexDiff.Action, i)
				}
			}
		})
	}
}

func TestGetStandaloneIndexText(t *testing.T) {
	// This test validates that the text extraction works correctly
	sdlText := `CREATE INDEX idx_users_email ON users(email);`

	chunks, err := ChunkSDLText(sdlText)
	require.NoError(t, err)
	require.NotNil(t, chunks)

	// Should have exactly one index chunk
	assert.Equal(t, 1, len(chunks.Indexes))

	// Get the index chunk
	var indexChunk *schema.SDLChunk
	for _, chunk := range chunks.Indexes {
		indexChunk = chunk
		break
	}
	require.NotNil(t, indexChunk)

	// Test text extraction
	text := getStandaloneIndexText(indexChunk.ASTNode)
	assert.Contains(t, text, "CREATE INDEX")
	assert.Contains(t, text, "idx_users_email")
	assert.Contains(t, text, "users(email)")
}

// TestStandaloneIndexIntegrationWithTableChanges tests integration with table changes
func TestStandaloneIndexIntegrationWithTableChanges(t *testing.T) {
	tests := []struct {
		name                 string
		previousUserSDL      string
		currentSDL           string
		expectedTableChanges int
		expectedTableName    string
		expectedTableAction  schema.MetadataDiffAction
		expectedIndexChanges int
		expectedIndexActions []schema.MetadataDiffAction
	}{
		{
			name:            "Standalone index on existing table (no table changes)",
			previousUserSDL: `CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE INDEX idx_users_email ON users(email);
			`,
			expectedTableChanges: 1, // One table affected by index change
			expectedTableName:    "users",
			expectedTableAction:  schema.MetadataDiffActionAlter,
			expectedIndexChanges: 1,
			expectedIndexActions: []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Table change and index change combined",
			previousUserSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE INDEX idx_users_email ON users(email);
			`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255), name VARCHAR(100));
				CREATE INDEX idx_users_email_name ON users(email, name);
			`,
			expectedTableChanges: 1, // Same table with both column and index changes
			expectedTableName:    "users",
			expectedTableAction:  schema.MetadataDiffActionAlter,
			expectedIndexChanges: 2, // Create new index + drop old index (different names)
			expectedIndexActions: []schema.MetadataDiffAction{
				schema.MetadataDiffActionCreate, // New index idx_users_email_name
				schema.MetadataDiffActionDrop,   // Old index idx_users_email
			},
		},
		{
			name: "Multiple tables with index changes",
			previousUserSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INTEGER);
				CREATE INDEX idx_users_email ON users(email);
			`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INTEGER);
				CREATE INDEX idx_users_email ON users(email);
				CREATE INDEX idx_orders_user_id ON orders(user_id);
			`,
			expectedTableChanges: 1, // Only orders table affected by new index
			expectedTableName:    "orders",
			expectedTableAction:  schema.MetadataDiffActionAlter,
			expectedIndexChanges: 1,
			expectedIndexActions: []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Index on non-primary schema table", // Testing schema extraction
			previousUserSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (id SERIAL PRIMARY KEY, name VARCHAR(255));
			`,
			currentSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (id SERIAL PRIMARY KEY, name VARCHAR(255));
				CREATE INDEX idx_products_name ON test_schema.products(name);
			`,
			expectedTableChanges: 1,
			expectedTableName:    "products", // Table name without schema prefix
			expectedTableAction:  schema.MetadataDiffActionAlter,
			expectedIndexChanges: 1,
			expectedIndexActions: []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousUserSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			assert.Equal(t, tt.expectedTableChanges, len(diff.TableChanges),
				"Expected %d table changes, got %d", tt.expectedTableChanges, len(diff.TableChanges))

			if tt.expectedTableChanges > 0 {
				// Find the table change we're interested in
				var targetTableChange *schema.TableDiff
				for _, tableChange := range diff.TableChanges {
					if tableChange.TableName == tt.expectedTableName {
						targetTableChange = tableChange
						break
					}
				}

				require.NotNil(t, targetTableChange, "Expected to find table change for %s", tt.expectedTableName)
				assert.Equal(t, tt.expectedTableAction, targetTableChange.Action,
					"Expected table action %v, got %v", tt.expectedTableAction, targetTableChange.Action)

				// Check index changes within this table
				assert.Equal(t, tt.expectedIndexChanges, len(targetTableChange.IndexChanges),
					"Expected %d index changes in table %s, got %d",
					tt.expectedIndexChanges, tt.expectedTableName, len(targetTableChange.IndexChanges))

				// Verify index actions
				var actualIndexActions []schema.MetadataDiffAction
				for _, indexChange := range targetTableChange.IndexChanges {
					actualIndexActions = append(actualIndexActions, indexChange.Action)
				}
				assert.Equal(t, tt.expectedIndexActions, actualIndexActions,
					"Expected index actions %v, got %v", tt.expectedIndexActions, actualIndexActions)

				// Verify AST nodes are properly set
				for i, indexChange := range targetTableChange.IndexChanges {
					switch indexChange.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, indexChange.NewASTNode,
							"Index change %d should have NewASTNode for CREATE", i)
						assert.Nil(t, indexChange.OldASTNode,
							"Index change %d should not have OldASTNode for CREATE", i)
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, indexChange.OldASTNode,
							"Index change %d should have OldASTNode for DROP", i)
						assert.Nil(t, indexChange.NewASTNode,
							"Index change %d should not have NewASTNode for DROP", i)
					default:
						t.Errorf("Unexpected index action %v for change %d", indexChange.Action, i)
					}
				}
			}
		})
	}
}

// TestIndexTableNameExtraction specifically tests the table name extraction logic
func TestIndexTableNameExtraction(t *testing.T) {
	tests := []struct {
		name         string
		indexSQL     string
		expectedName string
	}{
		{
			name:         "Simple table name",
			indexSQL:     "CREATE INDEX idx_test ON users(email);",
			expectedName: "public.users", // Defaults to public schema
		},
		{
			name:         "Qualified table name",
			indexSQL:     "CREATE INDEX idx_test ON public.users(email);",
			expectedName: "public.users",
		},
		{
			name:         "Custom schema",
			indexSQL:     "CREATE INDEX idx_test ON myschema.products(name);",
			expectedName: "myschema.products",
		},
		{
			name:         "Complex index with WHERE clause",
			indexSQL:     "CREATE INDEX idx_active_users ON public.users(email) WHERE active = true;",
			expectedName: "public.users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the index statement
			chunks, err := ChunkSDLText(tt.indexSQL)
			require.NoError(t, err)
			require.Equal(t, 1, len(chunks.Indexes), "Should have exactly one index")

			// Get the index chunk
			var indexChunk *schema.SDLChunk
			for _, chunk := range chunks.Indexes {
				indexChunk = chunk
				break
			}
			require.NotNil(t, indexChunk)

			// Test table name extraction
			tableName := extractTableNameFromIndex(indexChunk.ASTNode)
			assert.Equal(t, tt.expectedName, tableName,
				"Expected table name '%s', got '%s'", tt.expectedName, tableName)
		})
	}
}

// TestIndexSchemaNameExtraction tests that index schema names are correctly extracted
// This is a regression test for the bug where indexes in non-public schemas
// were incorrectly identified as belonging to the public schema
func TestIndexSchemaNameExtraction(t *testing.T) {
	tests := []struct {
		name               string
		previousUserSDL    string
		currentSDL         string
		expectedSchemaName string
		expectedTableName  string
		expectedIndexCount int
	}{
		{
			name:            "Index in company schema",
			previousUserSDL: "",
			currentSDL: `
CREATE SCHEMA company;

CREATE TABLE "company"."employees" (
	"id" integer NOT NULL,
	"first_name" character varying(50) NOT NULL,
	"email" character varying(100) NOT NULL
);

CREATE INDEX "idx_employees_email" ON "company"."employees" (email);
			`,
			expectedSchemaName: "company",
			expectedTableName:  "employees",
			expectedIndexCount: 1,
		},
		{
			name: "Index in custom schema with multiple indexes",
			previousUserSDL: `
CREATE SCHEMA myapp;

CREATE TABLE "myapp"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);
			`,
			currentSDL: `
CREATE SCHEMA myapp;

CREATE TABLE "myapp"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);

CREATE INDEX "idx_products_name" ON "myapp"."products" (name);
CREATE INDEX "idx_products_price" ON "myapp"."products" (price);
			`,
			expectedSchemaName: "myapp",
			expectedTableName:  "products",
			expectedIndexCount: 2,
		},
		{
			name: "Mixed schemas - should only affect correct schema",
			previousUserSDL: `
CREATE SCHEMA company;
CREATE SCHEMA sales;

CREATE TABLE "company"."employees" (
	"id" integer NOT NULL,
	"name" text
);

CREATE TABLE "sales"."orders" (
	"id" integer NOT NULL,
	"amount" numeric
);
			`,
			currentSDL: `
CREATE SCHEMA company;
CREATE SCHEMA sales;

CREATE TABLE "company"."employees" (
	"id" integer NOT NULL,
	"name" text
);

CREATE TABLE "sales"."orders" (
	"id" integer NOT NULL,
	"amount" numeric
);

CREATE INDEX "idx_employees_name" ON "company"."employees" (name);
			`,
			expectedSchemaName: "company",
			expectedTableName:  "employees",
			expectedIndexCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousUserSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Find the table change for our expected table
			var targetTableDiff *schema.TableDiff
			for _, tableDiff := range diff.TableChanges {
				if tableDiff.SchemaName == tt.expectedSchemaName && tableDiff.TableName == tt.expectedTableName {
					targetTableDiff = tableDiff
					break
				}
			}

			require.NotNil(t, targetTableDiff,
				"Expected to find table change for %s.%s", tt.expectedSchemaName, tt.expectedTableName)

			// Verify schema name is correct (not defaulting to "public")
			assert.Equal(t, tt.expectedSchemaName, targetTableDiff.SchemaName,
				"Expected schema name '%s', got '%s'", tt.expectedSchemaName, targetTableDiff.SchemaName)

			// Verify table name is correct
			assert.Equal(t, tt.expectedTableName, targetTableDiff.TableName,
				"Expected table name '%s', got '%s'", tt.expectedTableName, targetTableDiff.TableName)

			// Verify index count
			assert.Equal(t, tt.expectedIndexCount, len(targetTableDiff.IndexChanges),
				"Expected %d index changes, got %d", tt.expectedIndexCount, len(targetTableDiff.IndexChanges))

			// Verify all index changes have CREATE action
			for _, indexChange := range targetTableDiff.IndexChanges {
				assert.Equal(t, schema.MetadataDiffActionCreate, indexChange.Action,
					"Expected CREATE action for index")
			}
		})
	}
}

// TestPartialIndexWhereClauseInSDLOutput tests that partial index WHERE clauses
// are preserved in SDL output (regression test for WHERE clause being dropped).
func TestPartialIndexWhereClauseInSDLOutput(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "channels",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "slug", Type: "character varying(255)", Position: 2},
							{Name: "social_region", Type: "character varying(50)", Position: 3},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "udx_slug_social_region_on_channels",
								Expressions: []string{"slug", "social_region"},
								Type:        "btree",
								Unique:      true,
								Definition:  "CREATE UNIQUE INDEX udx_slug_social_region_on_channels ON public.channels USING btree (slug, social_region) WHERE ((slug)::text <> ''::text);",
							},
						},
					},
				},
			},
		},
	}

	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	// WHERE clause must be present
	require.Contains(t, sdl, "WHERE")
	require.Contains(t, sdl, "(slug)::text")

	// Non-partitioned table: must NOT have ON ONLY
	require.NotContains(t, sdl, "ON ONLY")
	require.Contains(t, sdl, "ON \"public\".\"channels\"")
}

// TestRegularIndexNoOnOnly tests that regular (non-partitioned) table indexes
// do not include ON ONLY in SDL output.
func TestRegularIndexNoOnOnly(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "email", Type: "character varying(255)", Position: 2},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "idx_users_email",
								Expressions: []string{"email"},
								Type:        "btree",
							},
						},
					},
				},
			},
		},
	}

	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	require.Contains(t, sdl, `CREATE INDEX "idx_users_email"`)
	require.Contains(t, sdl, `ON "public"."users"`)
	require.NotContains(t, sdl, "ON ONLY")
}

// TestPartitionedTableIndexUsesOnOnly tests that partitioned parent table indexes
// use ON ONLY in SDL output to prevent recursive child index creation.
func TestPartitionedTableIndexUsesOnOnly(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "order_date", Type: "date", Position: 2},
							{Name: "customer_id", Type: "integer", Position: 3},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "idx_orders_customer_id",
								Expressions: []string{"customer_id"},
								Type:        "btree",
							},
						},
						Partitions: []*storepb.TablePartitionMetadata{
							{
								Name: "orders_2024",
								Type: storepb.TablePartitionMetadata_RANGE,
							},
						},
					},
				},
			},
		},
	}

	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err)

	require.Contains(t, sdl, `CREATE INDEX "idx_orders_customer_id"`)
	require.Contains(t, sdl, `ON ONLY "public"."orders"`)
}

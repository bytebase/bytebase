package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestMaterializedViewSDLDiff(t *testing.T) {
	tests := []struct {
		name                            string
		previousSDL                     string
		currentSDL                      string
		expectedMaterializedViewChanges int
		expectedActions                 []schema.MetadataDiffAction
	}{
		{
			name:        "Create new materialized view",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			expectedMaterializedViewChanges: 1,
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop materialized view",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);
			`,
			expectedMaterializedViewChanges: 1,
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify materialized view (drop and recreate)",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name, 'active' as status
				FROM users
				WHERE active = true;
			`,
			expectedMaterializedViewChanges: 2, // Drop + Create
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionDrop, schema.MetadataDiffActionCreate},
		},
		{
			name: "No changes to materialized view",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			expectedMaterializedViewChanges: 0,
			expectedActions:                 []schema.MetadataDiffAction{},
		},
		{
			name: "Multiple materialized views with different changes",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW all_users_mv AS
				SELECT * FROM users;

				CREATE MATERIALIZED VIEW active_users_mv AS
				SELECT id, name FROM users WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);

				CREATE MATERIALIZED VIEW all_users_mv AS
				SELECT * FROM users;

				CREATE MATERIALIZED VIEW admin_users_mv AS
				SELECT id, name FROM users WHERE role = 'admin';
			`,
			expectedMaterializedViewChanges: 2, // Drop active_users_mv + Create admin_users_mv
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionCreate, schema.MetadataDiffActionDrop},
		},
		{
			name: "Schema-qualified materialized view names",
			previousSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
			`,
			currentSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW test_schema.product_summary_mv AS
				SELECT id, name FROM test_schema.products;
			`,
			expectedMaterializedViewChanges: 1,
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Materialized view with comment",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE MATERIALIZED VIEW user_summary_mv AS
				SELECT id, name FROM users;

				COMMENT ON MATERIALIZED VIEW user_summary_mv IS 'Summary of all users';
			`,
			expectedMaterializedViewChanges: 1,
			expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			assert.Equal(t, tt.expectedMaterializedViewChanges, len(diff.MaterializedViewChanges),
				"Expected %d materialized view changes, got %d", tt.expectedMaterializedViewChanges, len(diff.MaterializedViewChanges))

			// Check that the actions match expectations
			var actualActions []schema.MetadataDiffAction
			for _, mvDiff := range diff.MaterializedViewChanges {
				actualActions = append(actualActions, mvDiff.Action)
			}

			// Handle nil vs empty slice comparison
			if len(tt.expectedActions) == 0 && len(actualActions) == 0 {
				// Both are effectively empty - test passes
				t.Log("Both expected and actual actions are empty")
			} else {
				assert.ElementsMatch(t, tt.expectedActions, actualActions,
					"Expected actions %v, got %v", tt.expectedActions, actualActions)
			}

			// Verify AST nodes are properly set
			for i, mvDiff := range diff.MaterializedViewChanges {
				switch mvDiff.Action {
				case schema.MetadataDiffActionCreate:
					assert.NotNil(t, mvDiff.NewASTNode,
						"Materialized view diff %d should have NewASTNode for CREATE action", i)
					assert.Nil(t, mvDiff.OldASTNode,
						"Materialized view diff %d should not have OldASTNode for CREATE action", i)
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, mvDiff.OldASTNode,
						"Materialized view diff %d should have OldASTNode for DROP action", i)
					assert.Nil(t, mvDiff.NewASTNode,
						"Materialized view diff %d should not have NewASTNode for DROP action", i)
				default:
					t.Errorf("Unexpected action %v for materialized view diff %d", mvDiff.Action, i)
				}
			}
		})
	}
}

func TestMaterializedViewWithCommentParsing(t *testing.T) {
	tests := []struct {
		name               string
		sdl                string
		expectedMVCount    int
		expectedComment    string
		expectCommentInSDL bool
	}{
		{
			name: "Create materialized view with comment",
			sdl: `
				CREATE TABLE public.users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE MATERIALIZED VIEW public.user_summary_mv AS
				SELECT id, name FROM users;

				COMMENT ON MATERIALIZED VIEW public.user_summary_mv IS 'Summary of all users';
			`,
			expectedMVCount:    1,
			expectedComment:    "Summary of all users",
			expectCommentInSDL: true,
		},
		{
			name: "Create materialized view with schema-qualified name and comment",
			sdl: `
				CREATE TABLE public.table1 (
					id INTEGER PRIMARY KEY,
					name TEXT
				);

				CREATE MATERIALIZED VIEW "public"."table1_summary" AS
				SELECT t2.id AS table2_id, t2.name AS table2_name
				FROM "public"."table1" t2;

				COMMENT ON MATERIALIZED VIEW "public"."table1_summary" IS 'Summarizes table1 records grouped by table2 categories. Refresh periodically to keep data current.';
			`,
			expectedMVCount:    1,
			expectedComment:    "Summarizes table1 records grouped by table2 categories. Refresh periodically to keep data current.",
			expectCommentInSDL: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the SDL to get chunks
			chunks, err := ChunkSDLText(tt.sdl)
			require.NoError(t, err)
			require.NotNil(t, chunks)

			// Debug: Print what we got
			t.Logf("Found %d materialized views", len(chunks.MaterializedViews))
			for id, chunk := range chunks.MaterializedViews {
				t.Logf("  MV ID: %s", id)
				t.Logf("  Has ASTNode: %v", chunk.ASTNode != nil)
				t.Logf("  CommentStatements: %d", len(chunk.CommentStatements))
			}

			// Verify we parsed the correct number of materialized views
			assert.Equal(t, tt.expectedMVCount, len(chunks.MaterializedViews),
				"Expected %d materialized view(s), got %d", tt.expectedMVCount, len(chunks.MaterializedViews))

			// Get the materialized view chunk
			var mvChunk *schema.SDLChunk
			for _, chunk := range chunks.MaterializedViews {
				mvChunk = chunk
				break
			}
			require.NotNil(t, mvChunk, "Materialized view chunk should exist")

			// Get the full text including comments
			fullText := mvChunk.GetText()
			t.Logf("Full materialized view text:\n%s", fullText)

			// Verify comment is included in the SDL
			if tt.expectCommentInSDL {
				assert.Contains(t, fullText, "COMMENT ON MATERIALIZED VIEW",
					"Full text should contain COMMENT ON MATERIALIZED VIEW statement")
				assert.Contains(t, fullText, tt.expectedComment,
					"Comment should contain the expected text: %s", tt.expectedComment)

				// Verify comment statements count
				assert.Greater(t, len(mvChunk.CommentStatements), 0,
					"Materialized view should have comment statements")
			}

			// Verify CREATE MATERIALIZED VIEW is present
			assert.Contains(t, fullText, "CREATE MATERIALIZED VIEW",
				"Full text should contain CREATE MATERIALIZED VIEW statement")
		})
	}
}

func TestMaterializedViewMigrationGeneration(t *testing.T) {
	tests := []struct {
		name        string
		previousSDL string
		currentSDL  string
		wantCreate  bool
		wantDrop    bool
	}{
		{
			name:        "Create materialized view generates CREATE statement",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;
			`,
			wantCreate: true,
			wantDrop:   false,
		},
		{
			name: "Drop materialized view generates DROP statement",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
			`,
			wantCreate: false,
			wantDrop:   true,
		},
		{
			name: "Modify materialized view generates DROP and CREATE",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255),
					price DECIMAL
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name, price FROM products;
			`,
			wantCreate: true,
			wantDrop:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			migration, err := generateMigration(diff)
			require.NoError(t, err)

			if tt.wantCreate {
				assert.Contains(t, migration, "CREATE MATERIALIZED VIEW",
					"Expected migration to contain CREATE MATERIALIZED VIEW statement")
			}
			if tt.wantDrop {
				assert.Contains(t, migration, "DROP MATERIALIZED VIEW",
					"Expected migration to contain DROP MATERIALIZED VIEW statement")
			}
		})
	}
}

func TestMaterializedViewCommentMigrationGeneration(t *testing.T) {
	tests := []struct {
		name        string
		previousSDL string
		currentSDL  string
		wantComment bool
	}{
		{
			name:        "Create materialized view with comment generates COMMENT statement",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;

				COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
			`,
			wantComment: true,
		},
		{
			name: "Add comment to existing materialized view",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;

				COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
			`,
			wantComment: true,
		},
		{
			name: "Remove comment from materialized view",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;

				COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;
			`,
			wantComment: true, // Should generate COMMENT statement with empty string
		},
		{
			name: "Update comment on materialized view",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;

				COMMENT ON MATERIALIZED VIEW product_mv IS 'Old comment';
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);

				CREATE MATERIALIZED VIEW product_mv AS
				SELECT id, name FROM products;

				COMMENT ON MATERIALIZED VIEW product_mv IS 'New comment';
			`,
			wantComment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			migration, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration:\n%s", migration)

			if tt.wantComment {
				assert.Contains(t, migration, "COMMENT ON MATERIALIZED VIEW",
					"Expected migration to contain COMMENT ON MATERIALIZED VIEW statement")
			}
		})
	}
}

func TestMaterializedViewDependencyOrder(t *testing.T) {
	tests := []struct {
		name        string
		previousSDL string
		currentSDL  string
		description string
	}{
		{
			name:        "Create table, view, and materialized view with dependencies",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE customers (
					customer_id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255),
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE orders (
					order_id SERIAL PRIMARY KEY,
					customer_id INTEGER REFERENCES customers(customer_id),
					amount DECIMAL(10,2),
					order_date DATE
				);

				-- View depends on tables
				CREATE VIEW customer_stats_view AS
				SELECT
					c.customer_id,
					c.name,
					c.email,
					COUNT(o.order_id) as order_count,
					SUM(o.amount) as total_spent
				FROM customers c
				LEFT JOIN orders o ON c.customer_id = o.customer_id
				GROUP BY c.customer_id, c.name, c.email;

				-- Materialized view depends on the view above
				CREATE MATERIALIZED VIEW customer_segmentation_mv AS
				SELECT
					csv.customer_id,
					csv.name,
					csv.total_spent,
					CASE
						WHEN csv.total_spent >= 1000 THEN 'Premium'
						WHEN csv.total_spent >= 500 THEN 'Standard'
						ELSE 'Basic'
					END as segment
				FROM customer_stats_view csv;
			`,
			description: "Tests that objects are created in correct dependency order: tables -> view -> materialized view",
		},
		{
			name: "Drop table, view, and materialized view with dependencies",
			previousSDL: `
				CREATE TABLE customers (
					customer_id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255),
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE orders (
					order_id SERIAL PRIMARY KEY,
					customer_id INTEGER REFERENCES customers(customer_id),
					amount DECIMAL(10,2),
					order_date DATE
				);

				-- View depends on tables
				CREATE VIEW customer_stats_view AS
				SELECT
					c.customer_id,
					c.name,
					c.email,
					COUNT(o.order_id) as order_count,
					SUM(o.amount) as total_spent
				FROM customers c
				LEFT JOIN orders o ON c.customer_id = o.customer_id
				GROUP BY c.customer_id, c.name, c.email;

				-- Materialized view depends on the view above
				CREATE MATERIALIZED VIEW customer_segmentation_mv AS
				SELECT
					csv.customer_id,
					csv.name,
					csv.total_spent,
					CASE
						WHEN csv.total_spent >= 1000 THEN 'Premium'
						WHEN csv.total_spent >= 500 THEN 'Standard'
						ELSE 'Basic'
					END as segment
				FROM customer_stats_view csv;
			`,
			currentSDL:  ``,
			description: "Tests that objects are dropped in correct dependency order: materialized view -> view -> tables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			migration, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration:\n%s", migration)

			// Check if this is a CREATE or DROP test
			if tt.currentSDL != "" {
				// CREATE test: verify correct order for creation
				// Tables should come before views, and views should come before materialized views
				customersIdx := strings.Index(migration, "CREATE TABLE customers")
				ordersIdx := strings.Index(migration, "CREATE TABLE orders")
				viewIdx := strings.Index(migration, "CREATE VIEW customer_stats_view")
				mvIdx := strings.Index(migration, "CREATE MATERIALIZED VIEW customer_segmentation_mv")

				assert.NotEqual(t, -1, customersIdx, "customers table should be created")
				assert.NotEqual(t, -1, ordersIdx, "orders table should be created")
				assert.NotEqual(t, -1, viewIdx, "customer_stats_view should be created")
				assert.NotEqual(t, -1, mvIdx, "customer_segmentation_mv should be created")

				// Verify correct order for CREATE
				if customersIdx != -1 && viewIdx != -1 {
					assert.Less(t, customersIdx, viewIdx,
						"customers table must be created before customer_stats_view")
				}
				if ordersIdx != -1 && viewIdx != -1 {
					assert.Less(t, ordersIdx, viewIdx,
						"orders table must be created before customer_stats_view")
				}
				if viewIdx != -1 && mvIdx != -1 {
					assert.Less(t, viewIdx, mvIdx,
						"customer_stats_view must be created before customer_segmentation_mv")
				}
			} else {
				// DROP test: verify correct order for dropping
				// Materialized views should be dropped before views, and views before tables
				mvIdx := strings.Index(migration, "DROP MATERIALIZED VIEW")
				viewIdx := strings.Index(migration, "DROP VIEW")
				customersIdx := strings.Index(migration, "DROP TABLE")

				assert.NotEqual(t, -1, mvIdx, "customer_segmentation_mv should be dropped")
				assert.NotEqual(t, -1, viewIdx, "customer_stats_view should be dropped")
				assert.NotEqual(t, -1, customersIdx, "tables should be dropped")

				// Verify correct order for DROP (reverse of CREATE)
				if mvIdx != -1 && viewIdx != -1 {
					assert.Less(t, mvIdx, viewIdx,
						"customer_segmentation_mv must be dropped before customer_stats_view")
				}
				if viewIdx != -1 && customersIdx != -1 {
					assert.Less(t, viewIdx, customersIdx,
						"customer_stats_view must be dropped before tables")
				}
			}
		})
	}
}

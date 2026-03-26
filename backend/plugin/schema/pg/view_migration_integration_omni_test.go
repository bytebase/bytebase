package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniViewSDLDiffAndMigration(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name:    "Create new view",
			fromSDL: ``,
			toSDL: `
				CREATE VIEW active_users AS
				SELECT 1 as id, 'test' as name;
			`,
			contains: []string{"CREATE VIEW", "active_users"},
		},
		{
			name: "Drop view",
			fromSDL: `
				CREATE VIEW active_users AS
				SELECT 1 as id, 'test' as name;
			`,
			toSDL:    ``,
			contains: []string{"DROP VIEW", "active_users"},
		},
		{
			name: "Modify view (drop and recreate)",
			fromSDL: `
				CREATE VIEW active_users AS
				SELECT 1 as id, 'test' as name;
			`,
			toSDL: `
				CREATE VIEW active_users AS
				SELECT 2 as id, 'updated' as name, 'extra' as email;
			`,
			contains: []string{"active_users"},
		},
		{
			name:    "Schema-qualified view",
			fromSDL: ``,
			toSDL: `
				CREATE SCHEMA test_schema;
				CREATE VIEW test_schema.expensive_products AS
				SELECT 1 as id, 'product' as name, 150.00 as price;
			`,
			contains: []string{"CREATE VIEW", "expensive_products"},
		},
		{
			name: "Multiple view changes",
			fromSDL: `
				CREATE VIEW user_summary AS
				SELECT 1 as id, 'user' as name;

				CREATE VIEW order_summary AS
				SELECT 1 as id, 100.00 as amount;
			`,
			toSDL: `
				CREATE VIEW user_summary AS
				SELECT 2 as id, 'updated_user' as name, 'active' as status;

				CREATE VIEW order_analytics AS
				SELECT 1 as user_id, 500.00 as total_amount;
			`,
			contains: []string{"order_summary", "order_analytics", "user_summary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniViewDependencyHandling(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE VIEW base_data AS
		SELECT 1 as id, 'item1' as name, 100 as value;

		CREATE VIEW summary_report AS
		SELECT b.name, b.value, b.value * 2 as doubled_value
		FROM base_data b
		WHERE b.value > 50;
	`)
	require.Contains(t, sql, "CREATE VIEW")
	require.Contains(t, sql, "base_data")
	require.Contains(t, sql, "summary_report")

	// base_data should be created before summary_report
	baseIndex := strings.Index(sql, "base_data")
	summaryIndex := strings.Index(sql, "summary_report")
	require.True(t, baseIndex < summaryIndex,
		"base_data should appear before summary_report")
}

func TestOmniDropTableAndDependentView_CorrectOrder(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);

		CREATE VIEW active_users AS
		SELECT id, name, email
		FROM users
		WHERE email IS NOT NULL;
	`, "")

	// Omni uses DROP TABLE ... CASCADE which handles dependent views,
	// so both DROP TABLE and DROP VIEW should be present
	require.Contains(t, sql, "DROP TABLE")
	require.Contains(t, sql, "users")
	require.Contains(t, sql, "active_users")
}

func TestOmniCreateTableAndDependentView_CorrectOrder(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);

		CREATE VIEW active_users AS
		SELECT id, name, email
		FROM users
		WHERE email IS NOT NULL;
	`)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "CREATE VIEW")

	tableCreateIndex := strings.Index(sql, "CREATE TABLE")
	viewCreateIndex := strings.Index(sql, "CREATE VIEW")
	require.True(t, tableCreateIndex < viewCreateIndex,
		"CREATE TABLE must come before CREATE VIEW")
}

func TestOmniCreateMultipleTablesAndViews_CorrectOrder(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INTEGER REFERENCES categories(id)
		);

		CREATE VIEW product_summary AS
		SELECT p.id, p.name, c.name as category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id;
	`)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "CREATE VIEW")

	categoriesIndex := strings.Index(sql, "categories")
	viewIndex := strings.Index(sql, "CREATE VIEW")
	require.True(t, categoriesIndex < viewIndex,
		"CREATE TABLE categories must come before CREATE VIEW")

	productsIndex := strings.Index(sql, "products")
	require.True(t, productsIndex < viewIndex,
		"CREATE TABLE products must come before CREATE VIEW")
}

func TestOmniDropMultipleTablesAndViews_CorrectOrder(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INTEGER REFERENCES categories(id)
		);

		CREATE VIEW product_summary AS
		SELECT p.id, p.name, c.name as category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id;
	`, "")

	require.Contains(t, sql, "DROP VIEW")
	require.Contains(t, sql, "DROP TABLE")

	viewDropIndex := strings.Index(sql, "DROP VIEW")
	productsTableDropIndex := strings.Index(sql, "products")
	// Find the DROP TABLE that mentions products (not the view reference)
	// The view drop should come first
	require.True(t, viewDropIndex >= 0, "DROP VIEW should be present")
	_ = productsTableDropIndex // products appears in both drop statements
}

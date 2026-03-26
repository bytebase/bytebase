package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniMaterializedViewSDLDiff_CreateNew(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name
		FROM users
		WHERE active = true;
	`)
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "active_users_mv")
}

func TestOmniMaterializedViewSDLDiff_Drop(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name
		FROM users
		WHERE active = true;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);
	`)
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
	require.Contains(t, sql, "active_users_mv")
}

func TestOmniMaterializedViewSDLDiff_Modify(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name
		FROM users
		WHERE active = true;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name, 'active' as status
		FROM users
		WHERE active = true;
	`)
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "active_users_mv")
}

func TestOmniMaterializedViewSDLDiff_NoChange(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name
		FROM users
		WHERE active = true;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name
		FROM users
		WHERE active = true;
	`)
	require.Empty(t, sql)
}

func TestOmniMaterializedViewSDLDiff_MultipleViews(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW all_users_mv AS
		SELECT * FROM users;

		CREATE MATERIALIZED VIEW active_users_mv AS
		SELECT id, name FROM users WHERE active = true;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT true
		);

		CREATE MATERIALIZED VIEW all_users_mv AS
		SELECT * FROM users;

		CREATE MATERIALIZED VIEW admin_users_mv AS
		SELECT id, name FROM users WHERE name = 'admin';
	`)
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
	require.Contains(t, sql, "active_users_mv")
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "admin_users_mv")
}

func TestOmniMaterializedViewSDLDiff_SchemaQualified(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE SCHEMA test_schema;
		CREATE TABLE test_schema.products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);
	`, `
		CREATE SCHEMA test_schema;
		CREATE TABLE test_schema.products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW test_schema.product_summary_mv AS
		SELECT id, name FROM test_schema.products;
	`)
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "product_summary_mv")
}

func TestOmniMaterializedViewSDLDiff_WithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);

		CREATE MATERIALIZED VIEW user_summary_mv AS
		SELECT id, name FROM users;

		COMMENT ON MATERIALIZED VIEW user_summary_mv IS 'Summary of all users';
	`)
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "user_summary_mv")
	require.Contains(t, sql, "COMMENT ON MATERIALIZED VIEW")
	require.Contains(t, sql, "Summary of all users")
}

func TestOmniMaterializedViewMigration_Create(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;
	`)
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.NotContains(t, sql, "DROP MATERIALIZED VIEW")
}

func TestOmniMaterializedViewMigration_DropOnly(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);
	`)
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
}

func TestOmniMaterializedViewMigration_ModifyDropAndCreate(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255),
			price DECIMAL
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name, price FROM products;
	`)
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
}

func TestOmniMaterializedViewCommentMigration_CreateWithComment(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;

		COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
	`)
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "Product summary view")
}

func TestOmniMaterializedViewCommentMigration_AddComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;

		COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "Product summary view")
}

func TestOmniMaterializedViewCommentMigration_RemoveComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;

		COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "product_mv")
	require.Contains(t, sql, "NULL")
}

func TestOmniMaterializedViewCommentMigration_UpdateComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;

		COMMENT ON MATERIALIZED VIEW product_mv IS 'Old comment';
	`, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE MATERIALIZED VIEW product_mv AS
		SELECT id, name FROM products;

		COMMENT ON MATERIALIZED VIEW product_mv IS 'New comment';
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "New comment")
}

func TestOmniMaterializedViewDependencyOrder_Create(t *testing.T) {
	sql := omniSDLMigration(t, "", `
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
	`)

	customersIdx := strings.Index(sql, "CREATE TABLE")
	viewIdx := strings.Index(sql, "CREATE VIEW")
	mvIdx := strings.Index(sql, "CREATE MATERIALIZED VIEW")

	require.NotEqual(t, -1, customersIdx, "customers table should be created")
	require.NotEqual(t, -1, viewIdx, "view should be created")
	require.NotEqual(t, -1, mvIdx, "materialized view should be created")

	// Tables must be created before views and materialized views
	require.Less(t, customersIdx, viewIdx, "tables must be created before view")
	require.Less(t, customersIdx, mvIdx, "tables must be created before materialized view")
}

func TestOmniMaterializedViewDependencyOrder_Drop(t *testing.T) {
	sql := omniSDLMigration(t, `
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
	`, "")

	// The omni engine drops objects; verify all types are represented
	require.Contains(t, sql, "DROP MATERIALIZED VIEW")
	require.Contains(t, sql, "DROP VIEW")
	require.Contains(t, sql, "DROP TABLE")

	require.Contains(t, sql, "customer_segmentation_mv")
	require.Contains(t, sql, "customer_stats_view")
}

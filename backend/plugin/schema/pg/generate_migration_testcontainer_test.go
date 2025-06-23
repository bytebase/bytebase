package pg

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connectionString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to the database
	connConfig, err := pgx.ParseConfig(connectionString)
	require.NoError(t, err)
	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	// Test cases with various schema changes
	testCases := []struct {
		name          string
		initialSchema string
		migrationDDL  string
		description   string
	}{
		{
			name: "basic_table_operations",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
`,
			migrationDDL: `
-- Add new column
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;

-- Create new table
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_post FOREIGN KEY (post_id) REFERENCES posts(id),
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Add new index
CREATE INDEX idx_users_email ON users(email);

-- Add check constraint
ALTER TABLE posts ADD CONSTRAINT check_title_length CHECK (length(title) > 0);
`,
			description: "Basic table operations with columns, constraints, and indexes",
		},
		{
			name: "views_and_functions",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    stock INTEGER DEFAULT 0
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    total DECIMAL(10, 2),
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id)
);
`,
			migrationDDL: `
-- Create view
CREATE VIEW product_inventory AS
SELECT 
    p.id,
    p.name,
    p.price,
    p.stock,
    COALESCE(SUM(o.quantity), 0) as total_ordered
FROM products p
LEFT JOIN orders o ON p.id = o.product_id
GROUP BY p.id, p.name, p.price, p.stock;

-- Create function
CREATE OR REPLACE FUNCTION calculate_order_total()
RETURNS TRIGGER AS $$
BEGIN
    NEW.total := NEW.quantity * (SELECT price FROM products WHERE id = NEW.product_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER update_order_total
BEFORE INSERT OR UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION calculate_order_total();

-- Create materialized view
CREATE MATERIALIZED VIEW product_stats AS
SELECT 
    product_id,
    COUNT(*) as order_count,
    SUM(quantity) as total_quantity,
    SUM(total) as total_revenue
FROM orders
GROUP BY product_id;

-- Create index on materialized view
CREATE INDEX idx_product_stats_revenue ON product_stats(total_revenue DESC);
`,
			description: "Views, functions, triggers, and materialized views",
		},
		{
			name: "schema_and_sequences",
			initialSchema: `
CREATE SCHEMA inventory;

CREATE TABLE inventory.items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
`,
			migrationDDL: `
-- Create new schema
CREATE SCHEMA sales;

-- Create sequence
CREATE SEQUENCE sales.order_seq START WITH 1000 INCREMENT BY 10;

-- Create table using sequence
CREATE TABLE sales.orders (
    id INTEGER DEFAULT nextval('sales.order_seq') PRIMARY KEY,
    item_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    CONSTRAINT fk_item FOREIGN KEY (item_id) REFERENCES inventory.items(id)
);

-- Add enum type
CREATE TYPE inventory.item_status AS ENUM ('available', 'out_of_stock', 'discontinued');

-- Alter table to use enum
ALTER TABLE inventory.items ADD COLUMN status inventory.item_status DEFAULT 'available';
`,
			description: "Schemas, sequences, and custom types",
		},
		{
			name: "complex_dependencies",
			initialSchema: `
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER,
    CONSTRAINT fk_department FOREIGN KEY (department_id) REFERENCES departments(id)
);
`,
			migrationDDL: `
-- Create base view
CREATE VIEW department_employees AS
SELECT d.id as dept_id, d.name as dept_name, e.id as emp_id, e.name as emp_name
FROM departments d
LEFT JOIN employees e ON d.id = e.department_id;

-- Create dependent view
CREATE VIEW department_summary AS
SELECT dept_id, dept_name, COUNT(emp_id) as employee_count
FROM department_employees
GROUP BY dept_id, dept_name;

-- Create function that depends on table
CREATE OR REPLACE FUNCTION get_department_employees(dept_id INTEGER)
RETURNS TABLE(employee_id INTEGER, employee_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT id, name
    FROM employees
    WHERE department_id = dept_id;
END;
$$ LANGUAGE plpgsql;

-- Create another table with foreign key to existing table
CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER NOT NULL,
    lead_employee_id INTEGER,
    CONSTRAINT fk_project_dept FOREIGN KEY (department_id) REFERENCES departments(id),
    CONSTRAINT fk_project_lead FOREIGN KEY (lead_employee_id) REFERENCES employees(id)
);
`,
			description: "Complex dependencies between views, functions, and tables",
		},
		{
			name: "drop_indexes_and_constraints",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    price DECIMAL(10, 2),
    supplier_id INTEGER
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_price ON products(price);
CREATE UNIQUE INDEX idx_products_name ON products(name);

ALTER TABLE products ADD CONSTRAINT check_price_positive CHECK (price > 0);
ALTER TABLE products ADD CONSTRAINT check_name_length CHECK (length(name) >= 3);
`,
			migrationDDL: `
-- Drop indexes and constraints
DROP INDEX idx_products_category;
DROP INDEX idx_products_price;
DROP INDEX idx_products_name;
ALTER TABLE products DROP CONSTRAINT check_price_positive;
ALTER TABLE products DROP CONSTRAINT check_name_length;
`,
			description: "Drop indexes and constraints from tables",
		},
		{
			name: "drop_views_and_functions",
			initialSchema: `
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    sale_amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL
);

CREATE VIEW monthly_sales AS
SELECT 
    EXTRACT(YEAR FROM sale_date) as year,
    EXTRACT(MONTH FROM sale_date) as month,
    SUM(sale_amount) as total_sales
FROM sales
GROUP BY EXTRACT(YEAR FROM sale_date), EXTRACT(MONTH FROM sale_date);

CREATE MATERIALIZED VIEW top_products AS
SELECT 
    product_name,
    COUNT(*) as sale_count,
    SUM(sale_amount) as total_revenue
FROM sales
GROUP BY product_name
ORDER BY total_revenue DESC;

CREATE OR REPLACE FUNCTION get_monthly_total(year_param INTEGER, month_param INTEGER)
RETURNS DECIMAL AS $$
BEGIN
    RETURN (
        SELECT COALESCE(SUM(sale_amount), 0)
        FROM sales
        WHERE EXTRACT(YEAR FROM sale_date) = year_param
        AND EXTRACT(MONTH FROM sale_date) = month_param
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION calculate_discount(amount DECIMAL)
RETURNS DECIMAL AS $$
BEGIN
    RETURN amount * 0.1;
END;
$$ LANGUAGE plpgsql;
`,
			migrationDDL: `
-- Drop views and functions
DROP MATERIALIZED VIEW top_products;
DROP VIEW monthly_sales;
DROP FUNCTION get_monthly_total(INTEGER, INTEGER);
DROP FUNCTION calculate_discount(DECIMAL);
`,
			description: "Drop views and functions that depend on tables",
		},
		{
			name: "alter_table_columns",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    price DECIMAL(8, 2) NOT NULL,
    description TEXT,
    category VARCHAR(30),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_price ON products(price);
`,
			migrationDDL: `
-- Alter table operations
ALTER TABLE products 
    ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN stock_quantity INTEGER DEFAULT 0,
    ADD COLUMN weight DECIMAL(5, 2);

-- Change column types and constraints
ALTER TABLE products ALTER COLUMN name TYPE VARCHAR(100);
ALTER TABLE products ALTER COLUMN price TYPE DECIMAL(10, 2);
ALTER TABLE products ALTER COLUMN description SET NOT NULL;
ALTER TABLE products ALTER COLUMN category DROP NOT NULL;

-- Add constraints
ALTER TABLE products ADD CONSTRAINT check_price_positive CHECK (price > 0);
ALTER TABLE products ADD CONSTRAINT check_stock_non_negative CHECK (stock_quantity >= 0);

-- Add new index
CREATE INDEX idx_products_created_at ON products(created_at);
CREATE UNIQUE INDEX idx_products_name_category ON products(name, category);
`,
			description: "Alter table with column additions, type changes, and constraints",
		},
		{
			name: "drop_and_recreate_constraints",
			initialSchema: `
CREATE TABLE authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE
);

CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    author_id INTEGER NOT NULL,
    isbn VARCHAR(20) UNIQUE,
    published_year INTEGER,
    price DECIMAL(8, 2),
    CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id),
    CONSTRAINT check_year_valid CHECK (published_year >= 1000 AND published_year <= 2100),
    CONSTRAINT check_price_positive CHECK (price > 0)
);

CREATE INDEX idx_books_author ON books(author_id);
CREATE INDEX idx_books_year ON books(published_year);
`,
			migrationDDL: `
-- Drop and recreate foreign key with different options
ALTER TABLE books DROP CONSTRAINT fk_author;
ALTER TABLE books ADD CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE;

-- Drop and modify check constraints
ALTER TABLE books DROP CONSTRAINT check_year_valid;
ALTER TABLE books ADD CONSTRAINT check_year_extended CHECK (published_year >= 1000 AND published_year <= 2030);

-- Drop unique constraint and recreate as regular index
ALTER TABLE books DROP CONSTRAINT books_isbn_key;
CREATE INDEX idx_books_isbn ON books(isbn);

-- Add new constraints
ALTER TABLE books ADD CONSTRAINT check_title_length CHECK (length(title) >= 3);
`,
			description: "Drop and recreate constraints with different definitions",
		},
		{
			name: "drop_sequence_and_type",
			initialSchema: `
CREATE SCHEMA accounting;

CREATE TYPE accounting.transaction_status AS ENUM ('pending', 'completed', 'cancelled');

CREATE SEQUENCE accounting.transaction_seq START 1000;

CREATE TABLE accounting.simple_log (
    message TEXT NOT NULL,
    status accounting.transaction_status DEFAULT 'pending'
);
`,
			migrationDDL: `
-- Drop schema objects step by step
DROP TABLE accounting.simple_log;
DROP TYPE accounting.transaction_status;
DROP SEQUENCE accounting.transaction_seq;
DROP SCHEMA accounting;
`,
			description: "Drop schema with sequences and custom types",
		},
		{
			name: "mixed_operations_complex",
			initialSchema: `
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_email VARCHAR(100)
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    supplier_id INTEGER,
    price DECIMAL(10, 2),
    CONSTRAINT fk_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
);
`,
			migrationDDL: `
-- Mix of CREATE, ALTER, and DROP operations

-- 1. Create new table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT
);

-- 2. Alter existing table (add column)
ALTER TABLE products ADD COLUMN category_id INTEGER;

-- 3. Create foreign key to new table
ALTER TABLE products ADD CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id);

-- 4. Create new view
CREATE VIEW product_summary AS
SELECT 
    p.id,
    p.name as product_name,
    s.name as supplier_name,
    c.name as category_name,
    p.price
FROM products p
LEFT JOIN suppliers s ON p.supplier_id = s.id
LEFT JOIN categories c ON p.category_id = c.id;

-- 5. Drop old constraint and recreate with different action
ALTER TABLE products DROP CONSTRAINT fk_supplier;
ALTER TABLE products ADD CONSTRAINT fk_supplier_cascade FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE SET NULL;

-- 6. Add new constraints
ALTER TABLE products ADD CONSTRAINT check_price_range CHECK (price >= 0 AND price <= 10000);
ALTER TABLE suppliers ADD CONSTRAINT check_email_format CHECK (contact_email LIKE '%@%');

-- 7. Create indexes
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_price_range ON products(price) WHERE price > 100;

-- 8. Create function
CREATE OR REPLACE FUNCTION get_expensive_products(threshold DECIMAL DEFAULT 100)
RETURNS TABLE(product_name VARCHAR, supplier_name VARCHAR, price DECIMAL) AS $$
BEGIN
    RETURN QUERY
    SELECT p.name, s.name, p.price
    FROM products p
    JOIN suppliers s ON p.supplier_id = s.id
    WHERE p.price > threshold
    ORDER BY p.price DESC;
END;
$$ LANGUAGE plpgsql;
`,
			description: "Mixed operations: CREATE, ALTER, DROP with complex dependencies",
		},
		{
			name: "create_tables_with_fk",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_email ON users(email);
CREATE INDEX idx_username ON users(username);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_id ON posts(user_id);
`,
			migrationDDL: `
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS users;`,
			description: "Create tables with foreign key constraints",
		},
		{
			name: "multiple_foreign_keys",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_email ON users(email);
CREATE INDEX idx_username ON users(username);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_id ON posts(user_id);
`,
			migrationDDL: `
-- Add new column
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;

-- Create new table with multiple foreign keys
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_comment_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_post_user ON comments(post_id, user_id);

-- Add new index
CREATE INDEX idx_email_active ON users(email, is_active);

-- Add check constraint
ALTER TABLE posts ADD CONSTRAINT chk_title_length CHECK (LENGTH(title) > 0);
`,
			description: "Tables with multiple foreign key constraints",
		},
		{
			name: "drop_and_recreate_fk_constraints",
			initialSchema: `
CREATE TABLE authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100)
);

CREATE UNIQUE INDEX uk_email ON authors(email);

CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    author_id INTEGER NOT NULL,
    isbn VARCHAR(20),
    published_year INTEGER,
    price DECIMAL(8, 2),
    CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id),
    CONSTRAINT chk_year_valid CHECK (published_year >= 1000 AND published_year <= 2100),
    CONSTRAINT chk_price_positive CHECK (price > 0)
);

CREATE UNIQUE INDEX uk_isbn ON books(isbn);
CREATE INDEX idx_author ON books(author_id);
CREATE INDEX idx_year ON books(published_year);
`,
			migrationDDL: `
-- Drop and recreate foreign key with different options
ALTER TABLE books DROP CONSTRAINT fk_author;
ALTER TABLE books ADD CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE ON UPDATE CASCADE;

-- Drop and modify check constraints
ALTER TABLE books DROP CONSTRAINT chk_year_valid;
ALTER TABLE books ADD CONSTRAINT chk_year_extended CHECK (published_year >= 1000 AND published_year <= 2030);

-- Add new constraints
ALTER TABLE books ADD CONSTRAINT chk_title_length CHECK (LENGTH(title) >= 3);
`,
			description: "Drop and recreate foreign key constraints with different options",
		},
		{
			name: "self_referencing_foreign_keys",
			initialSchema: `
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    manager_id INTEGER
);

CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER,
    salary DECIMAL(10, 2),
    hire_date DATE,
    CONSTRAINT fk_dept FOREIGN KEY (department_id) REFERENCES departments(id)
);

CREATE INDEX idx_dept ON employees(department_id);

-- Add self-referencing foreign key
ALTER TABLE departments ADD CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES employees(id);
`,
			migrationDDL: `
-- Create base view
CREATE VIEW dept_employee_count AS
SELECT d.id AS dept_id, d.name AS dept_name, COUNT(e.id) AS emp_count
FROM departments d
LEFT JOIN employees e ON d.id = e.department_id
GROUP BY d.id, d.name;

-- Create dependent view
CREATE VIEW dept_summary AS
SELECT 
    dept_id,
    dept_name,
    emp_count,
    0 AS avg_salary,
    0 AS max_salary,
    0 AS min_salary
FROM dept_employee_count;

-- Create highly dependent view
CREATE VIEW dept_manager_summary AS
SELECT 
    ds.dept_id,
    ds.dept_name,
    ds.emp_count,
    ds.avg_salary,
    m.name AS manager_name
FROM dept_summary ds 
JOIN departments d ON ds.dept_id = d.id 
LEFT JOIN employees m ON d.manager_id = m.id;

-- Create function using views
CREATE OR REPLACE FUNCTION get_department_report(dept_name_pattern VARCHAR)
RETURNS TABLE(dept_id INTEGER, dept_name VARCHAR, emp_count BIGINT, avg_salary INTEGER, manager_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM dept_manager_summary
    WHERE dept_name LIKE '%' || dept_name_pattern || '%';
END;
$$ LANGUAGE plpgsql;
`,
			description: "Self-referencing foreign keys and complex view dependencies",
		},
		{
			name: "circular_foreign_key_dependencies",
			initialSchema: `
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    preferred_order_id INTEGER
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10, 2)
);
`,
			migrationDDL: `
-- Create circular foreign key dependencies
ALTER TABLE customers ADD CONSTRAINT fk_preferred_order FOREIGN KEY (preferred_order_id) REFERENCES orders(id) ON DELETE SET NULL;
ALTER TABLE orders ADD CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE;

-- Add more tables with complex relationships
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    CONSTRAINT fk_order_item FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX idx_order ON order_items(order_id);

-- Create trigger to update order total
CREATE OR REPLACE FUNCTION update_order_total()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE orders 
    SET total_amount = (
        SELECT SUM(quantity * unit_price) 
        FROM order_items 
        WHERE order_id = NEW.order_id
    )
    WHERE id = NEW.order_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_order_total
AFTER INSERT OR UPDATE ON order_items
FOR EACH ROW
EXECUTE FUNCTION update_order_total();
`,
			description: "Circular foreign key dependencies and triggers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new database for each test case
			dbName := fmt.Sprintf("test_%s", tc.name)
			_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
			require.NoError(t, err)
			defer func() {
				_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			}()

			// Connect to the test database
			testConnConfig := *connConfig
			testConnConfig.Database = dbName
			testDB := stdlib.OpenDB(testConnConfig)
			defer testDB.Close()

			// Step 1: Initialize the database schema and get schema result A
			_, err = testDB.Exec(tc.initialSchema)
			require.NoError(t, err)

			schemaA, err := getSyncMetadataForGenerateMigration(ctx, &testConnConfig, dbName)
			require.NoError(t, err)

			// Step 2: Do some migration and get schema result B
			_, err = testDB.Exec(tc.migrationDDL)
			require.NoError(t, err)

			schemaB, err := getSyncMetadataForGenerateMigration(ctx, &testConnConfig, dbName)
			require.NoError(t, err)

			// Step 3: Call generate migration to get the rollback DDL
			// Convert to model.DatabaseSchema
			dbSchemaA := model.NewDatabaseSchema(schemaA, nil, nil, storepb.Engine_POSTGRES, false)
			dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_POSTGRES, false)

			// Get diff from B to A (to generate rollback)
			diff, err := schema.GetDatabaseSchemaDiff(dbSchemaB, dbSchemaA)
			require.NoError(t, err)

			// Log the diff for debugging
			t.Logf("Test case: %s", tc.description)
			t.Logf("Schema changes: %d", len(diff.SchemaChanges))
			for _, sc := range diff.SchemaChanges {
				t.Logf("  Schema: %s, Action: %v", sc.SchemaName, sc.Action)
			}
			t.Logf("Table changes: %d", len(diff.TableChanges))
			for _, tc := range diff.TableChanges {
				t.Logf("  Table: %s.%s, Action: %v", tc.SchemaName, tc.TableName, tc.Action)
			}
			t.Logf("Sequence changes: %d", len(diff.SequenceChanges))
			for _, sc := range diff.SequenceChanges {
				t.Logf("  Sequence: %s.%s, Action: %v", sc.SchemaName, sc.SequenceName, sc.Action)
			}

			// Log schema changes for debugging
			if diff.SchemaChanges != nil {
				for _, schemaDiff := range diff.SchemaChanges {
					t.Logf("Schema diff: %s, Action: %v", schemaDiff.SchemaName, schemaDiff.Action)
					if schemaDiff.OldSchema != nil {
						t.Logf("  Old schema %s enum types: %d", schemaDiff.SchemaName, len(schemaDiff.OldSchema.EnumTypes))
						for _, enum := range schemaDiff.OldSchema.EnumTypes {
							t.Logf("    Old Enum: %s.%s", schemaDiff.SchemaName, enum.Name)
						}
					}
					if schemaDiff.NewSchema != nil {
						t.Logf("  New schema %s enum types: %d", schemaDiff.SchemaName, len(schemaDiff.NewSchema.EnumTypes))
						for _, enum := range schemaDiff.NewSchema.EnumTypes {
							t.Logf("    New Enum: %s.%s", schemaDiff.SchemaName, enum.Name)
						}
					}
				}
			}

			// Generate rollback migration
			rollbackDDL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err)

			t.Logf("Rollback DDL:\n%s", rollbackDDL)

			// Step 4: Run rollback DDL and get schema result C
			_, err = testDB.Exec(rollbackDDL)
			require.NoError(t, err)

			schemaC, err := getSyncMetadataForGenerateMigration(ctx, &testConnConfig, dbName)
			require.NoError(t, err)

			// Step 5: Compare schema result A and C to ensure they are the same
			normalizeMetadataForComparison(schemaA)
			normalizeMetadataForComparison(schemaC)

			// Use cmp with protocmp for proto message comparison
			if diff := cmp.Diff(schemaA, schemaC, protocmp.Transform()); diff != "" {
				t.Errorf("Schema mismatch after rollback (-want +got):\n%s", diff)
			}
		})
	}
}

// getSyncMetadataForGenerateMigration retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadataForGenerateMigration(ctx context.Context, connConfig *pgx.ConnConfig, dbName string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create a driver instance using the pg package
	driver := &pgdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: connConfig.User,
			Host:     connConfig.Host,
			Port:     fmt.Sprintf("%d", connConfig.Port),
			Database: dbName,
		},
		Password: connConfig.Password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0", // PostgreSQL 16
			DatabaseName:  dbName,
		},
	}

	// Open connection using the driver
	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Use SyncDBSchema to get the metadata
	pgDriver, ok := openedDriver.(*pgdb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to pg.Driver")
	}

	metadata, err := pgDriver.SyncDBSchema(ctx)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// normalizeMetadataForComparison normalizes metadata to ignore differences that don't affect schema equality
func normalizeMetadataForComparison(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it might differ
	metadata.Name = ""

	// Normalize schemas
	for _, schema := range metadata.Schemas {
		// Clear volatile fields (schemas don't have CreateTime/UpdateTime)

		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0

			// Sort columns by name for consistent comparison
			sortColumnsByName(table.Columns)

			// Sort indexes by name
			sortIndexesByName(table.Indexes)

			// Sort foreign keys by name
			sortForeignKeysByName(table.ForeignKeys)

			// Sort check constraints by name
			sortCheckConstraintsByName(table.CheckConstraints)
		}

		// Normalize views (no volatile fields to clear)

		// Normalize materialized views (no volatile fields to clear)

		// Normalize functions (no volatile fields to clear)

		// Normalize sequences (no volatile fields to clear)

		// Sort all collections for consistent comparison
		sortTablesByName(schema.Tables)
		sortViewsByName(schema.Views)
		sortMaterializedViewsByName(schema.MaterializedViews)
		sortFunctionsByName(schema.Functions)
		sortSequencesByName(schema.Sequences)
		sortEnumsByName(schema.EnumTypes)
	}

	// Sort schemas by name
	sortSchemasByName(metadata.Schemas)

	// Sort extensions by name
	sortExtensionsByName(metadata.Extensions)
}

// Sorting helper functions
func sortSchemasByName(schemas []*storepb.SchemaMetadata) {
	slices.SortFunc(schemas, func(a, b *storepb.SchemaMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortTablesByName(tables []*storepb.TableMetadata) {
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortColumnsByName(columns []*storepb.ColumnMetadata) {
	slices.SortFunc(columns, func(a, b *storepb.ColumnMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortIndexesByName(indexes []*storepb.IndexMetadata) {
	slices.SortFunc(indexes, func(a, b *storepb.IndexMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortForeignKeysByName(fks []*storepb.ForeignKeyMetadata) {
	slices.SortFunc(fks, func(a, b *storepb.ForeignKeyMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortCheckConstraintsByName(checks []*storepb.CheckConstraintMetadata) {
	slices.SortFunc(checks, func(a, b *storepb.CheckConstraintMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortViewsByName(views []*storepb.ViewMetadata) {
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortMaterializedViewsByName(mvs []*storepb.MaterializedViewMetadata) {
	slices.SortFunc(mvs, func(a, b *storepb.MaterializedViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortFunctionsByName(functions []*storepb.FunctionMetadata) {
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortSequencesByName(sequences []*storepb.SequenceMetadata) {
	slices.SortFunc(sequences, func(a, b *storepb.SequenceMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortEnumsByName(enums []*storepb.EnumTypeMetadata) {
	slices.SortFunc(enums, func(a, b *storepb.EnumTypeMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortExtensionsByName(extensions []*storepb.ExtensionMetadata) {
	slices.SortFunc(extensions, func(a, b *storepb.ExtensionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

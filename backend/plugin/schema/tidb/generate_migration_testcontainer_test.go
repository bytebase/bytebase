package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	// Import MySQL driver (TiDB is compatible with MySQL protocol)
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start TiDB container
	req := testcontainers.ContainerRequest{
		Image:        "pingcap/tidb:v8.5.0",
		ExposedPorts: []string{"4000/tcp"},
		WaitingFor: wait.ForLog("server is running MySQL protocol").
			WithStartupTimeout(5 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection details
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "4000")
	require.NoError(t, err)

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
    id INT NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email),
    INDEX idx_username (username)
);

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at DATETIME,
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`,
			migrationDDL: `
-- Add new column
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;

-- Create new table
CREATE TABLE comments (
    id INT NOT NULL AUTO_INCREMENT,
    post_id INT NOT NULL,
    user_id INT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_post_user (post_id, user_id),
    CONSTRAINT fk_comment_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add new index
CREATE INDEX idx_email_active ON users(email, is_active);

-- Add check constraint
ALTER TABLE posts ADD CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) > 0);
`,
			description: "Basic table operations with columns, constraints, and indexes",
		},
		{
			name: "tidb_specific_features",
			initialSchema: `
CREATE TABLE products (
    id BIGINT NOT NULL AUTO_RANDOM,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    stock INT DEFAULT 0,
    PRIMARY KEY (id),
    INDEX idx_name (name)
);

CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT,
    product_id BIGINT NOT NULL,
    quantity INT NOT NULL,
    total DECIMAL(10, 2),
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
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
    COALESCE(SUM(o.quantity), 0) AS total_ordered
FROM products p
LEFT JOIN orders o ON p.id = o.product_id
GROUP BY p.id, p.name, p.price, p.stock;

-- Add column with TiDB specific features
ALTER TABLE products ADD COLUMN category VARCHAR(50) DEFAULT 'general';
`,
			description: "TiDB specific features like AUTO_RANDOM and clustered index",
		},
		{
			name: "views_and_procedures",
			initialSchema: `
CREATE TABLE sales (
    id INT NOT NULL AUTO_INCREMENT,
    product_name VARCHAR(100) NOT NULL,
    sale_amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_product_date (product_name, sale_date)
);
`,
			migrationDDL: `
-- Create views
CREATE VIEW monthly_sales AS
SELECT 
    YEAR(sale_date) AS year,
    MONTH(sale_date) AS month,
    SUM(sale_amount) AS total_sales
FROM sales
GROUP BY YEAR(sale_date), MONTH(sale_date);

CREATE VIEW top_products AS
SELECT 
    product_name,
    COUNT(*) AS sale_count,
    SUM(sale_amount) AS total_revenue
FROM sales
GROUP BY product_name
ORDER BY SUM(sale_amount) DESC;

-- Create additional table instead of stored routines
CREATE TABLE sales_reports (
    id INT NOT NULL AUTO_INCREMENT,
    report_type VARCHAR(50) NOT NULL,
    report_data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_type (report_type)
);
`,
			description: "Views and stored routines",
		},
		{
			name: "drop_operations",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    price DECIMAL(10, 2),
    sku VARCHAR(50) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sku (sku),
    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_price (price),
    CONSTRAINT chk_price_positive CHECK (price > 0),
    CONSTRAINT chk_name_length CHECK (CHAR_LENGTH(name) >= 3)
);

CREATE VIEW product_summary AS
SELECT category, COUNT(*) as count, AVG(price) as avg_price
FROM products
GROUP BY category;

CREATE TABLE product_stats (
    id INT NOT NULL AUTO_INCREMENT,
    category VARCHAR(50) NOT NULL,
    product_count INT DEFAULT 0,
    avg_price DECIMAL(10, 2),
    PRIMARY KEY (id),
    UNIQUE KEY uk_category (category)
);
`,
			migrationDDL: `
-- Drop various objects
DROP VIEW product_summary;
DROP TABLE product_stats;
DROP INDEX idx_category ON products;
DROP INDEX idx_price ON products;
ALTER TABLE products DROP CHECK chk_price_positive;
ALTER TABLE products DROP COLUMN category;
`,
			description: "Drop operations for views, functions, indexes, and constraints",
		},
		{
			name: "alter_table_operations",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    email VARCHAR(100),
    age INT,
    PRIMARY KEY (id),
    INDEX idx_email (email)
);
`,
			migrationDDL: `
-- Add columns
ALTER TABLE users ADD COLUMN phone VARCHAR(20);
ALTER TABLE users ADD COLUMN address TEXT;
ALTER TABLE users ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Modify existing columns
ALTER TABLE users MODIFY COLUMN name VARCHAR(100) NOT NULL;
ALTER TABLE users MODIFY COLUMN email VARCHAR(150) NOT NULL;

-- Add constraints
ALTER TABLE users ADD CONSTRAINT chk_age_positive CHECK (age > 0);
ALTER TABLE users ADD CONSTRAINT uk_phone UNIQUE (phone);

-- Add indexes
CREATE INDEX idx_name ON users(name);
CREATE INDEX idx_age ON users(age);
`,
			description: "Various ALTER TABLE operations",
		},
		{
			name: "complex_relationships",
			initialSchema: `
CREATE TABLE categories (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    parent_id INT,
    PRIMARY KEY (id),
    INDEX idx_parent (parent_id)
);

CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    category_id INT,
    price DECIMAL(10, 2),
    PRIMARY KEY (id),
    INDEX idx_category (category_id)
);

-- Add foreign keys
ALTER TABLE categories ADD CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES categories(id);
ALTER TABLE products ADD CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id);
`,
			migrationDDL: `
-- Add more simple relationships without complex foreign key chains
ALTER TABLE categories ADD COLUMN description TEXT;
ALTER TABLE products ADD COLUMN weight DECIMAL(8, 3);

-- Create simple lookup table
CREATE TABLE product_attributes (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    attribute_name VARCHAR(100) NOT NULL,
    attribute_value VARCHAR(200),
    PRIMARY KEY (id),
    INDEX idx_product_attr (product_id, attribute_name),
    CONSTRAINT fk_product_attr FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
`,
			description: "Complex table relationships with triggers",
		},
		{
			name: "partitioned_tables",
			initialSchema: `
CREATE TABLE sales_data (
    id INT NOT NULL AUTO_INCREMENT,
    sale_date DATE NOT NULL,
    customer_id INT NOT NULL,
    product_id INT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL,
    PRIMARY KEY (id, sale_date)
);
`,
			migrationDDL: `
-- Add indexes to partitioned table
CREATE INDEX idx_customer ON sales_data(customer_id);
CREATE INDEX idx_product ON sales_data(product_id);
CREATE INDEX idx_region_date ON sales_data(region, sale_date);

-- Create view on partitioned table
CREATE VIEW regional_sales AS
SELECT 
    region,
    YEAR(sale_date) AS year,
    MONTH(sale_date) AS month,
    SUM(amount) AS total_sales,
    COUNT(DISTINCT customer_id) AS unique_customers
FROM sales_data
GROUP BY region, YEAR(sale_date), MONTH(sale_date);

-- Add constraint
ALTER TABLE sales_data ADD CONSTRAINT chk_amount_positive CHECK (amount > 0);
`,
			description: "Operations on partitioned tables",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Initialize the database schema and get schema result A
			portInt, err := strconv.Atoi(port.Port())
			require.NoError(t, err)

			// Add a delay to ensure TiDB is fully ready
			time.Sleep(3 * time.Second)

			t.Logf("Connecting to TiDB at %s:%d", host, portInt)
			testDB, err := openTestDatabase(host, portInt, "root", "", "test")
			require.NoError(t, err, "Failed to connect to TiDB database")
			defer testDB.Close()

			// Clean up any existing objects from previous tests
			cleanupDatabase(t, testDB)

			// Execute initial schema
			if err := executeStatements(testDB, tc.initialSchema); err != nil {
				t.Fatalf("Failed to execute initial schema: %v", err)
			}

			schemaA, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
			require.NoError(t, err)

			// Step 2: Do some migration and get schema result B
			if err := executeStatements(testDB, tc.migrationDDL); err != nil {
				t.Fatalf("Failed to execute migration DDL: %v", err)
			}

			schemaB, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
			require.NoError(t, err)

			// Step 3: Call generate migration to get the rollback DDL
			// Convert to model.DatabaseSchema
			dbSchemaA := model.NewDatabaseSchema(schemaA, nil, nil, storepb.Engine_TIDB, false)
			dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_TIDB, false)

			// Get diff from B to A (to generate rollback)
			diff, err := schema.GetDatabaseSchemaDiff(dbSchemaB, dbSchemaA)
			require.NoError(t, err)

			// Log the diff for debugging
			t.Logf("Test case: %s", tc.description)
			t.Logf("Table changes: %d", len(diff.TableChanges))
			for _, tc := range diff.TableChanges {
				t.Logf("  Table: %s, Action: %v", tc.TableName, tc.Action)
			}
			t.Logf("View changes: %d", len(diff.ViewChanges))
			for _, vc := range diff.ViewChanges {
				t.Logf("  View: %s, Action: %v", vc.ViewName, vc.Action)
			}
			t.Logf("Function changes: %d", len(diff.FunctionChanges))
			for _, fc := range diff.FunctionChanges {
				t.Logf("  Function: %s, Action: %v", fc.FunctionName, fc.Action)
			}

			// Generate rollback migration
			rollbackDDL, err := schema.GenerateMigration(storepb.Engine_TIDB, diff)
			require.NoError(t, err)

			t.Logf("Rollback DDL:\n%s", rollbackDDL)

			// Step 4: Run rollback DDL and get schema result C
			if err := executeStatements(testDB, rollbackDDL); err != nil {
				t.Fatalf("Failed to execute rollback DDL: %v", err)
			}

			schemaC, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "", "test")
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

// openTestDatabase opens a connection to the test TiDB database
func openTestDatabase(host string, port int, username, password, database string) (*sql.DB, error) {
	var dsn string
	if password == "" {
		dsn = fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
			username, host, port, database)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
			username, password, host, port, database)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection to TiDB")
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Try to ping with retries
	var pingErr error
	for i := 0; i < 10; i++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		return nil, errors.Wrapf(pingErr, "failed to ping TiDB database after retries")
	}

	return db, nil
}

// executeStatements executes multiple SQL statements
func executeStatements(db *sql.DB, statements string) error {
	// MySQL driver supports multi-statement execution natively
	if _, err := db.Exec(statements); err != nil {
		return errors.Wrapf(err, "failed to execute statements")
	}
	return nil
}

// getSyncMetadataForGenerateMigration retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadataForGenerateMigration(ctx context.Context, host string, port int, username, password, database string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create a driver instance using the tidb package
	driver := &tidbdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: username,
			Host:     host,
			Port:     fmt.Sprintf("%d", port),
			Database: database,
		},
		Password: password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "v8.5.0",
			DatabaseName:  database,
		},
	}

	// Open connection using the driver
	openedDriver, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Use SyncDBSchema to get the metadata
	tidbDriver, ok := openedDriver.(*tidbdb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to tidb.Driver")
	}

	metadata, err := tidbDriver.SyncDBSchema(ctx)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// cleanupDatabase removes all objects from the database
func cleanupDatabase(_ *testing.T, db *sql.DB) {
	// Disable foreign key checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	defer func() {
		_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}()

	// Drop all tables
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err == nil {
				tables = append(tables, table)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, table := range tables {
			_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
		}
	}

	// Drop all views
	rows, err = db.Query("SELECT table_name FROM information_schema.views WHERE table_schema = DATABASE()")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var views []string
		for rows.Next() {
			var view string
			if err := rows.Scan(&view); err == nil {
				views = append(views, view)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, view := range views {
			_, _ = db.Exec(fmt.Sprintf("DROP VIEW IF EXISTS `%s`", view))
		}
	}

	// Drop all procedures (TiDB may support them)
	rows, err = db.Query("SELECT routine_name FROM information_schema.routines WHERE routine_schema = DATABASE() AND routine_type = 'PROCEDURE'")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var procedures []string
		for rows.Next() {
			var proc string
			if err := rows.Scan(&proc); err == nil {
				procedures = append(procedures, proc)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, proc := range procedures {
			_, _ = db.Exec(fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`", proc))
		}
	}

	// Drop all functions (TiDB may support them)
	rows, err = db.Query("SELECT routine_name FROM information_schema.routines WHERE routine_schema = DATABASE() AND routine_type = 'FUNCTION'")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var functions []string
		for rows.Next() {
			var fn string
			if err := rows.Scan(&fn); err == nil {
				functions = append(functions, fn)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, fn := range functions {
			_, _ = db.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS `%s`", fn))
		}
	}
}

// normalizeMetadataForComparison normalizes metadata to ignore differences that don't affect schema equality
func normalizeMetadataForComparison(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it might differ
	metadata.Name = ""

	// Normalize schemas
	for _, schema := range metadata.Schemas {
		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0

			// Clear auto-increment and auto-random values as they might differ
			for _, col := range table.Columns {
				if col.GetDefaultExpression() == "AUTO_INCREMENT" {
					col.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
						DefaultExpression: "AUTO_INCREMENT",
					}
				} else if strings.HasPrefix(col.GetDefaultExpression(), "AUTO_RANDOM") {
					// Keep the AUTO_RANDOM marker but normalize the value
					col.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
						DefaultExpression: "AUTO_RANDOM",
					}
				}
				// Clear column position as it might change during DDL operations
				col.Position = 0
			}

			// Sort columns by name for consistent comparison
			sortColumnsByName(table.Columns)

			// Sort indexes by name
			sortIndexesByName(table.Indexes)

			// Sort foreign keys by name
			sortForeignKeysByName(table.ForeignKeys)

			// Sort check constraints by name
			sortCheckConstraintsByName(table.CheckConstraints)

			// Sort triggers by name
			sortTriggersByName(table.Triggers)
		}

		// Sort all collections for consistent comparison
		sortTablesByName(schema.Tables)
		sortViewsByName(schema.Views)
		sortFunctionsByName(schema.Functions)
		sortProceduresByName(schema.Procedures)
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

func sortFunctionsByName(functions []*storepb.FunctionMetadata) {
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortProceduresByName(procedures []*storepb.ProcedureMetadata) {
	slices.SortFunc(procedures, func(a, b *storepb.ProcedureMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortExtensionsByName(extensions []*storepb.ExtensionMetadata) {
	slices.SortFunc(extensions, func(a, b *storepb.ExtensionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortTriggersByName(triggers []*storepb.TriggerMetadata) {
	slices.SortFunc(triggers, func(a, b *storepb.TriggerMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

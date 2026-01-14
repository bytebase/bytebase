package mysql

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	// Import MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}
	type testCase struct {
		description  string
		databaseName string
		originalDDL  string
	}

	testCases := []testCase{
		{
			description:  "Basic tables with various column types",
			databaseName: "test_basic",
			originalDDL: `
CREATE TABLE users (
	id INT PRIMARY KEY AUTO_INCREMENT,
	username VARCHAR(50) NOT NULL UNIQUE,
	email VARCHAR(100) NOT NULL,
	age INT CHECK (age >= 18),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	profile JSON,
	is_active BOOLEAN DEFAULT TRUE,
	INDEX idx_email (email),
	INDEX idx_created_at (created_at)
);

CREATE TABLE posts (
	id INT PRIMARY KEY AUTO_INCREMENT,
	user_id INT NOT NULL,
	title VARCHAR(200) NOT NULL,
	content TEXT,
	published_at DATETIME,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	INDEX idx_user_published (user_id, published_at)
);
`,
		},
		{
			description:  "Generated columns and complex indexes",
			databaseName: "test_generated",
			originalDDL: `
CREATE TABLE products (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	price DECIMAL(10, 2) NOT NULL,
	tax_rate DECIMAL(5, 2) NOT NULL DEFAULT 0.08,
	price_with_tax DECIMAL(10, 2) AS (price * (1 + tax_rate)) STORED,
	description TEXT,
	tags JSON,
	FULLTEXT idx_fulltext (name, description)
);

CREATE TABLE inventory (
	id INT PRIMARY KEY AUTO_INCREMENT,
	product_id INT NOT NULL,
	warehouse VARCHAR(50) NOT NULL,
	quantity INT NOT NULL DEFAULT 0,
	last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	UNIQUE KEY uk_product_warehouse (product_id, warehouse),
	FOREIGN KEY (product_id) REFERENCES products(id)
);
`,
		},
		{
			description:  "Views and triggers",
			databaseName: "test_views_triggers",
			originalDDL: `
CREATE TABLE orders (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_number VARCHAR(20) NOT NULL UNIQUE,
	customer_name VARCHAR(100) NOT NULL,
	total_amount DECIMAL(10, 2) NOT NULL,
	status VARCHAR(20) DEFAULT 'pending',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_history (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_id INT NOT NULL,
	old_status VARCHAR(20),
	new_status VARCHAR(20),
	changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (order_id) REFERENCES orders(id)
);

CREATE VIEW pending_orders AS
SELECT id, order_number, customer_name, total_amount, created_at
FROM orders
WHERE status = 'pending'
ORDER BY created_at DESC;

CREATE TRIGGER order_status_change
AFTER UPDATE ON orders
FOR EACH ROW
BEGIN
	IF OLD.status != NEW.status THEN
		INSERT INTO order_history (order_id, old_status, new_status)
		VALUES (NEW.id, OLD.status, NEW.status);
	END IF;
END;
`,
		},
		{
			description:  "Stored procedures and functions",
			databaseName: "test_routines",
			originalDDL: `
CREATE TABLE accounts (
	id INT PRIMARY KEY AUTO_INCREMENT,
	account_number VARCHAR(20) NOT NULL UNIQUE,
	balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

DELIMITER $$

CREATE FUNCTION calculate_interest(principal DECIMAL(15, 2), rate DECIMAL(5, 4), years INT)
RETURNS DECIMAL(15, 2)
DETERMINISTIC
READS SQL DATA
BEGIN
	RETURN principal * POW(1 + rate, years);
END$$

CREATE PROCEDURE transfer_funds(
	IN from_account VARCHAR(20),
	IN to_account VARCHAR(20),
	IN amount DECIMAL(15, 2)
)
BEGIN
	DECLARE from_balance DECIMAL(15, 2);
	
	START TRANSACTION;
	
	SELECT balance INTO from_balance
	FROM accounts
	WHERE account_number = from_account
	FOR UPDATE;
	
	IF from_balance >= amount THEN
		UPDATE accounts
		SET balance = balance - amount
		WHERE account_number = from_account;
		
		UPDATE accounts
		SET balance = balance + amount
		WHERE account_number = to_account;
		
		COMMIT;
	ELSE
		ROLLBACK;
		SIGNAL SQLSTATE '45000'
		SET MESSAGE_TEXT = 'Insufficient funds';
	END IF;
END$$

DELIMITER ;
`,
		},
		{
			description:  "Partitioned tables",
			databaseName: "test_partitions",
			originalDDL: `
-- RANGE partition
CREATE TABLE sales (
	id INT NOT NULL AUTO_INCREMENT,
	sale_date DATE NOT NULL,
	product_id INT NOT NULL,
	quantity INT NOT NULL,
	amount DECIMAL(10, 2) NOT NULL,
	PRIMARY KEY (id, sale_date)
) PARTITION BY RANGE (YEAR(sale_date)) (
	PARTITION p2022 VALUES LESS THAN (2023),
	PARTITION p2023 VALUES LESS THAN (2024),
	PARTITION p2024 VALUES LESS THAN (2025),
	PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- HASH partition
CREATE TABLE employees (
	id INT NOT NULL,
	name VARCHAR(100) NOT NULL,
	department_id INT NOT NULL,
	hired_date DATE,
	PRIMARY KEY (id)
) PARTITION BY HASH(id) PARTITIONS 4;

-- LIST partition
CREATE TABLE customer_regions (
	id INT NOT NULL AUTO_INCREMENT,
	customer_name VARCHAR(100) NOT NULL,
	region VARCHAR(20) NOT NULL,
	sales_amount DECIMAL(10, 2),
	PRIMARY KEY (id, region)
) PARTITION BY LIST COLUMNS(region) (
	PARTITION p_north VALUES IN ('north', 'northeast', 'northwest'),
	PARTITION p_south VALUES IN ('south', 'southeast', 'southwest'),
	PARTITION p_east VALUES IN ('east'),
	PARTITION p_west VALUES IN ('west'),
	PARTITION p_central VALUES IN ('central')
);

-- KEY partition
CREATE TABLE user_sessions (
	session_id VARCHAR(64) NOT NULL,
	user_id INT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (session_id)
) PARTITION BY KEY() PARTITIONS 8;

-- RANGE COLUMNS partition
CREATE TABLE order_archive (
	order_id INT NOT NULL,
	order_date DATE NOT NULL,
	customer_id INT NOT NULL,
	status VARCHAR(20) NOT NULL,
	total_amount DECIMAL(10, 2),
	PRIMARY KEY (order_id, order_date)
) PARTITION BY RANGE COLUMNS(order_date) (
	PARTITION p_2022_q1 VALUES LESS THAN ('2022-04-01'),
	PARTITION p_2022_q2 VALUES LESS THAN ('2022-07-01'),
	PARTITION p_2022_q3 VALUES LESS THAN ('2022-10-01'),
	PARTITION p_2022_q4 VALUES LESS THAN ('2023-01-01'),
	PARTITION p_2023_and_later VALUES LESS THAN (MAXVALUE)
);
`,
		},
		{
			description:  "Events",
			databaseName: "test_events",
			originalDDL: `
CREATE TABLE daily_stats (
	id INT PRIMARY KEY AUTO_INCREMENT,
	stat_date DATE NOT NULL UNIQUE,
	total_orders INT DEFAULT 0,
	total_revenue DECIMAL(15, 2) DEFAULT 0.00,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE EVENT IF NOT EXISTS update_daily_stats
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
	INSERT INTO daily_stats (stat_date, total_orders, total_revenue)
	VALUES (CURDATE() - INTERVAL 1 DAY, 0, 0.00)
	ON DUPLICATE KEY UPDATE
		total_orders = VALUES(total_orders),
		total_revenue = VALUES(total_revenue);
`,
		},
		{
			description:  "Character sets and collations",
			databaseName: "test_charset",
			originalDDL: `
CREATE TABLE translations (
	id INT PRIMARY KEY AUTO_INCREMENT,
	language_code VARCHAR(5) CHARACTER SET ascii NOT NULL,
	content_key VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
	translation TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	UNIQUE KEY uk_lang_key (language_code, content_key)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`,
		},
	}

	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Start MySQL container
			container, err := testcontainer.GetTestMySQLContainer(ctx)
			require.NoError(t, err)
			defer container.Close(ctx)

			// Create test database
			_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", tc.databaseName))
			require.NoError(t, err)

			host := container.GetHost()
			port := container.GetPort()

			// Step 1: Initialize the database schema
			metadataA, err := initializeAndSyncDatabase(ctx, host, port, "root", "root-password", tc.databaseName, tc.originalDDL)
			require.NoError(t, err)

			// Step 2: Call GetDatabaseDefinition to generate the database definition X
			defCtx := schema.GetDefinitionContext{
				SkipBackupSchema: false,
				PrintHeader:      true,
			}
			definitionX, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, defCtx, metadataA)
			require.NoError(t, err)
			require.NotEmpty(t, definitionX)

			// Step 3: Create a new database to run the database definition X
			newDBName := fmt.Sprintf("%s_recreated", tc.databaseName)
			metadataB, err := createDatabaseAndSync(ctx, host, port, "root", "root-password", tc.databaseName, newDBName, definitionX)
			require.NoError(t, err)

			// Step 4: Compare the database metadata A and B, should be the same
			normalizeMetadata(metadataA)
			normalizeMetadata(metadataB)

			opts := []cmp.Option{
				protocmp.Transform(),
				protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
			}

			// Add custom ignored fields for specific test cases (for events test)
			if tc.description == "Events" {
				// Ignore time-specific fields that vary between runs
				opts = append(opts, protocmp.IgnoreFields(&storepb.EventMetadata{}, "time_zone", "sql_mode", "character_set_client"))
			}

			if diff := cmp.Diff(metadataA, metadataB, opts...); diff != "" {
				t.Errorf("Metadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// normalizeMetadata normalizes the metadata to ignore differences that don't affect schema equivalence
func normalizeMetadata(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it will differ between original and recreated
	metadata.Name = ""

	// Normalize AUTO_INCREMENT values to 0 as they can differ
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			// Clear runtime-specific values
			table.RowCount = 0
			table.DataSize = 0
			table.IndexSize = 0
			table.DataFree = 0

			// Normalize column defaults
			for _, column := range table.Columns {
				// MySQL might represent defaults differently
				if def := column.GetDefault(); def != "" {
					// Normalize CURRENT_TIMESTAMP variations
					if def == "CURRENT_TIMESTAMP" ||
						def == "current_timestamp()" ||
						def == "now()" {
						column.Default = "CURRENT_TIMESTAMP"
					}
				}
			}

			// Remove duplicate check constraints (sometimes appear as both message and string)
			seen := make(map[string]bool)
			var uniqueChecks []*storepb.CheckConstraintMetadata
			for _, check := range table.CheckConstraints {
				key := fmt.Sprintf("%s:%s", check.Name, check.Expression)
				if !seen[key] {
					seen[key] = true
					uniqueChecks = append(uniqueChecks, check)
				}
			}
			table.CheckConstraints = uniqueChecks
		}
	}
}

// TestGetDatabaseDefinitionWithConnectedDeps tests the ability to handle complex foreign key dependencies
func TestGetDatabaseDefinitionWithConnectedDeps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	const (
		databaseName = "test_complex_deps"
		originalDDL  = `
CREATE TABLE department (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	parent_id INT,
	FOREIGN KEY (parent_id) REFERENCES department(id) ON DELETE SET NULL
);

CREATE TABLE employee (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	department_id INT,
	manager_id INT,
	FOREIGN KEY (department_id) REFERENCES department(id) ON DELETE SET NULL,
	FOREIGN KEY (manager_id) REFERENCES employee(id) ON DELETE SET NULL
);

CREATE TABLE project (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	lead_id INT NOT NULL,
	department_id INT NOT NULL,
	FOREIGN KEY (lead_id) REFERENCES employee(id),
	FOREIGN KEY (department_id) REFERENCES department(id)
);

CREATE TABLE project_member (
	project_id INT NOT NULL,
	employee_id INT NOT NULL,
	role VARCHAR(50),
	PRIMARY KEY (project_id, employee_id),
	FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
	FOREIGN KEY (employee_id) REFERENCES employee(id) ON DELETE CASCADE
);
`
	)

	ctx := context.Background()

	// Start MySQL container
	container, err := testcontainer.GetTestMySQLContainer(ctx)
	require.NoError(t, err)
	defer container.Close(ctx)

	// Create test database
	_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", databaseName))
	require.NoError(t, err)

	host := container.GetHost()
	port := container.GetPort()

	// Step 1: Initialize the database schema
	metadataA, err := initializeAndSyncDatabase(ctx, host, port, "root", "root-password", databaseName, originalDDL)
	require.NoError(t, err)

	// Step 2: Generate definition
	defCtx := schema.GetDefinitionContext{
		SkipBackupSchema: false,
		PrintHeader:      true,
	}
	definitionX, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, defCtx, metadataA)
	require.NoError(t, err)

	// Step 3: Create new database and apply definition
	newDBName := fmt.Sprintf("%s_recreated", databaseName)
	metadataB, err := createDatabaseAndSync(ctx, host, port, "root", "root-password", databaseName, newDBName, definitionX)
	require.NoError(t, err)

	// Compare
	normalizeMetadata(metadataA)
	normalizeMetadata(metadataB)

	opts := []cmp.Option{
		protocmp.Transform(),
		protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
	}

	if diff := cmp.Diff(metadataA, metadataB, opts...); diff != "" {
		t.Errorf("Metadata mismatch (-want +got):\n%s", diff)
	}
}

// initializeAndSyncDatabase creates a database connection, executes DDL, and returns the metadata
func initializeAndSyncDatabase(ctx context.Context, host, port, username, password, database, ddl string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create driver instance
	driver := &mysqldb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: username,
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.0",
			DatabaseName:  database,
		},
	}

	// Open connection
	openedDriver, err := driver.Open(ctx, storepb.Engine_MYSQL, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Execute DDL
	_, err = openedDriver.Execute(ctx, ddl, db.ExecuteOptions{})
	if err != nil {
		return nil, err
	}

	// Wait for schema to be created
	time.Sleep(500 * time.Millisecond)

	// Sync metadata
	mysqlDriver, ok := openedDriver.(*mysqldb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to mysql.Driver")
	}

	return mysqlDriver.SyncDBSchema(ctx)
}

// createDatabaseAndSync creates a new database, applies DDL, and returns the metadata
func createDatabaseAndSync(ctx context.Context, host, port, username, password, sourceDB, targetDB, ddl string) (*storepb.DatabaseSchemaMetadata, error) {
	// First create the new database
	driver := &mysqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: username,
			Host:     host,
			Port:     port,
			Database: sourceDB,
		},
		Password: password,
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.0",
			DatabaseName:  sourceDB,
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_MYSQL, config)
	if err != nil {
		return nil, err
	}

	_, err = openedDriver.Execute(ctx, fmt.Sprintf("CREATE DATABASE `%s`", targetDB), db.ExecuteOptions{})
	openedDriver.Close(ctx)
	if err != nil {
		return nil, err
	}

	// Now connect to the new database and apply DDL
	return initializeAndSyncDatabase(ctx, host, port, username, password, targetDB, ddl)
}

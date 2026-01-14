package tidb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	// Import MySQL driver (TiDB is compatible with MySQL protocol)
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}

	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	defer container.Close(ctx)

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
			description:  "Views",
			databaseName: "test_views",
			originalDDL: `
CREATE TABLE orders (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_number VARCHAR(20) NOT NULL UNIQUE,
	customer_name VARCHAR(100) NOT NULL,
	total_amount DECIMAL(10, 2) NOT NULL,
	status VARCHAR(20) DEFAULT 'pending',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_items (
	id INT PRIMARY KEY AUTO_INCREMENT,
	order_id INT NOT NULL,
	product_name VARCHAR(100) NOT NULL,
	quantity INT NOT NULL,
	unit_price DECIMAL(10, 2) NOT NULL,
	FOREIGN KEY (order_id) REFERENCES orders(id)
);

CREATE VIEW pending_orders AS
SELECT id, order_number, customer_name, total_amount, created_at
FROM orders
WHERE status = 'pending'
ORDER BY created_at DESC;

CREATE VIEW order_summary AS
SELECT 
	o.id,
	o.order_number,
	o.customer_name,
	o.total_amount,
	COUNT(oi.id) AS item_count,
	SUM(oi.quantity) AS total_quantity
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
GROUP BY o.id, o.order_number, o.customer_name, o.total_amount;
`,
		},
		{
			description:  "TiDB specific features - AUTO_RANDOM",
			databaseName: "test_tidb_features",
			originalDDL: `
CREATE TABLE distributed_data (
	id BIGINT AUTO_RANDOM(5) PRIMARY KEY,
	data_type VARCHAR(50) NOT NULL,
	payload JSON,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_type_created (data_type, created_at)
);

CREATE TABLE users_autorand (
	id BIGINT AUTO_RANDOM PRIMARY KEY,
	username VARCHAR(100) NOT NULL UNIQUE,
	email VARCHAR(100) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`,
		},
		{
			description:  "Character sets and collations",
			databaseName: "test_charset",
			originalDDL: `
CREATE TABLE multilingual (
	id INT PRIMARY KEY AUTO_INCREMENT,
	content_utf8 VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
	content_bin VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
	content_unicode VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE charset_test (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) CHARACTER SET latin1,
	description TEXT CHARACTER SET utf8mb4
) DEFAULT CHARSET=utf8mb4;
`,
		},
		{
			description:  "TiDB clustered and non-clustered indexes",
			databaseName: "test_tidb_indexes",
			originalDDL: `
CREATE TABLE clustered_pk (
	id BIGINT PRIMARY KEY /*T![clustered_index] CLUSTERED */,
	name VARCHAR(100) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_name (name)
);

CREATE TABLE nonclustered_pk (
	id BIGINT PRIMARY KEY /*T![clustered_index] NONCLUSTERED */,
	code VARCHAR(50) NOT NULL UNIQUE,
	data JSON,
	INDEX idx_code (code)
);
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create database
			_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", tc.databaseName))
			require.NoError(t, err)

			// Execute original DDL
			_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("USE %s", tc.databaseName))
			require.NoError(t, err)
			err = executeMultiStatements(ctx, container.GetDB(), tc.originalDDL)
			require.NoError(t, err)

			// Get the original database metadata using SyncDBSchema
			originalMetadata, err := getMetadata(ctx, container.GetHost(), container.GetPort(), tc.databaseName)
			require.NoError(t, err)

			// Generate the database definition
			definition, err := generateDatabaseDefinition(ctx, container.GetHost(), container.GetPort(), tc.databaseName)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			// Create a new database and apply the generated definition
			newDatabaseName := fmt.Sprintf("%s_new", tc.databaseName)
			_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", newDatabaseName))
			require.NoError(t, err)

			// Apply the generated definition to the new database
			_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("USE %s", newDatabaseName))
			require.NoError(t, err)
			err = executeMultiStatements(ctx, container.GetDB(), definition)
			require.NoError(t, err)

			// Get the new database metadata
			newMetadata, err := getMetadata(ctx, container.GetHost(), container.GetPort(), newDatabaseName)
			require.NoError(t, err)

			// Compare the metadata
			// Ignore database name differences when comparing
			originalMetadata.Name = ""
			newMetadata.Name = ""
			diff := cmp.Diff(originalMetadata, newMetadata, protocmp.Transform())
			require.Empty(t, diff, "Database metadata should be identical")
		})
	}
}

// TestGetDatabaseDefinitionWithConnectedDeps tests the database definition generation
// with foreign key dependencies between tables
func TestGetDatabaseDefinitionWithConnectedDeps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}

	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	defer container.Close(ctx)

	databaseName := "test_deps"
	originalDDL := `
-- Create tables with complex foreign key dependencies
CREATE TABLE departments (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL UNIQUE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE employees (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	department_id INT NOT NULL,
	manager_id INT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE RESTRICT,
	FOREIGN KEY (manager_id) REFERENCES employees(id) ON DELETE SET NULL,
	INDEX idx_department (department_id),
	INDEX idx_manager (manager_id)
);

CREATE TABLE projects (
	id INT PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	department_id INT NOT NULL,
	lead_id INT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE,
	FOREIGN KEY (lead_id) REFERENCES employees(id) ON DELETE RESTRICT,
	INDEX idx_department (department_id),
	INDEX idx_lead (lead_id)
);

CREATE TABLE project_members (
	project_id INT NOT NULL,
	employee_id INT NOT NULL,
	role VARCHAR(50) NOT NULL,
	joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (project_id, employee_id),
	FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
	FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE
);
`

	// Create database
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", databaseName))
	require.NoError(t, err)

	// Execute original DDL
	_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("USE %s", databaseName))
	require.NoError(t, err)
	err = executeMultiStatements(ctx, container.GetDB(), originalDDL)
	require.NoError(t, err)

	// Get the original database metadata
	originalMetadata, err := getMetadata(ctx, container.GetHost(), container.GetPort(), databaseName)
	require.NoError(t, err)

	// Generate the database definition
	definition, err := generateDatabaseDefinition(ctx, container.GetHost(), container.GetPort(), databaseName)
	require.NoError(t, err)
	require.NotEmpty(t, definition)

	// Create a new database and apply the generated definition
	newDatabaseName := fmt.Sprintf("%s_new", databaseName)
	_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", newDatabaseName))
	require.NoError(t, err)

	// Apply the generated definition to the new database
	_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("USE %s", newDatabaseName))
	require.NoError(t, err)
	err = executeMultiStatements(ctx, container.GetDB(), definition)
	require.NoError(t, err)

	// Get the new database metadata
	newMetadata, err := getMetadata(ctx, container.GetHost(), container.GetPort(), newDatabaseName)
	require.NoError(t, err)

	// Compare the metadata
	// Ignore database name differences when comparing
	originalMetadata.Name = ""
	newMetadata.Name = ""
	diff := cmp.Diff(originalMetadata, newMetadata, protocmp.Transform())
	require.Empty(t, diff, "Database metadata should be identical")
}

// executeMultiStatements executes multiple SQL statements separated by semicolons
func executeMultiStatements(ctx context.Context, db *sql.DB, statements string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, statements); err != nil {
		return errors.New("failed to execute context in a transaction: " + err.Error())
	}

	return tx.Commit()
}

// getMetadata creates a database connection and retrieves metadata
func getMetadata(ctx context.Context, host, port, database string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create driver instance
	driver := &tidbdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.5.0",
			DatabaseName:  database,
		},
	}

	// Open connection
	openedDriver, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// No DDL execution since it's not used

	// Sync metadata
	tidbDriver, ok := openedDriver.(*tidbdb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to tidb.Driver")
	}

	return tidbDriver.SyncDBSchema(ctx)
}

// generateDatabaseDefinition generates the database definition using schema.GetDatabaseDefinition
func generateDatabaseDefinition(ctx context.Context, host, port, database string) (string, error) {
	// Create driver instance
	driver := &tidbdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "8.5.0",
			DatabaseName:  database,
		},
	}

	// Open connection
	openedDriver, err := driver.Open(ctx, storepb.Engine_TIDB, config)
	if err != nil {
		return "", err
	}
	defer openedDriver.Close(ctx)

	// Sync metadata first
	tidbDriver, ok := openedDriver.(*tidbdb.Driver)
	if !ok {
		return "", errors.New("failed to cast to tidb.Driver")
	}

	metadata, err := tidbDriver.SyncDBSchema(ctx)
	if err != nil {
		return "", err
	}

	// Generate definition
	return schema.GetDatabaseDefinition(storepb.Engine_TIDB, schema.GetDefinitionContext{PrintHeader: true}, metadata)
}

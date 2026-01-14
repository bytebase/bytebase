package tidb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	// Import MySQL driver (TiDB is compatible with MySQL protocol)
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TiDB testcontainer test in short mode")
	}

	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	host := container.GetHost()
	port := container.GetPort()

	type testCase struct {
		description string
		originalDDL string
	}

	testCases := []testCase{
		{
			description: "Basic tables with various column types",
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
			description: "Generated columns and complex indexes",
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
			description: "Views",
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
			description: "TiDB specific features - AUTO_RANDOM",
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
			description: "Character sets and collations",
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
			description: "TiDB clustered and non-clustered indexes",
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
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			// Create unique test database using UUID
			testDB := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
			_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDB))
			require.NoError(t, err)

			// Get the original database metadata using SyncDBSchema
			driver, err := createTiDBDriver(ctx, host, port, testDB)
			require.NoError(t, err)
			defer driver.Close(ctx)
			// Execute original DDL
			_, err = driver.Execute(ctx, tc.originalDDL, db.ExecuteOptions{})
			require.NoError(t, err)

			originalMetadata, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Generate the database definition
			definition, err := schema.GetDatabaseDefinition(storepb.Engine_TIDB, schema.GetDefinitionContext{PrintHeader: true}, originalMetadata)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			// Create a new database and apply the generated definition
			newTestDB := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
			_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", newTestDB))
			require.NoError(t, err)

			// Get the new database metadata
			newDriver, err := createTiDBDriver(ctx, host, port, newTestDB)
			require.NoError(t, err)
			defer newDriver.Close(ctx)

			require.NoError(t, err)
			_, err = newDriver.Execute(ctx, definition, db.ExecuteOptions{})
			require.NoError(t, err)

			newMetadata, err := newDriver.SyncDBSchema(ctx)
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
	t.Cleanup(func() { container.Close(ctx) })

	host := container.GetHost()
	port := container.GetPort()

	// Create unique test database using UUID
	testDB := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

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
	_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDB))
	require.NoError(t, err)

	// Get the original database metadata
	driver, err := createTiDBDriver(ctx, host, port, testDB)
	require.NoError(t, err)
	// Execute original DDL
	_, err = driver.Execute(ctx, originalDDL, db.ExecuteOptions{})
	require.NoError(t, err)
	defer driver.Close(ctx)

	originalMetadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Generate the database definition
	definition, err := schema.GetDatabaseDefinition(storepb.Engine_TIDB, schema.GetDefinitionContext{PrintHeader: true}, originalMetadata)
	require.NoError(t, err)
	require.NotEmpty(t, definition)

	// Create a new database and apply the generated definition
	newTestDB := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
	_, err = container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE `%s`", newTestDB))
	require.NoError(t, err)

	// Get the new database metadata
	newDriver, err := createTiDBDriver(ctx, host, port, newTestDB)
	require.NoError(t, err)
	defer newDriver.Close(ctx)

	// Apply the generated definition to the new database
	_, err = newDriver.Execute(ctx, definition, db.ExecuteOptions{})
	require.NoError(t, err)

	newMetadata, err := newDriver.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Compare the metadata
	// Ignore database name differences when comparing
	originalMetadata.Name = ""
	newMetadata.Name = ""
	diff := cmp.Diff(originalMetadata, newMetadata, protocmp.Transform())
	require.Empty(t, diff, "Database metadata should be identical")
}

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	// Import MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start MySQL container
	req := testcontainers.ContainerRequest{
		Image: "mysql:8.0",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test123",
			"MYSQL_DATABASE":      "testdb",
		},
		ExposedPorts: []string{"3306/tcp"},
		WaitingFor: wait.ForLog("ready for connections").
			WithOccurrence(2).
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
	port, err := container.MappedPort(ctx, "3306")
	require.NoError(t, err)

	// Test cases with various schema changes
	testCases := []struct {
		name          string
		initialSchema string
		migrationDDL  string
		description   string
	}{
		{
			name: "create_tables_with_fk",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email),
    INDEX idx_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at DATETIME,
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			migrationDDL: `
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS users;`,
			description: "Create tables with foreign key constraints",
		},
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at DATETIME,
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add new index
CREATE INDEX idx_email_active ON users(email, is_active);

-- Add check constraint (MySQL 8.0.16+)
ALTER TABLE posts ADD CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) > 0);
`,
			description: "Basic table operations with columns, constraints, and indexes",
		},
		{
			name: "views_and_triggers",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    stock INT DEFAULT 0,
    PRIMARY KEY (id),
    INDEX idx_name (name)
) ENGINE=InnoDB;

CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    total DECIMAL(10, 2),
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB;
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

-- Create another view
CREATE VIEW low_stock_products AS
SELECT * FROM products WHERE stock < 10;

-- Create trigger
CREATE TRIGGER update_order_total
BEFORE INSERT ON orders
FOR EACH ROW
BEGIN
    DECLARE product_price DECIMAL(10, 2);
    SELECT price INTO product_price FROM products WHERE id = NEW.product_id;
    SET NEW.total = NEW.quantity * product_price;
END;

-- Create stored procedure
CREATE PROCEDURE GetProductInventory(IN product_name VARCHAR(100))
BEGIN
    SELECT * FROM product_inventory
    WHERE name LIKE CONCAT('%', product_name, '%');
END;
`,
			description: "Views, triggers, and stored procedures",
		},
		{
			name: "stored_functions",
			initialSchema: `
CREATE TABLE sales (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_product_date (product_id, sale_date)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Create stored function
CREATE FUNCTION CalculateTotalSales(p_product_id INT, p_start_date DATE, p_end_date DATE)
RETURNS DECIMAL(10, 2)
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE total DECIMAL(10, 2);
    
    SELECT COALESCE(SUM(quantity * unit_price), 0) INTO total
    FROM sales
    WHERE product_id = p_product_id
    AND sale_date BETWEEN p_start_date AND p_end_date;
    
    RETURN total;
END;

-- Create another function
CREATE FUNCTION GetTaxAmount(amount DECIMAL(10, 2), tax_rate DECIMAL(5, 2))
RETURNS DECIMAL(10, 2)
DETERMINISTIC
NO SQL
BEGIN
    RETURN amount * (tax_rate / 100);
END;

-- Create table that uses functions
CREATE TABLE sales_summary (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    month_year VARCHAR(7) NOT NULL,
    total_sales DECIMAL(10, 2),
    tax_amount DECIMAL(10, 2),
    PRIMARY KEY (id),
    UNIQUE KEY uk_product_month (product_id, month_year)
) ENGINE=InnoDB;
`,
			description: "Stored functions and dependent tables",
		},
		{
			name: "drop_indexes_and_constraints",
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
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Drop indexes and constraints
DROP INDEX idx_category ON products;
DROP INDEX idx_price ON products;
DROP INDEX uk_sku ON products;
ALTER TABLE products DROP CHECK chk_price_positive;
ALTER TABLE products DROP CHECK chk_name_length;
`,
			description: "Drop indexes and constraints from tables",
		},
		{
			name: "drop_views_and_routines",
			initialSchema: `
CREATE TABLE sales (
    id INT NOT NULL AUTO_INCREMENT,
    product_name VARCHAR(100) NOT NULL,
    sale_amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

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

CREATE FUNCTION GetMonthlySalesTotal(year INT, month INT)
RETURNS DECIMAL(10, 2)
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE total DECIMAL(10, 2);
    SELECT COALESCE(SUM(sale_amount), 0) INTO total
    FROM sales
    WHERE YEAR(sale_date) = year AND MONTH(sale_date) = month;
    RETURN total;
END;

CREATE PROCEDURE CalculateDiscount(IN amount DECIMAL(10, 2), OUT discount DECIMAL(10, 2))
BEGIN
    SET discount = amount * 0.1;
END;
`,
			migrationDDL: `
-- Drop views and routines
DROP VIEW top_products;
DROP VIEW monthly_sales;
DROP FUNCTION GetMonthlySalesTotal;
DROP PROCEDURE CalculateDiscount;
`,
			description: "Drop views and stored routines",
		},
		{
			name: "alter_table_columns",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    price DECIMAL(8, 2) NOT NULL,
    description TEXT,
    category VARCHAR(30),
    is_active TINYINT(1) DEFAULT 1,
    PRIMARY KEY (id),
    INDEX idx_category (category),
    INDEX idx_price (price)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`,
			migrationDDL: `
-- Alter table operations
ALTER TABLE products ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE products ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
ALTER TABLE products ADD COLUMN stock_quantity INT DEFAULT 0;
ALTER TABLE products ADD COLUMN weight DECIMAL(5, 2);

-- Change column types and constraints
ALTER TABLE products MODIFY COLUMN name VARCHAR(100) NOT NULL;
ALTER TABLE products MODIFY COLUMN price DECIMAL(10, 2) NOT NULL;
ALTER TABLE products MODIFY COLUMN description TEXT NOT NULL;
ALTER TABLE products MODIFY COLUMN category VARCHAR(50);

-- Add constraints
ALTER TABLE products ADD CONSTRAINT chk_price_positive CHECK (price > 0);
ALTER TABLE products ADD CONSTRAINT chk_stock_non_negative CHECK (stock_quantity >= 0);

-- Add new indexes
CREATE INDEX idx_created_at ON products(created_at);
CREATE UNIQUE INDEX uk_name_category ON products(name, category);
`,
			description: "Alter table with column additions, type changes, and constraints",
		},
		{
			name: "drop_and_recreate_constraints",
			initialSchema: `
CREATE TABLE authors (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email)
) ENGINE=InnoDB;

CREATE TABLE books (
    id INT NOT NULL AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    author_id INT NOT NULL,
    isbn VARCHAR(20),
    published_year INT,
    price DECIMAL(8, 2),
    PRIMARY KEY (id),
    UNIQUE KEY uk_isbn (isbn),
    INDEX idx_author (author_id),
    INDEX idx_year (published_year),
    CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id),
    CONSTRAINT chk_year_valid CHECK (published_year >= 1000 AND published_year <= 2100),
    CONSTRAINT chk_price_positive CHECK (price > 0)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Drop and recreate foreign key with different options
ALTER TABLE books DROP FOREIGN KEY fk_author;
ALTER TABLE books ADD CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE ON UPDATE CASCADE;

-- Drop and modify check constraints
ALTER TABLE books DROP CHECK chk_year_valid;
ALTER TABLE books ADD CONSTRAINT chk_year_extended CHECK (published_year >= 1000 AND published_year <= 2030);

-- Add new constraints
ALTER TABLE books ADD CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) >= 3);
`,
			description: "Drop and recreate constraints with different definitions",
		},
		{
			name: "fulltext_and_spatial_indexes",
			initialSchema: `
CREATE TABLE articles (
    id INT NOT NULL AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    tags VARCHAR(500),
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE locations (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    position POINT NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Create fulltext index
ALTER TABLE articles ADD FULLTEXT idx_fulltext_content (title, content);

-- Create another fulltext index
CREATE FULLTEXT INDEX idx_fulltext_tags ON articles(tags);

-- Create spatial index (InnoDB supports this from MySQL 5.7+)
CREATE SPATIAL INDEX idx_spatial_position ON locations(position);

-- Create composite indexes
CREATE INDEX idx_title_tags ON articles(title(50), tags(50));
`,
			description: "FULLTEXT and SPATIAL indexes",
		},
		{
			name: "complex_view_dependencies",
			initialSchema: `
CREATE TABLE departments (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    manager_id INT,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE employees (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    department_id INT,
    salary DECIMAL(10, 2),
    hire_date DATE,
    PRIMARY KEY (id),
    INDEX idx_dept (department_id),
    CONSTRAINT fk_dept FOREIGN KEY (department_id) REFERENCES departments(id)
) ENGINE=InnoDB;

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
FROM dept_summary ds JOIN departments d ON ds.dept_id = d.id LEFT JOIN employees m ON d.manager_id = m.id;

-- Create procedure using views
CREATE PROCEDURE GetDepartmentReport(IN dept_name VARCHAR(100))
BEGIN
    SELECT * FROM dept_manager_summary
    WHERE dept_name LIKE CONCAT('%', dept_name, '%');
END;
`,
			description: "Complex view dependencies with multiple levels",
		},
		{
			name: "circular_foreign_key_dependencies",
			initialSchema: `
CREATE TABLE customers (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    preferred_order_id INT,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT,
    customer_id INT NOT NULL,
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10, 2),
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Create circular foreign key dependencies
ALTER TABLE customers ADD CONSTRAINT fk_preferred_order FOREIGN KEY (preferred_order_id) REFERENCES orders(id) ON DELETE SET NULL;
ALTER TABLE orders ADD CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE;

-- Add more tables with complex relationships
CREATE TABLE order_items (
    id INT NOT NULL AUTO_INCREMENT,
    order_id INT NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_order (order_id),
    CONSTRAINT fk_order_item FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Create trigger to update order total
CREATE TRIGGER trg_update_order_total
AFTER INSERT ON order_items
FOR EACH ROW
BEGIN
    UPDATE orders 
    SET total_amount = (
        SELECT SUM(quantity * unit_price) 
        FROM order_items 
        WHERE order_id = NEW.order_id
    )
    WHERE id = NEW.order_id;
END;
`,
			description: "Circular foreign key dependencies and triggers",
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
) ENGINE=InnoDB
PARTITION BY RANGE (YEAR(sale_date)) (
    PARTITION p2020 VALUES LESS THAN (2021),
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p_future VALUES LESS THAN MAXVALUE
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
		{
			name: "generated_columns",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 8.0,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Add generated columns
ALTER TABLE products ADD COLUMN price_with_tax DECIMAL(10, 2) AS (price * (1 + tax_rate / 100)) STORED;
ALTER TABLE products ADD COLUMN name_upper VARCHAR(100) AS (UPPER(name)) VIRTUAL;

-- Create indexes on generated columns
CREATE INDEX idx_price_with_tax ON products(price_with_tax);
CREATE INDEX idx_name_upper ON products(name_upper);

-- Create table with generated columns
CREATE TABLE order_summary (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    total_price DECIMAL(10, 2) AS (quantity * unit_price) STORED,
    order_date DATE NOT NULL,
    order_year INT AS (YEAR(order_date)) VIRTUAL,
    order_month INT AS (MONTH(order_date)) VIRTUAL,
    PRIMARY KEY (id),
    INDEX idx_year_month (order_year, order_month)
) ENGINE=InnoDB;
`,
			description: "Generated columns (virtual and stored)",
		},
		{
			name: "json_columns_and_indexes",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Add JSON columns
ALTER TABLE users ADD COLUMN preferences JSON;
ALTER TABLE users ADD COLUMN metadata JSON;

-- Create table with JSON columns and indexes
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    details JSON NOT NULL,
    tags JSON,
    PRIMARY KEY (id),
    CONSTRAINT chk_details_not_empty CHECK (JSON_TYPE(details) IS NOT NULL)
) ENGINE=InnoDB;

-- Create indexes on JSON fields (MySQL 8.0+)
CREATE INDEX idx_product_brand ON products((CAST(details->>'$.brand' AS CHAR(50))));
CREATE INDEX idx_product_price ON products((CAST(details->>'$.price' AS DECIMAL(10,2))));

-- Create generated column from JSON
ALTER TABLE products ADD COLUMN brand VARCHAR(50) AS (details->>'$.brand') VIRTUAL;
CREATE INDEX idx_brand ON products(brand);
`,
			description: "JSON columns and functional indexes",
		},
		{
			name: "complex_triggers",
			initialSchema: `
CREATE TABLE inventory (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_product (product_id)
) ENGINE=InnoDB;

CREATE TABLE inventory_log (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    old_quantity INT,
    new_quantity INT,
    change_type VARCHAR(20),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Create multiple triggers
CREATE TRIGGER trg_inventory_insert
AFTER INSERT ON inventory
FOR EACH ROW
BEGIN
    INSERT INTO inventory_log (product_id, old_quantity, new_quantity, change_type)
    VALUES (NEW.product_id, NULL, NEW.quantity, 'INSERT');
END;

CREATE TRIGGER trg_inventory_update
AFTER UPDATE ON inventory
FOR EACH ROW
BEGIN
    IF OLD.quantity != NEW.quantity THEN
        INSERT INTO inventory_log (product_id, old_quantity, new_quantity, change_type)
        VALUES (NEW.product_id, OLD.quantity, NEW.quantity, 'UPDATE');
    END IF;
END;

CREATE TRIGGER trg_inventory_delete
BEFORE DELETE ON inventory
FOR EACH ROW
BEGIN
    INSERT INTO inventory_log (product_id, old_quantity, new_quantity, change_type)
    VALUES (OLD.product_id, OLD.quantity, NULL, 'DELETE');
END;

-- Create view using log table
CREATE VIEW inventory_changes AS
SELECT 
    product_id,
    COUNT(*) AS change_count,
    MAX(changed_at) AS last_changed
FROM inventory_log
GROUP BY product_id;
`,
			description: "Multiple triggers and audit logging",
		},
		{
			name: "character_sets_and_collations",
			initialSchema: `
CREATE TABLE messages (
    id INT NOT NULL AUTO_INCREMENT,
    content TEXT NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`,
			migrationDDL: `
-- Add columns with different character sets and collations
ALTER TABLE messages ADD COLUMN title VARCHAR(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
ALTER TABLE messages ADD COLUMN summary VARCHAR(500) CHARACTER SET latin1 COLLATE latin1_swedish_ci;

-- Create table with mixed character sets
CREATE TABLE international_content (
    id INT NOT NULL AUTO_INCREMENT,
    english_text VARCHAR(1000) CHARACTER SET latin1,
    chinese_text VARCHAR(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    emoji_text VARCHAR(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

-- Create indexes on columns with specific collations
CREATE INDEX idx_title ON messages(title);
`,
			description: "Different character sets and collations",
		},
		{
			name: "events_and_advanced_features",
			initialSchema: `
CREATE TABLE system_stats (
    id INT NOT NULL AUTO_INCREMENT,
    stat_name VARCHAR(100) NOT NULL,
    stat_value DECIMAL(10, 2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Create event (requires event scheduler to be enabled)
CREATE EVENT IF NOT EXISTS evt_cleanup_old_stats
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM system_stats 
    WHERE recorded_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
END;

-- Create stored procedure with advanced features
CREATE PROCEDURE RecordSystemStat(
    IN p_stat_name VARCHAR(100),
    IN p_stat_value DECIMAL(10, 2)
)
BEGIN
    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        RESIGNAL;
    END;
    
    START TRANSACTION;
    
    INSERT INTO system_stats (stat_name, stat_value)
    VALUES (p_stat_name, p_stat_value);
    
    COMMIT;
END;

-- Create function with error handling
CREATE FUNCTION GetAverageStat(p_stat_name VARCHAR(100))
RETURNS DECIMAL(10, 2)
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE avg_value DECIMAL(10, 2);
    
    SELECT AVG(stat_value) INTO avg_value
    FROM system_stats
    WHERE stat_name = p_stat_name
    AND recorded_at >= DATE_SUB(NOW(), INTERVAL 7 DAY);
    
    RETURN COALESCE(avg_value, 0);
END;
`,
			description: "Events and advanced stored routines",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Initialize the database schema and get schema result A
			portInt, err := strconv.Atoi(port.Port())
			require.NoError(t, err)

			// Add a small delay to ensure MySQL is fully ready
			time.Sleep(2 * time.Second)

			t.Logf("Connecting to MySQL at %s:%d", host, portInt)
			testDB, err := openTestDatabase(host, portInt, "root", "test123", "testdb")
			require.NoError(t, err, "Failed to connect to MySQL database")
			defer testDB.Close()

			// Clean up any existing objects
			cleanupDatabase(t, testDB)

			// Execute initial schema
			if strings.TrimSpace(tc.initialSchema) != "" {
				if err := executeStatements(testDB, tc.initialSchema); err != nil {
					t.Fatalf("Failed to execute initial schema: %v", err)
				}
			}

			schemaA, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "test123", "testdb")
			require.NoError(t, err)

			// Step 2: Do some migration and get schema result B
			if err := executeStatements(testDB, tc.migrationDDL); err != nil {
				t.Fatalf("Failed to execute migration DDL: %v", err)
			}

			schemaB, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "test123", "testdb")
			require.NoError(t, err)

			// Step 3: Call generate migration to get the rollback DDL
			// Convert to model.DatabaseSchema
			dbSchemaA := model.NewDatabaseSchema(schemaA, nil, nil, storepb.Engine_MYSQL, false)
			dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_MYSQL, false)

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
			rollbackDDL, err := schema.GenerateMigration(storepb.Engine_MYSQL, diff)
			require.NoError(t, err)

			t.Logf("Rollback DDL:\n%s", rollbackDDL)

			// Step 4: Run rollback DDL and get schema result C
			if err := executeStatements(testDB, rollbackDDL); err != nil {
				t.Fatalf("Failed to execute rollback DDL: %v", err)
			}

			schemaC, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "test123", "testdb")
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

// openTestDatabase opens a connection to the test MySQL database
func openTestDatabase(host string, port int, username, password, database string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
		username, password, host, port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection to MySQL")
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Try to ping with retries
	var pingErr error
	for i := 0; i < 5; i++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		return nil, errors.Wrapf(pingErr, "failed to ping MySQL database after retries")
	}

	return db, nil
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

	// Drop all procedures
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

	// Drop all functions
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

	// Drop all events
	rows, err = db.Query("SELECT event_name FROM information_schema.events WHERE event_schema = DATABASE()")
	if err == nil {
		defer func() {
			rows.Close()
		}()
		var events []string
		for rows.Next() {
			var event string
			if err := rows.Scan(&event); err == nil {
				events = append(events, event)
			}
		}
		if err := rows.Err(); err != nil {
			return
		}
		for _, event := range events {
			_, _ = db.Exec(fmt.Sprintf("DROP EVENT IF EXISTS `%s`", event))
		}
	}
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
	// Create a driver instance using the mysql package
	driver := &mysqldb.Driver{}

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
			EngineVersion: "8.0",
			DatabaseName:  database,
		},
	}

	// Open connection using the driver
	openedDriver, err := driver.Open(ctx, storepb.Engine_MYSQL, config)
	if err != nil {
		return nil, err
	}
	defer openedDriver.Close(ctx)

	// Use SyncDBSchema to get the metadata
	mysqlDriver, ok := openedDriver.(*mysqldb.Driver)
	if !ok {
		return nil, errors.New("failed to cast to mysql.Driver")
	}

	metadata, err := mysqlDriver.SyncDBSchema(ctx)
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
		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0

			// Clear auto-increment values as they might differ
			for _, col := range table.Columns {
				if col.GetDefaultExpression() == "AUTO_INCREMENT" {
					// Keep the AUTO_INCREMENT marker but clear any current value
					col.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
						DefaultExpression: "AUTO_INCREMENT",
					}
				}
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

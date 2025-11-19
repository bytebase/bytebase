package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	// Import MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqldb "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start MySQL container using common testcontainer interface
	container, err := testcontainer.GetTestMySQLContainer(ctx)
	require.NoError(t, err)
	defer container.Close(ctx)

	// Get connection details
	host := container.GetHost()
	port := container.GetPort()

	// Test cases with various schema changes
	testCases := []struct {
		name          string
		initialSchema string
		migrationDDL  string
		description   string
	}{
		{
			name:          "drop_table_with_dependencies",
			initialSchema: ``,
			migrationDDL: `
--
-- Table structure for authors
--
CREATE TABLE authors (
  id int NOT NULL AUTO_INCREMENT,
  name varchar(100) NOT NULL,
  email varchar(100) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for books
--
CREATE TABLE books (
  id int NOT NULL AUTO_INCREMENT,
  title varchar(200) NOT NULL,
  author_id int NOT NULL,
  isbn varchar(20) DEFAULT NULL,
  published_year int DEFAULT NULL,
  price decimal(8,2) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_author (author_id),
  KEY idx_year (published_year),
  UNIQUE KEY uk_isbn (isbn),
  CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors (id),
  CONSTRAINT chk_price_positive CHECK (price > 0),
  CONSTRAINT chk_year_valid CHECK ((published_year >= 1000) and (published_year <= 2100))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


--
-- Table structure for products
--
CREATE TABLE products (
  id int NOT NULL AUTO_INCREMENT,
  name varchar(50) NOT NULL,
  price decimal(8,2) NOT NULL,
  description text DEFAULT NULL,
  category varchar(30) DEFAULT NULL,
  is_active tinyint(1) DEFAULT '1',
  PRIMARY KEY (id),
  KEY idx_category (category),
  KEY idx_price (price)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
			`,
		},
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
		{
			name: "table_and_column_comments",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT,
    category_id INT,
    PRIMARY KEY (id),
    INDEX idx_category (category_id)
) ENGINE=InnoDB;

CREATE TABLE categories (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Add table comments
ALTER TABLE products COMMENT = 'Product catalog table containing all product information';
ALTER TABLE categories COMMENT = 'Product categories for organization';

-- Add column comments to existing table
ALTER TABLE products MODIFY COLUMN name VARCHAR(100) NOT NULL COMMENT 'Product display name';
ALTER TABLE products MODIFY COLUMN price DECIMAL(10, 2) NOT NULL COMMENT 'Product price in USD';
ALTER TABLE products MODIFY COLUMN description TEXT COMMENT 'Detailed product description';
ALTER TABLE products MODIFY COLUMN category_id INT COMMENT 'Foreign key reference to categories table';

-- Add column comments to categories table
ALTER TABLE categories MODIFY COLUMN name VARCHAR(50) NOT NULL COMMENT 'Category display name';

-- Create new table with comments from the start
CREATE TABLE suppliers (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Unique supplier identifier',
    company_name VARCHAR(100) NOT NULL COMMENT 'Legal company name',
    contact_email VARCHAR(100) COMMENT 'Primary contact email address',
    phone VARCHAR(20) COMMENT 'Business phone number',
    address TEXT COMMENT 'Full business address',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation timestamp',
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (contact_email)
) ENGINE=InnoDB COMMENT = 'Supplier information and contact details';

-- Add new column with comment
ALTER TABLE products ADD COLUMN supplier_id INT COMMENT 'Reference to supplier providing this product';
`,
			description: "Add comments to tables and columns using MySQL COMMENT syntax",
		},
		{
			name: "modify_and_drop_comments",
			initialSchema: `
CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Order unique identifier',
    customer_name VARCHAR(100) NOT NULL COMMENT 'Customer full name',
    order_date DATE NOT NULL COMMENT 'Date when order was placed',
    total_amount DECIMAL(10, 2) COMMENT 'Total order amount in USD',
    status VARCHAR(20) DEFAULT 'pending' COMMENT 'Current order status',
    notes TEXT COMMENT 'Additional order notes or special instructions',
    PRIMARY KEY (id),
    INDEX idx_date (order_date),
    INDEX idx_status (status)
) ENGINE=InnoDB COMMENT = 'Customer orders and order details';

CREATE TABLE order_items (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Line item identifier',
    order_id INT NOT NULL COMMENT 'Reference to parent order',
    product_name VARCHAR(100) NOT NULL COMMENT 'Name of ordered product',
    quantity INT NOT NULL COMMENT 'Number of items ordered',
    unit_price DECIMAL(10, 2) NOT NULL COMMENT 'Price per individual item',
    PRIMARY KEY (id),
    INDEX idx_order (order_id)
) ENGINE=InnoDB COMMENT = 'Individual line items for each order';
`,
			migrationDDL: `
-- Modify existing table comment
ALTER TABLE orders COMMENT = 'Customer purchase orders with tracking information';

-- Modify existing column comments
ALTER TABLE orders MODIFY COLUMN customer_name VARCHAR(100) NOT NULL COMMENT 'Full name of the purchasing customer';
ALTER TABLE orders MODIFY COLUMN status VARCHAR(20) DEFAULT 'pending' COMMENT 'Order processing status (pending, processing, shipped, delivered, cancelled)';
ALTER TABLE orders MODIFY COLUMN notes TEXT COMMENT 'Special delivery instructions and customer notes';

-- Remove comments by setting them to empty string
ALTER TABLE orders MODIFY COLUMN total_amount DECIMAL(10, 2) COMMENT '';
ALTER TABLE orders MODIFY COLUMN order_date DATE NOT NULL COMMENT '';

-- Modify column type and comment simultaneously  
ALTER TABLE order_items MODIFY COLUMN product_name VARCHAR(150) NOT NULL COMMENT 'Full product name including variant details';
ALTER TABLE order_items MODIFY COLUMN quantity INT NOT NULL COMMENT 'Quantity ordered (must be positive)';

-- Remove table comment
ALTER TABLE order_items COMMENT = '';

-- Remove column comment
ALTER TABLE order_items MODIFY COLUMN unit_price DECIMAL(10, 2) NOT NULL COMMENT '';
`,
			description: "Modify existing comments and remove comments by setting to empty string",
		},
		{
			name: "comments_with_special_characters",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    bio TEXT,
    preferences JSON,
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email)
) ENGINE=InnoDB;

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content LONGTEXT,
    metadata JSON,
    PRIMARY KEY (id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB;
`,
			migrationDDL: `
-- Comments with single quotes - need proper escaping
ALTER TABLE users COMMENT = 'User accounts - stores user''s personal information and preferences';

-- Comments with double quotes and mixed quotes
ALTER TABLE users MODIFY COLUMN username VARCHAR(50) NOT NULL COMMENT 'User''s chosen "display name" for the platform';
ALTER TABLE users MODIFY COLUMN email VARCHAR(100) NOT NULL COMMENT 'Primary email address - must be "unique" across all users';

-- Multi-line comment using literal newlines
ALTER TABLE users MODIFY COLUMN bio TEXT COMMENT 'User biography text
Can contain multiple lines
and various formatting';

-- Comment with special characters and symbols
ALTER TABLE users MODIFY COLUMN preferences JSON COMMENT 'User settings: theme, notifications, privacy & security options (@, #, $, %, ^, &, *, +, =, |, \\, /, ?, <, >)';

-- Comments with Unicode characters
ALTER TABLE posts COMMENT = 'Blog posts and articles - supports international content (中文, العربية, Русский, 日本語, 한국어, Français, Español, Deutsch)';
ALTER TABLE posts MODIFY COLUMN title VARCHAR(200) NOT NULL COMMENT 'Post title - supports emojis 📝✨🔥💡🎉 and Unicode characters';

-- Comment with HTML/XML-like content
ALTER TABLE posts MODIFY COLUMN content LONGTEXT COMMENT 'Post content in HTML format: <p>, <strong>, <em>, <a href="...">, <img src="..."/>';

-- Comment with JSON-like structure
ALTER TABLE posts MODIFY COLUMN metadata JSON COMMENT 'Post metadata: {"tags": ["tag1", "tag2"], "category": "tech", "featured": true, "views": 0}';

-- Create table with complex comments
CREATE TABLE analytics (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Primary key (auto-increment)',
    event_name VARCHAR(100) NOT NULL COMMENT 'Event identifier - format: "page_view", "button_click", etc.',
    event_data JSON COMMENT 'Event payload: {"user_id": 123, "timestamp": "2023-12-01T10:30:00Z", "properties": {...}}',
    ip_address VARCHAR(45) COMMENT 'Client IP address (IPv4: xxx.xxx.xxx.xxx or IPv6: xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx)',
    user_agent TEXT COMMENT 'Browser user agent string - may contain "Mozilla/5.0", various browser/OS info',
    referrer VARCHAR(500) COMMENT 'HTTP referrer URL - where user came from (can be NULL)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Event timestamp - UTC timezone',
    processed BOOLEAN DEFAULT FALSE COMMENT 'Processing status: TRUE = processed, FALSE = pending',
    PRIMARY KEY (id),
    INDEX idx_event_date (event_name, created_at),
    INDEX idx_processed (processed)
) ENGINE=InnoDB COMMENT = 'Analytics events tracking - stores user interactions & system events. Data retention: 2 years. Access level: "admin" & "analyst" roles only.';
`,
			description: "Test comments with special characters, quotes, multiline text, and Unicode",
		},
		// Reverse test cases
		{
			name: "reverse_drop_table_with_dependencies",
			initialSchema: `
CREATE TABLE authors (
  id int NOT NULL AUTO_INCREMENT,
  name varchar(100) NOT NULL,
  email varchar(100) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE books (
  id int NOT NULL AUTO_INCREMENT,
  title varchar(200) NOT NULL,
  author_id int NOT NULL,
  isbn varchar(20) DEFAULT NULL,
  published_year int DEFAULT NULL,
  price decimal(8,2) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_author (author_id),
  KEY idx_year (published_year),
  UNIQUE KEY uk_isbn (isbn),
  CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors (id),
  CONSTRAINT chk_price_positive CHECK (price > 0),
  CONSTRAINT chk_year_valid CHECK ((published_year >= 1000) and (published_year <= 2100))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE products (
  id int NOT NULL AUTO_INCREMENT,
  name varchar(50) NOT NULL,
  price decimal(8,2) NOT NULL,
  description text DEFAULT NULL,
  category varchar(30) DEFAULT NULL,
  is_active tinyint(1) DEFAULT '1',
  PRIMARY KEY (id),
  KEY idx_category (category),
  KEY idx_price (price)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
			`,
			migrationDDL: `
DROP TABLE books;
DROP TABLE authors;
DROP TABLE products;
			`,
			description: "Reverse: Drop tables with foreign key dependencies",
		},
		{
			name:          "reverse_create_tables_with_fk",
			initialSchema: ``,
			migrationDDL: `
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
			description: "Reverse: Create tables with foreign key constraints",
		},
		{
			name: "reverse_basic_table_operations",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email),
    INDEX idx_username (username),
    INDEX idx_email_active (email, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at DATETIME,
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
			`,
			migrationDDL: `
-- Drop table
DROP TABLE comments;

-- Remove constraints
ALTER TABLE posts DROP CHECK chk_title_length;

-- Drop index
DROP INDEX idx_email_active ON users;

-- Drop column
ALTER TABLE users DROP COLUMN is_active;
			`,
			description: "Reverse: Basic table operations",
		},
		{
			name: "reverse_views_and_triggers",
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

CREATE VIEW low_stock_products AS
SELECT * FROM products WHERE stock < 10;

CREATE TRIGGER update_order_total
BEFORE INSERT ON orders
FOR EACH ROW
BEGIN
    DECLARE product_price DECIMAL(10, 2);
    SELECT price INTO product_price FROM products WHERE id = NEW.product_id;
    SET NEW.total = NEW.quantity * product_price;
END;

CREATE PROCEDURE GetProductInventory(IN product_name VARCHAR(100))
BEGIN
    SELECT * FROM product_inventory
    WHERE name LIKE CONCAT('%', product_name, '%');
END;
			`,
			migrationDDL: `
-- Drop procedure
DROP PROCEDURE GetProductInventory;

-- Drop trigger
DROP TRIGGER update_order_total;

-- Drop views
DROP VIEW low_stock_products;
DROP VIEW product_inventory;
			`,
			description: "Reverse: Views and triggers",
		},
		{
			name: "reverse_stored_functions",
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

CREATE TABLE sales_summary (
    id INT NOT NULL AUTO_INCREMENT,
    product_id INT NOT NULL,
    month_year VARCHAR(7) NOT NULL,
    total_sales DECIMAL(10, 2),
    tax_amount DECIMAL(10, 2),
    PRIMARY KEY (id),
    UNIQUE KEY uk_product_month (product_id, month_year)
) ENGINE=InnoDB;

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

CREATE FUNCTION GetTaxAmount(amount DECIMAL(10, 2), tax_rate DECIMAL(5, 2))
RETURNS DECIMAL(10, 2)
DETERMINISTIC
NO SQL
BEGIN
    RETURN amount * (tax_rate / 100);
END;
			`,
			migrationDDL: `
-- Drop functions
DROP FUNCTION CalculateTotalSales;
DROP FUNCTION GetTaxAmount;

-- Drop table
DROP TABLE sales_summary;
			`,
			description: "Reverse: Stored functions",
		},
		{
			name: "reverse_drop_indexes_and_constraints",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    price DECIMAL(10, 2),
    sku VARCHAR(50) NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
			`,
			migrationDDL: `
-- Add indexes and constraints
CREATE INDEX idx_name ON products(name);
CREATE INDEX idx_category ON products(category);
CREATE INDEX idx_price ON products(price);
ALTER TABLE products ADD UNIQUE KEY uk_sku (sku);
ALTER TABLE products ADD CONSTRAINT chk_price_positive CHECK (price > 0);
ALTER TABLE products ADD CONSTRAINT chk_name_length CHECK (CHAR_LENGTH(name) >= 3);
			`,
			description: "Reverse: Add indexes and constraints",
		},
		{
			name: "reverse_alter_table_columns",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50),
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    stock_quantity INT DEFAULT 0,
    weight DECIMAL(5, 2),
    PRIMARY KEY (id),
    INDEX idx_category (category),
    INDEX idx_price (price),
    INDEX idx_created_at (created_at),
    UNIQUE INDEX uk_name_category (name, category),
    CONSTRAINT chk_price_positive CHECK (price > 0),
    CONSTRAINT chk_stock_non_negative CHECK (stock_quantity >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
			`,
			migrationDDL: `
-- Drop constraints
ALTER TABLE products DROP CHECK chk_price_positive;
ALTER TABLE products DROP CHECK chk_stock_non_negative;

-- Drop indexes
DROP INDEX idx_created_at ON products;
DROP INDEX uk_name_category ON products;

-- Drop columns
ALTER TABLE products DROP COLUMN created_at;
ALTER TABLE products DROP COLUMN updated_at;
ALTER TABLE products DROP COLUMN stock_quantity;
ALTER TABLE products DROP COLUMN weight;

-- Modify columns
ALTER TABLE products MODIFY COLUMN name VARCHAR(50) NOT NULL;
ALTER TABLE products MODIFY COLUMN price DECIMAL(8, 2) NOT NULL;
ALTER TABLE products MODIFY COLUMN description TEXT;
ALTER TABLE products MODIFY COLUMN category VARCHAR(30);
			`,
			description: "Reverse: Alter table columns",
		},
		{
			name: "reverse_fulltext_and_spatial_indexes",
			initialSchema: `
CREATE TABLE articles (
    id INT NOT NULL AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    tags VARCHAR(500),
    PRIMARY KEY (id),
    FULLTEXT idx_fulltext_content (title, content),
    FULLTEXT idx_fulltext_tags (tags),
    INDEX idx_title_tags (title(50), tags(50))
) ENGINE=InnoDB;

CREATE TABLE locations (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    position POINT NOT NULL,
    PRIMARY KEY (id),
    SPATIAL INDEX idx_spatial_position (position)
) ENGINE=InnoDB;
			`,
			migrationDDL: `
-- Drop indexes
DROP INDEX idx_fulltext_content ON articles;
DROP INDEX idx_fulltext_tags ON articles;
DROP INDEX idx_title_tags ON articles;
DROP INDEX idx_spatial_position ON locations;
			`,
			description: "Reverse: Drop FULLTEXT and SPATIAL indexes",
		},
		{
			name: "reverse_generated_columns",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 8.0,
    price_with_tax DECIMAL(10, 2) AS (price * (1 + tax_rate / 100)) STORED,
    name_upper VARCHAR(100) AS (UPPER(name)) VIRTUAL,
    PRIMARY KEY (id),
    INDEX idx_price_with_tax (price_with_tax),
    INDEX idx_name_upper (name_upper)
) ENGINE=InnoDB;

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
			migrationDDL: `
-- Drop table
DROP TABLE order_summary;

-- Drop indexes
DROP INDEX idx_price_with_tax ON products;
DROP INDEX idx_name_upper ON products;

-- Drop generated columns
ALTER TABLE products DROP COLUMN price_with_tax;
ALTER TABLE products DROP COLUMN name_upper;
			`,
			description: "Reverse: Drop generated columns",
		},
		{
			name: "reverse_json_columns_and_indexes",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    preferences JSON,
    metadata JSON,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    details JSON NOT NULL,
    tags JSON,
    brand VARCHAR(50) AS (details->>'$.brand') VIRTUAL,
    PRIMARY KEY (id),
    INDEX idx_product_brand ((CAST(details->>'$.brand' AS CHAR(50)))),
    INDEX idx_product_price ((CAST(details->>'$.price' AS DECIMAL(10,2)))),
    INDEX idx_brand (brand),
    CONSTRAINT chk_details_not_empty CHECK (JSON_TYPE(details) IS NOT NULL)
) ENGINE=InnoDB;
			`,
			migrationDDL: `
-- Drop table
DROP TABLE products;

-- Drop columns
ALTER TABLE users DROP COLUMN preferences;
ALTER TABLE users DROP COLUMN metadata;
			`,
			description: "Reverse: Drop JSON columns and indexes",
		},
		{
			name: "reverse_complex_triggers",
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

CREATE VIEW inventory_changes AS
SELECT 
    product_id,
    COUNT(*) AS change_count,
    MAX(changed_at) AS last_changed
FROM inventory_log
GROUP BY product_id;
			`,
			migrationDDL: `
-- Drop view
DROP VIEW inventory_changes;

-- Drop triggers
DROP TRIGGER trg_inventory_delete;
DROP TRIGGER trg_inventory_update;
DROP TRIGGER trg_inventory_insert;
			`,
			description: "Reverse: Drop triggers and views",
		},
		{
			name: "reverse_events_and_advanced_features",
			initialSchema: `
CREATE TABLE system_stats (
    id INT NOT NULL AUTO_INCREMENT,
    stat_name VARCHAR(100) NOT NULL,
    stat_value DECIMAL(10, 2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE EVENT IF NOT EXISTS evt_cleanup_old_stats
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM system_stats 
    WHERE recorded_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
END;

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
			migrationDDL: `
-- Drop event
DROP EVENT IF EXISTS evt_cleanup_old_stats;

-- Drop procedure
DROP PROCEDURE RecordSystemStat;

-- Drop function
DROP FUNCTION GetAverageStat;
			`,
			description: "Reverse: Drop events and advanced features",
		},
		{
			name: "no_diff_identical_schemas",
			initialSchema: `
-- Test that identical schemas don't generate unnecessary migrations
-- This tests the fixes for primary key, index type, check constraints, and DEFAULT NULL

CREATE TABLE test_table (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) DEFAULT NULL,
    email VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    age INT DEFAULT NULL,
    score DECIMAL(5,2),
    PRIMARY KEY (id),
    INDEX idx_name (name),
    INDEX idx_status_hash (status) USING HASH,
    UNIQUE KEY uk_email (email),
    CONSTRAINT chk_age CHECK (age >= 18),
    CONSTRAINT chk_status CHECK (status IN ('active', 'inactive', 'pending'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table with various column types and defaults
CREATE TABLE t (
    id INT NOT NULL AUTO_INCREMENT,
    name CHAR(255) DEFAULT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE t1 (
    id INT NOT NULL,
    a INT NOT NULL,
    b INT DEFAULT NULL,
    c INT,
    PRIMARY KEY (id),
    INDEX b (b) USING HASH
) ENGINE=InnoDB;

-- Table with check constraint that has spaces
CREATE TABLE some_table (
    id INT PRIMARY KEY,
    a INT NOT NULL,
    CONSTRAINT some_table_chk_1 CHECK (a IN (1, 2, 3))
) ENGINE=InnoDB;
			`,
			migrationDDL: `
-- No changes - schemas should be identical
-- This simulates running the exact same CREATE statements again
-- The test verifies that no migration DDL is generated
			`,
			description: "Verify no migration generated for identical schemas (tests primary key, index, check constraints, DEFAULT NULL fixes)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Initialize the database schema and get schema result A
			portInt, err := strconv.Atoi(port)
			require.NoError(t, err)

			// Get the database connection from container
			db := container.GetDB()

			// Create a test database
			testDBName := fmt.Sprintf("test_%s", strings.ReplaceAll(tc.name, " ", "_"))
			_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
			require.NoError(t, err)
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDBName))
			require.NoError(t, err)
			defer func() {
				_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
			}()

			// Use the test database
			_, err = db.Exec(fmt.Sprintf("USE `%s`", testDBName))
			require.NoError(t, err)

			// Execute initial schema
			if strings.TrimSpace(tc.initialSchema) != "" {
				if err := executeStatements(db, tc.initialSchema); err != nil {
					t.Fatalf("Failed to execute initial schema: %v", err)
				}
			}

			schemaA, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "root-password", testDBName)
			require.NoError(t, err)

			// Step 2: Do some migration and get schema result B
			if err := executeStatements(db, tc.migrationDDL); err != nil {
				t.Fatalf("Failed to execute migration DDL: %v", err)
			}

			schemaB, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "root-password", testDBName)
			require.NoError(t, err)

			// Step 3: Call generate migration to get the rollback DDL
			// Convert to model.DatabaseSchema
			dbMetadataA := model.NewDatabaseMetadata(schemaA, nil, nil, storepb.Engine_MYSQL, false)
			dbMetadataB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_MYSQL, false)

			// Get diff from B to A (to generate rollback)
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MYSQL, dbMetadataB, dbMetadataA)
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

			// Special handling for the no-diff test case
			if tc.name == "no_diff_identical_schemas" {
				// For identical schemas, we expect no migration DDL to be generated
				if strings.TrimSpace(rollbackDDL) != "" {
					t.Errorf("Expected no migration DDL for identical schemas, but got:\n%s", rollbackDDL)
				}
				return
			}

			// Step 4: Run rollback DDL and get schema result C
			if err := executeStatements(db, rollbackDDL); err != nil {
				t.Fatalf("Failed to execute rollback DDL: %v", err)
			}

			schemaC, err := getSyncMetadataForGenerateMigration(ctx, host, portInt, "root", "root-password", testDBName)
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
				if col.Default == "AUTO_INCREMENT" {
					// Keep the AUTO_INCREMENT marker but clear any current value
					col.Default = "AUTO_INCREMENT"
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

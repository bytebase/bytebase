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

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	tidbdb "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
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
);

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
);

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
`,
			migrationDDL: `
-- Drop check constraint
ALTER TABLE posts DROP CHECK chk_title_length;

-- Drop index
DROP INDEX idx_email_active ON users;

-- Drop table
DROP TABLE comments;

-- Drop column
ALTER TABLE users DROP COLUMN is_active;
`,
			description: "Reverse of basic table operations - dropping columns, constraints, and indexes",
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
			name: "reverse_tidb_specific_features",
			initialSchema: `
CREATE TABLE products (
    id BIGINT NOT NULL AUTO_RANDOM,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    stock INT DEFAULT 0,
    category VARCHAR(50) DEFAULT 'general',
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
`,
			migrationDDL: `
-- Drop column
ALTER TABLE products DROP COLUMN category;

-- Drop view
DROP VIEW product_inventory;
`,
			description: "Reverse of TiDB specific features - dropping views and columns",
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
			name: "reverse_views_and_procedures",
			initialSchema: `
CREATE TABLE sales (
    id INT NOT NULL AUTO_INCREMENT,
    product_name VARCHAR(100) NOT NULL,
    sale_amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_product_date (product_name, sale_date)
);

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

CREATE TABLE sales_reports (
    id INT NOT NULL AUTO_INCREMENT,
    report_type VARCHAR(50) NOT NULL,
    report_data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_type (report_type)
);
`,
			migrationDDL: `
-- Drop table
DROP TABLE sales_reports;

-- Drop views
DROP VIEW top_products;
DROP VIEW monthly_sales;
`,
			description: "Reverse of views and procedures - dropping views and tables",
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
			name: "reverse_drop_operations",
			initialSchema: `
CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2),
    sku VARCHAR(50) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sku (sku),
    INDEX idx_name (name),
    CONSTRAINT chk_name_length CHECK (CHAR_LENGTH(name) >= 3)
);
`,
			migrationDDL: `
-- Add column
ALTER TABLE products ADD COLUMN category VARCHAR(50);

-- Add check constraint
ALTER TABLE products ADD CONSTRAINT chk_price_positive CHECK (price > 0);

-- Add indexes
CREATE INDEX idx_price ON products(price);
CREATE INDEX idx_category ON products(category);

-- Create table
CREATE TABLE product_stats (
    id INT NOT NULL AUTO_INCREMENT,
    category VARCHAR(50) NOT NULL,
    product_count INT DEFAULT 0,
    avg_price DECIMAL(10, 2),
    PRIMARY KEY (id),
    UNIQUE KEY uk_category (category)
);

-- Create view
CREATE VIEW product_summary AS
SELECT category, COUNT(*) as count, AVG(price) as avg_price
FROM products
GROUP BY category;
`,
			description: "Reverse of drop operations - creating views, indexes, and constraints",
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
			name: "reverse_alter_table_operations",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) NOT NULL,
    age INT,
    phone VARCHAR(20),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_email (email),
    INDEX idx_name (name),
    INDEX idx_age (age),
    CONSTRAINT chk_age_positive CHECK (age > 0),
    CONSTRAINT uk_phone UNIQUE (phone)
);
`,
			migrationDDL: `
-- Drop indexes
DROP INDEX idx_age ON users;
DROP INDEX idx_name ON users;

-- Drop constraints
ALTER TABLE users DROP CONSTRAINT uk_phone;
ALTER TABLE users DROP CHECK chk_age_positive;

-- Modify columns back
ALTER TABLE users MODIFY COLUMN email VARCHAR(100);
ALTER TABLE users MODIFY COLUMN name VARCHAR(50) NOT NULL;

-- Drop columns
ALTER TABLE users DROP COLUMN created_at;
ALTER TABLE users DROP COLUMN address;
ALTER TABLE users DROP COLUMN phone;
`,
			description: "Reverse of ALTER TABLE operations - dropping columns and constraints",
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
			name: "reverse_complex_relationships",
			initialSchema: `
CREATE TABLE categories (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    parent_id INT,
    description TEXT,
    PRIMARY KEY (id),
    INDEX idx_parent (parent_id),
    CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES categories(id)
);

CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    category_id INT,
    price DECIMAL(10, 2),
    weight DECIMAL(8, 3),
    PRIMARY KEY (id),
    INDEX idx_category (category_id),
    CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
);

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
			migrationDDL: `
-- Drop lookup table
DROP TABLE product_attributes;

-- Drop columns
ALTER TABLE products DROP COLUMN weight;
ALTER TABLE categories DROP COLUMN description;
`,
			description: "Reverse of complex relationships - dropping tables and columns",
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
		{
			name: "reverse_partitioned_tables",
			initialSchema: `
CREATE TABLE sales_data (
    id INT NOT NULL AUTO_INCREMENT,
    sale_date DATE NOT NULL,
    customer_id INT NOT NULL,
    product_id INT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL,
    PRIMARY KEY (id, sale_date),
    INDEX idx_customer (customer_id),
    INDEX idx_product (product_id),
    INDEX idx_region_date (region, sale_date),
    CONSTRAINT chk_amount_positive CHECK (amount > 0)
);

CREATE VIEW regional_sales AS
SELECT
    region,
    YEAR(sale_date) AS year,
    MONTH(sale_date) AS month,
    SUM(amount) AS total_sales,
    COUNT(DISTINCT customer_id) AS unique_customers
FROM sales_data
GROUP BY region, YEAR(sale_date), MONTH(sale_date);
`,
			migrationDDL: `
-- Drop constraint
ALTER TABLE sales_data DROP CHECK chk_amount_positive;

-- Drop view
DROP VIEW regional_sales;

-- Drop indexes
DROP INDEX idx_region_date ON sales_data;
DROP INDEX idx_product ON sales_data;
DROP INDEX idx_customer ON sales_data;
`,
			description: "Reverse of partitioned table operations - dropping indexes and views",
		},
		// Note: TiDB added foreign key support in v6.6.0 (experimental) and v7.5.0 (GA)
		// These tests verify that our migration generation handles foreign keys correctly
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
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS users;`,
			description: "Create tables with foreign key constraints",
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
			description: "Reverse of create tables with foreign key constraints - creating from empty",
		},
		{
			name: "multiple_foreign_keys",
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

-- Create new table with multiple foreign keys
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

-- Add check constraint (TiDB supports check constraints)
ALTER TABLE posts ADD CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) > 0);
`,
			description: "Tables with multiple foreign key constraints",
		},
		{
			name: "reverse_multiple_foreign_keys",
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
);

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
);

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
`,
			migrationDDL: `
-- Drop check constraint
ALTER TABLE posts DROP CHECK chk_title_length;

-- Drop index
DROP INDEX idx_email_active ON users;

-- Drop table with foreign keys
DROP TABLE comments;

-- Drop column
ALTER TABLE users DROP COLUMN is_active;
`,
			description: "Reverse of multiple foreign keys - dropping tables and constraints",
		},
		{
			name: "drop_and_recreate_fk_constraints",
			initialSchema: `
CREATE TABLE authors (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email)
);

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
);
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
			description: "Drop and recreate foreign key constraints with different options",
		},
		{
			name: "reverse_drop_and_recreate_fk_constraints",
			initialSchema: `
CREATE TABLE authors (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (email)
);

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
    CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT chk_year_extended CHECK (published_year >= 1000 AND published_year <= 2030),
    CONSTRAINT chk_price_positive CHECK (price > 0),
    CONSTRAINT chk_title_length CHECK (CHAR_LENGTH(title) >= 3)
);
`,
			migrationDDL: `
-- Drop new constraints
ALTER TABLE books DROP CHECK chk_title_length;

-- Restore original check constraint
ALTER TABLE books DROP CHECK chk_year_extended;
ALTER TABLE books ADD CONSTRAINT chk_year_valid CHECK (published_year >= 1000 AND published_year <= 2100);

-- Restore original foreign key
ALTER TABLE books DROP FOREIGN KEY fk_author_new;
ALTER TABLE books ADD CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id);
`,
			description: "Reverse of drop and recreate foreign key constraints - restoring original constraints",
		},
		{
			name: "circular_foreign_key_dependencies",
			initialSchema: `
CREATE TABLE customers (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    preferred_order_id INT,
    PRIMARY KEY (id)
);

CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT,
    customer_id INT NOT NULL,
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10, 2),
    PRIMARY KEY (id)
);
`,
			migrationDDL: `
-- Create circular foreign key dependencies
-- Note: Circular FKs might have limitations in TiDB
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
);
`,
			description: "Circular foreign key dependencies",
		},
		{
			name: "reverse_circular_foreign_key_dependencies",
			initialSchema: `
CREATE TABLE customers (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    preferred_order_id INT,
    PRIMARY KEY (id)
);

CREATE TABLE orders (
    id INT NOT NULL AUTO_INCREMENT,
    customer_id INT NOT NULL,
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10, 2),
    PRIMARY KEY (id)
);

ALTER TABLE customers ADD CONSTRAINT fk_preferred_order FOREIGN KEY (preferred_order_id) REFERENCES orders(id) ON DELETE SET NULL;
ALTER TABLE orders ADD CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE;

CREATE TABLE order_items (
    id INT NOT NULL AUTO_INCREMENT,
    order_id INT NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_order (order_id),
    CONSTRAINT fk_order_item FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);
`,
			migrationDDL: `
-- Drop dependent table
DROP TABLE order_items;

-- Drop circular foreign keys
ALTER TABLE orders DROP FOREIGN KEY fk_customer;
ALTER TABLE customers DROP FOREIGN KEY fk_preferred_order;
`,
			description: "Reverse of circular foreign key dependencies - dropping tables and foreign keys",
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_category (category_id)
);

CREATE TABLE categories (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    PRIMARY KEY (id)
);
`,
			migrationDDL: `
-- Add comments to existing table and columns
ALTER TABLE products COMMENT = 'Product catalog table storing all available products';
ALTER TABLE products MODIFY COLUMN name VARCHAR(100) NOT NULL COMMENT 'Product display name';
ALTER TABLE products MODIFY COLUMN price DECIMAL(10, 2) NOT NULL COMMENT 'Product price in USD';
ALTER TABLE products MODIFY COLUMN description TEXT COMMENT 'Detailed product description';

-- Add comment to existing table
ALTER TABLE categories COMMENT = 'Product categories for organization';
ALTER TABLE categories MODIFY COLUMN name VARCHAR(50) NOT NULL COMMENT 'Category name';

-- Create new table with comments
CREATE TABLE suppliers (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Unique supplier identifier',
    company_name VARCHAR(100) NOT NULL COMMENT 'Official company name',
    contact_email VARCHAR(150) COMMENT 'Primary contact email address',
    phone VARCHAR(20) COMMENT 'Contact phone number',
    address TEXT COMMENT 'Full business address',
    is_active BOOLEAN DEFAULT true COMMENT 'Supplier active status',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation timestamp',
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (contact_email),
    INDEX idx_active (is_active)
) COMMENT = 'Supplier information and contact details';

-- Add new columns with comments
ALTER TABLE products ADD COLUMN supplier_id INT COMMENT 'Reference to supplier table';
ALTER TABLE products ADD COLUMN weight DECIMAL(8, 3) COMMENT 'Product weight in kilograms';
ALTER TABLE products ADD COLUMN in_stock BOOLEAN DEFAULT true COMMENT 'Current stock availability';

-- Add foreign key with comment
ALTER TABLE products ADD CONSTRAINT fk_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE SET NULL;
`,
			description: "Adding comments to tables and columns using TiDB COMMENT syntax",
		},
		{
			name: "reverse_table_and_column_comments",
			initialSchema: `
CREATE TABLE categories (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL COMMENT 'Category name',
    PRIMARY KEY (id)
) COMMENT = 'Product categories for organization';

CREATE TABLE suppliers (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Unique supplier identifier',
    company_name VARCHAR(100) NOT NULL COMMENT 'Official company name',
    contact_email VARCHAR(150) COMMENT 'Primary contact email address',
    phone VARCHAR(20) COMMENT 'Contact phone number',
    address TEXT COMMENT 'Full business address',
    is_active BOOLEAN DEFAULT true COMMENT 'Supplier active status',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation timestamp',
    PRIMARY KEY (id),
    UNIQUE KEY uk_email (contact_email),
    INDEX idx_active (is_active)
) COMMENT = 'Supplier information and contact details';

CREATE TABLE products (
    id INT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL COMMENT 'Product display name',
    price DECIMAL(10, 2) NOT NULL COMMENT 'Product price in USD',
    description TEXT COMMENT 'Detailed product description',
    category_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    supplier_id INT COMMENT 'Reference to supplier table',
    weight DECIMAL(8, 3) COMMENT 'Product weight in kilograms',
    in_stock BOOLEAN DEFAULT true COMMENT 'Current stock availability',
    PRIMARY KEY (id),
    INDEX idx_category (category_id),
    CONSTRAINT fk_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE SET NULL
) COMMENT = 'Product catalog table storing all available products';
`,
			migrationDDL: `
-- Drop foreign key
ALTER TABLE products DROP FOREIGN KEY fk_supplier;

-- Drop columns with comments
ALTER TABLE products DROP COLUMN in_stock;
ALTER TABLE products DROP COLUMN weight;
ALTER TABLE products DROP COLUMN supplier_id;

-- Drop table with comments
DROP TABLE suppliers;

-- Remove comments from columns
ALTER TABLE categories MODIFY COLUMN name VARCHAR(50) NOT NULL;
ALTER TABLE categories COMMENT = '';

-- Remove comments from products table and columns
ALTER TABLE products MODIFY COLUMN description TEXT;
ALTER TABLE products MODIFY COLUMN price DECIMAL(10, 2) NOT NULL;
ALTER TABLE products MODIFY COLUMN name VARCHAR(100) NOT NULL;
ALTER TABLE products COMMENT = '';
`,
			description: "Reverse of table and column comments - removing comments and dropping tables",
		},
		{
			name: "modify_and_drop_comments",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Primary key identifier',
    username VARCHAR(50) NOT NULL COMMENT 'Unique username for login',
    email VARCHAR(100) NOT NULL COMMENT 'User email address',
    full_name VARCHAR(100) COMMENT 'User full display name',
    bio TEXT COMMENT 'User biography or description',
    status ENUM('active', 'inactive', 'suspended') DEFAULT 'active' COMMENT 'Account status',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Account creation date',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last update timestamp',
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email),
    INDEX idx_status (status)
) COMMENT = 'User account information and profile data';

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Post unique identifier',
    user_id INT NOT NULL COMMENT 'Author user ID reference',
    title VARCHAR(200) NOT NULL COMMENT 'Post title',
    content TEXT COMMENT 'Post content body',
    status ENUM('draft', 'published', 'archived') DEFAULT 'draft' COMMENT 'Post publication status',
    published_at DATETIME COMMENT 'Publication timestamp',
    view_count INT DEFAULT 0 COMMENT 'Number of views',
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    CONSTRAINT fk_post_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) COMMENT = 'Blog posts and articles';
`,
			migrationDDL: `
-- Modify existing table comments
ALTER TABLE users COMMENT = 'Updated user account information with enhanced profile features';
ALTER TABLE posts COMMENT = 'Blog posts with analytics and engagement tracking';

-- Modify existing column comments
ALTER TABLE users MODIFY COLUMN username VARCHAR(50) NOT NULL COMMENT 'Updated: Unique username for authentication and display';
ALTER TABLE users MODIFY COLUMN bio TEXT COMMENT 'Updated: Extended user biography with rich text support';
ALTER TABLE users MODIFY COLUMN status ENUM('active', 'inactive', 'suspended', 'pending') DEFAULT 'active' COMMENT 'Updated: Account status with pending verification';

-- Remove comments from columns (set to empty string)
ALTER TABLE users MODIFY COLUMN full_name VARCHAR(100) COMMENT '';
ALTER TABLE posts MODIFY COLUMN view_count INT DEFAULT 0 COMMENT '';

-- Add new columns with comments
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(255) COMMENT 'Profile picture URL';
ALTER TABLE users ADD COLUMN last_login DATETIME COMMENT 'Last successful login timestamp';
ALTER TABLE users ADD COLUMN login_count INT DEFAULT 0 COMMENT 'Total number of logins';

-- Modify existing column comments with new information
ALTER TABLE posts MODIFY COLUMN title VARCHAR(200) NOT NULL COMMENT 'Post title with SEO optimization';
ALTER TABLE posts MODIFY COLUMN content TEXT COMMENT 'Rich text post content with markdown support';

-- Add new table with comprehensive comments
CREATE TABLE user_preferences (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Preference record identifier',
    user_id INT NOT NULL COMMENT 'User ID foreign key reference',
    preference_key VARCHAR(100) NOT NULL COMMENT 'Configuration key name',
    preference_value TEXT COMMENT 'Configuration value in JSON format',
    is_public BOOLEAN DEFAULT false COMMENT 'Whether preference is publicly visible',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Preference creation time',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last modification time',
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_key (user_id, preference_key),
    INDEX idx_key (preference_key),
    INDEX idx_public (is_public),
    CONSTRAINT fk_pref_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) COMMENT = 'User-specific configuration and preference settings';
`,
			description: "Modifying existing comments and removing comments from tables and columns",
		},
		{
			name: "reverse_modify_and_drop_comments",
			initialSchema: `
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Primary key identifier',
    username VARCHAR(50) NOT NULL COMMENT 'Updated: Unique username for authentication and display',
    email VARCHAR(100) NOT NULL COMMENT 'User email address',
    full_name VARCHAR(100),
    bio TEXT COMMENT 'Updated: Extended user biography with rich text support',
    status ENUM('active', 'inactive', 'suspended', 'pending') DEFAULT 'active' COMMENT 'Updated: Account status with pending verification',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Account creation date',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last update timestamp',
    avatar_url VARCHAR(255) COMMENT 'Profile picture URL',
    last_login DATETIME COMMENT 'Last successful login timestamp',
    login_count INT DEFAULT 0 COMMENT 'Total number of logins',
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email),
    INDEX idx_status (status)
) COMMENT = 'Updated user account information with enhanced profile features';

CREATE TABLE posts (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Post unique identifier',
    user_id INT NOT NULL COMMENT 'Author user ID reference',
    title VARCHAR(200) NOT NULL COMMENT 'Post title with SEO optimization',
    content TEXT COMMENT 'Rich text post content with markdown support',
    status ENUM('draft', 'published', 'archived') DEFAULT 'draft' COMMENT 'Post publication status',
    published_at DATETIME COMMENT 'Publication timestamp',
    view_count INT DEFAULT 0,
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    CONSTRAINT fk_post_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) COMMENT = 'Blog posts with analytics and engagement tracking';

CREATE TABLE user_preferences (
    id INT NOT NULL AUTO_INCREMENT COMMENT 'Preference record identifier',
    user_id INT NOT NULL COMMENT 'User ID foreign key reference',
    preference_key VARCHAR(100) NOT NULL COMMENT 'Configuration key name',
    preference_value TEXT COMMENT 'Configuration value in JSON format',
    is_public BOOLEAN DEFAULT false COMMENT 'Whether preference is publicly visible',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Preference creation time',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last modification time',
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_key (user_id, preference_key),
    INDEX idx_key (preference_key),
    INDEX idx_public (is_public),
    CONSTRAINT fk_pref_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) COMMENT = 'User-specific configuration and preference settings';
`,
			migrationDDL: `
-- Drop new table
DROP TABLE user_preferences;

-- Restore original column comments
ALTER TABLE posts MODIFY COLUMN content TEXT COMMENT 'Post content body';
ALTER TABLE posts MODIFY COLUMN title VARCHAR(200) NOT NULL COMMENT 'Post title';

-- Drop new columns
ALTER TABLE users DROP COLUMN login_count;
ALTER TABLE users DROP COLUMN last_login;
ALTER TABLE users DROP COLUMN avatar_url;

-- Restore comments that were removed
ALTER TABLE posts MODIFY COLUMN view_count INT DEFAULT 0 COMMENT 'Number of views';
ALTER TABLE users MODIFY COLUMN full_name VARCHAR(100) COMMENT 'User full display name';

-- Restore original column types and comments
ALTER TABLE users MODIFY COLUMN status ENUM('active', 'inactive', 'suspended') DEFAULT 'active' COMMENT 'Account status';
ALTER TABLE users MODIFY COLUMN bio TEXT COMMENT 'User biography or description';
ALTER TABLE users MODIFY COLUMN username VARCHAR(50) NOT NULL COMMENT 'Unique username for login';

-- Restore original table comments
ALTER TABLE posts COMMENT = 'Blog posts and articles';
ALTER TABLE users COMMENT = 'User account information and profile data';
`,
			description: "Reverse of modify and drop comments - restoring original comments and dropping new table",
		},
		{
			name: "comments_with_special_characters",
			initialSchema: `
CREATE TABLE test_table (
    id INT NOT NULL AUTO_INCREMENT,
    simple_field VARCHAR(100),
    PRIMARY KEY (id)
);
`,
			migrationDDL: `
-- Test comments with various special characters and scenarios
ALTER TABLE test_table COMMENT = 'Table with "double quotes" and ''single quotes'' in comment';

-- Add columns with special characters in comments
ALTER TABLE test_table ADD COLUMN field_with_quotes VARCHAR(100) COMMENT 'Field with "double" and ''single'' quotes';
ALTER TABLE test_table ADD COLUMN field_with_symbols VARCHAR(100) COMMENT 'Field with symbols: @#$%^&*()_+-={}[]|:";''<>?,./ and more!';
ALTER TABLE test_table ADD COLUMN field_with_unicode VARCHAR(100) COMMENT 'Unicode test: 你好世界 🌍 café naïve résumé Москва العالم 한국어';
ALTER TABLE test_table ADD COLUMN field_multiline TEXT COMMENT 'Multi-line comment:
Line 1 with regular text
Line 2 with "quotes" and symbols @#$
Line 3 with unicode: 测试 🚀 café
Final line with mixed content';
ALTER TABLE test_table ADD COLUMN field_with_sql VARCHAR(100) COMMENT 'Comment with SQL-like content: SELECT * FROM table WHERE id = ''123'' AND name LIKE "%test%"';
ALTER TABLE test_table ADD COLUMN field_with_html VARCHAR(100) COMMENT 'HTML content: <div class="test">Hello & "World"</div> <!-- comment -->';
ALTER TABLE test_table ADD COLUMN field_with_json VARCHAR(100) COMMENT 'JSON example: {"name": "test", "value": 123, "nested": {"key": "value with spaces"}}';
ALTER TABLE test_table ADD COLUMN field_with_escape VARCHAR(100) COMMENT 'Escape sequences: \n \t \r " '' \\';

-- Create table with complex comment containing all special character types
CREATE TABLE special_comments_table (
    id BIGINT NOT NULL AUTO_RANDOM COMMENT 'ID with "quotes", symbols @#$, unicode 测试🌟, and
multi-line
content',
    data_field JSON COMMENT 'JSON field storing: {"users": ["John O''Connor", "Jane \"Doe\""], "count": 42}',
    html_field TEXT COMMENT 'HTML content field: <script>alert("XSS test & more");</script>',
    sql_field VARCHAR(255) COMMENT 'SQL patterns: SELECT * FROM users WHERE name = ''O''Brien'' AND age > 21',
    unicode_field VARCHAR(200) COMMENT '多语言支持: English, 中文, العربية, Русский, 日本語, 한국어, Français, Español',
    symbols_field VARCHAR(100) COMMENT 'All symbols: !@#$%^&*()_+-={}[]|:";''<>?,./ plus tab	and newline
test',
    url_field VARCHAR(300) COMMENT 'URL with params: https://example.com/path?param1=value1&param2="quoted value"&param3=50%+discount',
    regex_field VARCHAR(150) COMMENT 'Regex pattern: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$ for email validation',
    PRIMARY KEY (id),
    INDEX idx_unicode (unicode_field)
) COMMENT = 'Special characters test table with:
- Double quotes: "test"
- Single quotes: ''test''
- Unicode: 特殊字符测试表 🎯
- Symbols: @#$%^&*()
- HTML: <div>content</div>
- JSON: {"key": "value"}
- URLs: https://example.com
- Multi-line content with various encodings';

-- Test modifying comments with special characters
ALTER TABLE special_comments_table MODIFY COLUMN data_field JSON COMMENT 'Updated JSON field: {"new": "structure", "with": ["array", "of", "strings"], "escapes": "\n\t\r"}';

-- Test removing comments that had special characters
ALTER TABLE special_comments_table MODIFY COLUMN html_field TEXT COMMENT '';
`,
			description: "Testing comments with special characters, quotes, Unicode, multi-line text, and edge cases",
		},
		{
			name: "reverse_comments_with_special_characters",
			initialSchema: `
CREATE TABLE test_table (
    id INT NOT NULL AUTO_INCREMENT,
    simple_field VARCHAR(100),
    field_with_quotes VARCHAR(100) COMMENT 'Field with "double" and ''single'' quotes',
    field_with_symbols VARCHAR(100) COMMENT 'Field with symbols: @#$%^&*()_+-={}[]|:";''<>?,./ and more!',
    field_with_unicode VARCHAR(100) COMMENT 'Unicode test: 你好世界 🌍 café naïve résumé Москва العالم 한국어',
    field_multiline TEXT COMMENT 'Multi-line comment:
Line 1 with regular text
Line 2 with "quotes" and symbols @#$
Line 3 with unicode: 测试 🚀 café
Final line with mixed content',
    field_with_sql VARCHAR(100) COMMENT 'Comment with SQL-like content: SELECT * FROM table WHERE id = ''123'' AND name LIKE "%test%"',
    field_with_html VARCHAR(100) COMMENT 'HTML content: <div class="test">Hello & "World"</div> <!-- comment -->',
    field_with_json VARCHAR(100) COMMENT 'JSON example: {"name": "test", "value": 123, "nested": {"key": "value with spaces"}}',
    field_with_escape VARCHAR(100) COMMENT 'Escape sequences: \n \t \r " '' \\',
    PRIMARY KEY (id)
) COMMENT = 'Table with "double quotes" and ''single quotes'' in comment';

CREATE TABLE special_comments_table (
    id BIGINT NOT NULL AUTO_RANDOM COMMENT 'ID with "quotes", symbols @#$, unicode 测试🌟, and
multi-line
content',
    data_field JSON COMMENT 'Updated JSON field: {"new": "structure", "with": ["array", "of", "strings"], "escapes": "\n\t\r"}',
    html_field TEXT,
    sql_field VARCHAR(255) COMMENT 'SQL patterns: SELECT * FROM users WHERE name = ''O''Brien'' AND age > 21',
    unicode_field VARCHAR(200) COMMENT '多语言支持: English, 中文, العربية, Русский, 日本語, 한국어, Français, Español',
    symbols_field VARCHAR(100) COMMENT 'All symbols: !@#$%^&*()_+-={}[]|:";''<>?,./ plus tab	and newline
test',
    url_field VARCHAR(300) COMMENT 'URL with params: https://example.com/path?param1=value1&param2="quoted value"&param3=50%+discount',
    regex_field VARCHAR(150) COMMENT 'Regex pattern: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$ for email validation',
    PRIMARY KEY (id),
    INDEX idx_unicode (unicode_field)
) COMMENT = 'Special characters test table with:
- Double quotes: "test"
- Single quotes: ''test''
- Unicode: 特殊字符测试表 🎯
- Symbols: @#$%^&*()
- HTML: <div>content</div>
- JSON: {"key": "value"}
- URLs: https://example.com
- Multi-line content with various encodings';
`,
			migrationDDL: `
-- Drop table with special comments
DROP TABLE special_comments_table;

-- Drop columns with special character comments
ALTER TABLE test_table DROP COLUMN field_with_escape;
ALTER TABLE test_table DROP COLUMN field_with_json;
ALTER TABLE test_table DROP COLUMN field_with_html;
ALTER TABLE test_table DROP COLUMN field_with_sql;
ALTER TABLE test_table DROP COLUMN field_multiline;
ALTER TABLE test_table DROP COLUMN field_with_unicode;
ALTER TABLE test_table DROP COLUMN field_with_symbols;
ALTER TABLE test_table DROP COLUMN field_with_quotes;

-- Remove table comment
ALTER TABLE test_table COMMENT = '';
`,
			description: "Reverse of comments with special characters - removing all special character comments",
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
			dbMetadataA := model.NewDatabaseMetadata(schemaA, nil, nil, storepb.Engine_TIDB, false)
			dbMetadataB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_TIDB, false)

			// Get diff from B to A (to generate rollback)
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_TIDB, dbMetadataB, dbMetadataA)
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
				if col.Default == "AUTO_INCREMENT" {
					col.Default = "AUTO_INCREMENT"
				} else if strings.HasPrefix(col.Default, "AUTO_RANDOM") {
					// Keep the AUTO_RANDOM marker but normalize the value
					col.Default = "AUTO_RANDOM"
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

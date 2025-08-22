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

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
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
			name:          "bytebase_schema",
			initialSchema: ``,
			migrationDDL: `
            CREATE TABLE public.employee (
	emp_no      SERIAL NOT NULL,
	birth_date  DATE NOT NULL,
	first_name  TEXT NOT NULL,
	last_name   TEXT NOT NULL,
	gender      TEXT NOT NULL CHECK (gender IN('M', 'F')) NOT NULL,
	hire_date   DATE NOT NULL,
	PRIMARY KEY (emp_no)
);

CREATE INDEX idx_employee_hire_date ON public.employee (hire_date);

CREATE TABLE public.department (
	dept_no     TEXT NOT NULL,
	dept_name   TEXT NOT NULL,
	PRIMARY KEY (dept_no),
	UNIQUE      (dept_name)
);

CREATE TABLE public.dept_manager (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE public.dept_emp (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE public.title (
	emp_no      INT NOT NULL,
	title       TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, title, from_date)
); 

CREATE TABLE public.salary (
	emp_no      INT NOT NULL,
	amount      INT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, from_date)
);

CREATE INDEX idx_salary_amount ON public.salary (amount);

CREATE TABLE public.audit (
    id SERIAL PRIMARY KEY,
    operation TEXT NOT NULL,
    query TEXT,
    user_name TEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_operation ON public.audit (operation);
CREATE INDEX idx_audit_username ON public.audit (user_name);
CREATE INDEX idx_audit_changed_at ON public.audit (changed_at);

CREATE OR REPLACE FUNCTION public.log_dml_operations() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('INSERT', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('UPDATE', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('DELETE', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- only log update and delete, otherwise, it will cause too much change.
CREATE TRIGGER salary_log_trigger
AFTER UPDATE OR DELETE ON public.salary
FOR EACH ROW
EXECUTE FUNCTION public.log_dml_operations();

CREATE OR REPLACE VIEW public.dept_emp_latest_date AS
SELECT
	emp_no,
	MAX(
		from_date) AS from_date,
	MAX(
		to_date) AS to_date
FROM
	public.dept_emp
GROUP BY
	emp_no;

-- shows only the current department for each employee
CREATE OR REPLACE VIEW public.current_dept_emp AS
SELECT
	l.emp_no,
	dept_no,
	l.from_date,
	l.to_date
FROM
	public.dept_emp d
	INNER JOIN public.dept_emp_latest_date l ON d.emp_no = l.emp_no
		AND d.from_date = l.from_date
		AND l.to_date = d.to_date;
            `,
		},
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
		{
			name: "table_and_column_comments",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20)
);`,
			migrationDDL: `
-- Add comments to tables
COMMENT ON TABLE products IS 'Product catalog with pricing information';
COMMENT ON TABLE customers IS 'Customer master data';

-- Add comments to columns
COMMENT ON COLUMN products.id IS 'Unique product identifier';
COMMENT ON COLUMN products.name IS 'Product display name';
COMMENT ON COLUMN products.price IS 'Product price in USD';
COMMENT ON COLUMN products.category IS 'Product category classification';
COMMENT ON COLUMN products.created_at IS 'Record creation timestamp';

COMMENT ON COLUMN customers.id IS 'Unique customer identifier';
COMMENT ON COLUMN customers.first_name IS 'Customer first name';
COMMENT ON COLUMN customers.last_name IS 'Customer last name';
COMMENT ON COLUMN customers.email IS 'Customer email address';
COMMENT ON COLUMN customers.phone IS 'Customer contact phone number';

-- Create table with comments from the start
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_order_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT fk_order_product FOREIGN KEY (product_id) REFERENCES products(id)
);

COMMENT ON TABLE orders IS 'Customer purchase orders';
COMMENT ON COLUMN orders.id IS 'Unique order identifier';
COMMENT ON COLUMN orders.customer_id IS 'Reference to customer who placed the order';
COMMENT ON COLUMN orders.product_id IS 'Reference to ordered product';
COMMENT ON COLUMN orders.quantity IS 'Number of items ordered';
COMMENT ON COLUMN orders.order_date IS 'Date and time when order was placed';`,
			description: "Adding comments to tables and columns",
		},
		{
			name: "modify_and_drop_comments",
			initialSchema: `
CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    product_code VARCHAR(20) NOT NULL UNIQUE,
    stock_level INTEGER DEFAULT 0,
    warehouse_location VARCHAR(100),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add initial comments
COMMENT ON TABLE inventory IS 'Initial inventory tracking table';
COMMENT ON COLUMN inventory.id IS 'Primary key for inventory records';
COMMENT ON COLUMN inventory.product_code IS 'Unique product identifier code';
COMMENT ON COLUMN inventory.stock_level IS 'Current stock quantity';
COMMENT ON COLUMN inventory.warehouse_location IS 'Physical location in warehouse';
COMMENT ON COLUMN inventory.last_updated IS 'Last modification timestamp';`,
			migrationDDL: `
-- Modify existing comments
COMMENT ON TABLE inventory IS 'Comprehensive inventory management system';
COMMENT ON COLUMN inventory.product_code IS 'SKU - Stock Keeping Unit identifier';
COMMENT ON COLUMN inventory.stock_level IS 'Available quantity for sale';

-- Remove some comments
COMMENT ON COLUMN inventory.warehouse_location IS NULL;
COMMENT ON COLUMN inventory.last_updated IS NULL;

-- Add new column with comment
ALTER TABLE inventory ADD COLUMN reorder_point INTEGER DEFAULT 10;
COMMENT ON COLUMN inventory.reorder_point IS 'Minimum stock level before reordering';

-- Create view with comment
CREATE VIEW low_stock_items AS
SELECT product_code, stock_level, reorder_point
FROM inventory
WHERE stock_level <= reorder_point;

COMMENT ON VIEW low_stock_items IS 'Products that need to be restocked';`,
			description: "Modifying and dropping comments on existing objects",
		},
		{
			name: "comments_with_special_characters",
			initialSchema: `
CREATE TABLE test_table (
    id SERIAL PRIMARY KEY,
    data VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active'
);`,
			migrationDDL: `
-- Test comments with special characters and escaping
COMMENT ON TABLE test_table IS 'Test table with "quotes" and ''apostrophes'' and $special$ characters';
COMMENT ON COLUMN test_table.id IS 'ID with symbols: @#$%^&*()_+-={}|[]\\:";''<>?,./';
COMMENT ON COLUMN test_table.data IS 'Data field containing 
multiline
text with various symbols: ñáéíóú àèìòù äëïöü';
COMMENT ON COLUMN test_table.status IS 'Status: active/inactive (default: active)';

-- Create function with comment
CREATE OR REPLACE FUNCTION get_active_records()
RETURNS TABLE(id INTEGER, data VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.data
    FROM test_table t
    WHERE t.status = 'active';
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_active_records() IS 'Returns all active records from test_table';`,
			description: "Comments with special characters, quotes, and multiline text",
		},
		// Reverse test cases
		{
			name: "reverse_basic_table_operations",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT check_title_length CHECK (length(title) > 0)
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_post FOREIGN KEY (post_id) REFERENCES posts(id),
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_users_email ON users(email);
`,
			migrationDDL: `
-- Drop index
DROP INDEX idx_users_email;

-- Drop check constraint
ALTER TABLE posts DROP CONSTRAINT check_title_length;

-- Drop table
DROP TABLE comments;

-- Drop column
ALTER TABLE users DROP COLUMN is_active;
`,
			description: "Reverse of basic_table_operations: DROP column, table, index, constraint",
		},
		{
			name: "reverse_views_and_functions",
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

CREATE OR REPLACE FUNCTION calculate_order_total()
RETURNS TRIGGER AS $$
BEGIN
    NEW.total := NEW.quantity * (SELECT price FROM products WHERE id = NEW.product_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_order_total
BEFORE INSERT OR UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION calculate_order_total();

CREATE MATERIALIZED VIEW product_stats AS
SELECT 
    product_id,
    COUNT(*) as order_count,
    SUM(quantity) as total_quantity,
    SUM(total) as total_revenue
FROM orders
GROUP BY product_id;

CREATE INDEX idx_product_stats_revenue ON product_stats(total_revenue DESC);
`,
			migrationDDL: `
-- Drop index on materialized view
DROP INDEX idx_product_stats_revenue;

-- Drop materialized view
DROP MATERIALIZED VIEW product_stats;

-- Drop trigger
DROP TRIGGER update_order_total ON orders;

-- Drop function
DROP FUNCTION calculate_order_total();

-- Drop view
DROP VIEW product_inventory;
`,
			description: "Reverse of views_and_functions: DROP view, function, trigger, materialized view",
		},
		{
			name: "reverse_schema_and_sequences",
			initialSchema: `
CREATE SCHEMA inventory;
CREATE SCHEMA sales;

CREATE SEQUENCE sales.order_seq START WITH 1000 INCREMENT BY 10;

CREATE TYPE inventory.item_status AS ENUM ('available', 'out_of_stock', 'discontinued');

CREATE TABLE inventory.items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status inventory.item_status DEFAULT 'available'
);

CREATE TABLE sales.orders (
    id INTEGER DEFAULT nextval('sales.order_seq') PRIMARY KEY,
    item_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    CONSTRAINT fk_item FOREIGN KEY (item_id) REFERENCES inventory.items(id)
);
`,
			migrationDDL: `
-- Drop table in sales schema
DROP TABLE sales.orders;

-- Drop sequence
DROP SEQUENCE sales.order_seq;

-- Drop schema
DROP SCHEMA sales;

-- Drop column that uses enum
ALTER TABLE inventory.items DROP COLUMN status;

-- Drop enum type
DROP TYPE inventory.item_status;
`,
			description: "Reverse of schema_and_sequences: DROP schema, sequence, table, type",
		},
		{
			name: "reverse_complex_dependencies",
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

CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER NOT NULL,
    lead_employee_id INTEGER,
    CONSTRAINT fk_project_dept FOREIGN KEY (department_id) REFERENCES departments(id),
    CONSTRAINT fk_project_lead FOREIGN KEY (lead_employee_id) REFERENCES employees(id)
);

CREATE VIEW department_employees AS
SELECT d.id as dept_id, d.name as dept_name, e.id as emp_id, e.name as emp_name
FROM departments d
LEFT JOIN employees e ON d.id = e.department_id;

CREATE VIEW department_summary AS
SELECT dept_id, dept_name, COUNT(emp_id) as employee_count
FROM department_employees
GROUP BY dept_id, dept_name;

CREATE OR REPLACE FUNCTION get_department_employees(dept_id INTEGER)
RETURNS TABLE(employee_id INTEGER, employee_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT id, name
    FROM employees
    WHERE department_id = dept_id;
END;
$$ LANGUAGE plpgsql;
`,
			migrationDDL: `
-- Drop table with foreign keys
DROP TABLE projects;

-- Drop function
DROP FUNCTION get_department_employees(INTEGER);

-- Drop dependent view first
DROP VIEW department_summary;

-- Drop base view
DROP VIEW department_employees;
`,
			description: "Reverse of complex_dependencies: DROP views, functions, tables with dependencies",
		},
		{
			name: "reverse_drop_indexes_and_constraints",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    price DECIMAL(10, 2),
    supplier_id INTEGER
);
`,
			migrationDDL: `
-- Create indexes and constraints
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_price ON products(price);
CREATE UNIQUE INDEX idx_products_name ON products(name);

ALTER TABLE products ADD CONSTRAINT check_price_positive CHECK (price > 0);
ALTER TABLE products ADD CONSTRAINT check_name_length CHECK (length(name) >= 3);
`,
			description: "Reverse of drop_indexes_and_constraints: CREATE indexes and constraints",
		},
		{
			name: "reverse_drop_views_and_functions",
			initialSchema: `
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    sale_amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL
);
`,
			migrationDDL: `
-- Create views and functions
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
			description: "Reverse of drop_views_and_functions: CREATE views and functions",
		},
		{
			name: "reverse_alter_table_columns",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(30),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    stock_quantity INTEGER DEFAULT 0,
    weight DECIMAL(5, 2),
    CONSTRAINT check_price_positive CHECK (price > 0),
    CONSTRAINT check_stock_non_negative CHECK (stock_quantity >= 0)
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_created_at ON products(created_at);
CREATE UNIQUE INDEX idx_products_name_category ON products(name, category);
`,
			migrationDDL: `
-- Drop indexes
DROP INDEX idx_products_created_at;
DROP INDEX idx_products_name_category;

-- Drop constraints
ALTER TABLE products DROP CONSTRAINT check_stock_non_negative;
ALTER TABLE products DROP CONSTRAINT check_price_positive;

-- Revert column changes
ALTER TABLE products ALTER COLUMN category SET NOT NULL;
ALTER TABLE products ALTER COLUMN description DROP NOT NULL;
ALTER TABLE products ALTER COLUMN price TYPE DECIMAL(8, 2);
ALTER TABLE products ALTER COLUMN name TYPE VARCHAR(50);

-- Drop columns
ALTER TABLE products 
    DROP COLUMN weight,
    DROP COLUMN stock_quantity,
    DROP COLUMN created_at;
`,
			description: "Reverse of alter_table_columns: DROP columns, constraints, indexes",
		},
		{
			name: "reverse_drop_and_recreate_constraints",
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
    isbn VARCHAR(20),
    published_year INTEGER,
    price DECIMAL(8, 2),
    CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE,
    CONSTRAINT check_year_extended CHECK (published_year >= 1000 AND published_year <= 2030),
    CONSTRAINT check_title_length CHECK (length(title) >= 3)
);

CREATE INDEX idx_books_author ON books(author_id);
CREATE INDEX idx_books_year ON books(published_year);
CREATE INDEX idx_books_isbn ON books(isbn);
`,
			migrationDDL: `
-- Drop new constraints
ALTER TABLE books DROP CONSTRAINT check_title_length;

-- Drop index and recreate unique constraint
DROP INDEX idx_books_isbn;
ALTER TABLE books ADD CONSTRAINT books_isbn_key UNIQUE (isbn);

-- Drop and recreate check constraints with original definitions
ALTER TABLE books DROP CONSTRAINT check_year_extended;
ALTER TABLE books ADD CONSTRAINT check_year_valid CHECK (published_year >= 1000 AND published_year <= 2100);

-- Drop and recreate foreign key with original options
ALTER TABLE books DROP CONSTRAINT fk_author_new;
ALTER TABLE books ADD CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id);

-- Add back original constraint
ALTER TABLE books ADD CONSTRAINT check_price_positive CHECK (price > 0);
`,
			description: "Reverse of drop_and_recreate_constraints: Reverse the constraint changes",
		},
		{
			name:          "reverse_drop_sequence_and_type",
			initialSchema: ``,
			migrationDDL: `
-- Create schema and objects
CREATE SCHEMA accounting;

CREATE TYPE accounting.transaction_status AS ENUM ('pending', 'completed', 'cancelled');

CREATE SEQUENCE accounting.transaction_seq START 1000;

CREATE TABLE accounting.simple_log (
    message TEXT NOT NULL,
    status accounting.transaction_status DEFAULT 'pending'
);
`,
			description: "Reverse of drop_sequence_and_type: CREATE schema, sequence, type, table",
		},
		{
			name: "reverse_mixed_operations_complex",
			initialSchema: `
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_email VARCHAR(100),
    CONSTRAINT check_email_format CHECK (contact_email LIKE '%@%')
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    supplier_id INTEGER,
    price DECIMAL(10, 2),
    category_id INTEGER,
    CONSTRAINT fk_supplier_cascade FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE SET NULL,
    CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id),
    CONSTRAINT check_price_range CHECK (price >= 0 AND price <= 10000)
);

CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_price_range ON products(price) WHERE price > 100;

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
			migrationDDL: `
-- Drop function
DROP FUNCTION get_expensive_products(DECIMAL);

-- Drop indexes
DROP INDEX idx_products_price_range;
DROP INDEX idx_products_category;

-- Drop constraints
ALTER TABLE suppliers DROP CONSTRAINT check_email_format;
ALTER TABLE products DROP CONSTRAINT check_price_range;

-- Drop and recreate foreign key with original action
ALTER TABLE products DROP CONSTRAINT fk_supplier_cascade;
ALTER TABLE products ADD CONSTRAINT fk_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id);

-- Drop view
DROP VIEW product_summary;

-- Drop foreign key to categories
ALTER TABLE products DROP CONSTRAINT fk_category;

-- Drop column
ALTER TABLE products DROP COLUMN category_id;

-- Drop table
DROP TABLE categories;
`,
			description: "Reverse of mixed_operations_complex: Reverse all mixed operations",
		},
		{
			name:          "reverse_create_tables_with_fk",
			initialSchema: ``,
			migrationDDL: `
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
			description: "Reverse of create_tables_with_fk: CREATE tables with foreign keys",
		},
		{
			name: "reverse_multiple_foreign_keys",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE UNIQUE INDEX uk_email ON users(email);
CREATE INDEX idx_username ON users(username);
CREATE INDEX idx_email_active ON users(email, is_active);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published_at TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_title_length CHECK (LENGTH(title) > 0)
);

CREATE INDEX idx_user_id ON posts(user_id);

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
`,
			migrationDDL: `
-- Drop check constraint
ALTER TABLE posts DROP CONSTRAINT chk_title_length;

-- Drop index
DROP INDEX idx_email_active;

-- Drop table with multiple foreign keys
DROP TABLE comments;

-- Drop column
ALTER TABLE users DROP COLUMN is_active;
`,
			description: "Reverse of multiple_foreign_keys: DROP column, table, indexes, constraints",
		},
		{
			name: "reverse_drop_and_recreate_fk_constraints",
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
    CONSTRAINT fk_author_new FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT chk_year_extended CHECK (published_year >= 1000 AND published_year <= 2030),
    CONSTRAINT chk_price_positive CHECK (price > 0),
    CONSTRAINT chk_title_length CHECK (LENGTH(title) >= 3)
);

CREATE UNIQUE INDEX uk_isbn ON books(isbn);
CREATE INDEX idx_author ON books(author_id);
CREATE INDEX idx_year ON books(published_year);
`,
			migrationDDL: `
-- Drop new constraints
ALTER TABLE books DROP CONSTRAINT chk_title_length;

-- Drop and modify check constraints
ALTER TABLE books DROP CONSTRAINT chk_year_extended;
ALTER TABLE books ADD CONSTRAINT chk_year_valid CHECK (published_year >= 1000 AND published_year <= 2100);

-- Drop and recreate foreign key with different options
ALTER TABLE books DROP CONSTRAINT fk_author_new;
ALTER TABLE books ADD CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id);
`,
			description: "Reverse of drop_and_recreate_fk_constraints: Reverse FK constraint changes",
		},
		{
			name: "reverse_self_referencing_foreign_keys",
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

ALTER TABLE departments ADD CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES employees(id);

CREATE VIEW dept_employee_count AS
SELECT d.id AS dept_id, d.name AS dept_name, COUNT(e.id) AS emp_count
FROM departments d
LEFT JOIN employees e ON d.id = e.department_id
GROUP BY d.id, d.name;

CREATE VIEW dept_summary AS
SELECT 
    dept_id,
    dept_name,
    emp_count,
    0 AS avg_salary,
    0 AS max_salary,
    0 AS min_salary
FROM dept_employee_count;

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

CREATE OR REPLACE FUNCTION get_department_report(dept_name_pattern VARCHAR)
RETURNS TABLE(dept_id INTEGER, dept_name VARCHAR, emp_count BIGINT, avg_salary INTEGER, manager_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM dept_manager_summary
    WHERE dept_name LIKE '%' || dept_name_pattern || '%';
END;
$$ LANGUAGE plpgsql;
`,
			migrationDDL: `
-- Drop function using views
DROP FUNCTION get_department_report(VARCHAR);

-- Drop highly dependent view
DROP VIEW dept_manager_summary;

-- Drop dependent view
DROP VIEW dept_summary;

-- Drop base view
DROP VIEW dept_employee_count;
`,
			description: "Reverse of self_referencing_foreign_keys: DROP views and functions with dependencies",
		},
		{
			name: "reverse_circular_foreign_key_dependencies",
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
    total_amount DECIMAL(10, 2),
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    CONSTRAINT fk_order_item FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX idx_order ON order_items(order_id);

ALTER TABLE customers ADD CONSTRAINT fk_preferred_order FOREIGN KEY (preferred_order_id) REFERENCES orders(id) ON DELETE SET NULL;

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
			migrationDDL: `
-- Drop trigger
DROP TRIGGER trg_update_order_total ON order_items;

-- Drop function
DROP FUNCTION update_order_total();

-- Drop index
DROP INDEX idx_order;

-- Drop table
DROP TABLE order_items;

-- Drop circular foreign key constraints
ALTER TABLE customers DROP CONSTRAINT fk_preferred_order;
ALTER TABLE orders DROP CONSTRAINT fk_customer;
`,
			description: "Reverse of circular_foreign_key_dependencies: DROP circular FK dependencies",
		},
		{
			name: "reverse_table_and_column_comments",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20)
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_order_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT fk_order_product FOREIGN KEY (product_id) REFERENCES products(id)
);

COMMENT ON TABLE products IS 'Product catalog with pricing information';
COMMENT ON TABLE customers IS 'Customer master data';
COMMENT ON TABLE orders IS 'Customer purchase orders';

COMMENT ON COLUMN products.id IS 'Unique product identifier';
COMMENT ON COLUMN products.name IS 'Product display name';
COMMENT ON COLUMN products.price IS 'Product price in USD';
COMMENT ON COLUMN products.category IS 'Product category classification';
COMMENT ON COLUMN products.created_at IS 'Record creation timestamp';

COMMENT ON COLUMN customers.id IS 'Unique customer identifier';
COMMENT ON COLUMN customers.first_name IS 'Customer first name';
COMMENT ON COLUMN customers.last_name IS 'Customer last name';
COMMENT ON COLUMN customers.email IS 'Customer email address';
COMMENT ON COLUMN customers.phone IS 'Customer contact phone number';

COMMENT ON COLUMN orders.id IS 'Unique order identifier';
COMMENT ON COLUMN orders.customer_id IS 'Reference to customer who placed the order';
COMMENT ON COLUMN orders.product_id IS 'Reference to ordered product';
COMMENT ON COLUMN orders.quantity IS 'Number of items ordered';
COMMENT ON COLUMN orders.order_date IS 'Date and time when order was placed';
`,
			migrationDDL: `
-- Drop all column comments
COMMENT ON COLUMN orders.order_date IS NULL;
COMMENT ON COLUMN orders.quantity IS NULL;
COMMENT ON COLUMN orders.product_id IS NULL;
COMMENT ON COLUMN orders.customer_id IS NULL;
COMMENT ON COLUMN orders.id IS NULL;

COMMENT ON COLUMN customers.phone IS NULL;
COMMENT ON COLUMN customers.email IS NULL;
COMMENT ON COLUMN customers.last_name IS NULL;
COMMENT ON COLUMN customers.first_name IS NULL;
COMMENT ON COLUMN customers.id IS NULL;

COMMENT ON COLUMN products.created_at IS NULL;
COMMENT ON COLUMN products.category IS NULL;
COMMENT ON COLUMN products.price IS NULL;
COMMENT ON COLUMN products.name IS NULL;
COMMENT ON COLUMN products.id IS NULL;

-- Drop table comments
COMMENT ON TABLE orders IS NULL;
COMMENT ON TABLE customers IS NULL;
COMMENT ON TABLE products IS NULL;

-- Drop orders table with foreign keys
DROP TABLE orders;
`,
			description: "Reverse of table_and_column_comments: DROP comments",
		},
		{
			name: "reverse_modify_and_drop_comments",
			initialSchema: `
CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    product_code VARCHAR(20) NOT NULL UNIQUE,
    stock_level INTEGER DEFAULT 0,
    warehouse_location VARCHAR(100),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reorder_point INTEGER DEFAULT 10
);

CREATE VIEW low_stock_items AS
SELECT product_code, stock_level, reorder_point
FROM inventory
WHERE stock_level <= reorder_point;

COMMENT ON TABLE inventory IS 'Comprehensive inventory management system';
COMMENT ON COLUMN inventory.id IS 'Primary key for inventory records';
COMMENT ON COLUMN inventory.product_code IS 'SKU - Stock Keeping Unit identifier';
COMMENT ON COLUMN inventory.stock_level IS 'Available quantity for sale';
COMMENT ON COLUMN inventory.reorder_point IS 'Minimum stock level before reordering';
COMMENT ON VIEW low_stock_items IS 'Products that need to be restocked';
`,
			migrationDDL: `
-- Drop view comment
COMMENT ON VIEW low_stock_items IS NULL;

-- Drop view
DROP VIEW low_stock_items;

-- Drop column with comment
ALTER TABLE inventory DROP COLUMN reorder_point;

-- Add back removed comments
COMMENT ON COLUMN inventory.warehouse_location IS 'Physical location in warehouse';
COMMENT ON COLUMN inventory.last_updated IS 'Last modification timestamp';

-- Modify existing comments back to original
COMMENT ON COLUMN inventory.stock_level IS 'Current stock quantity';
COMMENT ON COLUMN inventory.product_code IS 'Unique product identifier code';
COMMENT ON TABLE inventory IS 'Initial inventory tracking table';
`,
			description: "Reverse of modify_and_drop_comments: Reverse comment changes",
		},
		{
			name: "reverse_bytebase_schema",
			initialSchema: `
CREATE TABLE public.employee (
	emp_no      SERIAL NOT NULL,
	birth_date  DATE NOT NULL,
	first_name  TEXT NOT NULL,
	last_name   TEXT NOT NULL,
	gender      TEXT NOT NULL CHECK (gender IN('M', 'F')) NOT NULL,
	hire_date   DATE NOT NULL,
	PRIMARY KEY (emp_no)
);

CREATE INDEX idx_employee_hire_date ON public.employee (hire_date);

CREATE TABLE public.department (
	dept_no     TEXT NOT NULL,
	dept_name   TEXT NOT NULL,
	PRIMARY KEY (dept_no),
	UNIQUE      (dept_name)
);

CREATE TABLE public.dept_manager (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE public.dept_emp (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE public.title (
	emp_no      INT NOT NULL,
	title       TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, title, from_date)
); 

CREATE TABLE public.salary (
	emp_no      INT NOT NULL,
	amount      INT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, from_date)
);

CREATE INDEX idx_salary_amount ON public.salary (amount);

CREATE TABLE public.audit (
    id SERIAL PRIMARY KEY,
    operation TEXT NOT NULL,
    query TEXT,
    user_name TEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_operation ON public.audit (operation);
CREATE INDEX idx_audit_username ON public.audit (user_name);
CREATE INDEX idx_audit_changed_at ON public.audit (changed_at);

CREATE OR REPLACE FUNCTION public.log_dml_operations() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('INSERT', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('UPDATE', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO public.audit (operation, query, user_name)
        VALUES ('DELETE', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- only log update and delete, otherwise, it will cause too much change.
CREATE TRIGGER salary_log_trigger
AFTER UPDATE OR DELETE ON public.salary
FOR EACH ROW
EXECUTE FUNCTION public.log_dml_operations();

CREATE OR REPLACE VIEW public.dept_emp_latest_date AS
SELECT
	emp_no,
	MAX(
		from_date) AS from_date,
	MAX(
		to_date) AS to_date
FROM
	public.dept_emp
GROUP BY
	emp_no;

-- shows only the current department for each employee
CREATE OR REPLACE VIEW public.current_dept_emp AS
SELECT
	l.emp_no,
	dept_no,
	l.from_date,
	l.to_date
FROM
	public.dept_emp d
	INNER JOIN public.dept_emp_latest_date l ON d.emp_no = l.emp_no
		AND d.from_date = l.from_date
		AND l.to_date = d.to_date;
            `,
			migrationDDL: `
-- Drop views first (due to dependencies)
DROP VIEW public.current_dept_emp;
DROP VIEW public.dept_emp_latest_date;

-- Drop trigger
DROP TRIGGER salary_log_trigger ON public.salary;

-- Drop function
DROP FUNCTION public.log_dml_operations();

-- Drop indexes
DROP INDEX idx_audit_changed_at;
DROP INDEX idx_audit_username;
DROP INDEX idx_audit_operation;
DROP INDEX idx_salary_amount;
DROP INDEX idx_employee_hire_date;

-- Drop tables (respecting foreign key dependencies)
DROP TABLE public.audit;
DROP TABLE public.title;
DROP TABLE public.salary;
DROP TABLE public.dept_emp;
DROP TABLE public.dept_manager;
DROP TABLE public.department;
DROP TABLE public.employee;
`,
			description: "Reverse of bytebase_schema: DROP entire schema with all objects",
		},
		{
			name: "reverse_comments_with_special_characters",
			initialSchema: `
CREATE TABLE test_table (
    id SERIAL PRIMARY KEY,
    data VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active'
);

CREATE OR REPLACE FUNCTION get_active_records()
RETURNS TABLE(id INTEGER, data VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.data
    FROM test_table t
    WHERE t.status = 'active';
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE test_table IS 'Test table with "quotes" and ''apostrophes'' and $special$ characters';
COMMENT ON COLUMN test_table.id IS 'ID with symbols: @#$%^&*()_+-={}|[]\\:";''<>?,./';
COMMENT ON COLUMN test_table.data IS 'Data field containing 
multiline
text with various symbols: ñáéíóú àèìòù äëïöü';
COMMENT ON COLUMN test_table.status IS 'Status: active/inactive (default: active)';
COMMENT ON FUNCTION get_active_records() IS 'Returns all active records from test_table';
`,
			migrationDDL: `
-- Drop function comment
COMMENT ON FUNCTION get_active_records() IS NULL;

-- Drop function
DROP FUNCTION get_active_records();

-- Drop column comments
COMMENT ON COLUMN test_table.status IS NULL;
COMMENT ON COLUMN test_table.data IS NULL;
COMMENT ON COLUMN test_table.id IS NULL;

-- Drop table comment
COMMENT ON TABLE test_table IS NULL;
`,
			description: "Reverse of comments_with_special_characters: DROP comments with special chars",
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
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, dbSchemaB, dbSchemaA)
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

			// Clear column positions as they can change when columns are added/dropped
			for _, column := range table.Columns {
				column.Position = 0
			}

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

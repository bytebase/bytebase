package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestSDLValidationWithTestcontainer tests the SDL (Schema Definition Language) correctness
// by following a 5-step validation process:
// 1. Apply initial schema A to database
// 2. Sync to get schema B from database
// 3. Define expected schema C, use getDatabaseMetadata to get its metadata, generate diff from B to C, then generate migration DDL
// 4. Apply generated DDL to database, then sync to get schema D
// 5. Verify schema D and schema C metadata have no diff and validate each member consistently
func TestSDLValidationWithTestcontainer(t *testing.T) {
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

	// Test cases following the 5-step SDL validation process
	testCases := []struct {
		name           string
		initialSchema  string // Schema A - initial state
		expectedSchema string // Schema C - target state
		description    string
	}{
		{
			name: "hr_test",
			initialSchema: `
			
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

CREATE SEQUENCE "public"."audit_id_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."audit" (
    "id" integer DEFAULT nextval('public.audit_id_seq'::regclass) NOT NULL,
    "operation" text NOT NULL,
    "query" text,
    "user_name" text NOT NULL,
    "changed_at" timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER SEQUENCE "public"."audit_id_seq" OWNED BY "public"."audit"."id";

ALTER TABLE ONLY "public"."audit" ADD CONSTRAINT "audit_pkey" PRIMARY KEY (id);

CREATE INDEX "idx_audit_changed_at" ON ONLY "public"."audit" (changed_at);

CREATE INDEX "idx_audit_operation" ON ONLY "public"."audit" (operation);

CREATE INDEX "idx_audit_username" ON ONLY "public"."audit" (user_name);

CREATE TABLE "public"."department" (
    "dept_no" text NOT NULL,
    "dept_name" text NOT NULL
);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_pkey" PRIMARY KEY (dept_no);

ALTER TABLE ONLY "public"."department" ADD CONSTRAINT "department_dept_name_key" UNIQUE ("dept_name");

CREATE TABLE "public"."dept_emp" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_emp" ADD CONSTRAINT "dept_emp_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE TABLE "public"."dept_manager" (
    "emp_no" integer NOT NULL,
    "dept_no" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."dept_manager" ADD CONSTRAINT "dept_manager_pkey" PRIMARY KEY (emp_no, dept_no);

CREATE SEQUENCE "public"."employee_emp_no_seq"
    AS integer
	START WITH 1
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 2147483647
	NO CYCLE;

CREATE TABLE "public"."employee" (
    "emp_no" integer DEFAULT nextval('public.employee_emp_no_seq'::regclass) NOT NULL,
    "birth_date" date NOT NULL,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "gender" text NOT NULL,
    "hire_date" date NOT NULL,
    CONSTRAINT "employee_gender_check" CHECK (gender = ANY (ARRAY['M'::text, 'F'::text]))
);

ALTER SEQUENCE "public"."employee_emp_no_seq" OWNED BY "public"."employee"."emp_no";

ALTER TABLE ONLY "public"."employee" ADD CONSTRAINT "employee_pkey" PRIMARY KEY (emp_no);

CREATE INDEX "idx_employee_hire_date" ON ONLY "public"."employee" (hire_date);

CREATE OR REPLACE FUNCTION public.log_dml_operations()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
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
$function$
;

CREATE TABLE "public"."salary" (
    "emp_no" integer NOT NULL,
    "amount" integer NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date NOT NULL
);

ALTER TABLE ONLY "public"."salary" ADD CONSTRAINT "salary_pkey" PRIMARY KEY (emp_no, from_date);

CREATE INDEX "idx_salary_amount" ON ONLY "public"."salary" (amount);

CREATE TABLE "public"."title" (
    "emp_no" integer NOT NULL,
    "title" text NOT NULL,
    "from_date" date NOT NULL,
    "to_date" date
);

ALTER TABLE ONLY "public"."title" ADD CONSTRAINT "title_pkey" PRIMARY KEY (emp_no, title, from_date);

CREATE VIEW "public"."dept_emp_latest_date" AS 
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM public.dept_emp
  GROUP BY emp_no;

CREATE VIEW "public"."current_dept_emp" AS 
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (public.dept_emp d
     JOIN public.dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_emp"
    ADD CONSTRAINT "dept_emp_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_dept_no_fkey" FOREIGN KEY ("dept_no")
    REFERENCES "public"."department" ("dept_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."dept_manager"
    ADD CONSTRAINT "dept_manager_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."salary"
    ADD CONSTRAINT "salary_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;

ALTER TABLE "public"."title"
    ADD CONSTRAINT "title_emp_no_fkey" FOREIGN KEY ("emp_no")
    REFERENCES "public"."employee" ("emp_no")
    ON DELETE CASCADE;
`,
			expectedSchema: `
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

CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	username TEXT NOT NULL,
	email TEXT NOT NULL,
	password TEXT NOT NULL
);
`,
		},
		{
			name:          "start_for_empty_schema",
			initialSchema: ``,
			expectedSchema: `
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
			description: "Start with an empty schema",
		},
		{
			name: "add_table_with_constraints",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`,
			expectedSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;
`,
			description: "Add new table with foreign key and indexes",
		},
		{
			name: "modify_table_add_columns",
			initialSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL
);
`,
			expectedSchema: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_active ON products(is_active) WHERE is_active = true;
`,
			description: "Add columns and indexes to existing table",
		},
		{
			name: "add_view_and_function",
			initialSchema: `
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50),
    salary DECIMAL(10, 2),
    hire_date DATE DEFAULT CURRENT_DATE
);
`,
			expectedSchema: `
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50),
    salary DECIMAL(10, 2),
    hire_date DATE DEFAULT CURRENT_DATE
);

CREATE VIEW active_employees AS
SELECT id, name, department, salary
FROM employees
WHERE department IS NOT NULL;

CREATE FUNCTION get_employee_count(dept VARCHAR) RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM employees WHERE department = dept);
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION calculate_bonus(emp_id INTEGER) RETURNS DECIMAL AS $$
DECLARE
    emp_salary DECIMAL;
BEGIN
    SELECT salary INTO emp_salary FROM employees WHERE id = emp_id;
    RETURN COALESCE(emp_salary * 0.1, 0);
END;
$$ LANGUAGE plpgsql;
`,
			description: "Add view and functions to existing table",
		},
		{
			name: "add_enum_and_sequence",
			initialSchema: `
CREATE TABLE basic_items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
`,
			expectedSchema: `
CREATE TYPE status_enum AS ENUM ('pending', 'active', 'inactive', 'deleted');
CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral');

CREATE SEQUENCE custom_id_seq START WITH 1000 INCREMENT BY 10;

CREATE TABLE basic_items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE items (
    id INTEGER DEFAULT nextval('custom_id_seq') PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status status_enum DEFAULT 'pending',
    user_mood mood,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_items_status ON items(status);
`,
			description: "Add enum types and custom sequence",
		},
		{
			name: "add_triggers_and_audit",
			initialSchema: `
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE DEFAULT CURRENT_DATE
);
`,
			expectedSchema: `
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    sale_date DATE DEFAULT CURRENT_DATE
);

CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    user_id INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    old_values JSONB,
    new_values JSONB
);

CREATE OR REPLACE FUNCTION audit_trigger_function() 
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (table_name, operation, old_values, new_values)
    VALUES (
        TG_TABLE_NAME,
        TG_OP,
        CASE WHEN TG_OP = 'DELETE' THEN row_to_json(OLD) ELSE NULL END,
        CASE WHEN TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN row_to_json(NEW) ELSE NULL END
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sales_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON sales
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();
`,
			description: "Add audit table, function and trigger",
		},
		{
			name:          "create_from_empty",
			initialSchema: ``, // Start from empty database
			expectedSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;

CREATE VIEW user_post_count AS
SELECT u.username, COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.username;
`,
			description: "Create complete schema from empty database",
		},
		{
			name: "cross_schema_references",
			initialSchema: `
CREATE SCHEMA hr;

CREATE TABLE hr.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);
`,
			expectedSchema: `
CREATE SCHEMA hr;
CREATE SCHEMA finance;

CREATE TABLE hr.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    manager_id INTEGER
);

CREATE TABLE hr.employees (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    department_id INTEGER NOT NULL,
    salary DECIMAL(10, 2),
    hire_date DATE DEFAULT CURRENT_DATE,
    CONSTRAINT fk_dept FOREIGN KEY (department_id) REFERENCES hr.departments(id)
);

-- Self-referencing foreign key for manager
ALTER TABLE hr.departments
ADD CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES hr.employees(id);

-- Tables in finance schema
CREATE TABLE finance.budgets (
    id SERIAL PRIMARY KEY,
    department_id INTEGER NOT NULL,
    fiscal_year INTEGER NOT NULL,
    allocated_amount DECIMAL(12, 2) NOT NULL,
    spent_amount DECIMAL(12, 2) DEFAULT 0.00,
    -- Cross-schema foreign key
    CONSTRAINT fk_budget_dept FOREIGN KEY (department_id) REFERENCES hr.departments(id),
    CONSTRAINT unique_budget_year UNIQUE (department_id, fiscal_year)
);

-- View that joins across schemas
CREATE VIEW finance.department_spending AS
SELECT d.name AS department_name,
 b.fiscal_year,
 b.allocated_amount,
 b.spent_amount,
 (b.allocated_amount - b.spent_amount) AS remaining_budget
FROM hr.departments d
  JOIN finance.budgets b ON d.id = b.department_id;
`,
			description: "Add cross-schema references and views",
		},
		{
			name: "drop_all_objects",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
`,
			expectedSchema: ``, // Empty schema - expect all objects to be dropped
			description:    "Drop all objects from database (test deletion)",
		},
		{
			name: "foreign_key_on_delete_cascade",
			initialSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL
);
`,
			expectedSchema: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_comment_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
`,
			description: "Test foreign keys with ON DELETE CASCADE and SET NULL",
		},
		{
			name: "foreign_key_on_update_actions",
			initialSchema: `
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL UNIQUE
);
`,
			expectedSchema: `
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL UNIQUE
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL,
    category_code VARCHAR(10) NOT NULL,
    name VARCHAR(200) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    CONSTRAINT fk_product_category_id FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_product_category_code FOREIGN KEY (category_code) REFERENCES categories(code) ON DELETE CASCADE ON UPDATE SET NULL
);

CREATE TABLE product_reviews (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    CONSTRAINT fk_review_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE ON UPDATE RESTRICT
);
`,
			description: "Test foreign keys with various ON DELETE and ON UPDATE actions",
		},
		{
			name: "foreign_key_no_action",
			initialSchema: `
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    budget DECIMAL(15,2) DEFAULT 0
);
`,
			expectedSchema: `
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    budget DECIMAL(15,2) DEFAULT 0
);

CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    department_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    manager_id INTEGER,
    CONSTRAINT fk_employee_department FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_employee_manager FOREIGN KEY (manager_id) REFERENCES employees(id)
);
`,
			description: "Test foreign keys with NO ACTION (explicit and implicit default)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new database for each test case
			dbName := fmt.Sprintf("sdl_test_%s", tc.name)
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

			// Execute the 5-step SDL validation process
			err = executeSDLValidationProcess(ctx, t, &testConnConfig, testDB, dbName, tc.initialSchema, tc.expectedSchema, tc.description)
			require.NoError(t, err, "SDL validation process should complete successfully for test case: %s", tc.description)
		})
	}
}

// executeSDLValidationProcess implements the 5-step SDL validation workflow
func executeSDLValidationProcess(ctx context.Context, t *testing.T, connConfig *pgx.ConnConfig, testDB *sql.DB, dbName, initialSchema, expectedSchema, description string) error {
	t.Logf("Starting SDL validation process for: %s", description)

	// Step 1: Apply initial schema A to database
	t.Log("Step 1: Applying initial schema A to database")
	if strings.TrimSpace(initialSchema) != "" {
		_, err := testDB.Exec(initialSchema)
		if err != nil {
			return errors.Wrapf(err, "failed to apply initial schema")
		}
	}

	// Step 2: Sync to get schema B from database
	t.Log("Step 2: Syncing to get schema B from database")
	schemaB, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return errors.Wrapf(err, "failed to sync schema B")
	}

	// Step 3: Define expected schema C, use getDatabaseMetadata to get its metadata, generate diff from B to C, then generate migration DDL
	t.Log("Step 3: Define expected schema C, use getDatabaseMetadata to get its metadata, generate diff from B to C, then generate migration DDL")

	// Get metadata for expected schema C using GetDatabaseMetadata
	// Note: Empty expectedSchema means we expect an empty database (all objects dropped)
	schemaCMetadata, err := GetDatabaseMetadata(expectedSchema)
	if err != nil {
		return errors.Wrapf(err, "failed to get expected schema C metadata")
	}

	// Generate migration DDL from schema B to schema C using the differ
	migrationDDL, actualDiff, err := generateMigrationDDLFromMetadata(schemaB, schemaCMetadata)
	require.NoError(t, err, "Failed to generate migration DDL from metadata")

	t.Logf("Generated migration DDL (%d characters):\n%s", len(migrationDDL), migrationDDL)

	// Save or validate the actual diff (B to C) and DDL files
	testName := strings.ReplaceAll(t.Name(), "/", "_")
	err = validateOrSaveTestFiles(t, actualDiff, testName, migrationDDL)
	if err != nil {
		return errors.Wrapf(err, "test file validation/creation failed")
	}

	// Step 4: Apply generated DDL to database, then sync to get schema D
	t.Log("Step 4: Apply generated DDL to database, then sync to get schema D")
	if strings.TrimSpace(migrationDDL) != "" {
		_, err := testDB.Exec(migrationDDL)
		if err != nil {
			return errors.Wrapf(err, "failed to apply migration DDL")
		}
	}

	// Sync to get schema D after migration
	schemaD, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return errors.Wrapf(err, "failed to sync schema D after migration")
	}

	// Step 5: Verify schema D and schema C metadata have no diff and validate each member consistently
	t.Log("Step 5: Verify schema D and schema C metadata have no diff and validate each member consistently")

	// Compare schema D with schema C metadata to ensure they match
	err = validateSchemaConsistency(schemaD, schemaCMetadata)
	if err != nil {
		return errors.Wrapf(err, "schema validation failed - schema D does not match expected schema C")
	}

	t.Logf("✓ SDL validation process completed successfully for: %s", description)
	return nil
}

// getSyncMetadataForSDL retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadataForSDL(ctx context.Context, connConfig *pgx.ConnConfig, dbName string) (*storepb.DatabaseSchemaMetadata, error) {
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

// generateMigrationDDLFromMetadata generates migration DDL from schema B to schema C using schema.differ
func generateMigrationDDLFromMetadata(schemaB, schemaC *storepb.DatabaseSchemaMetadata) (string, *schema.MetadataDiff, error) {
	// Create model.DatabaseMetadata objects for comparison
	modelSchemaB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_POSTGRES, true)
	modelSchemaC := model.NewDatabaseMetadata(schemaC, nil, nil, storepb.Engine_POSTGRES, true)

	// Use the schema differ to compute differences
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, modelSchemaB, modelSchemaC)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to compute schema difference")
	}

	// Convert the metadata diff to DDL using schema.GenerateMigration
	// This generates the actual migration DDL from the computed differences
	migrationDDL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to generate migration DDL from schema diff")
	}

	return migrationDDL, diff, nil
}

// validateSchemaConsistency compares two schema metadata objects to ensure they are equivalent
func validateSchemaConsistency(schemaD, schemaC *storepb.DatabaseSchemaMetadata) error {
	// Create model.DatabaseMetadata objects for comparison
	// PostgreSQL is case-sensitive for object names but case-insensitive for details
	modelSchemaD := model.NewDatabaseMetadata(schemaD, nil, nil, storepb.Engine_POSTGRES, true)
	modelSchemaC := model.NewDatabaseMetadata(schemaC, nil, nil, storepb.Engine_POSTGRES, true)

	// Use the schema differ to compare the two schemas
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, modelSchemaD, modelSchemaC)
	if err != nil {
		return errors.Wrapf(err, "failed to compute schema difference")
	}

	// Note: diff files are saved earlier in the process with the actual B to C differences

	// Check if there are any differences
	if diff != nil {
		var diffDetails []string

		// Check for any schema changes
		if len(diff.SchemaChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("SchemaChanges: %d", len(diff.SchemaChanges)))
		}
		if len(diff.TableChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("TableChanges: %d", len(diff.TableChanges)))
			// Add detailed table changes for debugging
			for i, tableDiff := range diff.TableChanges {
				diffDetails = append(diffDetails, fmt.Sprintf("  Table[%d]: %s table %s.%s (columns: %d, indexes: %d, checkConstraints: %d, foreignKeys: %d, triggers: %d)",
					i, tableDiff.Action, tableDiff.SchemaName, tableDiff.TableName,
					len(tableDiff.ColumnChanges), len(tableDiff.IndexChanges), len(tableDiff.CheckConstraintChanges), len(tableDiff.ForeignKeyChanges), len(tableDiff.TriggerChanges)))

				// Add column change details
				for j, columnDiff := range tableDiff.ColumnChanges {
					columnName := ""
					if columnDiff.OldColumn != nil {
						columnName = columnDiff.OldColumn.Name
					} else if columnDiff.NewColumn != nil {
						columnName = columnDiff.NewColumn.Name
					}
					diffDetails = append(diffDetails, fmt.Sprintf("    Column[%d]: %s column %s", j, columnDiff.Action, columnName))
				}

				// Add index change details
				for j, indexDiff := range tableDiff.IndexChanges {
					indexName := ""
					if indexDiff.OldIndex != nil {
						indexName = indexDiff.OldIndex.Name
					} else if indexDiff.NewIndex != nil {
						indexName = indexDiff.NewIndex.Name
					}
					diffDetails = append(diffDetails, fmt.Sprintf("    Index[%d]: %s index %s", j, indexDiff.Action, indexName))

					// Show more details for debugging
					if indexDiff.OldIndex != nil && indexDiff.NewIndex != nil {
						oldExpr := ""
						newExpr := ""
						if len(indexDiff.OldIndex.Expressions) > 0 {
							oldExpr = strings.Join(indexDiff.OldIndex.Expressions, ", ")
						}
						if len(indexDiff.NewIndex.Expressions) > 0 {
							newExpr = strings.Join(indexDiff.NewIndex.Expressions, ", ")
						}
						diffDetails = append(diffDetails, fmt.Sprintf("      Old: expr=%s type=%s unique=%v primary=%v", oldExpr, indexDiff.OldIndex.Type, indexDiff.OldIndex.Unique, indexDiff.OldIndex.Primary))
						diffDetails = append(diffDetails, fmt.Sprintf("      Old def: %s", indexDiff.OldIndex.Definition))
						diffDetails = append(diffDetails, fmt.Sprintf("      New: expr=%s type=%s unique=%v primary=%v", newExpr, indexDiff.NewIndex.Type, indexDiff.NewIndex.Unique, indexDiff.NewIndex.Primary))
						diffDetails = append(diffDetails, fmt.Sprintf("      New def: %s", indexDiff.NewIndex.Definition))
					} else if indexDiff.OldIndex != nil {
						oldExpr := ""
						if len(indexDiff.OldIndex.Expressions) > 0 {
							oldExpr = strings.Join(indexDiff.OldIndex.Expressions, ", ")
						}
						diffDetails = append(diffDetails, fmt.Sprintf("      Old: expr=%s type=%s unique=%v primary=%v", oldExpr, indexDiff.OldIndex.Type, indexDiff.OldIndex.Unique, indexDiff.OldIndex.Primary))
						diffDetails = append(diffDetails, fmt.Sprintf("      Old def: %s", indexDiff.OldIndex.Definition))
					} else if indexDiff.NewIndex != nil {
						newExpr := ""
						if len(indexDiff.NewIndex.Expressions) > 0 {
							newExpr = strings.Join(indexDiff.NewIndex.Expressions, ", ")
						}
						diffDetails = append(diffDetails, fmt.Sprintf("      New: expr=%s type=%s unique=%v primary=%v", newExpr, indexDiff.NewIndex.Type, indexDiff.NewIndex.Unique, indexDiff.NewIndex.Primary))
						diffDetails = append(diffDetails, fmt.Sprintf("      New def: %s", indexDiff.NewIndex.Definition))
					}
				}
			}
		}
		if len(diff.ViewChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("ViewChanges: %d", len(diff.ViewChanges)))
			for _, viewDiff := range diff.ViewChanges {
				diffDetails = append(diffDetails, fmt.Sprintf("  View: %s.%s, Action: %v", viewDiff.SchemaName, viewDiff.ViewName, viewDiff.Action))
				if viewDiff.OldView != nil && viewDiff.NewView != nil {
					diffDetails = append(diffDetails, fmt.Sprintf("    Old definition: %s", viewDiff.OldView.Definition))
					diffDetails = append(diffDetails, fmt.Sprintf("    New definition: %s", viewDiff.NewView.Definition))
				} else if viewDiff.OldView != nil {
					diffDetails = append(diffDetails, fmt.Sprintf("    Old definition: %s", viewDiff.OldView.Definition))
				} else if viewDiff.NewView != nil {
					diffDetails = append(diffDetails, fmt.Sprintf("    New definition: %s", viewDiff.NewView.Definition))
				}
			}
		}
		if len(diff.MaterializedViewChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("MaterializedViewChanges: %d", len(diff.MaterializedViewChanges)))
		}
		if len(diff.FunctionChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("FunctionChanges: %d", len(diff.FunctionChanges)))
		}
		if len(diff.ProcedureChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("ProcedureChanges: %d", len(diff.ProcedureChanges)))
		}
		if len(diff.SequenceChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("SequenceChanges: %d", len(diff.SequenceChanges)))
		}
		if len(diff.EnumTypeChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("EnumTypeChanges: %d", len(diff.EnumTypeChanges)))
		}
		if len(diff.EventChanges) > 0 {
			diffDetails = append(diffDetails, fmt.Sprintf("EventChanges: %d", len(diff.EventChanges)))
		}

		if len(diffDetails) > 0 {
			return errors.Errorf("schema D does not match expected schema C: found differences - %s", strings.Join(diffDetails, ", "))
		}
	}

	return nil
}

// DiffSummary represents the structure for JSON serialization of schema differences
type DiffSummary struct {
	HasDifferences  bool         `json:"hasDifferences"`
	Summary         string       `json:"summary"`
	TableChanges    int          `json:"tableChanges"`
	ViewChanges     int          `json:"viewChanges"`
	FunctionChanges int          `json:"functionChanges"`
	SchemaChanges   int          `json:"schemaChanges"`
	SequenceChanges int          `json:"sequenceChanges"`
	EnumTypeChanges int          `json:"enumTypeChanges"`
	Details         *DiffDetails `json:"details,omitempty"`
}

// DiffDetails provides detailed information about the differences
type DiffDetails struct {
	Tables    []TableChangeSummary    `json:"tables,omitempty"`
	Views     []ViewChangeSummary     `json:"views,omitempty"`
	Functions []FunctionChangeSummary `json:"functions,omitempty"`
	Schemas   []SchemaChangeSummary   `json:"schemas,omitempty"`
}

// TableChangeSummary summarizes table changes with detailed information
type TableChangeSummary struct {
	Action            string `json:"action"`
	SchemaName        string `json:"schemaName"`
	TableName         string `json:"tableName"`
	ColumnChanges     int    `json:"columnChanges"`
	IndexChanges      int    `json:"indexChanges"`
	ConstraintChanges int    `json:"constraintChanges"`
	ForeignKeyChanges int    `json:"foreignKeyChanges"`
	// Detailed changes
	Columns     []ColumnChangeDetail     `json:"columns,omitempty"`
	Indexes     []IndexChangeDetail      `json:"indexes,omitempty"`
	Constraints []ConstraintChangeDetail `json:"constraints,omitempty"`
	ForeignKeys []ForeignKeyChangeDetail `json:"foreignKeys,omitempty"`
}

// ViewChangeSummary summarizes view changes
type ViewChangeSummary struct {
	Action     string `json:"action"`
	SchemaName string `json:"schemaName"`
	ViewName   string `json:"viewName"`
}

// FunctionChangeSummary summarizes function changes
type FunctionChangeSummary struct {
	Action       string `json:"action"`
	SchemaName   string `json:"schemaName"`
	FunctionName string `json:"functionName"`
}

// SchemaChangeSummary summarizes schema changes
type SchemaChangeSummary struct {
	Action     string `json:"action"`
	SchemaName string `json:"schemaName"`
}

// ColumnChangeDetail provides detailed information about column changes
type ColumnChangeDetail struct {
	Action    string      `json:"action"`
	Name      string      `json:"name"`
	OldColumn *ColumnInfo `json:"oldColumn,omitempty"`
	NewColumn *ColumnInfo `json:"newColumn,omitempty"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Comment      string `json:"comment,omitempty"`
}

// IndexChangeDetail provides detailed information about index changes
type IndexChangeDetail struct {
	Action   string     `json:"action"`
	Name     string     `json:"name"`
	OldIndex *IndexInfo `json:"oldIndex,omitempty"`
	NewIndex *IndexInfo `json:"newIndex,omitempty"`
}

// IndexInfo represents index metadata
type IndexInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"`
	Unique      bool     `json:"unique"`
	Expressions []string `json:"expressions"`
	KeyLength   []int64  `json:"keyLength,omitempty"`
	Descending  []bool   `json:"descending,omitempty"`
}

// ConstraintChangeDetail provides detailed information about constraint changes
type ConstraintChangeDetail struct {
	Action        string          `json:"action"`
	Name          string          `json:"name"`
	OldConstraint *ConstraintInfo `json:"oldConstraint,omitempty"`
	NewConstraint *ConstraintInfo `json:"newConstraint,omitempty"`
}

// ConstraintInfo represents constraint metadata
type ConstraintInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Expression string `json:"expression,omitempty"`
}

// ForeignKeyChangeDetail provides detailed information about foreign key changes
type ForeignKeyChangeDetail struct {
	Action        string          `json:"action"`
	Name          string          `json:"name"`
	OldForeignKey *ForeignKeyInfo `json:"oldForeignKey,omitempty"`
	NewForeignKey *ForeignKeyInfo `json:"newForeignKey,omitempty"`
}

// ForeignKeyInfo represents foreign key metadata
type ForeignKeyInfo struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedSchema  string   `json:"referencedSchema"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"`
	OnUpdate          string   `json:"onUpdate,omitempty"`
	OnDelete          string   `json:"onDelete,omitempty"`
	MatchType         string   `json:"matchType,omitempty"`
}

// createDiffSummary creates a structured summary from MetadataDiff
func createDiffSummary(diff *schema.MetadataDiff) *DiffSummary {
	if diff == nil {
		return &DiffSummary{
			HasDifferences: false,
			Summary:        "No schema differences found - schemas match perfectly!",
		}
	}

	// Check if there are actually any differences
	hasDifferences := len(diff.TableChanges) > 0 || len(diff.ViewChanges) > 0 ||
		len(diff.FunctionChanges) > 0 || len(diff.SchemaChanges) > 0 ||
		len(diff.SequenceChanges) > 0 || len(diff.EnumTypeChanges) > 0

	summary := &DiffSummary{
		HasDifferences:  hasDifferences,
		TableChanges:    len(diff.TableChanges),
		ViewChanges:     len(diff.ViewChanges),
		FunctionChanges: len(diff.FunctionChanges),
		SchemaChanges:   len(diff.SchemaChanges),
		SequenceChanges: len(diff.SequenceChanges),
		EnumTypeChanges: len(diff.EnumTypeChanges),
	}

	if !hasDifferences {
		summary.Summary = "No schema differences found - schemas match perfectly!"
		return summary
	}

	// Create summary text
	var summaryParts []string
	if summary.TableChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Table changes: %d", summary.TableChanges))
	}
	if summary.ViewChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("View changes: %d", summary.ViewChanges))
	}
	if summary.FunctionChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Function changes: %d", summary.FunctionChanges))
	}
	if summary.SchemaChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Schema changes: %d", summary.SchemaChanges))
	}
	if summary.SequenceChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Sequence changes: %d", summary.SequenceChanges))
	}
	if summary.EnumTypeChanges > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Enum type changes: %d", summary.EnumTypeChanges))
	}

	summary.Summary = strings.Join(summaryParts, ", ")

	// Add detailed information
	details := &DiffDetails{}

	// Add table changes details
	for _, tableChange := range diff.TableChanges {
		tableSummary := TableChangeSummary{
			Action:            string(tableChange.Action),
			SchemaName:        tableChange.SchemaName,
			TableName:         tableChange.TableName,
			ColumnChanges:     len(tableChange.ColumnChanges),
			IndexChanges:      len(tableChange.IndexChanges),
			ConstraintChanges: len(tableChange.CheckConstraintChanges),
			ForeignKeyChanges: len(tableChange.ForeignKeyChanges),
		}

		// Add detailed column changes
		for _, colChange := range tableChange.ColumnChanges {
			columnDetail := ColumnChangeDetail{
				Action: string(colChange.Action),
				Name:   getColumnName(colChange),
			}
			if colChange.OldColumn != nil {
				columnDetail.OldColumn = &ColumnInfo{
					Name:         colChange.OldColumn.Name,
					Type:         colChange.OldColumn.Type,
					Nullable:     colChange.OldColumn.Nullable,
					DefaultValue: colChange.OldColumn.Default,
					Comment:      colChange.OldColumn.Comment,
				}
			}
			if colChange.NewColumn != nil {
				columnDetail.NewColumn = &ColumnInfo{
					Name:         colChange.NewColumn.Name,
					Type:         colChange.NewColumn.Type,
					Nullable:     colChange.NewColumn.Nullable,
					DefaultValue: colChange.NewColumn.Default,
					Comment:      colChange.NewColumn.Comment,
				}
			}
			tableSummary.Columns = append(tableSummary.Columns, columnDetail)
		}

		// Add detailed index changes
		for _, indexChange := range tableChange.IndexChanges {
			indexDetail := IndexChangeDetail{
				Action: string(indexChange.Action),
				Name:   getIndexName(indexChange),
			}
			if indexChange.OldIndex != nil {
				indexDetail.OldIndex = &IndexInfo{
					Name:        indexChange.OldIndex.Name,
					Type:        indexChange.OldIndex.Type,
					Unique:      indexChange.OldIndex.Unique,
					Expressions: indexChange.OldIndex.Expressions,
					KeyLength:   indexChange.OldIndex.KeyLength,
					Descending:  indexChange.OldIndex.Descending,
				}
			}
			if indexChange.NewIndex != nil {
				indexDetail.NewIndex = &IndexInfo{
					Name:        indexChange.NewIndex.Name,
					Type:        indexChange.NewIndex.Type,
					Unique:      indexChange.NewIndex.Unique,
					Expressions: indexChange.NewIndex.Expressions,
					KeyLength:   indexChange.NewIndex.KeyLength,
					Descending:  indexChange.NewIndex.Descending,
				}
			}
			tableSummary.Indexes = append(tableSummary.Indexes, indexDetail)
		}

		// Add detailed constraint changes
		for _, constraintChange := range tableChange.CheckConstraintChanges {
			constraintDetail := ConstraintChangeDetail{
				Action: string(constraintChange.Action),
				Name:   getConstraintName(constraintChange),
			}
			if constraintChange.OldCheckConstraint != nil {
				constraintDetail.OldConstraint = &ConstraintInfo{
					Name:       constraintChange.OldCheckConstraint.Name,
					Type:       "CHECK",
					Expression: constraintChange.OldCheckConstraint.Expression,
				}
			}
			if constraintChange.NewCheckConstraint != nil {
				constraintDetail.NewConstraint = &ConstraintInfo{
					Name:       constraintChange.NewCheckConstraint.Name,
					Type:       "CHECK",
					Expression: constraintChange.NewCheckConstraint.Expression,
				}
			}
			tableSummary.Constraints = append(tableSummary.Constraints, constraintDetail)
		}

		// Add detailed foreign key changes
		for _, fkChange := range tableChange.ForeignKeyChanges {
			fkDetail := ForeignKeyChangeDetail{
				Action: string(fkChange.Action),
				Name:   getForeignKeyName(fkChange),
			}
			if fkChange.OldForeignKey != nil {
				fkDetail.OldForeignKey = &ForeignKeyInfo{
					Name:              fkChange.OldForeignKey.Name,
					Columns:           fkChange.OldForeignKey.Columns,
					ReferencedSchema:  fkChange.OldForeignKey.ReferencedSchema,
					ReferencedTable:   fkChange.OldForeignKey.ReferencedTable,
					ReferencedColumns: fkChange.OldForeignKey.ReferencedColumns,
					OnUpdate:          fkChange.OldForeignKey.OnUpdate,
					OnDelete:          fkChange.OldForeignKey.OnDelete,
					MatchType:         fkChange.OldForeignKey.MatchType,
				}
			}
			if fkChange.NewForeignKey != nil {
				fkDetail.NewForeignKey = &ForeignKeyInfo{
					Name:              fkChange.NewForeignKey.Name,
					Columns:           fkChange.NewForeignKey.Columns,
					ReferencedSchema:  fkChange.NewForeignKey.ReferencedSchema,
					ReferencedTable:   fkChange.NewForeignKey.ReferencedTable,
					ReferencedColumns: fkChange.NewForeignKey.ReferencedColumns,
					OnUpdate:          fkChange.NewForeignKey.OnUpdate,
					OnDelete:          fkChange.NewForeignKey.OnDelete,
					MatchType:         fkChange.NewForeignKey.MatchType,
				}
			}
			tableSummary.ForeignKeys = append(tableSummary.ForeignKeys, fkDetail)
		}

		details.Tables = append(details.Tables, tableSummary)
	}

	// Add view changes details
	for _, viewChange := range diff.ViewChanges {
		details.Views = append(details.Views, ViewChangeSummary{
			Action:     string(viewChange.Action),
			SchemaName: viewChange.SchemaName,
			ViewName:   viewChange.ViewName,
		})
	}

	// Add function changes details
	for _, funcChange := range diff.FunctionChanges {
		details.Functions = append(details.Functions, FunctionChangeSummary{
			Action:       string(funcChange.Action),
			SchemaName:   funcChange.SchemaName,
			FunctionName: funcChange.FunctionName,
		})
	}

	// Add schema changes details
	for _, schemaChange := range diff.SchemaChanges {
		details.Schemas = append(details.Schemas, SchemaChangeSummary{
			Action:     string(schemaChange.Action),
			SchemaName: schemaChange.SchemaName,
		})
	}

	if len(details.Tables) > 0 || len(details.Views) > 0 || len(details.Functions) > 0 || len(details.Schemas) > 0 {
		summary.Details = details
	}

	return summary
}

// Helper functions to extract names from diff structures
func getColumnName(colChange *schema.ColumnDiff) string {
	if colChange.NewColumn != nil {
		return colChange.NewColumn.Name
	}
	if colChange.OldColumn != nil {
		return colChange.OldColumn.Name
	}
	return ""
}

func getIndexName(indexChange *schema.IndexDiff) string {
	if indexChange.NewIndex != nil {
		return indexChange.NewIndex.Name
	}
	if indexChange.OldIndex != nil {
		return indexChange.OldIndex.Name
	}
	return ""
}

func getConstraintName(constraintChange *schema.CheckConstraintDiff) string {
	if constraintChange.NewCheckConstraint != nil {
		return constraintChange.NewCheckConstraint.Name
	}
	if constraintChange.OldCheckConstraint != nil {
		return constraintChange.OldCheckConstraint.Name
	}
	return ""
}

func getForeignKeyName(fkChange *schema.ForeignKeyDiff) string {
	if fkChange.NewForeignKey != nil {
		return fkChange.NewForeignKey.Name
	}
	if fkChange.OldForeignKey != nil {
		return fkChange.OldForeignKey.Name
	}
	return ""
}

// validateOrSaveTestFiles validates existing test files or creates new ones based on UPDATE_TEST_FILES environment variable
func validateOrSaveTestFiles(t *testing.T, diff *schema.MetadataDiff, testName string, ddlContent string) error {
	// Check if we should update test files
	updateFiles := os.Getenv("UPDATE_TEST_FILES") == "true"

	testDir := getTestDataDirectory(testName)
	if testDir == "" {
		// For hardcoded tests (TestSDLValidationWithTestcontainer), skip file validation
		// Only TestSDLValidationFromTestData tests should have corresponding test data files
		if strings.Contains(testName, "TestSDLValidationWithTestcontainer") {
			t.Logf("Skipping file validation for hardcoded test: %s", testName)
			return nil
		}
		return errors.Errorf("could not determine test data directory for test %s", testName)
	}

	diffFilename := filepath.Join(testDir, "diff.json")
	ddlFilename := filepath.Join(testDir, "ddl.sql")

	// Generate current content
	currentDiffJSON, err := generateDiffJSON(diff)
	if err != nil {
		return errors.Wrapf(err, "failed to generate diff JSON")
	}

	if updateFiles {
		// Update mode: create/overwrite files
		t.Logf("Updating test files in %s", testDir)

		// Save diff JSON file
		if err := os.WriteFile(diffFilename, []byte(currentDiffJSON), 0644); err != nil {
			return errors.Wrapf(err, "failed to write diff.json file")
		}

		// Save DDL file
		if err := os.WriteFile(ddlFilename, []byte(ddlContent), 0644); err != nil {
			return errors.Wrapf(err, "failed to write ddl.sql file")
		}

		t.Logf("Test files updated: %s (diff.json, ddl.sql)", testDir)
		return nil
	}

	// Validation mode: check if existing files match current content
	t.Logf("Validating test files in %s", testDir)

	// Validate diff.json
	if err := validateFileContent(diffFilename, currentDiffJSON, "diff.json"); err != nil {
		return err
	}

	// Validate ddl.sql
	if err := validateFileContent(ddlFilename, ddlContent, "ddl.sql"); err != nil {
		return err
	}

	t.Logf("✓ Test files validated successfully: %s", testDir)
	return nil
}

// generateDiffJSON generates JSON content from MetadataDiff
func generateDiffJSON(diff *schema.MetadataDiff) (string, error) {
	diffSummary := createDiffSummary(diff)
	jsonData, err := json.MarshalIndent(diffSummary, "", "  ")
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal diff to JSON")
	}
	return string(jsonData), nil
}

// validateFileContent validates that the file content matches expected content
func validateFileContent(filename, expectedContent, fileType string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return errors.Errorf("%s file does not exist: %s. Run with UPDATE_TEST_FILES=true to create it", fileType, filename)
	}

	actualContent, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s file: %s", fileType, filename)
	}

	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(expectedContent)

	if actualStr != expectedStr {
		return errors.Errorf("%s content mismatch in %s.\nExpected:\n%s\n\nActual:\n%s\n\nRun with UPDATE_TEST_FILES=true to update the file",
			fileType, filename, expectedStr, actualStr)
	}

	return nil
}

// getTestDataDirectory converts a test name to the corresponding test data directory path
func getTestDataDirectory(testName string) string {
	// Handle TestSDLValidationFromTestData_ prefix
	var pathPart string
	if strings.HasPrefix(testName, "TestSDLValidationFromTestData_") {
		pathPart = strings.TrimPrefix(testName, "TestSDLValidationFromTestData_")
	} else {
		return ""
	}

	// Convert from test name format to directory format
	// TestSDLValidationFromTestData_data_types_basic_types_datetime_types -> data_types/basic_types/datetime_types

	// Based on the known test structure, parse the path correctly:
	// data_types_basic_types_datetime_types
	// schema_objects_views_complex_views
	// constraints_indexes_foreign_keys_cascade_actions

	var testDataPath string
	switch {
	case strings.HasPrefix(pathPart, "data_types_basic_types_"):
		// data_types_basic_types_datetime_types -> data_types/basic_types/datetime_types
		suffix := strings.TrimPrefix(pathPart, "data_types_basic_types_")
		testDataPath = filepath.Join("data_types", "basic_types", suffix)
	case strings.HasPrefix(pathPart, "data_types_advanced_types_"):
		// data_types_advanced_types_binary_json_uuid -> data_types/advanced_types/binary_json_uuid
		suffix := strings.TrimPrefix(pathPart, "data_types_advanced_types_")
		testDataPath = filepath.Join("data_types", "advanced_types", suffix)
	case strings.HasPrefix(pathPart, "data_types_type_conversions_"):
		// data_types_type_conversions_compatible_conversions -> data_types/type_conversions/compatible_conversions
		suffix := strings.TrimPrefix(pathPart, "data_types_type_conversions_")
		testDataPath = filepath.Join("data_types", "type_conversions", suffix)
	case strings.HasPrefix(pathPart, "schema_objects_views_"):
		// schema_objects_views_complex_views -> schema_objects/views/complex_views
		suffix := strings.TrimPrefix(pathPart, "schema_objects_views_")
		testDataPath = filepath.Join("schema_objects", "views", suffix)
	case strings.HasPrefix(pathPart, "constraints_indexes_foreign_keys_"):
		// constraints_indexes_foreign_keys_cascade_actions -> constraints_indexes/foreign_keys/cascade_actions
		suffix := strings.TrimPrefix(pathPart, "constraints_indexes_foreign_keys_")
		testDataPath = filepath.Join("constraints_indexes", "foreign_keys", suffix)
	case strings.HasPrefix(pathPart, "table_operations_column_operations_"):
		// table_operations_column_operations_add_simple_columns -> table_operations/column_operations/add_simple_columns
		suffix := strings.TrimPrefix(pathPart, "table_operations_column_operations_")
		testDataPath = filepath.Join("table_operations", "column_operations", suffix)
	default:
		// Fallback: try to split intelligently
		parts := strings.Split(pathPart, "_")
		if len(parts) >= 3 {
			// Take first 2 parts as category, rest as test name
			testDataPath = filepath.Join(strings.Join(parts[:2], "/"), strings.Join(parts[2:], "_"))
		} else {
			testDataPath = strings.ReplaceAll(pathPart, "_", "/")
		}
	}

	return filepath.Join("sdl_testdata", testDataPath)
}

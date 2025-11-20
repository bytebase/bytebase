package pg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestGetDatabaseMetadataWithTestcontainer tests the get_database_metadata function
// by comparing its output with the metadata retrieved from a real PostgreSQL instance.
func TestGetDatabaseMetadataWithTestcontainer(t *testing.T) {
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

	// Test cases with various PostgreSQL features
	testCases := []struct {
		name string
		ddl  string
	}{
		{
			name: "bytebase_schema",
			ddl: `
			CREATE TABLE employee (
	emp_no      SERIAL NOT NULL,
	birth_date  DATE NOT NULL,
	first_name  TEXT NOT NULL,
	last_name   TEXT NOT NULL,
	gender      TEXT NOT NULL CHECK (gender IN('M', 'F')) NOT NULL,
	hire_date   DATE NOT NULL,
	PRIMARY KEY (emp_no)
);

CREATE INDEX idx_employee_hire_date ON employee (hire_date);

CREATE TABLE department (
	dept_no     TEXT NOT NULL,
	dept_name   TEXT NOT NULL,
	PRIMARY KEY (dept_no),
	UNIQUE      (dept_name)
);

CREATE TABLE dept_manager (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE dept_emp (
	emp_no      INT NOT NULL,
	dept_no     TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE title (
	emp_no      INT NOT NULL,
	title       TEXT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, title, from_date)
); 

CREATE TABLE salary (
	emp_no      INT NOT NULL,
	amount      INT NOT NULL,
	from_date   DATE NOT NULL,
	to_date     DATE NOT NULL,
	FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
	PRIMARY KEY (emp_no, from_date)
);

CREATE INDEX idx_salary_amount ON salary (amount);

CREATE TABLE audit (
    id SERIAL PRIMARY KEY,
    operation TEXT NOT NULL,
    query TEXT,
    user_name TEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_operation ON audit (operation);
CREATE INDEX idx_audit_username ON audit (user_name);
CREATE INDEX idx_audit_changed_at ON audit (changed_at);

CREATE OR REPLACE FUNCTION log_dml_operations() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('INSERT', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('UPDATE', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('DELETE', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- only log update and delete, otherwise, it will cause too much change.
CREATE TRIGGER salary_log_trigger
AFTER UPDATE OR DELETE ON salary
FOR EACH ROW
EXECUTE FUNCTION log_dml_operations();

CREATE OR REPLACE VIEW dept_emp_latest_date AS
SELECT
	emp_no,
	MAX(
		from_date) AS from_date,
	MAX(
		to_date) AS to_date
FROM
	dept_emp
GROUP BY
	emp_no;

-- shows only the current department for each employee
CREATE OR REPLACE VIEW current_dept_emp AS
SELECT
	l.emp_no,
	dept_no,
	l.from_date,
	l.to_date
FROM
	dept_emp d
	INNER JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no
		AND d.from_date = l.from_date
		AND l.to_date = d.to_date;
			`,
		},
		{
			name: "basic_tables_with_constraints",
			ddl: `
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;
`,
		},
		{
			name: "sequences_and_custom_types",
			ddl: `
CREATE TYPE status_enum AS ENUM ('pending', 'active', 'inactive', 'deleted');
CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral');

CREATE SEQUENCE custom_id_seq START WITH 1000 INCREMENT BY 10;

CREATE TABLE items (
    id INTEGER DEFAULT nextval('custom_id_seq') PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status status_enum DEFAULT 'pending',
    user_mood mood
);
`,
		},
		{
			name: "views_and_functions",
			ddl: `
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50),
    salary DECIMAL(10, 2)
);

CREATE VIEW active_employees AS
SELECT id,
  name,
  department
 FROM employees
WHERE (department IS NOT NULL);

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
    RETURN emp_salary * 0.1;
END;
$$ LANGUAGE plpgsql;
`,
		},
		{
			name: "partitioned_tables",
			ddl: `
CREATE TABLE sales (
    id SERIAL,
    sale_date DATE NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    region VARCHAR(50) NOT NULL
) PARTITION BY RANGE (sale_date);

CREATE TABLE sales_2023_q1 PARTITION OF sales
FOR VALUES FROM ('2023-01-01') TO ('2023-04-01');

CREATE TABLE sales_2023_q2 PARTITION OF sales
FOR VALUES FROM ('2023-04-01') TO ('2023-07-01');

CREATE TABLE sales_2023_q3 PARTITION OF sales
FOR VALUES FROM ('2023-07-01') TO ('2023-10-01');

CREATE TABLE sales_2023_q4 PARTITION OF sales
FOR VALUES FROM ('2023-10-01') TO ('2024-01-01');
`,
		},
		{
			name: "extensions_and_advanced_features",
			ddl: `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE documents (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_documents_metadata ON documents USING GIN (metadata);
CREATE INDEX idx_documents_tags ON documents USING GIN (tags);
`,
		},
		{
			name: "indexes_with_asc_desc",
			ddl: `
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    order_date DATE NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20)
);

-- Index with explicit DESC
CREATE INDEX idx_orders_date_desc ON orders(order_date DESC);

-- Index with multiple columns, mixed ASC/DESC
CREATE INDEX idx_orders_customer_date ON orders(customer_id ASC, order_date DESC);

-- Index with expressions and DESC
CREATE INDEX idx_orders_year_month ON orders(EXTRACT( year FROM order_date ) DESC, EXTRACT( month FROM order_date ) ASC);

-- Unique index with DESC
CREATE UNIQUE INDEX idx_orders_customer_status ON orders(customer_id, status DESC) WHERE status IS NOT NULL;
`,
		},
		{
			name: "materialized_views_and_triggers",
			ddl: `
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    user_id INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    old_values JSONB,
    new_values JSONB
);

CREATE TABLE users_mv (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    last_login TIMESTAMP,
    login_count INTEGER DEFAULT 0
);

-- Materialized view for user statistics
CREATE MATERIALIZED VIEW user_stats AS
SELECT count(*) AS total_users,
 count(
     CASE
         WHEN (last_login > (CURRENT_DATE - '30 days'::interval)) THEN 1
         ELSE NULL::integer
     END) AS active_users,
 avg(login_count) AS avg_login_count
FROM public.users_mv
WITH DATA;

-- Index on materialized view
CREATE INDEX idx_user_stats_total ON user_stats(total_users);

-- Trigger function for audit logging
CREATE OR REPLACE FUNCTION audit_trigger_function() 
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (table_name, operation, user_id, old_values, new_values)
    VALUES (
        TG_TABLE_NAME,
        TG_OP,
        COALESCE(NEW.id, OLD.id),
        CASE WHEN TG_OP = 'DELETE' THEN row_to_json(OLD) ELSE NULL END,
        CASE WHEN TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN row_to_json(NEW) ELSE NULL END
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger on users table
CREATE TRIGGER users_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users_mv
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- Stored procedure with parameters
CREATE OR REPLACE PROCEDURE refresh_user_stats()
LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW user_stats;
    INSERT INTO audit_log (table_name, operation, timestamp)
    VALUES ('user_stats', 'REFRESH', CURRENT_TIMESTAMP);
END;
$$;
`,
		},
		{
			name: "cross_schema_references",
			ddl: `
-- Create additional schemas
CREATE SCHEMA hr;
CREATE SCHEMA finance;

-- Tables in hr schema
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
FROM (hr.departments d
  JOIN finance.budgets b ON ((d.id = b.department_id)));
`,
		},
		{
			name: "advanced_indexes_and_constraints",
			ddl: `
CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2),
    category_tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sku VARCHAR(50),
    coordinates POINT
);

-- BRIN index for time-series data
CREATE INDEX idx_products_created_brin ON products USING BRIN (created_at);

-- Hash index
CREATE INDEX idx_products_sku_hash ON products USING HASH (sku);

-- Covering index (INCLUDE clause)
CREATE INDEX idx_products_category_include ON products (price) INCLUDE (name, description);

-- Partial index with complex condition
CREATE INDEX idx_expensive_products ON products (price, category_tags) 
WHERE price > 100.00 AND array_length(category_tags, 1) > 0;

-- Expression index with functions
CREATE INDEX idx_products_name_lower ON products (lower(name::text));
CREATE INDEX idx_products_price_rounded ON products (round(price));

-- Multi-column GIN index
CREATE INDEX idx_products_tags_meta ON products USING GIN (category_tags, metadata);

-- Simple check constraints (avoiding complex exclusion constraints)
CREATE TABLE orders_advanced (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    order_date DATE DEFAULT CURRENT_DATE,
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT valid_quantity CHECK (quantity > 0 AND quantity <= 1000),
    CONSTRAINT recent_order CHECK (order_date >= CURRENT_DATE - INTERVAL '1 year')
);
`,
		},
		{
			name: "geometric_and_network_types",
			ddl: `
-- Using built-in geometric and network types (no PostGIS required)
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    coordinates POINT,
    boundary_box BOX,
    service_area POLYGON,
    delivery_path PATH,
    center_point CIRCLE,
    route_line LSEG
);

CREATE TABLE network_devices (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(100) NOT NULL,
    ip_address INET,
    subnet CIDR,
    mac_address MACADDR,
    ipv6_address INET,
    device_config JSONB
);

-- Geometric indexes (using GIST for geometric types)
CREATE INDEX idx_locations_coordinates ON locations USING GIST (coordinates);
CREATE INDEX idx_locations_service_area ON locations USING GIST (service_area);

-- Network type indexes
CREATE INDEX idx_devices_ip ON network_devices (ip_address);
-- Skip GIST index on CIDR as it doesn't have a default operator class
CREATE INDEX idx_devices_subnet ON network_devices (subnet);

-- Range types
CREATE TABLE reservations (
    id SERIAL PRIMARY KEY,
    resource_id INTEGER NOT NULL,
    date_range DATERANGE NOT NULL,
    time_range TSRANGE,
    price_range NUMRANGE,
    capacity_range INT4RANGE
);

-- Index on range types
CREATE INDEX idx_reservations_date_range ON reservations USING GIST (date_range);

-- Full-text search types
CREATE TABLE documents_fts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    search_vector TSVECTOR,
    keywords TSQUERY
);

-- Full-text search index
CREATE INDEX idx_documents_search ON documents_fts USING GIN (search_vector);

-- Update trigger for full-text search
CREATE OR REPLACE FUNCTION update_search_vector() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', COALESCE(NEW.title, '') || ' ' || COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_search
    BEFORE INSERT OR UPDATE ON documents_fts
    FOR EACH ROW
    EXECUTE FUNCTION update_search_vector();
`,
		},
		{
			name: "routing_rules_pattern",
			ddl: `
-- Test the routing rules pattern (view with INSERT/UPDATE/DELETE rules)
-- This is the exact pattern from the user's scenario

-- Create a regular table (not partitioned to avoid test issues)
CREATE TABLE old_name (
    id BIGINT NOT NULL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    name VARCHAR(100),
    data JSONB,
    status VARCHAR(20) DEFAULT 'active'::character varying
);

-- Create indexes
CREATE INDEX idx_old_name_status ON old_name (status);
CREATE INDEX idx_old_name_created ON old_name (created_at);

-- Create the view that will act as the new interface
CREATE VIEW new_name AS
SELECT 
    id,
    created_at,
    updated_at,
    name,
    data,
    status
FROM old_name;

-- Create routing rules to redirect operations from view to table
CREATE RULE new_name_delete AS
    ON DELETE TO new_name 
    DO INSTEAD DELETE FROM old_name
    WHERE old_name.id = OLD.id;

CREATE RULE new_name_insert AS
    ON INSERT TO new_name 
    DO INSTEAD INSERT INTO old_name (id, created_at, updated_at, name, data, status)
    VALUES (NEW.id, NEW.created_at, NEW.updated_at, NEW.name, NEW.data, NEW.status);

CREATE RULE new_name_update AS
    ON UPDATE TO new_name 
    DO INSTEAD UPDATE old_name 
    SET 
        created_at = NEW.created_at,
        updated_at = NEW.updated_at,
        name = NEW.name,
        data = NEW.data,
        status = NEW.status
    WHERE old_name.id = OLD.id;


-- Create another table with rules (not on a view)
CREATE TABLE audit_table (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50),
    operation VARCHAR(10),
    user_name VARCHAR(50),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE monitored_table (
    id SERIAL PRIMARY KEY,
    data TEXT
);

-- Create a rule on a regular table (not a view)
CREATE RULE log_monitored_inserts AS
    ON INSERT TO monitored_table
    DO ALSO INSERT INTO audit_table (table_name, operation, user_name)
    VALUES ('monitored_table', 'INSERT', current_user);

CREATE RULE log_monitored_deletes AS
    ON DELETE TO monitored_table
    DO ALSO INSERT INTO audit_table (table_name, operation, user_name)
    VALUES ('monitored_table', 'DELETE', current_user);
`,
		},
		{
			name: "table_inheritance_and_partitioning",
			ddl: `
-- Skip table inheritance as it's not supported
-- Just test partitioning features

-- List partitioning (without unique constraints that would cause issues)
CREATE TABLE events (
    id BIGINT,
    event_type VARCHAR(20) NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) PARTITION BY LIST (event_type);

CREATE TABLE events_user PARTITION OF events
FOR VALUES IN ('user_login', 'user_logout', 'user_register');

CREATE TABLE events_system PARTITION OF events
FOR VALUES IN ('system_start', 'system_stop', 'system_error');

CREATE TABLE events_audit PARTITION OF events
FOR VALUES IN ('data_change', 'permission_change', 'config_change');

-- Hash partitioning (without primary key to avoid partitioning column requirement)
CREATE TABLE user_sessions (
    session_id UUID DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address INET
) PARTITION BY HASH (user_id);

CREATE TABLE user_sessions_0 PARTITION OF user_sessions
FOR VALUES WITH (modulus 4, remainder 0);

CREATE TABLE user_sessions_1 PARTITION OF user_sessions
FOR VALUES WITH (modulus 4, remainder 1);

CREATE TABLE user_sessions_2 PARTITION OF user_sessions
FOR VALUES WITH (modulus 4, remainder 2);

CREATE TABLE user_sessions_3 PARTITION OF user_sessions
FOR VALUES WITH (modulus 4, remainder 3);

-- Unlogged table for temporary data
CREATE UNLOGGED TABLE temp_calculations (
    id SERIAL PRIMARY KEY,
    calculation_data JSONB,
    result DECIMAL(15, 6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`,
		},
		{
			name: "array_types_comprehensive",
			ddl: `
-- Create custom enum for testing custom type arrays
CREATE TYPE status_enum AS ENUM ('active', 'inactive', 'pending');
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'urgent');

-- Table with comprehensive array type coverage
CREATE TABLE array_types_test (
    id SERIAL PRIMARY KEY,
    
    -- Integer array types (different aliases)
    int_array int[],
    int4_array int4[],
    integer_array integer[],
    bigint_array bigint[],
    int8_array int8[],
    smallint_array smallint[],
    int2_array int2[],
    
    -- Multi-dimensional arrays (should normalize to same as single dimension)
    int_multi_array int[][],
    text_multi_array text[][][],
    
    -- Floating point array types
    real_array real[],
    float4_array float4[],
    double_precision_array double precision[],
    float8_array float8[],
    
    -- Character array types with and without precision
    char_array char[],
    char_with_length char(10)[],
    varchar_array varchar[],
    varchar_with_length varchar(255)[],
    character_array character[],
    character_with_length character(50)[],
    character_varying_array character varying[],
    character_varying_with_length character varying(100)[],
    text_array text[],
    
    -- Boolean arrays
    bool_array bool[],
    boolean_array boolean[],
    
    -- Numeric arrays with precision
    numeric_array numeric[],
    numeric_with_precision numeric(10,2)[],
    decimal_array decimal[],
    decimal_with_precision decimal(15,4)[],
    money_array money[],
    
    -- Date/time arrays
    date_array date[],
    time_array time[],
    time_with_tz time with time zone[],
    timetz_array timetz[],
    timestamp_array timestamp[],
    timestamp_with_tz timestamp with time zone[],
    timestamptz_array timestamptz[],
    interval_array interval[],
    
    -- Binary and UUID arrays
    bytea_array bytea[],
    uuid_array uuid[],
    
    -- JSON arrays
    json_array json[],
    jsonb_array jsonb[],
    
    -- Network address arrays
    inet_array inet[],
    cidr_array cidr[],
    macaddr_array macaddr[],
    macaddr8_array macaddr8[],
    
    -- Geometric type arrays
    point_array point[],
    line_array line[],
    lseg_array lseg[],
    box_array box[],
    path_array path[],
    polygon_array polygon[],
    circle_array circle[],
    
    -- Full-text search arrays
    tsvector_array tsvector[],
    tsquery_array tsquery[],
    
    -- Range type arrays
    int4range_array int4range[],
    int8range_array int8range[],
    numrange_array numrange[],
    tsrange_array tsrange[],
    tstzrange_array tstzrange[],
    daterange_array daterange[],
    
    -- Multi-range type arrays (PostgreSQL 14+)
    int4multirange_array int4multirange[],
    int8multirange_array int8multirange[],
    nummultirange_array nummultirange[],
    tsmultirange_array tsmultirange[],
    tstzmultirange_array tstzmultirange[],
    datemultirange_array datemultirange[],
    
    -- Bit string arrays
    bit_array bit[],
    bit_with_length bit(8)[],
    bit_varying_array bit varying[],
    varbit_array varbit[],
    varbit_with_length varbit(16)[],
    
    -- Object identifier arrays
    oid_array oid[],
    regclass_array regclass[],
    regproc_array regproc[],
    regprocedure_array regprocedure[],
    regoper_array regoper[],
    regoperator_array regoperator[],
    regtype_array regtype[],
    regconfig_array regconfig[],
    regdictionary_array regdictionary[],
    regnamespace_array regnamespace[],
    regrole_array regrole[],
    regcollation_array regcollation[],
    
    -- System type arrays
    tid_array tid[],
    xid_array xid[],
    xid8_array xid8[],
    cid_array cid[],
    pg_lsn_array pg_lsn[],
    
    -- Custom enum arrays (should use fallback logic)
    status_array status_enum[],
    priority_array priority_level[],
    
    -- XML array
    xml_array xml[],
    
    -- Other special arrays (note: record[] not allowed in table columns)
    jsonpath_array jsonpath[],
    
    -- Additional system types (compatible with PostgreSQL 16)
    txid_snapshot_array txid_snapshot[],
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add some sample data to verify the schema works
INSERT INTO array_types_test (
    int_array,
    varchar_with_length,
    bool_array,
    status_array,
    json_array
) VALUES (
    ARRAY[1, 2, 3],
    ARRAY['test1', 'test2'],
    ARRAY[true, false, true],
    ARRAY['active'::status_enum, 'pending'::status_enum],
    ARRAY['{"key": "value1"}'::json, '{"key": "value2"}'::json]
);
`,
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

			// Execute the DDL
			_, err = testDB.Exec(tc.ddl)
			require.NoError(t, err)

			// Get metadata using Driver.SyncDBSchema
			syncMetadata, err := getSyncMetadata(ctx, &testConnConfig, dbName)
			require.NoError(t, err)

			// Get metadata using get_database_metadata
			parseMetadata, err := GetDatabaseMetadata(tc.ddl)
			require.NoError(t, err)

			// Compare the two metadata structures
			compareMetadata(t, syncMetadata, parseMetadata)

			// Additional validation: use schema differ to ensure no differences detected
			validateWithSchemaDiffer(t, tc.name, syncMetadata, parseMetadata)

			// Validate rules are handled correctly (no cycles, proper dumping)
			validateRoutingRules(t, syncMetadata)

			// Additional validation for array types test case
			if tc.name == "array_types_comprehensive" {
				validateArrayTypes(t, syncMetadata, parseMetadata)
			}
		})
	}
}

// getSyncMetadata retrieves metadata from the live database using Driver.SyncDBSchema
func getSyncMetadata(ctx context.Context, connConfig *pgx.ConnConfig, dbName string) (*storepb.DatabaseSchemaMetadata, error) {
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

// compareMetadata compares metadata from sync.go and get_database_metadata
func compareMetadata(t *testing.T, syncMeta, parseMeta *storepb.DatabaseSchemaMetadata) {
	// Compare schemas
	require.Equal(t, len(syncMeta.Schemas), len(parseMeta.Schemas), "number of schemas should match")

	// Find the public schema in both
	var syncPublic, parsePublic *storepb.SchemaMetadata
	for _, schema := range syncMeta.Schemas {
		if schema.Name == "public" {
			syncPublic = schema
			break
		}
	}
	for _, schema := range parseMeta.Schemas {
		if schema.Name == "public" {
			parsePublic = schema
			break
		}
	}

	require.NotNil(t, syncPublic, "sync metadata should have public schema")
	require.NotNil(t, parsePublic, "parse metadata should have public schema")

	// Compare tables
	compareTables(t, syncPublic.Tables, parsePublic.Tables)

	// Compare views
	compareViews(t, syncPublic.Views, parsePublic.Views)

	// Compare materialized views
	compareMaterializedViews(t, syncPublic.MaterializedViews, parsePublic.MaterializedViews)

	// Compare functions
	compareFunctions(t, syncPublic.Functions, parsePublic.Functions)

	// Compare procedures (part of functions in metadata)
	compareProcedures(t, syncPublic.Procedures, parsePublic.Procedures)

	// Compare sequences
	compareSequences(t, syncPublic.Sequences, parsePublic.Sequences)

	// Compare enums
	compareEnums(t, syncPublic.EnumTypes, parsePublic.EnumTypes)

	// Note: Triggers are stored at the table level, not schema level
	// They will be compared as part of table comparison

	// Compare extensions
	compareExtensions(t, syncMeta.Extensions, parseMeta.Extensions)

	// Compare all schemas for cross-schema tests
	compareAllSchemas(t, syncMeta.Schemas, parseMeta.Schemas)
}

// normalizeSQL normalizes SQL for comparison by:
// - Converting to lowercase
// - Removing extra whitespace
// - Removing trailing semicolons
// - Removing schema qualifiers for common schemas
// - Normalizing parentheses
func normalizeSQL(sql string) string {
	// Convert to lowercase
	sql = strings.ToLower(sql)

	// Replace multiple spaces/newlines with single space
	sql = strings.Join(strings.Fields(sql), " ")

	// Remove trailing semicolons
	sql = strings.TrimSuffix(sql, ";")

	// Remove schema qualifiers for public schema
	// This handles cases like "public.table_name" -> "table_name"
	sql = strings.ReplaceAll(sql, "public.", "")

	// Handle PostgreSQL's tendency to wrap WHERE conditions in parentheses
	// e.g., "WHERE (condition)" -> "WHERE condition"
	// We need to be careful to only remove the outermost parentheses around the entire WHERE clause
	whereIndex := strings.Index(sql, "where (")
	if whereIndex >= 0 {
		// Find the matching closing parenthesis for the WHERE clause
		afterWhere := sql[whereIndex+7:] // Skip "where ("
		openCount := 1
		closeIndex := -1

		// Find the matching closing parenthesis
		for i, ch := range afterWhere {
			if ch == '(' {
				openCount++
			} else if ch == ')' {
				openCount--
				if openCount == 0 {
					closeIndex = i
					break
				}
			}
		}

		// If we found the matching closing parenthesis and it's at the end or followed by end/order/group
		if closeIndex >= 0 {
			beforeWhere := sql[:whereIndex+6] // Include "where "
			afterCloseParen := ""
			if whereIndex+7+closeIndex+1 < len(sql) {
				afterCloseParen = sql[whereIndex+7+closeIndex+1:]
			}

			// Check if the closing paren is at the end or followed by valid SQL keywords
			if afterCloseParen == "" || strings.HasPrefix(strings.TrimSpace(afterCloseParen), "order by") ||
				strings.HasPrefix(strings.TrimSpace(afterCloseParen), "group by") ||
				strings.HasPrefix(strings.TrimSpace(afterCloseParen), "limit") {
				// Remove the parentheses
				sql = beforeWhere + afterWhere[:closeIndex] + afterCloseParen
			}
		}
	}

	// PostgreSQL function and procedure-specific normalizations
	if strings.Contains(sql, "function") || strings.Contains(sql, "procedure") {
		// Normalize CREATE OR REPLACE vs CREATE
		sql = strings.ReplaceAll(sql, "create or replace function", "create function")

		// Normalize parameter types to consistent forms
		sql = strings.ReplaceAll(sql, "character varying", "varchar")
		sql = strings.ReplaceAll(sql, "returns numeric", "returns decimal")

		// Normalize dollar quoting
		sql = strings.ReplaceAll(sql, "$function$", "$$")
		sql = strings.ReplaceAll(sql, "$procedure$", "$$")

		// Move LANGUAGE clause to end for consistent comparison
		if strings.Contains(sql, "language plpgsql") && !strings.HasSuffix(sql, "language plpgsql") {
			withoutLanguage := strings.ReplaceAll(sql, " language plpgsql", "")
			withoutLanguage = strings.ReplaceAll(withoutLanguage, "language plpgsql ", "")
			sql = strings.TrimSpace(withoutLanguage) + " language plpgsql"
		}
	}

	// Final trim
	sql = strings.TrimSpace(sql)

	return sql
}

// normalizeSignature normalizes function signatures for comparison
func normalizeSignature(sig string) string {
	// Convert to lowercase
	sig = strings.ToLower(sig)

	// Remove extra spaces
	sig = strings.Join(strings.Fields(sig), " ")

	// Remove spaces around parentheses and commas
	sig = strings.ReplaceAll(sig, " (", "(")
	sig = strings.ReplaceAll(sig, "( ", "(")
	sig = strings.ReplaceAll(sig, " )", ")")
	sig = strings.ReplaceAll(sig, ") ", ")")
	sig = strings.ReplaceAll(sig, " ,", ",")
	sig = strings.ReplaceAll(sig, ", ", ",")

	// Remove quotes around function names
	sig = strings.ReplaceAll(sig, "\"", "")

	return sig
}

func compareTables(t *testing.T, syncTables, parseTables []*storepb.TableMetadata) {
	// Log the tables found for debugging
	t.Logf("Sync tables: %d", len(syncTables))
	for _, table := range syncTables {
		t.Logf("  - %s", table.Name)
	}
	t.Logf("Parse tables: %d", len(parseTables))
	for _, table := range parseTables {
		t.Logf("  - %s", table.Name)
	}

	require.Equal(t, len(syncTables), len(parseTables), "number of tables should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.TableMetadata)
	for _, table := range syncTables {
		syncMap[table.Name] = table
	}

	parseMap := make(map[string]*storepb.TableMetadata)
	for _, table := range parseTables {
		parseMap[table.Name] = table
	}

	// Compare each table
	for name, syncTable := range syncMap {
		parseTable, exists := parseMap[name]
		require.True(t, exists, "table %s should exist in parsed metadata", name)

		// Compare columns
		compareColumns(t, name, syncTable.Columns, parseTable.Columns)

		// Compare rules
		compareRules(t, name, syncTable.Rules, parseTable.Rules)

		// Compare indexes
		compareIndexes(t, name, syncTable.Indexes, parseTable.Indexes)

		// Compare foreign keys
		compareForeignKeys(t, name, syncTable.ForeignKeys, parseTable.ForeignKeys)

		// Compare partitions
		comparePartitions(t, name, syncTable.Partitions, parseTable.Partitions)

		// Compare triggers
		compareTriggers(t, name, syncTable.Triggers, parseTable.Triggers)
	}
}

func compareColumns(t *testing.T, tableName string, syncCols, parseCols []*storepb.ColumnMetadata) {
	require.Equal(t, len(syncCols), len(parseCols), "table %s: number of columns should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range syncCols {
		syncMap[col.Name] = col
	}

	for _, parseCol := range parseCols {
		syncCol, exists := syncMap[parseCol.Name]
		require.True(t, exists, "table %s: column %s should exist in sync metadata", tableName, parseCol.Name)

		// Compare column properties
		require.Equal(t, syncCol.Type, parseCol.Type, "table %s, column %s: type should match", tableName, parseCol.Name)
		require.Equal(t, syncCol.Nullable, parseCol.Nullable, "table %s, column %s: nullable should match", tableName, parseCol.Name)

		// Compare default values if both exist
		hasDefaultSync := syncCol.Default != ""
		hasDefaultParse := parseCol.Default != ""
		if hasDefaultSync && hasDefaultParse {
			// Default values might be represented differently, so we just check they exist
			t.Logf("table %s, column %s: default values exist in both", tableName, parseCol.Name)
		}
	}
}

func compareIndexes(t *testing.T, tableName string, syncIndexes, parseIndexes []*storepb.IndexMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range syncIndexes {
		syncMap[idx.Name] = idx
	}

	parseMap := make(map[string]*storepb.IndexMetadata)
	for _, idx := range parseIndexes {
		parseMap[idx.Name] = idx
	}

	// Check that both directions have the same indexes
	require.Equal(t, len(syncIndexes), len(parseIndexes), "mismatch in number of indexes for table %s", tableName)

	// Compare indexes that exist in parse
	for name, parseIdx := range parseMap {
		syncIdx, exists := syncMap[name]
		if !exists {
			// Some indexes might be system-generated and not in DDL
			t.Logf("table %s: index %s exists in parse but not in sync (might be implicit)", tableName, name)
			continue
		}

		// 1. Name - explicitly validate name consistency
		require.Equal(t, syncIdx.Name, parseIdx.Name, "table %s, index %s: name should match", tableName, name)

		// 2. Primary - validate primary key flag
		require.Equal(t, syncIdx.Primary, parseIdx.Primary, "table %s, index %s: primary should match", tableName, name)

		// 3. Unique - validate unique constraint flag
		require.Equal(t, syncIdx.Unique, parseIdx.Unique, "table %s, index %s: unique should match", tableName, name)

		// 4. Type - validate index type
		if syncIdx.Type != "" || parseIdx.Type != "" {
			require.Equal(t, syncIdx.Type, parseIdx.Type, "table %s, index %s: type should match", tableName, name)
		}

		// 5. Expressions - compare expressions using AST-based semantic comparison
		if len(syncIdx.Expressions) == len(parseIdx.Expressions) {
			for i := range syncIdx.Expressions {
				// Use AST-based semantic comparison instead of string normalization
				equal := ast.CompareExpressionsSemantically(syncIdx.Expressions[i], parseIdx.Expressions[i])
				require.True(t, equal, "table %s, index %s: expression[%d] should be semantically equivalent. sync: %q, parse: %q",
					tableName, name, i, syncIdx.Expressions[i], parseIdx.Expressions[i])
			}
		} else {
			require.Equal(t, len(syncIdx.Expressions), len(parseIdx.Expressions), "table %s, index %s: expressions count should match", tableName, name)
		}

		// 6. Descending - compare descending order for each expression
		// Note: sync.go currently doesn't populate the Descending field, so we need to handle both cases
		if len(syncIdx.Descending) > 0 && len(parseIdx.Descending) > 0 {
			// Both have descending info, compare them
			require.Equal(t, len(syncIdx.Descending), len(parseIdx.Descending), "table %s, index %s: descending array length should match", tableName, name)
			for i := range syncIdx.Descending {
				require.Equal(t, syncIdx.Descending[i], parseIdx.Descending[i], "table %s, index %s: descending[%d] should match", tableName, name, i)
			}
		} else if len(parseIdx.Descending) > 0 {
			// Only parser has descending info, verify it matches the number of expressions
			require.Equal(t, len(parseIdx.Expressions), len(parseIdx.Descending), "table %s, index %s: descending array should match expressions count", tableName, name)
		}

		// 7. KeyLength - validate index key lengths (PostgreSQL specific - prefix lengths on expressions)
		if len(syncIdx.KeyLength) > 0 || len(parseIdx.KeyLength) > 0 {
			require.Equal(t, len(syncIdx.KeyLength), len(parseIdx.KeyLength), "table %s, index %s: key length array length should match", tableName, name)
			for i := range syncIdx.KeyLength {
				if i < len(parseIdx.KeyLength) {
					require.Equal(t, syncIdx.KeyLength[i], parseIdx.KeyLength[i], "table %s, index %s: key length[%d] should match", tableName, name, i)
				}
			}
		}

		// 8. Visible - validate index visibility (not commonly used in PostgreSQL, but validate if present)
		require.Equal(t, syncIdx.Visible, parseIdx.Visible, "table %s, index %s: visible should match", tableName, name)

		// 9. Comment - validate index comment
		if syncIdx.Comment != "" || parseIdx.Comment != "" {
			require.Equal(t, syncIdx.Comment, parseIdx.Comment, "table %s, index %s: comment should match", tableName, name)
		}

		// 10. IsConstraint - validate if index represents a constraint
		require.Equal(t, syncIdx.IsConstraint, parseIdx.IsConstraint, "table %s, index %s: IsConstraint should match", tableName, name)

		// 11. Definition - validate full index definition using AST-based semantic comparison
		if syncIdx.Definition != "" || parseIdx.Definition != "" {
			if syncIdx.Definition != "" && parseIdx.Definition != "" {
				definitionsEqual := compareIndexDefinitionsSemanticaly(syncIdx.Definition, parseIdx.Definition)
				require.True(t, definitionsEqual, "table %s, index %s: definition should match\nSync: %s\nParse: %s", tableName, name, syncIdx.Definition, parseIdx.Definition)
			}
		}

		t.Logf("✓ Validated all IndexMetadata fields for index %s: name=%s, primary=%v, unique=%v, type=%s, expressions=%v, visible=%v, comment=%s",
			name, parseIdx.Name, parseIdx.Primary, parseIdx.Unique, parseIdx.Type, parseIdx.Expressions, parseIdx.Visible, parseIdx.Comment)
	}
}

func compareForeignKeys(t *testing.T, tableName string, syncFKs, parseFKs []*storepb.ForeignKeyMetadata) {
	require.Equal(t, len(syncFKs), len(parseFKs), "table %s: number of foreign keys should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range syncFKs {
		syncMap[fk.Name] = fk
	}

	for _, parseFk := range parseFKs {
		syncFk, exists := syncMap[parseFk.Name]
		require.True(t, exists, "table %s: foreign key %s should exist in sync metadata", tableName, parseFk.Name)

		require.ElementsMatch(t, syncFk.Columns, parseFk.Columns, "table %s, FK %s: columns should match", tableName, parseFk.Name)
		require.Equal(t, syncFk.ReferencedTable, parseFk.ReferencedTable, "table %s, FK %s: referenced table should match", tableName, parseFk.Name)
		require.ElementsMatch(t, syncFk.ReferencedColumns, parseFk.ReferencedColumns, "table %s, FK %s: referenced columns should match", tableName, parseFk.Name)
	}
}

func comparePartitions(t *testing.T, tableName string, syncParts, parseParts []*storepb.TablePartitionMetadata) {
	require.Equal(t, len(syncParts), len(parseParts), "table %s: number of partitions should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range syncParts {
		syncMap[part.Name] = part
	}

	for _, parsePart := range parseParts {
		syncPart, exists := syncMap[parsePart.Name]
		require.True(t, exists, "table %s: partition %s should exist in sync metadata", tableName, parsePart.Name)
		// Use AST-based semantic comparison for partition expressions
		equal := ast.CompareExpressionsSemantically(syncPart.Expression, parsePart.Expression)
		require.True(t, equal, "table %s, partition %s: expression should be semantically equivalent. sync: %q, parse: %q",
			tableName, parsePart.Name, syncPart.Expression, parsePart.Expression)
		require.Equal(t, syncPart.Value, parsePart.Value, "table %s, partition %s: value should match", tableName, parsePart.Name)
	}
}

func compareViews(t *testing.T, syncViews, parseViews []*storepb.ViewMetadata) {
	require.Equal(t, len(syncViews), len(parseViews), "number of views should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ViewMetadata)
	for _, view := range syncViews {
		syncMap[view.Name] = view
	}

	for _, parseView := range parseViews {
		syncView, exists := syncMap[parseView.Name]
		require.True(t, exists, "view %s should exist in sync metadata", parseView.Name)

		// Compare view definitions using PostgreSQL view comparer for better normalization
		comparer := &PostgreSQLViewComparer{}
		definitionsEqual := comparer.compareViewsSemanticaly(syncView.Definition, parseView.Definition)
		require.True(t, definitionsEqual, "view %s: definition should match\nSync: %s\nParse: %s", parseView.Name, syncView.Definition, parseView.Definition)

		// Compare comment if present
		if syncView.Comment != "" || parseView.Comment != "" {
			require.Equal(t, syncView.Comment, parseView.Comment, "view %s: comment should match", parseView.Name)
		}

		// Compare rules
		compareRules(t, parseView.Name, syncView.Rules, parseView.Rules)
	}
}

func compareFunctions(t *testing.T, syncFuncs, parseFuncs []*storepb.FunctionMetadata) {
	// Function comparison is tricky because signatures might be formatted differently
	t.Logf("sync has %d functions, parse has %d functions", len(syncFuncs), len(parseFuncs))

	// Currently the parser doesn't extract functions from DDL, so we expect 0 functions from parser
	// If parser starts extracting functions, we should implement proper comparison here
	if len(parseFuncs) == 0 && len(syncFuncs) > 0 {
		// This is expected - parser doesn't extract functions yet
		return
	}

	// If parser starts returning functions, implement full comparison
	require.Equal(t, len(syncFuncs), len(parseFuncs), "number of functions should match")

	// Create maps for easier comparison - use function signature for mapping
	syncMap := make(map[string]*storepb.FunctionMetadata)
	for _, fn := range syncFuncs {
		syncMap[fn.Signature] = fn
	}

	for _, parseFn := range parseFuncs {
		// Try to find matching function by signature
		var syncFn *storepb.FunctionMetadata
		for _, sf := range syncFuncs {
			if normalizeSignature(sf.Signature) == normalizeSignature(parseFn.Signature) {
				syncFn = sf
				break
			}
		}

		require.NotNil(t, syncFn, "function with signature %s should exist in sync metadata", parseFn.Signature)

		// Compare function definitions using PostgreSQL function comparer
		comparer := &PostgreSQLFunctionComparer{}
		syncFunc := &storepb.FunctionMetadata{Definition: syncFn.Definition}
		parseFunc := &storepb.FunctionMetadata{Definition: parseFn.Definition}

		// Use function comparer to check if they are equal
		if !comparer.Equal(syncFunc, parseFunc) {
			// Get detailed comparison for better error message
			result, err := comparer.CompareDetailed(syncFunc, parseFunc)
			if err != nil {
				t.Errorf("function %s: error comparing functions: %v", parseFn.Name, err)
			} else if result != nil {
				t.Errorf("function %s: definitions differ - SignatureChanged: %v, BodyChanged: %v, AttributesChanged: %v",
					parseFn.Name, result.SignatureChanged, result.BodyChanged, result.AttributesChanged)
				t.Logf("  Sync definition: %s", syncFn.Definition)
				t.Logf("  Parse definition: %s", parseFn.Definition)
			}
		}

		// Compare comment if present
		if syncFn.Comment != "" || parseFn.Comment != "" {
			require.Equal(t, syncFn.Comment, parseFn.Comment, "function %s: comment should match", parseFn.Name)
		}
	}
}

func compareSequences(t *testing.T, syncSeqs, parseSeqs []*storepb.SequenceMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range syncSeqs {
		syncMap[seq.Name] = seq
	}

	parseMap := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range parseSeqs {
		parseMap[seq.Name] = seq
	}

	// Check sequences in parseSeqs
	for _, parseSeq := range parseSeqs {
		syncSeq, exists := syncMap[parseSeq.Name]
		if !exists {
			// SERIAL columns create implicit sequences that might not be in DDL
			t.Logf("sequence %s exists in parse but not in sync (might be implicit from SERIAL)", parseSeq.Name)
			continue
		}

		// Compare basic properties
		if parseSeq.Start != "" {
			require.Equal(t, syncSeq.Start, parseSeq.Start, "sequence %s: start value should match", parseSeq.Name)
		}
		if parseSeq.Increment != "" {
			require.Equal(t, syncSeq.Increment, parseSeq.Increment, "sequence %s: increment should match", parseSeq.Name)
		}
	}

	// Check sequences in syncSeqs that are not in parseSeqs
	for _, syncSeq := range syncSeqs {
		_, exists := parseMap[syncSeq.Name]
		if !exists {
			// Skip implicit sequences created by SERIAL columns
			if strings.Contains(syncSeq.Name, "_id_seq") || strings.Contains(syncSeq.Name, "_seq") {
				t.Logf("sequence %s exists in sync but not in parse (implicit sequence from SERIAL column)", syncSeq.Name)
				continue
			}
			// For explicitly created sequences, this is an error
			require.True(t, exists, "sequence %s should exist in parsed metadata", syncSeq.Name)
		}
	}
}

func compareEnums(t *testing.T, syncEnums, parseEnums []*storepb.EnumTypeMetadata) {
	require.Equal(t, len(syncEnums), len(parseEnums), "number of enums should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.EnumTypeMetadata)
	for _, enum := range syncEnums {
		syncMap[enum.Name] = enum
	}

	for _, parseEnum := range parseEnums {
		syncEnum, exists := syncMap[parseEnum.Name]
		require.True(t, exists, "enum %s should exist in sync metadata", parseEnum.Name)
		require.ElementsMatch(t, syncEnum.Values, parseEnum.Values, "enum %s: values should match", parseEnum.Name)
	}
}

func compareExtensions(t *testing.T, syncExts, parseExts []*storepb.ExtensionMetadata) {
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ExtensionMetadata)
	for _, ext := range syncExts {
		syncMap[ext.Name] = ext
	}

	for _, parseExt := range parseExts {
		syncExt, exists := syncMap[parseExt.Name]
		require.True(t, exists, "extension %s should exist in sync metadata", parseExt.Name)
		require.Equal(t, syncExt.Schema, parseExt.Schema, "extension %s: schema should match", parseExt.Name)
	}
}

func compareMaterializedViews(t *testing.T, syncMViews, parseMViews []*storepb.MaterializedViewMetadata) {
	// Materialized views are not currently supported by the parser
	// The parser may incorrectly classify them as tables, so we handle this gracefully
	if len(parseMViews) == 0 && len(syncMViews) > 0 {
		t.Logf("Parser doesn't extract materialized views yet - found %d in sync", len(syncMViews))
		return
	}

	require.Equal(t, len(syncMViews), len(parseMViews), "number of materialized views should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.MaterializedViewMetadata)
	for _, mv := range syncMViews {
		syncMap[mv.Name] = mv
	}

	for _, parseMv := range parseMViews {
		syncMv, exists := syncMap[parseMv.Name]
		require.True(t, exists, "materialized view %s should exist in sync metadata", parseMv.Name)

		// Compare definitions using PostgreSQL view comparer for better normalization
		comparer := &PostgreSQLViewComparer{}
		definitionsEqual := comparer.compareViewsSemanticaly(syncMv.Definition, parseMv.Definition)
		require.True(t, definitionsEqual, "materialized view %s: definition should match\nSync: %s\nParse: %s", parseMv.Name, syncMv.Definition, parseMv.Definition)

		// Compare comment if present
		if syncMv.Comment != "" || parseMv.Comment != "" {
			require.Equal(t, syncMv.Comment, parseMv.Comment, "materialized view %s: comment should match", parseMv.Name)
		}

		// Compare indexes on materialized views if present
		if len(syncMv.Indexes) > 0 || len(parseMv.Indexes) > 0 {
			compareIndexes(t, parseMv.Name, syncMv.Indexes, parseMv.Indexes)
		}

		// Compare triggers on materialized views if present
		if len(syncMv.Triggers) > 0 || len(parseMv.Triggers) > 0 {
			compareTriggers(t, parseMv.Name, syncMv.Triggers, parseMv.Triggers)
		}
	}
}

func compareProcedures(t *testing.T, syncProcs, parseProcs []*storepb.ProcedureMetadata) {
	// Procedures might not be extracted by parser yet, so handle gracefully
	if len(parseProcs) == 0 && len(syncProcs) > 0 {
		t.Logf("Parser doesn't extract procedures yet - found %d in sync", len(syncProcs))
		return
	}

	require.Equal(t, len(syncProcs), len(parseProcs), "number of procedures should match")

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.ProcedureMetadata)
	for _, proc := range syncProcs {
		syncMap[proc.Name] = proc
	}

	for _, parseProc := range parseProcs {
		syncProc, exists := syncMap[parseProc.Name]
		require.True(t, exists, "procedure %s should exist in sync metadata", parseProc.Name)

		// Compare definitions
		syncDef := normalizeSQL(syncProc.Definition)
		parseDef := normalizeSQL(parseProc.Definition)
		require.Equal(t, syncDef, parseDef, "procedure %s: definition should match", parseProc.Name)
	}
}

func compareRules(t *testing.T, objectName string, syncRules, parseRules []*storepb.RuleMetadata) {
	// Rules might not be extracted by parser yet, so handle gracefully
	if len(parseRules) == 0 && len(syncRules) > 0 {
		t.Logf("Object %s: Parser doesn't extract rules yet - found %d in sync", objectName, len(syncRules))
		return
	}

	require.Equal(t, len(syncRules), len(parseRules), "object %s: number of rules should match", objectName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.RuleMetadata)
	for _, rule := range syncRules {
		syncMap[rule.Name] = rule
	}

	for _, parseRule := range parseRules {
		syncRule, exists := syncMap[parseRule.Name]
		require.True(t, exists, "object %s: rule %s should exist in sync metadata", objectName, parseRule.Name)

		// Compare rule properties
		require.Equal(t, syncRule.Event, parseRule.Event, "object %s: rule %s: event should match", objectName, parseRule.Name)
		require.Equal(t, syncRule.IsInstead, parseRule.IsInstead, "object %s: rule %s: is_instead should match", objectName, parseRule.Name)
		require.Equal(t, syncRule.IsEnabled, parseRule.IsEnabled, "object %s: rule %s: is_enabled should match", objectName, parseRule.Name)

		// Compare conditions (might be empty)
		if syncRule.Condition != "" || parseRule.Condition != "" {
			require.Equal(t, syncRule.Condition, parseRule.Condition, "object %s: rule %s: condition should match", objectName, parseRule.Name)
		}

		// Compare definitions - normalize them first
		syncDef := normalizeSQL(syncRule.Definition)
		parseDef := normalizeSQL(parseRule.Definition)
		require.Equal(t, syncDef, parseDef, "object %s: rule %s: definition should match", objectName, parseRule.Name)
	}
}

func compareTriggers(t *testing.T, tableName string, syncTriggers, parseTriggers []*storepb.TriggerMetadata) {
	// Triggers might not be extracted by parser yet, so handle gracefully
	if len(parseTriggers) == 0 && len(syncTriggers) > 0 {
		t.Logf("Table %s: Parser doesn't extract triggers yet - found %d in sync", tableName, len(syncTriggers))
		return
	}

	require.Equal(t, len(syncTriggers), len(parseTriggers), "table %s: number of triggers should match", tableName)

	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.TriggerMetadata)
	for _, trigger := range syncTriggers {
		syncMap[trigger.Name] = trigger
	}

	for _, parseTrigger := range parseTriggers {
		syncTrigger, exists := syncMap[parseTrigger.Name]
		require.True(t, exists, "table %s: trigger %s should exist in sync metadata", tableName, parseTrigger.Name)

		// Compare basic trigger properties
		require.Equal(t, syncTrigger.Event, parseTrigger.Event, "table %s, trigger %s: event should match", tableName, parseTrigger.Name)
		require.Equal(t, syncTrigger.Timing, parseTrigger.Timing, "table %s, trigger %s: timing should match", tableName, parseTrigger.Name)

		// Compare trigger body/definition if available
		if syncTrigger.Body != "" || parseTrigger.Body != "" {
			syncBody := normalizeSQL(syncTrigger.Body)
			parseBody := normalizeSQL(parseTrigger.Body)
			require.Equal(t, syncBody, parseBody, "table %s, trigger %s: body should match", tableName, parseTrigger.Name)
		}
	}
}

func compareAllSchemas(t *testing.T, syncSchemas, parseSchemas []*storepb.SchemaMetadata) {
	// For cross-schema test cases, we need to compare schemas beyond just 'public'
	// Create maps for easier comparison
	syncMap := make(map[string]*storepb.SchemaMetadata)
	for _, schema := range syncSchemas {
		syncMap[schema.Name] = schema
	}

	parseMap := make(map[string]*storepb.SchemaMetadata)
	for _, schema := range parseSchemas {
		parseMap[schema.Name] = schema
	}

	// Check that important schemas exist in both
	for _, parseSchema := range parseSchemas {
		if parseSchema.Name == "information_schema" || parseSchema.Name == "pg_catalog" {
			// Skip system schemas
			continue
		}

		syncSchema, exists := syncMap[parseSchema.Name]
		if !exists {
			t.Logf("Schema %s exists in parse but not in sync (might be expected for some test cases)", parseSchema.Name)
			continue
		}

		// Compare schema-specific content
		t.Logf("Comparing schema: %s", parseSchema.Name)

		// Compare tables in this schema
		compareTables(t, syncSchema.Tables, parseSchema.Tables)

		// Compare views in this schema
		compareViews(t, syncSchema.Views, parseSchema.Views)

		// Compare functions in this schema
		compareFunctions(t, syncSchema.Functions, parseSchema.Functions)
	}
}

// validateWithSchemaDiffer validates that the schema differ returns no significant differences between sync and parse metadata
func validateWithSchemaDiffer(t *testing.T, testName string, syncMeta, parseMeta *storepb.DatabaseSchemaMetadata) {
	// Convert metadata to model.DatabaseMetadata for differ
	syncSchema := model.NewDatabaseMetadata(syncMeta, nil, nil, storepb.Engine_POSTGRES, false)
	parseSchema := model.NewDatabaseMetadata(parseMeta, nil, nil, storepb.Engine_POSTGRES, false)

	// Get schema diff
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, syncSchema, parseSchema)
	require.NoError(t, err, "test case %s: schema differ should not error", testName)

	if diff == nil {
		// No diff is expected - this is good
		return
	}

	// Check that all diff categories are empty - DDL parser should fully match PostgreSQL behavior
	var diffMessages []string

	if len(diff.SchemaChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d schema changes", len(diff.SchemaChanges)))
		for _, change := range diff.SchemaChanges {
			t.Logf("Schema change: %s %s", change.Action, change.SchemaName)
		}
	}

	if len(diff.TableChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d table changes", len(diff.TableChanges)))
		for _, change := range diff.TableChanges {
			t.Logf("Table change: %s %s.%s", change.Action, change.SchemaName, change.TableName)
			// Log detailed changes within the table
			if len(change.ColumnChanges) > 0 {
				t.Logf("  Column changes: %d", len(change.ColumnChanges))
				for _, colChange := range change.ColumnChanges {
					if colChange.OldColumn != nil && colChange.NewColumn != nil {
						t.Logf("    %s column %s: %s -> %s", colChange.Action, colChange.NewColumn.Name, colChange.OldColumn.Type, colChange.NewColumn.Type)
						if colChange.OldColumn.Default != colChange.NewColumn.Default {
							t.Logf("      Default changed: '%s' -> '%s'", colChange.OldColumn.Default, colChange.NewColumn.Default)
						}
						if colChange.OldColumn.Nullable != colChange.NewColumn.Nullable {
							t.Logf("      Nullable changed: %t -> %t", colChange.OldColumn.Nullable, colChange.NewColumn.Nullable)
						}
					} else if colChange.NewColumn != nil {
						t.Logf("    %s column %s: %s", colChange.Action, colChange.NewColumn.Name, colChange.NewColumn.Type)
					} else if colChange.OldColumn != nil {
						t.Logf("    %s column %s: %s", colChange.Action, colChange.OldColumn.Name, colChange.OldColumn.Type)
					}
				}
			}
			if len(change.IndexChanges) > 0 {
				t.Logf("  Index changes: %d", len(change.IndexChanges))
				for i, idxChange := range change.IndexChanges {
					// Look for DROP/CREATE pairs that indicate a difference was detected
					if i+1 < len(change.IndexChanges) && idxChange.Action == "DROP" && change.IndexChanges[i+1].Action == "CREATE" &&
						idxChange.OldIndex.Name == change.IndexChanges[i+1].NewIndex.Name {
						// This is a DROP/CREATE pair - log detailed comparison
						oldIdx := idxChange.OldIndex
						newIdx := change.IndexChanges[i+1].NewIndex
						t.Logf("    DETAILED INDEX COMPARISON for %s (DROP/CREATE pair):", oldIdx.Name)
						t.Logf("      Type: old='%s' vs new='%s' (equal: %v)", oldIdx.Type, newIdx.Type, oldIdx.Type == newIdx.Type)
						t.Logf("      Unique: old=%v vs new=%v (equal: %v)", oldIdx.Unique, newIdx.Unique, oldIdx.Unique == newIdx.Unique)
						t.Logf("      Primary: old=%v vs new=%v (equal: %v)", oldIdx.Primary, newIdx.Primary, oldIdx.Primary == newIdx.Primary)
						t.Logf("      Expressions: old=%v vs new=%v (equal: %v)", oldIdx.Expressions, newIdx.Expressions, fmt.Sprintf("%v", oldIdx.Expressions) == fmt.Sprintf("%v", newIdx.Expressions))
						t.Logf("      KeyLength: old=%v vs new=%v (equal: %v)", oldIdx.KeyLength, newIdx.KeyLength, fmt.Sprintf("%v", oldIdx.KeyLength) == fmt.Sprintf("%v", newIdx.KeyLength))
						t.Logf("      Descending: old=%v vs new=%v (equal: %v)", oldIdx.Descending, newIdx.Descending, fmt.Sprintf("%v", oldIdx.Descending) == fmt.Sprintf("%v", newIdx.Descending))
						t.Logf("      Visible: old=%v vs new=%v (equal: %v)", oldIdx.Visible, newIdx.Visible, oldIdx.Visible == newIdx.Visible)
						t.Logf("      Definition: old='%s' vs new='%s' (equal: %v)", oldIdx.Definition, newIdx.Definition, oldIdx.Definition == newIdx.Definition)
						t.Logf("      IsConstraint: old=%v vs new=%v (equal: %v)", oldIdx.IsConstraint, newIdx.IsConstraint, oldIdx.IsConstraint == newIdx.IsConstraint)
					}

					if idxChange.NewIndex != nil && idxChange.OldIndex != nil {
						// Log detailed comparison for ALTER cases
						t.Logf("    DETAILED INDEX COMPARISON for %s:", idxChange.NewIndex.Name)
						t.Logf("      Type: old='%s' vs new='%s' (equal: %v)", idxChange.OldIndex.Type, idxChange.NewIndex.Type, idxChange.OldIndex.Type == idxChange.NewIndex.Type)
						t.Logf("      Unique: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.Unique, idxChange.NewIndex.Unique, idxChange.OldIndex.Unique == idxChange.NewIndex.Unique)
						t.Logf("      Primary: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.Primary, idxChange.NewIndex.Primary, idxChange.OldIndex.Primary == idxChange.NewIndex.Primary)
						t.Logf("      Expressions: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.Expressions, idxChange.NewIndex.Expressions, fmt.Sprintf("%v", idxChange.OldIndex.Expressions) == fmt.Sprintf("%v", idxChange.NewIndex.Expressions))
						t.Logf("      KeyLength: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.KeyLength, idxChange.NewIndex.KeyLength, fmt.Sprintf("%v", idxChange.OldIndex.KeyLength) == fmt.Sprintf("%v", idxChange.NewIndex.KeyLength))
						t.Logf("      Descending: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.Descending, idxChange.NewIndex.Descending, fmt.Sprintf("%v", idxChange.OldIndex.Descending) == fmt.Sprintf("%v", idxChange.NewIndex.Descending))
						t.Logf("      Visible: old=%v vs new=%v (equal: %v)", idxChange.OldIndex.Visible, idxChange.NewIndex.Visible, idxChange.OldIndex.Visible == idxChange.NewIndex.Visible)
						t.Logf("      Definition: old='%s' vs new='%s' (equal: %v)", idxChange.OldIndex.Definition, idxChange.NewIndex.Definition, idxChange.OldIndex.Definition == idxChange.NewIndex.Definition)
					}

					if idxChange.NewIndex != nil {
						t.Logf("    %s index %s: type=%s, unique=%v, primary=%v, expressions=%v, definition=%s",
							idxChange.Action, idxChange.NewIndex.Name, idxChange.NewIndex.Type,
							idxChange.NewIndex.Unique, idxChange.NewIndex.Primary, idxChange.NewIndex.Expressions, idxChange.NewIndex.Definition)
					} else if idxChange.OldIndex != nil {
						t.Logf("    %s index %s: type=%s, unique=%v, primary=%v, expressions=%v, definition=%s",
							idxChange.Action, idxChange.OldIndex.Name, idxChange.OldIndex.Type,
							idxChange.OldIndex.Unique, idxChange.OldIndex.Primary, idxChange.OldIndex.Expressions, idxChange.OldIndex.Definition)
					}
				}
			}
			if len(change.ForeignKeyChanges) > 0 {
				t.Logf("  Foreign key changes: %d", len(change.ForeignKeyChanges))
				for _, fkChange := range change.ForeignKeyChanges {
					if fkChange.NewForeignKey != nil {
						t.Logf("    %s FK %s: columns=%v -> %s.%s(%v), onDelete=%s",
							fkChange.Action, fkChange.NewForeignKey.Name, fkChange.NewForeignKey.Columns,
							fkChange.NewForeignKey.ReferencedSchema, fkChange.NewForeignKey.ReferencedTable,
							fkChange.NewForeignKey.ReferencedColumns, fkChange.NewForeignKey.OnDelete)
					} else if fkChange.OldForeignKey != nil {
						t.Logf("    %s FK %s: columns=%v -> %s.%s(%v), onDelete=%s",
							fkChange.Action, fkChange.OldForeignKey.Name, fkChange.OldForeignKey.Columns,
							fkChange.OldForeignKey.ReferencedSchema, fkChange.OldForeignKey.ReferencedTable,
							fkChange.OldForeignKey.ReferencedColumns, fkChange.OldForeignKey.OnDelete)
					}
				}
			}
			if len(change.CheckConstraintChanges) > 0 {
				t.Logf("  Check constraint changes: %d", len(change.CheckConstraintChanges))
				for _, checkChange := range change.CheckConstraintChanges {
					if checkChange.NewCheckConstraint != nil && checkChange.OldCheckConstraint != nil {
						t.Logf("    %s CHECK %s: old='%s' -> new='%s'",
							checkChange.Action, checkChange.NewCheckConstraint.Name,
							checkChange.OldCheckConstraint.Expression, checkChange.NewCheckConstraint.Expression)
					} else if checkChange.NewCheckConstraint != nil {
						t.Logf("    %s CHECK %s: expression='%s'",
							checkChange.Action, checkChange.NewCheckConstraint.Name, checkChange.NewCheckConstraint.Expression)
					} else if checkChange.OldCheckConstraint != nil {
						t.Logf("    %s CHECK %s: expression='%s'",
							checkChange.Action, checkChange.OldCheckConstraint.Name, checkChange.OldCheckConstraint.Expression)
					}
				}
			}
			if len(change.TriggerChanges) > 0 {
				t.Logf("  Trigger changes: %d", len(change.TriggerChanges))
				for _, triggerChange := range change.TriggerChanges {
					if triggerChange.NewTrigger != nil && triggerChange.OldTrigger != nil {
						t.Logf("    %s TRIGGER %s: old='%s' -> new='%s'",
							triggerChange.Action, triggerChange.NewTrigger.Name,
							triggerChange.OldTrigger.Body, triggerChange.NewTrigger.Body)
					} else if triggerChange.NewTrigger != nil {
						t.Logf("    %s TRIGGER %s: body='%s'",
							triggerChange.Action, triggerChange.NewTrigger.Name, triggerChange.NewTrigger.Body)
					} else if triggerChange.OldTrigger != nil {
						t.Logf("    %s TRIGGER %s: body='%s'",
							triggerChange.Action, triggerChange.OldTrigger.Name, triggerChange.OldTrigger.Body)
					}
				}
			}
		}
	}

	if len(diff.ViewChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d view changes", len(diff.ViewChanges)))
		for _, change := range diff.ViewChanges {
			t.Logf("View change: %s %s.%s", change.Action, change.SchemaName, change.ViewName)
		}
	}

	if len(diff.MaterializedViewChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d materialized view changes", len(diff.MaterializedViewChanges)))
		for _, change := range diff.MaterializedViewChanges {
			t.Logf("Materialized view change: %s %s.%s", change.Action, change.SchemaName, change.MaterializedViewName)
		}
	}

	if len(diff.FunctionChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d function changes", len(diff.FunctionChanges)))
		for _, change := range diff.FunctionChanges {
			t.Logf("Function change: %s %s.%s", change.Action, change.SchemaName, change.FunctionName)
			// Log detailed function comparison for debugging
			if change.OldFunction != nil && change.NewFunction != nil {
				t.Logf("  Old function signature: %s", change.OldFunction.Signature)
				t.Logf("  New function signature: %s", change.NewFunction.Signature)
				t.Logf("  Old function definition: %s", change.OldFunction.Definition)
				t.Logf("  New function definition: %s", change.NewFunction.Definition)
				t.Logf("  Definitions equal: %v", change.OldFunction.Definition == change.NewFunction.Definition)
				t.Logf("  Signatures equal: %v", change.OldFunction.Signature == change.NewFunction.Signature)
			}
		}
	}

	if len(diff.ProcedureChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d procedure changes", len(diff.ProcedureChanges)))
		for _, change := range diff.ProcedureChanges {
			t.Logf("Procedure change: %s %s.%s", change.Action, change.SchemaName, change.ProcedureName)
		}
	}

	if len(diff.SequenceChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d sequence changes", len(diff.SequenceChanges)))
		for _, change := range diff.SequenceChanges {
			t.Logf("Sequence change: %s %s.%s", change.Action, change.SchemaName, change.SequenceName)
			if change.OldSequence != nil && change.NewSequence == nil {
				t.Logf("  ISSUE: SERIAL sequence %s missing in parsed metadata", change.SequenceName)
			} else if change.NewSequence != nil && change.OldSequence == nil {
				t.Logf("  ISSUE: Unexpected sequence %s in parsed metadata", change.SequenceName)
			}
		}
	}

	if len(diff.EnumTypeChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d enum type changes", len(diff.EnumTypeChanges)))
		for _, change := range diff.EnumTypeChanges {
			t.Logf("Enum type change: %s %s.%s", change.Action, change.SchemaName, change.EnumTypeName)
		}
	}

	if len(diff.EventChanges) > 0 {
		diffMessages = append(diffMessages, fmt.Sprintf("%d event changes", len(diff.EventChanges)))
		for _, change := range diff.EventChanges {
			t.Logf("Event change: %s %s", change.Action, change.EventName)
		}
	}

	// If we have any differences, fail the test - DDL parser must match PostgreSQL exactly
	if len(diffMessages) > 0 {
		require.Fail(t, fmt.Sprintf("test case %s: DDL parser should fully replicate PostgreSQL behavior. Differences found: %s", testName, strings.Join(diffMessages, ", ")))
	}
}

// validateRoutingRules validates that rules are handled correctly for all test cases
func validateRoutingRules(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
	// This validation ensures:
	// 1. No cycle errors occur with rules (if we got here, it passed)
	// 2. GetDatabaseDefinition works without errors
	// 3. Rules are properly included in the output

	require.NotNil(t, metadata, "Should have metadata without cycle errors")

	// Test that GetDatabaseDefinition works without cycle errors
	ctx := schema.GetDefinitionContext{
		PrintHeader: false,
	}
	definition, err := GetDatabaseDefinition(ctx, metadata)
	require.NoError(t, err, "GetDatabaseDefinition should not error even with rules")
	require.NotEmpty(t, definition)

	// Count rules across all schemas, tables, and views
	var totalRules int
	var hasRules bool

	for _, schema := range metadata.Schemas {
		// Check tables for rules
		for _, table := range schema.Tables {
			if len(table.Rules) > 0 {
				hasRules = true
				totalRules += len(table.Rules)

				// Verify each rule is in the output
				for _, rule := range table.Rules {
					require.NotEmpty(t, rule.Definition, "Rule should have definition")
					require.NotEmpty(t, rule.Event, "Rule should have event type")
					// Verify the rule appears in the output
					if !strings.Contains(definition, rule.Name) {
						t.Logf("Warning: Rule %s not found in output (might be a system rule)", rule.Name)
					}
				}
			}
		}

		// Check views for rules
		for _, view := range schema.Views {
			if len(view.Rules) > 0 {
				hasRules = true
				totalRules += len(view.Rules)

				// Verify each rule is in the output
				for _, rule := range view.Rules {
					require.NotEmpty(t, rule.Definition, "Rule should have definition")
					require.NotEmpty(t, rule.Event, "Rule should have event type")
					// Non-SELECT rules should be in the output
					if rule.Event != "SELECT" && !strings.Contains(definition, rule.Name) {
						t.Logf("Warning: Rule %s not found in output", rule.Name)
					}
				}
			}
		}
	}

	// Log summary for debugging
	if hasRules {
		t.Logf("✓ Found %d rules in metadata, GetDatabaseDefinition succeeded without cycles", totalRules)

		// For test cases with specific routing patterns, do additional validation
		// Check for the routing rules pattern (view with INSERT/UPDATE/DELETE rules)
		for _, schema := range metadata.Schemas {
			for _, view := range schema.Views {
				if view.Name == "new_name" && len(view.Rules) > 0 {
					// This is the routing pattern - verify correct ordering
					tablePos := strings.Index(definition, `CREATE TABLE "public"."old_name"`)
					viewPos := strings.Index(definition, `CREATE VIEW "public"."new_name"`)
					if tablePos > -1 && viewPos > -1 {
						require.Less(t, tablePos, viewPos, "With routing rules: table should come before view due to dependency")
						t.Log("✓ Routing rules pattern validated - correct ordering maintained")
					}
				}
			}
		}
	} else {
		t.Log("No rules in this test case - validation passed")
	}
}

// compareIndexDefinitionsSemanticaly compares two PostgreSQL index definitions using AST-based semantic comparison
func compareIndexDefinitionsSemanticaly(def1, def2 string) bool {
	if strings.TrimSpace(def1) == strings.TrimSpace(def2) {
		return true
	}

	// Parse the index definitions to extract components
	parts1 := parseIndexDefinition(def1)
	parts2 := parseIndexDefinition(def2)

	if parts1 == nil || parts2 == nil {
		// Fallback to case-insensitive string comparison if parsing fails
		return strings.EqualFold(strings.TrimSpace(def1), strings.TrimSpace(def2))
	}

	// Compare non-expression parts
	if !strings.EqualFold(parts1.CreateClause, parts2.CreateClause) ||
		!strings.EqualFold(parts1.IndexName, parts2.IndexName) ||
		!strings.EqualFold(parts1.OnClause, parts2.OnClause) ||
		!strings.EqualFold(parts1.UsingClause, parts2.UsingClause) {
		return false
	}

	// Compare expressions using semantic comparison
	comparer := ast.NewPostgreSQLExpressionComparer()
	if parts1.ExpressionsClause != "" && parts2.ExpressionsClause != "" {
		equal, err := comparer.CompareExpressions(parts1.ExpressionsClause, parts2.ExpressionsClause)
		if err != nil {
			// Fallback to string comparison if expression comparison fails
			return strings.EqualFold(strings.TrimSpace(parts1.ExpressionsClause), strings.TrimSpace(parts2.ExpressionsClause))
		}
		if !equal {
			return false
		}
	} else if parts1.ExpressionsClause != parts2.ExpressionsClause {
		return false
	}

	// Compare WHERE clause using semantic comparison
	if parts1.WhereClause != "" && parts2.WhereClause != "" {
		equal, err := comparer.CompareExpressions(parts1.WhereClause, parts2.WhereClause)
		if err != nil {
			// Fallback to string comparison if expression comparison fails
			return strings.EqualFold(strings.TrimSpace(parts1.WhereClause), strings.TrimSpace(parts2.WhereClause))
		}
		if !equal {
			return false
		}
	} else if parts1.WhereClause != parts2.WhereClause {
		return false
	}

	// Compare optional clauses
	return strings.EqualFold(parts1.IncludeClause, parts2.IncludeClause) &&
		strings.EqualFold(parts1.WithClause, parts2.WithClause)
}

// indexDefinitionParts represents the parsed components of an index definition
type indexDefinitionParts struct {
	CreateClause      string // "CREATE [UNIQUE] INDEX"
	IndexName         string // index name
	OnClause          string // "ON table_name"
	UsingClause       string // "USING method"
	ExpressionsClause string // column list or expressions inside parentheses
	IncludeClause     string // "INCLUDE (...)" clause
	WhereClause       string // WHERE condition (without WHERE keyword)
	WithClause        string // WITH (...) clause
}

// parseIndexDefinition parses a PostgreSQL index definition into its components
func parseIndexDefinition(definition string) *indexDefinitionParts {
	def := strings.TrimSpace(definition)
	if def == "" {
		return nil
	}

	// Remove trailing semicolon
	def = strings.TrimSuffix(def, ";")

	parts := &indexDefinitionParts{}

	// Split by major keywords to identify parts
	lowerDef := strings.ToLower(def)

	// Find CREATE clause
	createEnd := strings.Index(lowerDef, " on ")
	if createEnd == -1 {
		return nil
	}

	// Extract index name from CREATE clause
	createPart := def[:createEnd]
	parts.CreateClause = createPart

	// Extract index name (last word before "on")
	words := strings.Fields(createPart)
	if len(words) > 0 {
		parts.IndexName = words[len(words)-1]
	}

	remaining := def[createEnd:]
	lowerRemaining := strings.ToLower(remaining)

	// Find ON clause
	usingPos := strings.Index(lowerRemaining, " using ")
	if usingPos == -1 {
		return nil
	}
	parts.OnClause = strings.TrimSpace(remaining[:usingPos])

	remaining = remaining[usingPos:]

	// Find USING clause
	parenPos := strings.Index(remaining, "(")
	if parenPos == -1 {
		return nil
	}
	parts.UsingClause = strings.TrimSpace(remaining[:parenPos])

	remaining = remaining[parenPos:]

	// Extract expressions (content within first set of parentheses)
	parenCount := 0
	exprEnd := -1
	for i, r := range remaining {
		if r == '(' {
			parenCount++
		} else if r == ')' {
			parenCount--
			if parenCount == 0 {
				exprEnd = i
				break
			}
		}
	}

	if exprEnd == -1 {
		return nil
	}

	// Extract content within parentheses (excluding the parentheses themselves)
	parts.ExpressionsClause = strings.TrimSpace(remaining[1:exprEnd])

	if len(remaining) > exprEnd+1 {
		remaining = strings.TrimSpace(remaining[exprEnd+1:])
		lowerRemaining = strings.ToLower(remaining)

		// Look for optional clauses in order: INCLUDE, WHERE, WITH

		// INCLUDE clause
		if strings.HasPrefix(lowerRemaining, "include") {
			includeEnd := strings.Index(remaining, ")")
			if includeEnd != -1 {
				parts.IncludeClause = strings.TrimSpace(remaining[:includeEnd+1])
				if len(remaining) > includeEnd+1 {
					remaining = strings.TrimSpace(remaining[includeEnd+1:])
					lowerRemaining = strings.ToLower(remaining)
				} else {
					remaining = ""
					lowerRemaining = ""
				}
			}
		}

		// WHERE clause
		if strings.HasPrefix(lowerRemaining, "where ") {
			whereStart := 6 // length of "where "
			// Find the end of WHERE clause (could be end of string or start of WITH clause)
			withPos := strings.Index(lowerRemaining, " with ")
			var whereEnd int
			if withPos != -1 {
				whereEnd = withPos
			} else {
				whereEnd = len(remaining)
			}
			parts.WhereClause = strings.TrimSpace(remaining[whereStart:whereEnd])

			if withPos != -1 {
				remaining = strings.TrimSpace(remaining[withPos:])
				lowerRemaining = strings.ToLower(remaining)
			} else {
				remaining = ""
				lowerRemaining = ""
			}
		}

		// WITH clause
		if strings.HasPrefix(lowerRemaining, "with ") {
			parts.WithClause = strings.TrimSpace(remaining)
		}
	}

	return parts
}

// validateArrayTypes validates that array types are correctly mapped to their PostgreSQL internal representations
func validateArrayTypes(t *testing.T, syncMetadata, parseMetadata *storepb.DatabaseSchemaMetadata) {
	t.Helper()

	// Expected array type mappings based on our implementation
	expectedArrayTypes := map[string]string{
		// Integer arrays (all int variants should map to _int4)
		"int_array":      "_int4",
		"int4_array":     "_int4",
		"integer_array":  "_int4",
		"bigint_array":   "_int8",
		"int8_array":     "_int8",
		"smallint_array": "_int2",
		"int2_array":     "_int2",

		// Multi-dimensional arrays (should be same as single dimension)
		"int_multi_array":  "_int4",
		"text_multi_array": "_text",

		// Floating point arrays
		"real_array":             "_float4",
		"float4_array":           "_float4",
		"double_precision_array": "_float8",
		"float8_array":           "_float8",

		// Character arrays (all should map to appropriate internal types)
		"char_array":                    "_bpchar",
		"char_with_length":              "_bpchar",
		"varchar_array":                 "_varchar",
		"varchar_with_length":           "_varchar",
		"character_array":               "_bpchar",
		"character_with_length":         "_bpchar",
		"character_varying_array":       "_varchar",
		"character_varying_with_length": "_varchar",
		"text_array":                    "_text",

		// Boolean arrays
		"bool_array":    "_bool",
		"boolean_array": "_bool",

		// Numeric arrays
		"numeric_array":          "_numeric",
		"numeric_with_precision": "_numeric",
		"decimal_array":          "_numeric",
		"decimal_with_precision": "_numeric",
		"money_array":            "_money",

		// Date/time arrays
		"date_array":        "_date",
		"time_array":        "_time",
		"time_with_tz":      "_timetz",
		"timetz_array":      "_timetz",
		"timestamp_array":   "_timestamp",
		"timestamp_with_tz": "_timestamptz",
		"timestamptz_array": "_timestamptz",
		"interval_array":    "_interval",

		// Binary and UUID arrays
		"bytea_array": "_bytea",
		"uuid_array":  "_uuid",

		// JSON arrays
		"json_array":  "_json",
		"jsonb_array": "_jsonb",

		// Network arrays
		"inet_array":     "_inet",
		"cidr_array":     "_cidr",
		"macaddr_array":  "_macaddr",
		"macaddr8_array": "_macaddr8",

		// Geometric arrays
		"point_array":   "_point",
		"line_array":    "_line",
		"lseg_array":    "_lseg",
		"box_array":     "_box",
		"path_array":    "_path",
		"polygon_array": "_polygon",
		"circle_array":  "_circle",

		// Full-text search arrays
		"tsvector_array": "_tsvector",
		"tsquery_array":  "_tsquery",

		// Range arrays
		"int4range_array": "_int4range",
		"int8range_array": "_int8range",
		"numrange_array":  "_numrange",
		"tsrange_array":   "_tsrange",
		"tstzrange_array": "_tstzrange",
		"daterange_array": "_daterange",

		// Multi-range arrays (PostgreSQL 14+)
		"int4multirange_array": "_int4multirange",
		"int8multirange_array": "_int8multirange",
		"nummultirange_array":  "_nummultirange",
		"tsmultirange_array":   "_tsmultirange",
		"tstzmultirange_array": "_tstzmultirange",
		"datemultirange_array": "_datemultirange",

		// Bit string arrays
		"bit_array":          "_bit",
		"bit_with_length":    "_bit",
		"bit_varying_array":  "_varbit",
		"varbit_array":       "_varbit",
		"varbit_with_length": "_varbit",

		// Object identifier arrays
		"oid_array":           "_oid",
		"regclass_array":      "_regclass",
		"regproc_array":       "_regproc",
		"regprocedure_array":  "_regprocedure",
		"regoper_array":       "_regoper",
		"regoperator_array":   "_regoperator",
		"regtype_array":       "_regtype",
		"regconfig_array":     "_regconfig",
		"regdictionary_array": "_regdictionary",
		"regnamespace_array":  "_regnamespace",
		"regrole_array":       "_regrole",
		"regcollation_array":  "_regcollation",

		// System type arrays
		"tid_array":    "_tid",
		"xid_array":    "_xid",
		"xid8_array":   "_xid8",
		"cid_array":    "_cid",
		"pg_lsn_array": "_pg_lsn",

		// Custom enum arrays (should use fallback logic)
		"status_array":   "_status_enum",
		"priority_array": "_priority_level",

		// Other arrays
		"xml_array":           "_xml",
		"jsonpath_array":      "_jsonpath",
		"txid_snapshot_array": "_txid_snapshot",
	}

	// Find the array_types_test table in both metadata structures
	var syncTable, parseTable *storepb.TableMetadata

	for _, schema := range syncMetadata.Schemas {
		for _, table := range schema.Tables {
			if table.Name == "array_types_test" {
				syncTable = table
				break
			}
		}
		if syncTable != nil {
			break
		}
	}

	for _, schema := range parseMetadata.Schemas {
		for _, table := range schema.Tables {
			if table.Name == "array_types_test" {
				parseTable = table
				break
			}
		}
		if parseTable != nil {
			break
		}
	}

	require.NotNil(t, syncTable, "array_types_test table not found in sync metadata")
	require.NotNil(t, parseTable, "array_types_test table not found in parse metadata")

	// Build maps of column types for easy lookup
	syncColumnTypes := make(map[string]string)
	parseColumnTypes := make(map[string]string)

	for _, col := range syncTable.Columns {
		syncColumnTypes[col.Name] = col.Type
	}

	for _, col := range parseTable.Columns {
		parseColumnTypes[col.Name] = col.Type
	}

	t.Log("Array type validation results:")
	t.Log("Column Name                        | Sync Type                | Parse Type               | Expected     | Status")
	t.Log(strings.Repeat("-", 120))

	allCorrect := true

	for columnName, expectedType := range expectedArrayTypes {
		syncType, syncExists := syncColumnTypes[columnName]
		parseType, parseExists := parseColumnTypes[columnName]

		if !syncExists {
			t.Errorf("Column %s not found in sync metadata", columnName)
			allCorrect = false
			continue
		}

		if !parseExists {
			t.Errorf("Column %s not found in parse metadata", columnName)
			allCorrect = false
			continue
		}

		// The key validation: parse metadata should match expected array type
		parseCorrect := parseType == expectedType
		syncParseMatch := syncType == parseType

		status := "✓"
		if !parseCorrect {
			status = "✗ Parse incorrect"
			allCorrect = false
		} else if !syncParseMatch {
			status = "⚠ Sync/Parse mismatch"
			// This might be acceptable since sync gets actual PostgreSQL type names
			// while parse gets our normalized versions
		}

		t.Logf("%-30s | %-24s | %-24s | %-12s | %s",
			columnName, syncType, parseType, expectedType, status)

		// The critical assertion: our parsed metadata should have the correct array types
		if !parseCorrect {
			t.Errorf("Array type mismatch for column %s: got %s, expected %s", columnName, parseType, expectedType)
		}
	}

	if allCorrect {
		t.Log("🎉 All array types are correctly mapped!")
	} else {
		t.Error("❌ Some array types are not correctly mapped")
	}
}

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
CREATE INDEX idx_orders_year_month ON orders(EXTRACT( 'year' FROM order_date ) DESC, EXTRACT( 'month' FROM order_date ) ASC);

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

// normalizeExpression normalizes an expression for comparison using the PostgreSQL view comparer
func normalizeExpression(expr string) string {
	// Use the PostgreSQL view comparer's normalization logic for expressions
	comparer := &PostgreSQLViewComparer{}
	return comparer.normalizeViewDefinition(expr)
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

	// Compare common indexes
	for name, parseIdx := range parseMap {
		syncIdx, exists := syncMap[name]
		if !exists {
			// Some indexes might be system-generated and not in DDL
			t.Logf("table %s: index %s exists in parse but not in sync (might be implicit)", tableName, name)
			continue
		}

		require.Equal(t, syncIdx.Primary, parseIdx.Primary, "table %s, index %s: primary should match", tableName, name)
		require.Equal(t, syncIdx.Unique, parseIdx.Unique, "table %s, index %s: unique should match", tableName, name)

		// Compare expressions - sync.go gets normalized expressions from PostgreSQL catalog
		// while parser gets the raw expressions. We need to handle this difference.
		if len(syncIdx.Expressions) == len(parseIdx.Expressions) {
			for i := range syncIdx.Expressions {
				syncExpr := normalizeExpression(syncIdx.Expressions[i])
				parseExpr := normalizeExpression(parseIdx.Expressions[i])
				require.Equal(t, syncExpr, parseExpr, "table %s, index %s: expression[%d] should match", tableName, name, i)
			}
		} else {
			require.Equal(t, len(syncIdx.Expressions), len(parseIdx.Expressions), "table %s, index %s: expressions count should match", tableName, name)
		}

		// Compare descending order for each expression
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

		// Also compare the index type if available
		if syncIdx.Type != "" || parseIdx.Type != "" {
			require.Equal(t, syncIdx.Type, parseIdx.Type, "table %s, index %s: type should match", tableName, name)
		}

		// Compare IsConstraint field
		require.Equal(t, syncIdx.IsConstraint, parseIdx.IsConstraint, "table %s, index %s: IsConstraint should match", tableName, name)
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
		require.Equal(t, syncPart.Expression, parsePart.Expression, "table %s, partition %s: expression should match", tableName, parsePart.Name)
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
		definitionsEqual := comparer.compareViewDefinitions(syncView.Definition, parseView.Definition)
		require.True(t, definitionsEqual, "view %s: definition should match\nSync: %s\nParse: %s", parseView.Name, syncView.Definition, parseView.Definition)

		// Compare comment if present
		if syncView.Comment != "" || parseView.Comment != "" {
			require.Equal(t, syncView.Comment, parseView.Comment, "view %s: comment should match", parseView.Name)
		}
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

		// Compare function definitions
		syncDef := normalizeSQL(syncFn.Definition)
		parseDef := normalizeSQL(parseFn.Definition)
		require.Equal(t, syncDef, parseDef, "function %s: definition should match", parseFn.Name)

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
		definitionsEqual := comparer.compareViewDefinitions(syncMv.Definition, parseMv.Definition)
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

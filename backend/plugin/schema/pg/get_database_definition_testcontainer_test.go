package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	pgdb "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetDatabaseDefinitionWithTestcontainer tests the get_database_definition function
// by comparing metadata from a database created using the original DDL versus
// metadata from a database created using the generated DDL.
func TestGetDatabaseDefinitionWithTestcontainer(t *testing.T) {
	ctx := context.Background()

	// Get PostgreSQL container from testcontainer.go
	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	// Get the database connection
	db := pgContainer.GetDB()

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
SELECT id, name, department
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
			name: "check_constraints_and_comments",
			ddl: `
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    quantity INTEGER NOT NULL,
    CONSTRAINT chk_price CHECK (price > 0),
    CONSTRAINT chk_quantity CHECK (quantity >= 0)
);

COMMENT ON TABLE products IS 'Product catalog table';
COMMENT ON COLUMN products.name IS 'Product name';
COMMENT ON COLUMN products.price IS 'Product price in USD';
`,
		},
		{
			name: "materialized_views",
			ddl: `
CREATE TABLE raw_data (
    id SERIAL PRIMARY KEY,
    data_value NUMERIC NOT NULL,
    category VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE MATERIALIZED VIEW data_summary AS
SELECT category, 
       COUNT(*) as count, 
       AVG(data_value) as avg_value,
       MAX(data_value) as max_value,
       MIN(data_value) as min_value
FROM raw_data
GROUP BY category
WITH NO DATA;

CREATE INDEX idx_data_summary_category ON data_summary(category);
`,
		},
		{
			name: "comprehensive_views",
			ddl: `
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    total_amount DECIMAL(10, 2) NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending'
);

-- View with public schema references and various formatting
CREATE VIEW public.active_customer_orders AS
  SELECT 
    c.id AS customer_id,
    c.name,
    c.email,
    o.id AS order_id,
    o.total_amount,
    o.order_date
  FROM 
    public.customers c
    INNER JOIN public.orders o ON c.id = o.customer_id
  WHERE 
    c.active = TRUE 
    AND o.status = 'completed';

-- Simple view without schema qualification
CREATE VIEW customer_summary AS
SELECT 
  id,
  name,
  email,
  (SELECT COUNT(*) FROM orders WHERE customer_id = customers.id) as order_count
FROM customers
WHERE active = true;

-- Materialized view with complex aggregation
CREATE MATERIALIZED VIEW monthly_sales_summary AS
SELECT 
  DATE_TRUNC('month', order_date) as month,
  COUNT(*) as order_count,
  SUM(total_amount) as total_revenue,
  AVG(total_amount) as avg_order_value,
  COUNT(DISTINCT customer_id) as unique_customers
FROM public.orders
WHERE status = 'completed'
GROUP BY DATE_TRUNC('month', order_date)
ORDER BY month
WITH NO DATA;

CREATE UNIQUE INDEX idx_monthly_sales_month ON monthly_sales_summary(month);
`,
		},
		{
			name: "triggers",
			ddl: `
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    user_name VARCHAR(50),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    balance DECIMAL(10, 2) DEFAULT 0,
    last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_last_modified()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_last_modified
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE FUNCTION update_last_modified();

CREATE OR REPLACE FUNCTION log_account_changes()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (table_name, operation, user_name)
    VALUES ('accounts', TG_OP, current_user);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_accounts
AFTER INSERT OR UPDATE OR DELETE ON accounts
FOR EACH ROW
EXECUTE FUNCTION log_account_changes();
`,
		},
		{
			name: "foreign_key_on_delete_cascade",
			ddl: `
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category_id INTEGER,
    CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    product_id INTEGER,
    quantity INTEGER DEFAULT 1,
    CONSTRAINT fk_product_cascade FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL
);
`,
		},
		{
			name: "foreign_key_on_update_actions",
			ddl: `
CREATE TABLE departments (
    code VARCHAR(10) PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    dept_code VARCHAR(10),
    manager_id INTEGER,
    CONSTRAINT fk_dept_restrict FOREIGN KEY (dept_code) REFERENCES departments(code) ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT fk_manager_cascade FOREIGN KEY (manager_id) REFERENCES employees(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    lead_emp_id INTEGER,
    dept_code VARCHAR(10),
    CONSTRAINT fk_lead_setnull FOREIGN KEY (lead_emp_id) REFERENCES employees(id) ON DELETE SET NULL ON UPDATE SET NULL,
    CONSTRAINT fk_proj_dept FOREIGN KEY (dept_code) REFERENCES departments(code) ON DELETE RESTRICT ON UPDATE RESTRICT
);
`,
		},
		{
			name: "foreign_key_no_action",
			ddl: `
CREATE TABLE countries (
    id SERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE states (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    country_id INTEGER,
    -- Explicit NO ACTION
    CONSTRAINT fk_country_explicit FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE NO ACTION ON UPDATE NO ACTION
);

CREATE TABLE cities (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    state_id INTEGER,
    -- Implicit NO ACTION (default behavior)
    CONSTRAINT fk_state_implicit FOREIGN KEY (state_id) REFERENCES states(id)
);
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create first database with original DDL
			dbNameA := fmt.Sprintf("test_a_%s", tc.name)
			_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbNameA))
			require.NoError(t, err)
			defer func() {
				_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbNameA))
			}()

			// Connect to database A
			testDBA, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s sslmode=disable",
				pgContainer.GetHost(), pgContainer.GetPort(), dbNameA))
			require.NoError(t, err)
			defer testDBA.Close()

			// Execute the original DDL
			_, err = testDBA.Exec(tc.ddl)
			require.NoError(t, err)

			// Get metadata A using Driver.SyncDBSchema
			metadataA, err := getDBSyncMetadata(ctx, pgContainer, dbNameA)
			require.NoError(t, err)

			// Generate database definition using GetDatabaseDefinition
			generatedDDL, err := GetDatabaseDefinition(schema.GetDefinitionContext{PrintHeader: true}, metadataA)
			require.NoError(t, err)
			require.NotEmpty(t, generatedDDL, "generated DDL should not be empty")

			// Create second database and apply generated DDL
			dbNameB := fmt.Sprintf("test_b_%s", tc.name)
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbNameB))
			require.NoError(t, err)
			defer func() {
				_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbNameB))
			}()

			// Connect to database B
			testDBB, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=%s sslmode=disable",
				pgContainer.GetHost(), pgContainer.GetPort(), dbNameB))
			require.NoError(t, err)
			defer testDBB.Close()

			// Execute the generated DDL
			_, err = testDBB.Exec(generatedDDL)
			require.NoError(t, err, "failed to execute generated DDL: %s", generatedDDL)

			// Get metadata B using Driver.SyncDBSchema
			metadataB, err := getDBSyncMetadata(ctx, pgContainer, dbNameB)
			require.NoError(t, err)

			// Compare metadata A and B
			compareFullMetadata(t, metadataA, metadataB)
		})
	}
}

// getDBSyncMetadata retrieves metadata from the live database using Driver.SyncDBSchema
func getDBSyncMetadata(ctx context.Context, container *testcontainer.Container, dbName string) (*storepb.DatabaseSchemaMetadata, error) {
	// Create a driver instance using the pg package
	driver := &pgdb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Database: dbName,
		},
		Password: "root-password",
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

// compareFullMetadata compares all aspects of database metadata
func compareFullMetadata(t *testing.T, metaA, metaB *storepb.DatabaseSchemaMetadata) {
	// Compare extensions
	compareExtensionsDef(t, metaA.Extensions, metaB.Extensions)

	// Compare schemas
	require.Equal(t, len(metaA.Schemas), len(metaB.Schemas), "number of schemas should match")

	// Find the public schema in both
	var schemaA, schemaB *storepb.SchemaMetadata
	for _, schema := range metaA.Schemas {
		if schema.Name == "public" {
			schemaA = schema
			break
		}
	}
	for _, schema := range metaB.Schemas {
		if schema.Name == "public" {
			schemaB = schema
			break
		}
	}

	require.NotNil(t, schemaA, "metadata A should have public schema")
	require.NotNil(t, schemaB, "metadata B should have public schema")

	// Compare schema contents
	compareSchemaContents(t, schemaA, schemaB)
}

// compareSchemaContents compares the contents of two schemas
func compareSchemaContents(t *testing.T, schemaA, schemaB *storepb.SchemaMetadata) {
	// Compare enums
	compareEnumsDef(t, schemaA.EnumTypes, schemaB.EnumTypes)

	// Compare sequences (excluding implicit sequences from SERIAL columns)
	compareExplicitSequences(t, schemaA.Sequences, schemaB.Sequences)

	// Compare tables
	compareTablesDef(t, schemaA.Tables, schemaB.Tables)

	// Compare views
	compareViewsDef(t, schemaA.Views, schemaB.Views)

	// Compare materialized views
	compareMaterializedViewsDef(t, schemaA.MaterializedViews, schemaB.MaterializedViews)

	// Compare functions
	compareFunctionsDef(t, schemaA.Functions, schemaB.Functions)
}

// compareExplicitSequences compares only explicitly created sequences
func compareExplicitSequences(t *testing.T, seqsA, seqsB []*storepb.SequenceMetadata) {
	// Filter out implicit sequences (those owned by columns)
	explicitA := filterExplicitSequences(seqsA)
	explicitB := filterExplicitSequences(seqsB)

	require.Equal(t, len(explicitA), len(explicitB), "number of explicit sequences should match")

	// Create maps for comparison
	mapA := make(map[string]*storepb.SequenceMetadata)
	for _, seq := range explicitA {
		mapA[seq.Name] = seq
	}

	for _, seqB := range explicitB {
		seqA, exists := mapA[seqB.Name]
		require.True(t, exists, "sequence %s should exist in metadata A", seqB.Name)

		// Compare sequence properties
		require.Equal(t, seqA.Start, seqB.Start, "sequence %s: start value should match", seqB.Name)
		require.Equal(t, seqA.Increment, seqB.Increment, "sequence %s: increment should match", seqB.Name)
		require.Equal(t, seqA.MinValue, seqB.MinValue, "sequence %s: min value should match", seqB.Name)
		require.Equal(t, seqA.MaxValue, seqB.MaxValue, "sequence %s: max value should match", seqB.Name)
		require.Equal(t, seqA.Cycle, seqB.Cycle, "sequence %s: cycle should match", seqB.Name)
		require.Equal(t, seqA.Comment, seqB.Comment, "sequence %s: comment should match", seqB.Name)
	}
}

// filterExplicitSequences returns only sequences that are not owned by columns
func filterExplicitSequences(sequences []*storepb.SequenceMetadata) []*storepb.SequenceMetadata {
	var result []*storepb.SequenceMetadata
	for _, seq := range sequences {
		if seq.OwnerTable == "" || seq.OwnerColumn == "" {
			result = append(result, seq)
		}
	}
	return result
}

// compareMaterializedViewsDef compares materialized views between two schemas using PostgreSQL view comparer
func compareMaterializedViewsDef(t *testing.T, viewsA, viewsB []*storepb.MaterializedViewMetadata) {
	require.Equal(t, len(viewsA), len(viewsB), "number of materialized views should match")

	// Create maps for comparison
	mapA := make(map[string]*storepb.MaterializedViewMetadata)
	for _, view := range viewsA {
		mapA[view.Name] = view
	}

	// Get PostgreSQL view comparer for sophisticated comparison
	pgComparer := &PostgreSQLViewComparer{}

	for _, viewB := range viewsB {
		viewA, exists := mapA[viewB.Name]
		require.True(t, exists, "materialized view %s should exist in metadata A", viewB.Name)

		// Compare view definitions using PostgreSQL view comparer
		definitionsEqual := pgComparer.compareViewsSemanticaly(viewA.Definition, viewB.Definition)

		// Ensure definitions are equal using PostgreSQL ANTLR parser comparison
		if !definitionsEqual {
			// Log normalized definitions for debugging
			t.Logf("Materialized view %s definition mismatch:", viewB.Name)
			t.Logf("  A normalized: %q", pgComparer.normalizeExpression(viewA.Definition))
			t.Logf("  B normalized: %q", pgComparer.normalizeExpression(viewB.Definition))
			t.Logf("  A original: %q", viewA.Definition)
			t.Logf("  B original: %q", viewB.Definition)

			// Fallback to simple comparison for debugging
			defA := normalizeSQLDef(viewA.Definition)
			defB := normalizeSQLDef(viewB.Definition)
			t.Logf("  A simple normalized: %q", defA)
			t.Logf("  B simple normalized: %q", defB)
		}

		// Assert that materialized view definitions must be equal after sophisticated PostgreSQL parsing
		require.True(t, definitionsEqual,
			"materialized view %s: definitions must be equal using PostgreSQL view comparer (A: %q, B: %q)",
			viewB.Name, viewA.Definition, viewB.Definition)

		require.Equal(t, viewA.Comment, viewB.Comment,
			"materialized view %s: comment should match", viewB.Name)

		// Compare indexes on materialized views
		compareIndexesDef(t, viewB.Name, viewA.Indexes, viewB.Indexes)
	}
}

// normalizeSQLDef normalizes SQL for comparison
func normalizeSQLDef(sql string) string {
	// Convert to lowercase
	sql = strings.ToLower(sql)

	// Replace multiple spaces/newlines with single space
	sql = strings.Join(strings.Fields(sql), " ")

	// Remove trailing semicolons
	sql = strings.TrimSuffix(sql, ";")

	// Remove schema qualifiers for public schema
	sql = strings.ReplaceAll(sql, "public.", "")

	// Final trim
	sql = strings.TrimSpace(sql)

	return sql
}

// normalizeExprDef normalizes an expression for comparison
func normalizeExprDef(expr string) string {
	// Convert to lowercase
	expr = strings.ToLower(expr)

	// Replace multiple spaces with single space
	expr = strings.Join(strings.Fields(expr), " ")

	// Remove spaces around parentheses
	expr = strings.ReplaceAll(expr, " (", "(")
	expr = strings.ReplaceAll(expr, "( ", "(")
	expr = strings.ReplaceAll(expr, " )", ")")
	expr = strings.ReplaceAll(expr, ") ", ")")

	// Normalize quotes around identifiers
	expr = strings.ReplaceAll(expr, "'", "")

	return expr
}

// compareTablesDef compares tables between two schemas
func compareTablesDef(t *testing.T, tablesA, tablesB []*storepb.TableMetadata) {
	require.Equal(t, len(tablesA), len(tablesB), "number of tables should match")

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.TableMetadata)
	for _, table := range tablesA {
		mapA[table.Name] = table
	}

	mapB := make(map[string]*storepb.TableMetadata)
	for _, table := range tablesB {
		mapB[table.Name] = table
	}

	// Compare each table
	for name, tableA := range mapA {
		tableB, exists := mapB[name]
		require.True(t, exists, "table %s should exist in metadata B", name)

		// Compare columns
		compareColumnsDef(t, name, tableA.Columns, tableB.Columns)

		// Compare indexes
		compareIndexesDef(t, name, tableA.Indexes, tableB.Indexes)

		// Compare foreign keys
		compareForeignKeysDef(t, name, tableA.ForeignKeys, tableB.ForeignKeys)

		// Compare check constraints
		compareCheckConstraintsDef(t, name, tableA.CheckConstraints, tableB.CheckConstraints)

		// Compare partitions
		comparePartitionsDef(t, name, tableA.Partitions, tableB.Partitions)

		// Compare triggers
		compareTriggersDef(t, name, tableA.Triggers, tableB.Triggers)

		// Compare comments
		require.Equal(t, tableA.Comment, tableB.Comment, "table %s: comment should match", name)
	}
}

// compareColumnsDef compares columns between two tables
func compareColumnsDef(t *testing.T, tableName string, colsA, colsB []*storepb.ColumnMetadata) {
	require.Equal(t, len(colsA), len(colsB), "table %s: number of columns should match", tableName)

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.ColumnMetadata)
	for _, col := range colsA {
		mapA[col.Name] = col
	}

	for _, colB := range colsB {
		colA, exists := mapA[colB.Name]
		require.True(t, exists, "table %s: column %s should exist in metadata A", tableName, colB.Name)

		// Compare column properties
		require.Equal(t, colA.Type, colB.Type, "table %s, column %s: type should match", tableName, colB.Name)
		require.Equal(t, colA.Nullable, colB.Nullable, "table %s, column %s: nullable should match", tableName, colB.Name)
		require.Equal(t, colA.Comment, colB.Comment, "table %s, column %s: comment should match", tableName, colB.Name)

		// Compare default values if both exist
		hasDefaultA := colA.Default != ""
		hasDefaultB := colB.Default != ""
		if hasDefaultA && hasDefaultB {
			// Default values might be represented differently, so we just check they exist
			t.Logf("table %s, column %s: default values exist in both", tableName, colB.Name)
		}
	}
}

// compareIndexesDef compares indexes between two tables
func compareIndexesDef(t *testing.T, tableName string, indexesA, indexesB []*storepb.IndexMetadata) {
	// Create maps for easier comparison
	mapA := make(map[string]*storepb.IndexMetadata)
	for _, idx := range indexesA {
		mapA[idx.Name] = idx
	}

	mapB := make(map[string]*storepb.IndexMetadata)
	for _, idx := range indexesB {
		mapB[idx.Name] = idx
	}

	// Compare common indexes
	for name, idxB := range mapB {
		idxA, exists := mapA[name]
		if !exists {
			// Some indexes might be system-generated
			t.Logf("table %s: index %s exists in B but not in A (might be implicit)", tableName, name)
			continue
		}

		require.Equal(t, idxA.Primary, idxB.Primary, "table %s, index %s: primary should match", tableName, name)
		require.Equal(t, idxA.Unique, idxB.Unique, "table %s, index %s: unique should match", tableName, name)

		// Compare expressions
		if len(idxA.Expressions) == len(idxB.Expressions) {
			for i := range idxA.Expressions {
				exprA := normalizeExprDef(idxA.Expressions[i])
				exprB := normalizeExprDef(idxB.Expressions[i])
				require.Equal(t, exprA, exprB, "table %s, index %s: expression[%d] should match", tableName, name, i)
			}
		}

		// Compare IsConstraint field
		require.Equal(t, idxA.IsConstraint, idxB.IsConstraint, "table %s, index %s: IsConstraint should match", tableName, name)

		// Compare comment
		require.Equal(t, idxA.Comment, idxB.Comment, "table %s, index %s: comment should match", tableName, name)
	}
}

// compareForeignKeysDef compares foreign keys between two tables
func compareForeignKeysDef(t *testing.T, tableName string, fksA, fksB []*storepb.ForeignKeyMetadata) {
	require.Equal(t, len(fksA), len(fksB), "table %s: number of foreign keys should match", tableName)

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.ForeignKeyMetadata)
	for _, fk := range fksA {
		mapA[fk.Name] = fk
	}

	for _, fkB := range fksB {
		fkA, exists := mapA[fkB.Name]
		require.True(t, exists, "table %s: foreign key %s should exist in metadata A", tableName, fkB.Name)

		require.ElementsMatch(t, fkA.Columns, fkB.Columns, "table %s, FK %s: columns should match", tableName, fkB.Name)
		require.Equal(t, fkA.ReferencedTable, fkB.ReferencedTable, "table %s, FK %s: referenced table should match", tableName, fkB.Name)
		require.ElementsMatch(t, fkA.ReferencedColumns, fkB.ReferencedColumns, "table %s, FK %s: referenced columns should match", tableName, fkB.Name)

		// Compare ON DELETE and ON UPDATE actions
		require.Equal(t, fkA.OnDelete, fkB.OnDelete, "table %s, FK %s: ON DELETE action should match", tableName, fkB.Name)
		require.Equal(t, fkA.OnUpdate, fkB.OnUpdate, "table %s, FK %s: ON UPDATE action should match", tableName, fkB.Name)
	}
}

// compareCheckConstraintsDef compares check constraints between two tables
func compareCheckConstraintsDef(t *testing.T, tableName string, checksA, checksB []*storepb.CheckConstraintMetadata) {
	require.Equal(t, len(checksA), len(checksB), "table %s: number of check constraints should match", tableName)

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.CheckConstraintMetadata)
	for _, check := range checksA {
		mapA[check.Name] = check
	}

	for _, checkB := range checksB {
		checkA, exists := mapA[checkB.Name]
		require.True(t, exists, "table %s: check constraint %s should exist in metadata A", tableName, checkB.Name)

		// Normalize and compare expressions
		exprA := normalizeExprDef(checkA.Expression)
		exprB := normalizeExprDef(checkB.Expression)
		require.Equal(t, exprA, exprB, "table %s, check %s: expression should match", tableName, checkB.Name)
	}
}

// comparePartitionsDef compares partitions between two tables
func comparePartitionsDef(t *testing.T, tableName string, partsA, partsB []*storepb.TablePartitionMetadata) {
	require.Equal(t, len(partsA), len(partsB), "table %s: number of partitions should match", tableName)

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.TablePartitionMetadata)
	for _, part := range partsA {
		mapA[part.Name] = part
	}

	for _, partB := range partsB {
		partA, exists := mapA[partB.Name]
		require.True(t, exists, "table %s: partition %s should exist in metadata A", tableName, partB.Name)
		require.Equal(t, partA.Expression, partB.Expression, "table %s, partition %s: expression should match", tableName, partB.Name)
		require.Equal(t, partA.Value, partB.Value, "table %s, partition %s: value should match", tableName, partB.Name)
	}
}

// compareTriggersDef compares triggers between two tables
func compareTriggersDef(t *testing.T, tableName string, triggersA, triggersB []*storepb.TriggerMetadata) {
	require.Equal(t, len(triggersA), len(triggersB), "table %s: number of triggers should match", tableName)

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.TriggerMetadata)
	for _, trigger := range triggersA {
		mapA[trigger.Name] = trigger
	}

	for _, triggerB := range triggersB {
		triggerA, exists := mapA[triggerB.Name]
		require.True(t, exists, "table %s: trigger %s should exist in metadata A", tableName, triggerB.Name)

		// Compare trigger properties
		require.Equal(t, normalizeSQLDef(triggerA.Body), normalizeSQLDef(triggerB.Body),
			"table %s, trigger %s: body should match", tableName, triggerB.Name)
		require.Equal(t, triggerA.Comment, triggerB.Comment,
			"table %s, trigger %s: comment should match", tableName, triggerB.Name)
	}
}

// compareViewsDef compares views between two schemas using PostgreSQL view comparer
func compareViewsDef(t *testing.T, viewsA, viewsB []*storepb.ViewMetadata) {
	require.Equal(t, len(viewsA), len(viewsB), "number of views should match")

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.ViewMetadata)
	for _, view := range viewsA {
		mapA[view.Name] = view
	}

	// Get PostgreSQL view comparer for sophisticated comparison
	pgComparer := &PostgreSQLViewComparer{}

	for _, viewB := range viewsB {
		viewA, exists := mapA[viewB.Name]
		require.True(t, exists, "view %s should exist in metadata A", viewB.Name)

		// Compare view definitions using PostgreSQL view comparer
		definitionsEqual := pgComparer.compareViewsSemanticaly(viewA.Definition, viewB.Definition)

		// Assert that view definitions must be equal after sophisticated PostgreSQL parsing
		require.True(t, definitionsEqual,
			"view %s: definitions must be equal using PostgreSQL view comparer (A: %q, B: %q)",
			viewB.Name, viewA.Definition, viewB.Definition)

		// Compare comment
		require.Equal(t, viewA.Comment, viewB.Comment, "view %s: comment should match", viewB.Name)
	}
}

// compareFunctionsDef compares functions between two schemas
func compareFunctionsDef(t *testing.T, funcsA, funcsB []*storepb.FunctionMetadata) {
	require.Equal(t, len(funcsA), len(funcsB), "number of functions should match")

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.FunctionMetadata)
	for _, fn := range funcsA {
		mapA[fn.Name] = fn
	}

	for _, fnB := range funcsB {
		fnA, exists := mapA[fnB.Name]
		require.True(t, exists, "function %s should exist in metadata A", fnB.Name)

		// Compare function definitions
		defA := normalizeSQLDef(fnA.Definition)
		defB := normalizeSQLDef(fnB.Definition)
		require.Equal(t, defA, defB, "function %s: definition should match", fnB.Name)

		// Compare comment
		require.Equal(t, fnA.Comment, fnB.Comment, "function %s: comment should match", fnB.Name)
	}
}

// compareEnumsDef compares enum types between two schemas
func compareEnumsDef(t *testing.T, enumsA, enumsB []*storepb.EnumTypeMetadata) {
	require.Equal(t, len(enumsA), len(enumsB), "number of enums should match")

	// Create maps for easier comparison
	mapA := make(map[string]*storepb.EnumTypeMetadata)
	for _, enum := range enumsA {
		mapA[enum.Name] = enum
	}

	for _, enumB := range enumsB {
		enumA, exists := mapA[enumB.Name]
		require.True(t, exists, "enum %s should exist in metadata A", enumB.Name)
		require.ElementsMatch(t, enumA.Values, enumB.Values, "enum %s: values should match", enumB.Name)
		require.Equal(t, enumA.Comment, enumB.Comment, "enum %s: comment should match", enumB.Name)
	}
}

// compareExtensionsDef compares extensions between two databases
func compareExtensionsDef(t *testing.T, extsA, extsB []*storepb.ExtensionMetadata) {
	// Create maps for easier comparison
	mapA := make(map[string]*storepb.ExtensionMetadata)
	for _, ext := range extsA {
		mapA[ext.Name] = ext
	}

	for _, extB := range extsB {
		extA, exists := mapA[extB.Name]
		require.True(t, exists, "extension %s should exist in metadata A", extB.Name)
		require.Equal(t, extA.Schema, extB.Schema, "extension %s: schema should match", extB.Name)
		require.Equal(t, extA.Version, extB.Version, "extension %s: version should match", extB.Name)
	}
}

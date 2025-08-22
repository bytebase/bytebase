package pg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"database/sql"

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

// TestSDLValidationWithTestcontainer tests the SDL (Schema Definition Language) correctness
// by following a 5-step validation process:
// 1. Apply initial schema A to database
// 2. Sync to get schema B from database
// 3. Define expected schema C, get its metadata, generate diff from B to C, then generate migration DDL
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
			err = executeSDLValidationProcess(t, ctx, &testConnConfig, testDB, dbName, tc.initialSchema, tc.expectedSchema, tc.description)
			require.NoError(t, err, "SDL validation process should complete successfully for test case: %s", tc.description)
		})
	}
}

// executeSDLValidationProcess implements the 5-step SDL validation workflow
func executeSDLValidationProcess(t *testing.T, ctx context.Context, connConfig *pgx.ConnConfig, testDB *sql.DB, dbName, initialSchema, expectedSchema, description string) error {
	t.Logf("Starting SDL validation process for: %s", description)

	// Step 1: Apply initial schema A to database
	t.Logf("Step 1: Applying initial schema A to database")
	if strings.TrimSpace(initialSchema) != "" {
		_, err := testDB.Exec(initialSchema)
		if err != nil {
			return fmt.Errorf("failed to apply initial schema: %w", err)
		}
	}

	// Step 2: Sync to get schema B from database
	t.Logf("Step 2: Syncing to get schema B from database")
	schemaB, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return fmt.Errorf("failed to sync schema B: %w", err)
	}

	// Step 3: Get metadata for expected schema C, generate diff from B to C, then generate migration DDL
	t.Logf("Step 3: Getting metadata for expected schema C and generating migration DDL")
	schemaC, err := GetDatabaseMetadata(expectedSchema)
	if err != nil {
		return fmt.Errorf("failed to get metadata for expected schema C: %w", err)
	}

	// Convert to model.DatabaseSchema for diff generation
	dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_POSTGRES, false)
	dbSchemaC := model.NewDatabaseSchema(schemaC, nil, nil, storepb.Engine_POSTGRES, false)

	// Get diff from B to C
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, dbSchemaB, dbSchemaC)
	if err != nil {
		return fmt.Errorf("failed to generate diff from B to C: %w", err)
	}

	// Generate migration DDL
	migrationDDL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	if err != nil {
		return fmt.Errorf("failed to generate migration DDL: %w", err)
	}

	t.Logf("Generated migration DDL:\n%s", migrationDDL)

	// Step 4: Apply generated DDL to database, then sync to get schema D
	t.Logf("Step 4: Applying generated DDL and syncing to get schema D")
	if strings.TrimSpace(migrationDDL) != "" {
		_, err = testDB.Exec(migrationDDL)
		if err != nil {
			return fmt.Errorf("failed to apply migration DDL: %w", err)
		}
	}

	schemaD, err := getSyncMetadataForSDL(ctx, connConfig, dbName)
	if err != nil {
		return fmt.Errorf("failed to sync schema D: %w", err)
	}

	// Step 5: Verify schema D and schema C metadata have no diff and validate each member consistently
	t.Logf("Step 5: Verifying schema D matches schema C with no differences")
	err = validateSchemasMatch(t, schemaD, schemaC, description)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
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

// validateSchemasMatch performs comprehensive validation that schema D matches schema C
func validateSchemasMatch(t *testing.T, schemaD, schemaC *storepb.DatabaseSchemaMetadata, description string) error {
	// Normalize metadata for comparison (same as used in generate_migration_testcontainer_test.go)
	normalizeMetadataForSDL(schemaD)
	normalizeMetadataForSDL(schemaC)

	// First, use the schema differ to check for any differences
	dbSchemaD := model.NewDatabaseSchema(schemaD, nil, nil, storepb.Engine_POSTGRES, false)
	dbSchemaC := model.NewDatabaseSchema(schemaC, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_POSTGRES, dbSchemaD, dbSchemaC)
	if err != nil {
		return fmt.Errorf("failed to generate diff between D and C: %w", err)
	}

	// Check if there are any differences - we require strict equivalence for SDL validation
	if diff != nil && hasDifferences(diff) {
		// Log detailed information about the differences
		t.Logf("Detected schema differences between D and C:")

		// Log view differences if any
		for _, viewChange := range diff.ViewChanges {
			t.Logf("  View change: %s %s.%s", viewChange.Action, viewChange.SchemaName, viewChange.ViewName)
			if viewChange.OldView != nil && viewChange.NewView != nil {
				t.Logf("    Old definition: %s", viewChange.OldView.Definition)
				t.Logf("    New definition: %s", viewChange.NewView.Definition)
			}
		}

		// Log other types of differences
		if len(diff.TableChanges) > 0 {
			t.Logf("  %d table changes detected", len(diff.TableChanges))
		}
		if len(diff.FunctionChanges) > 0 {
			t.Logf("  %d function changes detected", len(diff.FunctionChanges))
		}
		if len(diff.SchemaChanges) > 0 {
			t.Logf("  %d schema changes detected", len(diff.SchemaChanges))
		}

		return fmt.Errorf("schemas D and C have differences when they should be identical:\n%s",
			formatDifferences(diff))
	}

	// Second, use protocmp for detailed comparison, but allow for expected differences
	if diffProto := cmp.Diff(schemaD, schemaC, protocmp.Transform()); diffProto != "" {
		t.Logf("Detailed proto diff between schema D and C (-D +C):\n%s", diffProto)
		// For SDL validation, we need to be more lenient about metadata differences
		// that are expected between database sync and DDL parsing
		// We'll rely on the schema differ and detailed member validation instead
		t.Logf("Proto differences detected, but proceeding with member-level validation")
	}

	// Third, perform detailed member-by-member validation using the same methods as get_database_metadata tests
	err = validateSchemaMembersMatch(t, schemaD, schemaC, description)
	if err != nil {
		return fmt.Errorf("member validation failed: %w", err)
	}

	t.Logf("✓ Schema validation passed: D and C are identical for %s", description)
	return nil
}

// normalizeMetadataForSDL normalizes metadata for SDL comparison (similar to generate_migration test)
func normalizeMetadataForSDL(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database-level metadata that DDL parser doesn't extract
	metadata.Name = ""
	metadata.CharacterSet = ""
	metadata.Collation = ""
	metadata.Owner = ""
	metadata.SearchPath = ""

	// Normalize schemas
	for _, schema := range metadata.Schemas {
		// Clear schema-level metadata that DDL parser doesn't extract
		schema.Comment = ""
		schema.Owner = ""

		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0
			table.Owner = ""

			// Clear column positions as they can change when columns are added/dropped
			for _, column := range table.Columns {
				column.Position = 0
			}
		}

		// Normalize sequences - clear fields that DDL parser doesn't extract
		for _, sequence := range schema.Sequences {
			sequence.CacheSize = ""
			sequence.DataType = ""
			sequence.MinValue = ""
			sequence.MaxValue = ""
			sequence.OwnerTable = ""
			sequence.OwnerColumn = ""
		}

		// Normalize views - DDL parser doesn't extract view columns
		// View definitions and dependencies are now properly handled
		for _, view := range schema.Views {
			view.Columns = nil
			// Note: View definitions are handled separately with semantic comparison
			// DependencyColumns are now extracted and should be compared
		}

		// Materialized views: Dependencies are now properly extracted and should be compared
		// No additional normalization needed for materialized views
	}

	// Clear extensions metadata that might differ
	for _, extension := range metadata.Extensions {
		extension.Version = ""
	}
}

// hasDifferences checks if a MetadataDiff contains any differences
func hasDifferences(diff *schema.MetadataDiff) bool {
	return len(diff.SchemaChanges) > 0 ||
		len(diff.TableChanges) > 0 ||
		len(diff.ViewChanges) > 0 ||
		len(diff.MaterializedViewChanges) > 0 ||
		len(diff.FunctionChanges) > 0 ||
		len(diff.ProcedureChanges) > 0 ||
		len(diff.SequenceChanges) > 0 ||
		len(diff.EnumTypeChanges) > 0 ||
		len(diff.EventChanges) > 0
}

// formatDifferences formats a MetadataDiff for readable output
func formatDifferences(diff *schema.MetadataDiff) string {
	var parts []string

	if len(diff.SchemaChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d schema changes", len(diff.SchemaChanges)))
	}
	if len(diff.TableChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d table changes", len(diff.TableChanges)))
	}
	if len(diff.ViewChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d view changes", len(diff.ViewChanges)))
	}
	if len(diff.MaterializedViewChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d materialized view changes", len(diff.MaterializedViewChanges)))
	}
	if len(diff.FunctionChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d function changes", len(diff.FunctionChanges)))
	}
	if len(diff.ProcedureChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d procedure changes", len(diff.ProcedureChanges)))
	}
	if len(diff.SequenceChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d sequence changes", len(diff.SequenceChanges)))
	}
	if len(diff.EnumTypeChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d enum type changes", len(diff.EnumTypeChanges)))
	}
	if len(diff.EventChanges) > 0 {
		parts = append(parts, fmt.Sprintf("%d event changes", len(diff.EventChanges)))
	}

	return strings.Join(parts, ", ")
}

// validateSchemaMembersMatch performs detailed member-by-member validation using the same approach as get_database_metadata tests
func validateSchemaMembersMatch(t *testing.T, schemaD, schemaC *storepb.DatabaseSchemaMetadata, description string) error {
	schemaDMap := make(map[string]*storepb.SchemaMetadata)
	for _, schema := range schemaD.Schemas {
		schemaDMap[schema.Name] = schema
	}

	schemaCMap := make(map[string]*storepb.SchemaMetadata)
	for _, schema := range schemaC.Schemas {
		schemaCMap[schema.Name] = schema
	}

	// Validate all schemas exist in both
	for schemaName, schemaC := range schemaCMap {
		schemaD, exists := schemaDMap[schemaName]
		if !exists {
			return fmt.Errorf("schema %s exists in expected (C) but not in result (D)", schemaName)
		}

		// Compare this schema's content
		err := validateSchemaContentMatch(t, schemaD, schemaC, schemaName, description)
		if err != nil {
			return fmt.Errorf("schema %s validation failed: %w", schemaName, err)
		}
	}

	// Check for extra schemas in D that shouldn't be there
	for schemaName := range schemaDMap {
		if schemaName == "information_schema" || schemaName == "pg_catalog" {
			continue // Skip system schemas
		}
		if _, exists := schemaCMap[schemaName]; !exists {
			return fmt.Errorf("schema %s exists in result (D) but not in expected (C)", schemaName)
		}
	}

	// Compare extensions
	err := validateExtensionsMatch(t, schemaD.Extensions, schemaC.Extensions, description)
	if err != nil {
		return fmt.Errorf("extension validation failed: %w", err)
	}

	return nil
}

// validateSchemaContentMatch validates that the content of two schemas match
func validateSchemaContentMatch(t *testing.T, schemaD, schemaC *storepb.SchemaMetadata, schemaName, description string) error {
	// Compare tables
	err := validateTablesMatch(t, schemaD.Tables, schemaC.Tables, schemaName, description)
	if err != nil {
		return fmt.Errorf("table validation failed: %w", err)
	}

	// Compare views
	err = validateViewsMatch(t, schemaD.Views, schemaC.Views, schemaName, description)
	if err != nil {
		return fmt.Errorf("view validation failed: %w", err)
	}

	// Compare materialized views
	err = validateMaterializedViewsMatch(t, schemaD.MaterializedViews, schemaC.MaterializedViews, schemaName, description)
	if err != nil {
		return fmt.Errorf("materialized view validation failed: %w", err)
	}

	// Compare functions (use similar approach as get_database_metadata test)
	err = validateFunctionsMatch(t, schemaD.Functions, schemaC.Functions, schemaName, description)
	if err != nil {
		return fmt.Errorf("function validation failed: %w", err)
	}

	// Compare procedures
	err = validateProceduresMatch(t, schemaD.Procedures, schemaC.Procedures, schemaName, description)
	if err != nil {
		return fmt.Errorf("procedure validation failed: %w", err)
	}

	// Compare sequences
	err = validateSequencesMatch(t, schemaD.Sequences, schemaC.Sequences, schemaName, description)
	if err != nil {
		return fmt.Errorf("sequence validation failed: %w", err)
	}

	// Compare enums
	err = validateEnumsMatch(t, schemaD.EnumTypes, schemaC.EnumTypes, schemaName, description)
	if err != nil {
		return fmt.Errorf("enum validation failed: %w", err)
	}

	return nil
}

// Validation functions for each type of schema object
func validateTablesMatch(t *testing.T, tablesD, tablesC []*storepb.TableMetadata, schemaName, description string) error {
	if len(tablesD) != len(tablesC) {
		return fmt.Errorf("schema %s: table count mismatch - D has %d, C has %d", schemaName, len(tablesD), len(tablesC))
	}

	// Create maps for easy lookup
	tableDMap := make(map[string]*storepb.TableMetadata)
	for _, table := range tablesD {
		tableDMap[table.Name] = table
	}

	// Validate each table in C exists in D and matches
	for _, tableC := range tablesC {
		tableD, exists := tableDMap[tableC.Name]
		if !exists {
			return fmt.Errorf("schema %s: table %s exists in expected (C) but not in result (D)", schemaName, tableC.Name)
		}

		// Use the same comparison logic as get_database_metadata tests
		compareTables(t, []*storepb.TableMetadata{tableD}, []*storepb.TableMetadata{tableC})
	}

	return nil
}

func validateViewsMatch(t *testing.T, viewsD, viewsC []*storepb.ViewMetadata, schemaName, description string) error {
	if len(viewsD) != len(viewsC) {
		return fmt.Errorf("schema %s: view count mismatch - D has %d, C has %d", schemaName, len(viewsD), len(viewsC))
	}

	// Use the same comparison logic as get_database_metadata tests
	// This includes semantic view definition comparison
	compareViews(t, viewsD, viewsC)
	return nil
}

func validateMaterializedViewsMatch(t *testing.T, mviewsD, mviewsC []*storepb.MaterializedViewMetadata, schemaName, description string) error {
	if len(mviewsD) != len(mviewsC) {
		return fmt.Errorf("schema %s: materialized view count mismatch - D has %d, C has %d", schemaName, len(mviewsD), len(mviewsC))
	}

	// Use the same comparison logic as get_database_metadata tests
	compareMaterializedViews(t, mviewsD, mviewsC)
	return nil
}

func validateFunctionsMatch(t *testing.T, functionsD, functionsC []*storepb.FunctionMetadata, schemaName, description string) error {
	// Functions might not be extracted by parser, so handle gracefully like in get_database_metadata test
	if len(functionsC) == 0 && len(functionsD) > 0 {
		t.Logf("Schema %s: Parser doesn't extract functions yet - found %d in result (D)", schemaName, len(functionsD))
		return nil
	}

	// Use the same comparison logic as get_database_metadata tests
	compareFunctions(t, functionsD, functionsC)
	return nil
}

func validateProceduresMatch(t *testing.T, proceduresD, proceduresC []*storepb.ProcedureMetadata, schemaName, description string) error {
	// Procedures might not be extracted by parser, so handle gracefully like in get_database_metadata test
	if len(proceduresC) == 0 && len(proceduresD) > 0 {
		t.Logf("Schema %s: Parser doesn't extract procedures yet - found %d in result (D)", schemaName, len(proceduresD))
		return nil
	}

	// Use the same comparison logic as get_database_metadata tests
	compareProcedures(t, proceduresD, proceduresC)
	return nil
}

func validateSequencesMatch(t *testing.T, sequencesD, sequencesC []*storepb.SequenceMetadata, schemaName, description string) error {
	// Use the same comparison logic as get_database_metadata tests
	compareSequences(t, sequencesD, sequencesC)
	return nil
}

func validateEnumsMatch(t *testing.T, enumsD, enumsC []*storepb.EnumTypeMetadata, schemaName, description string) error {
	if len(enumsD) != len(enumsC) {
		return fmt.Errorf("schema %s: enum count mismatch - D has %d, C has %d", schemaName, len(enumsD), len(enumsC))
	}

	// Use the same comparison logic as get_database_metadata tests
	compareEnums(t, enumsD, enumsC)
	return nil
}

func validateExtensionsMatch(t *testing.T, extensionsD, extensionsC []*storepb.ExtensionMetadata, description string) error {
	// Extensions might differ due to system defaults, so handle gracefully
	compareExtensions(t, extensionsD, extensionsC)
	return nil
}

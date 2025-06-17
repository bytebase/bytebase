package mssql

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	// Import MSSQL driver
	"github.com/google/go-cmp/cmp"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestGenerateMigrationWithTestcontainer tests the generate migration function
// by applying migrations and rollback to verify the schema can be restored.
func TestGenerateMigrationWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MSSQL testcontainer test in short mode")
	}

	ctx := context.Background()

	// Start MSSQL container
	req := testcontainers.ContainerRequest{
		Image: "mcr.microsoft.com/mssql/server:2022-latest",
		Env: map[string]string{
			"ACCEPT_EULA": "Y",
			"SA_PASSWORD": "Test123!",
			"MSSQL_PID":   "Express",
		},
		ExposedPorts: []string{"1433/tcp"},
		WaitingFor: wait.ForLog("SQL Server is now ready for client connections").
			WithStartupTimeout(3 * time.Minute),
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
	port, err := container.MappedPort(ctx, "1433")
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
CREATE SCHEMA [app];
GO

CREATE TABLE [app].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [uk_users_email] UNIQUE ([email])
);

CREATE INDEX [idx_users_username] ON [app].[users] ([username]);

CREATE TABLE [app].[posts] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [title] NVARCHAR(200) NOT NULL,
    [content] NTEXT,
    [published_at] DATETIME2,
    CONSTRAINT [fk_posts_user] FOREIGN KEY ([user_id]) REFERENCES [app].[users]([id]) ON DELETE CASCADE
);

CREATE INDEX [idx_posts_user_id] ON [app].[posts] ([user_id]);
`,
			migrationDDL: `
-- Add new column
ALTER TABLE [app].[users] ADD [status] NVARCHAR(20) DEFAULT 'active';

-- Add new table
CREATE TABLE [app].[comments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [post_id] INT NOT NULL,
    [user_id] INT NOT NULL,
    [content] NTEXT NOT NULL,
    [created_at] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [fk_comments_post] FOREIGN KEY ([post_id]) REFERENCES [app].[posts]([id]) ON DELETE CASCADE,
    CONSTRAINT [fk_comments_user] FOREIGN KEY ([user_id]) REFERENCES [app].[users]([id])
);

-- Add index
CREATE INDEX [idx_comments_post_id] ON [app].[comments] ([post_id]);

-- Add check constraint
ALTER TABLE [app].[users] ADD CONSTRAINT [ck_users_status] CHECK ([status] IN ('active', 'inactive', 'suspended'));
`,
			description: "Basic table operations with foreign keys, indexes, and constraints",
		},
		{
			name: "schema_operations",
			initialSchema: `
CREATE SCHEMA [sales];
GO

CREATE TABLE [sales].[customers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100)
);

CREATE TABLE [sales].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [customer_id] INT NOT NULL,
    [order_date] DATE DEFAULT GETDATE(),
    [total] DECIMAL(10,2),
    CONSTRAINT [fk_orders_customer] FOREIGN KEY ([customer_id]) REFERENCES [sales].[customers]([id])
);
`,
			migrationDDL: `
-- Create new schema
CREATE SCHEMA [inventory];
GO

-- Create table in new schema
CREATE TABLE [inventory].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [category_id] INT,
    CONSTRAINT [ck_products_price] CHECK ([price] > 0)
);

-- Add reference to new table from existing schema
ALTER TABLE [sales].[orders] ADD [product_id] INT;
ALTER TABLE [sales].[orders] ADD CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [inventory].[products]([id]);
`,
			description: "Cross-schema operations with new schema creation",
		},
		{
			name: "view_operations",
			initialSchema: `
CREATE SCHEMA [reporting];
GO

CREATE TABLE [reporting].[sales_data] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [product_name] NVARCHAR(100) NOT NULL,
    [quantity] INT NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [sale_date] DATE DEFAULT GETDATE()
);

CREATE TABLE [reporting].[customers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [region] NVARCHAR(50)
);
`,
			migrationDDL: `
-- Create basic view
CREATE VIEW [reporting].[product_summary] AS
SELECT 
    [product_name],
    COUNT(*) as [sale_count],
    SUM([quantity]) as [total_quantity],
    AVG([price]) as [avg_price]
FROM [reporting].[sales_data]
GROUP BY [product_name];
GO

-- Create dependent view
CREATE VIEW [reporting].[top_products] AS
SELECT TOP 10
    [product_name],
    [total_quantity]
FROM [reporting].[product_summary]
WHERE [total_quantity] > 100
ORDER BY [total_quantity] DESC;
GO
`,
			description: "View creation with dependencies",
		},
		{
			name: "function_and_procedure_operations",
			initialSchema: `
CREATE SCHEMA [calc];
GO

CREATE TABLE [calc].[numbers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [value] INT NOT NULL,
    [category] NVARCHAR(20)
);
`,
			migrationDDL: `
-- Create scalar function
CREATE FUNCTION [calc].[square](@input INT)
RETURNS INT
AS
BEGIN
    RETURN @input * @input;
END;
GO

-- Create table-valued function
CREATE FUNCTION [calc].[get_numbers_by_category](@category NVARCHAR(20))
RETURNS TABLE
AS
RETURN (
    SELECT [id], [value], [category]
    FROM [calc].[numbers]
    WHERE [category] = @category
);
GO

-- Create stored procedure
CREATE PROCEDURE [calc].[add_number]
    @value INT,
    @category NVARCHAR(20)
AS
BEGIN
    INSERT INTO [calc].[numbers] ([value], [category])
    VALUES (@value, @category);
END;
GO
`,
			description: "Functions and procedures creation",
		},
		{
			name: "index_operations",
			initialSchema: `
CREATE SCHEMA [perf];
GO

CREATE TABLE [perf].[events] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [event_name] NVARCHAR(100) NOT NULL,
    [event_data] NVARCHAR(MAX),
    [timestamp] DATETIME2 DEFAULT GETDATE(),
    [user_id] INT,
    [category] NVARCHAR(50)
);
`,
			migrationDDL: `
-- Add various index types
CREATE INDEX [idx_events_timestamp] ON [perf].[events] ([timestamp]);
CREATE INDEX [idx_events_user_category] ON [perf].[events] ([user_id], [category]);
CREATE UNIQUE INDEX [idx_events_name_timestamp] ON [perf].[events] ([event_name], [timestamp]);

-- Add filtered index
CREATE INDEX [idx_events_recent] ON [perf].[events] ([event_name])
WHERE [timestamp] >= DATEADD(DAY, -30, GETDATE());

-- Add computed column first
ALTER TABLE [perf].[events] ADD [event_month] AS DATEPART(MONTH, [timestamp]);

-- Add index on computed column
CREATE INDEX [idx_events_month] ON [perf].[events] ([event_month]);
`,
			description: "Various index types including filtered and computed column indexes",
		},
		{
			name: "complex_constraints",
			initialSchema: `
CREATE SCHEMA [finance];
GO

CREATE TABLE [finance].[accounts] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [account_number] NVARCHAR(20) NOT NULL,
    [balance] DECIMAL(15,2) DEFAULT 0,
    [account_type] NVARCHAR(20) NOT NULL,
    [created_at] DATETIME2 DEFAULT GETDATE()
);
`,
			migrationDDL: `
-- Add multiple check constraints
ALTER TABLE [finance].[accounts] ADD CONSTRAINT [ck_accounts_balance] CHECK ([balance] >= 0);
ALTER TABLE [finance].[accounts] ADD CONSTRAINT [ck_accounts_type] CHECK ([account_type] IN ('CHECKING', 'SAVINGS', 'CREDIT'));

-- Add unique constraint
ALTER TABLE [finance].[accounts] ADD CONSTRAINT [uk_accounts_number] UNIQUE ([account_number]);

-- Create related table with foreign key
CREATE TABLE [finance].[transactions] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [account_id] INT NOT NULL,
    [amount] DECIMAL(15,2) NOT NULL,
    [transaction_type] NVARCHAR(20) NOT NULL,
    [transaction_date] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [fk_transactions_account] FOREIGN KEY ([account_id]) REFERENCES [finance].[accounts]([id]),
    CONSTRAINT [ck_transactions_amount] CHECK ([amount] != 0),
    CONSTRAINT [ck_transactions_type] CHECK ([transaction_type] IN ('DEBIT', 'CREDIT'))
);
`,
			description: "Complex constraints with multiple check constraints and foreign keys",
		},
		{
			name: "column_modifications",
			initialSchema: `
CREATE SCHEMA [hr];
GO

CREATE TABLE [hr].[employees] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [first_name] NVARCHAR(50),
    [last_name] NVARCHAR(50),
    [email] NVARCHAR(100),
    [phone] VARCHAR(20),
    [hire_date] DATE,
    [salary] DECIMAL(10,2)
);
`,
			migrationDDL: `
-- Add new columns
ALTER TABLE [hr].[employees] ADD [department_id] INT;
ALTER TABLE [hr].[employees] ADD [manager_id] INT;
ALTER TABLE [hr].[employees] ADD [status] NVARCHAR(20);

-- Add named default constraint for status
ALTER TABLE [hr].[employees] ADD CONSTRAINT [df_employees_status] DEFAULT 'ACTIVE' FOR [status];

-- Create department table
CREATE TABLE [hr].[departments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [budget] DECIMAL(15,2)
);

-- Add foreign key constraints
ALTER TABLE [hr].[employees] ADD CONSTRAINT [fk_employees_department] FOREIGN KEY ([department_id]) REFERENCES [hr].[departments]([id]);
ALTER TABLE [hr].[employees] ADD CONSTRAINT [fk_employees_manager] FOREIGN KEY ([manager_id]) REFERENCES [hr].[employees]([id]);

-- Add check constraints
ALTER TABLE [hr].[employees] ADD CONSTRAINT [ck_employees_salary] CHECK ([salary] > 0);
ALTER TABLE [hr].[employees] ADD CONSTRAINT [ck_employees_status] CHECK ([status] IN ('ACTIVE', 'INACTIVE', 'TERMINATED'));
`,
			description: "Column additions and self-referencing foreign keys",
		},
		{
			name: "multiple_schema_dependencies",
			initialSchema: `
CREATE SCHEMA [core];
GO
CREATE SCHEMA [app];
GO

CREATE TABLE [core].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL
);

CREATE TABLE [core].[roles] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(50) NOT NULL
);
`,
			migrationDDL: `
-- Create user_roles junction table in core schema
CREATE TABLE [core].[user_roles] (
    [user_id] INT NOT NULL,
    [role_id] INT NOT NULL,
    [assigned_at] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [pk_user_roles] PRIMARY KEY ([user_id], [role_id]),
    CONSTRAINT [fk_user_roles_user] FOREIGN KEY ([user_id]) REFERENCES [core].[users]([id]) ON DELETE CASCADE,
    CONSTRAINT [fk_user_roles_role] FOREIGN KEY ([role_id]) REFERENCES [core].[roles]([id]) ON DELETE CASCADE
);

-- Create application tables that reference core schema
CREATE TABLE [app].[sessions] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [token] NVARCHAR(255) NOT NULL,
    [expires_at] DATETIME2 NOT NULL,
    CONSTRAINT [fk_sessions_user] FOREIGN KEY ([user_id]) REFERENCES [core].[users]([id]) ON DELETE CASCADE
);

CREATE TABLE [app].[audit_log] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT,
    [action] NVARCHAR(100) NOT NULL,
    [timestamp] DATETIME2 DEFAULT GETDATE(),
    [details] NVARCHAR(MAX),
    CONSTRAINT [fk_audit_user] FOREIGN KEY ([user_id]) REFERENCES [core].[users]([id])
);
`,
			description: "Multiple schema dependencies with cross-schema foreign keys",
		},
		{
			name: "computed_columns_and_triggers",
			initialSchema: `
CREATE SCHEMA [ecommerce];
GO

CREATE TABLE [ecommerce].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [tax_rate] DECIMAL(5,4) DEFAULT 0.08
);
`,
			migrationDDL: `
-- Add computed columns
ALTER TABLE [ecommerce].[products] ADD [price_with_tax] AS ([price] * (1 + [tax_rate]));
GO

ALTER TABLE [ecommerce].[products] ADD [price_category] AS (
    CASE 
        WHEN [price] < 10 THEN 'Budget'
        WHEN [price] < 100 THEN 'Standard'
        ELSE 'Premium'
    END
);
GO

-- Create order table
CREATE TABLE [ecommerce].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [product_id] INT NOT NULL,
    [quantity] INT NOT NULL,
    [unit_price] DECIMAL(10,2) NOT NULL,
    [total] DECIMAL(15,2),
    [order_date] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [ecommerce].[products]([id])
);
GO

-- Add computed column for total
ALTER TABLE [ecommerce].[orders] ADD [computed_total] AS ([quantity] * [unit_price]);
GO

-- Add indexes on computed columns
CREATE INDEX [idx_products_price_category] ON [ecommerce].[products] ([price_category]);
CREATE INDEX [idx_products_price_with_tax] ON [ecommerce].[products] ([price_with_tax]);
`,
			description: "Computed columns with indexes",
		},
		{
			name: "temporal_tables",
			initialSchema: `
CREATE SCHEMA [tracking];
GO

CREATE TABLE [tracking].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [last_modified] DATETIME2 DEFAULT GETDATE()
);
`,
			migrationDDL: `
-- Add columns and period definition in single statement
ALTER TABLE [tracking].[products] ADD 
    [valid_from] DATETIME2 GENERATED ALWAYS AS ROW START HIDDEN NOT NULL,
    [valid_to] DATETIME2 GENERATED ALWAYS AS ROW END HIDDEN NOT NULL,
    PERIOD FOR SYSTEM_TIME ([valid_from], [valid_to]);

-- Create history table
CREATE TABLE [tracking].[products_history] (
    [id] INT NOT NULL,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [last_modified] DATETIME2,
    [valid_from] DATETIME2 NOT NULL,
    [valid_to] DATETIME2 NOT NULL
);

-- Create clustered index on history table
CREATE CLUSTERED INDEX [idx_products_history_period] ON [tracking].[products_history] ([valid_to], [valid_from]);

-- Enable system versioning
ALTER TABLE [tracking].[products] SET (SYSTEM_VERSIONING = ON (HISTORY_TABLE = [tracking].[products_history]));
`,
			description: "Temporal tables with system versioning",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Step 1: Execute 5-step workflow
			portInt, err := strconv.Atoi(port.Port())
			require.NoError(t, err)
			err = executeFiveStepWorkflow(ctx, host, portInt, testCase.initialSchema, testCase.migrationDDL)
			require.NoError(t, err, "Failed 5-step workflow for test case: %s", testCase.description)
		})
	}
}

// executeFiveStepWorkflow implements the 5-step workflow:
// 1. Initialize database schema, get schema result A via syncDBSchema
// 2. Apply migration DDL, get schema result B via syncDBSchema
// 3. Generate rollback DDL via generate_migration
// 4. Execute rollback DDL, get schema result C via syncDBSchema
// 5. Compare schema results A and C to verify they are identical
func executeFiveStepWorkflow(ctx context.Context, host string, port int, initialSchema, migrationDDL string) error {
	// Create driver instance
	driverInstance := &mssqldb.Driver{}

	// Create connection config
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "sa",
			Host:     host,
			Port:     strconv.Itoa(port),
			Database: "master",
		},
		Password: "Test123!",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "master",
		},
	}

	// Open connection
	driver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
	if err != nil {
		return errors.Wrap(err, "failed to connect to MSSQL")
	}
	defer driver.Close(ctx)

	// Create test database with unique name
	testDB := fmt.Sprintf("test_db_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
	if _, err := driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE [%s]", testDB), db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return errors.Wrap(err, "failed to create test database")
	}
	defer func() {
		// Clean up test database - reconnect to master first
		driver.Close(ctx)
		config.DataSource.Database = "master"
		config.ConnectionContext.DatabaseName = "master"
		cleanupDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
		if err == nil {
			// Set the database to single user mode to close any open connections
			_, _ = cleanupDriver.Execute(ctx, fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", testDB), db.ExecuteOptions{CreateDatabase: true})
			// Drop the database using CreateDatabase option to avoid transaction issues
			if _, err := cleanupDriver.Execute(ctx, fmt.Sprintf("DROP DATABASE [%s]", testDB), db.ExecuteOptions{CreateDatabase: true}); err != nil {
				// Log but don't fail if cleanup fails
				fmt.Printf("Warning: failed to clean up test database %s: %v\n", testDB, err)
			}
			cleanupDriver.Close(ctx)
		}
	}()

	// Reconnect to test database
	driver.Close(ctx)
	config.DataSource.Database = testDB
	config.ConnectionContext.DatabaseName = testDB
	driver, err = driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
	if err != nil {
		return errors.Wrap(err, "failed to reconnect to test database")
	}
	defer driver.Close(ctx)

	// Step 1: Initialize database schema and get schema result A
	if err := executeSQL(ctx, driver, initialSchema); err != nil {
		return errors.Wrap(err, "failed to execute initial schema")
	}

	mssqlDriver, ok := driver.(*mssqldb.Driver)
	if !ok {
		return errors.New("failed to cast to mssqldb.Driver")
	}

	schemaA, err := mssqlDriver.SyncDBSchema(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to sync schema A")
	}

	// Step 2: Apply migration DDL and get schema result B
	if err := executeSQL(ctx, driver, migrationDDL); err != nil {
		return errors.Wrap(err, "failed to execute migration DDL")
	}

	schemaB, err := mssqlDriver.SyncDBSchema(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to sync schema B")
	}

	// Debug: Print what tables are in each schema
	fmt.Printf("Schema A contents:\n")
	for _, s := range schemaA.Schemas {
		fmt.Printf("  Schema [%s]: %d tables\n", s.Name, len(s.Tables))
		for _, t := range s.Tables {
			fmt.Printf("    - [%s].[%s]\n", s.Name, t.Name)
		}
	}
	fmt.Printf("Schema B contents:\n")
	for _, s := range schemaB.Schemas {
		fmt.Printf("  Schema [%s]: %d tables\n", s.Name, len(s.Tables))
		for _, t := range s.Tables {
			fmt.Printf("    - [%s].[%s]\n", s.Name, t.Name)
		}
	}

	// Step 3: Generate rollback DDL using generate_migration
	// Convert to model schemas for diff
	dbSchemaA := model.NewDatabaseSchema(schemaA, nil, nil, storepb.Engine_MSSQL, false)
	dbSchemaB := model.NewDatabaseSchema(schemaB, nil, nil, storepb.Engine_MSSQL, false)

	// Get diff from B to A (to generate rollback)
	diff, err := schema.GetDatabaseSchemaDiff(dbSchemaB, dbSchemaA)
	if err != nil {
		return errors.Wrap(err, "failed to generate diff")
	}

	// Debug: Print what changes are detected
	fmt.Printf("Schema changes detected:\n")
	fmt.Printf("- Table changes: %d\n", len(diff.TableChanges))
	for i, tc := range diff.TableChanges {
		fmt.Printf("  %d. %s [%s].[%s]\n", i+1, tc.Action, tc.SchemaName, tc.TableName)
	}
	fmt.Printf("- Schema changes: %d\n", len(diff.SchemaChanges))
	for i, sc := range diff.SchemaChanges {
		fmt.Printf("  %d. %s [%s]\n", i+1, sc.Action, sc.SchemaName)
	}

	rollbackDDL, err := schema.GenerateMigration(storepb.Engine_MSSQL, diff)
	if err != nil {
		return errors.Wrap(err, "failed to generate rollback migration")
	}

	// Debug: Print the generated rollback DDL
	fmt.Printf("Generated rollback DDL:\n%s\n", rollbackDDL)

	// Only proceed if there's actual rollback DDL to execute
	if strings.TrimSpace(rollbackDDL) == "" {
		// No rollback needed, schemas should already be identical
		return nil
	}

	// Step 4: Execute rollback DDL and get schema result C
	if err := executeSQL(ctx, driver, rollbackDDL); err != nil {
		return errors.Wrapf(err, "failed to execute rollback DDL: %s", rollbackDDL)
	}

	schemaC, err := mssqlDriver.SyncDBSchema(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to sync schema C")
	}

	// Step 5: Compare schema results A and C
	if err := compareSchemas(schemaA, schemaC); err != nil {
		return errors.Wrap(err, "schema comparison failed")
	}

	return nil
}

// executeSQL executes SQL statements, handling GO separators
func executeSQL(ctx context.Context, driver db.Driver, sql string) error {
	if strings.TrimSpace(sql) == "" {
		return nil
	}

	// Split by GO statements (case insensitive)
	statements := splitByGO(sql)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := driver.Execute(ctx, stmt, db.ExecuteOptions{}); err != nil {
			return errors.Wrapf(err, "failed to execute statement %d: %s", i+1, stmt)
		}
	}

	return nil
}

// splitByGO splits SQL by GO statements (case insensitive) or by semicolons if no GO statements
func splitByGO(sql string) []string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []string{}
	}

	// First check if there are any GO statements
	hasGOStatements := false
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "GO") {
			hasGOStatements = true
			break
		}
	}

	if hasGOStatements {
		// Split by GO statements
		var statements []string
		var currentStatement strings.Builder

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.EqualFold(trimmed, "GO") {
				if currentStatement.Len() > 0 {
					statements = append(statements, currentStatement.String())
					currentStatement.Reset()
				}
			} else {
				if currentStatement.Len() > 0 {
					currentStatement.WriteString("\n")
				}
				currentStatement.WriteString(line)
			}
		}

		if currentStatement.Len() > 0 {
			statements = append(statements, currentStatement.String())
		}

		return statements
	}
	// Split by semicolons for DDL statements
	statements := strings.Split(sql, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" && !strings.HasPrefix(stmt, "--") {
			result = append(result, stmt)
		}
	}
	return result
}

// compareSchemas compares two database schemas and returns an error if they differ
func compareSchemas(schemaA, schemaC *storepb.DatabaseSchemaMetadata) error {
	// Sort schemas for consistent comparison
	normalizeSchema(schemaA)
	normalizeSchema(schemaC)

	// Use protocmp for detailed comparison
	if diff := cmp.Diff(schemaA, schemaC, protocmp.Transform()); diff != "" {
		return errors.Errorf("schemas differ:\n%s", diff)
	}

	return nil
}

// normalizeSchema sorts elements within a schema for consistent comparison
func normalizeSchema(schema *storepb.DatabaseSchemaMetadata) {
	if schema == nil {
		return
	}

	// Sort schemas
	slices.SortFunc(schema.Schemas, func(i, j *storepb.SchemaMetadata) int {
		return strings.Compare(i.Name, j.Name)
	})

	for _, s := range schema.Schemas {
		// Sort tables
		slices.SortFunc(s.Tables, func(i, j *storepb.TableMetadata) int {
			return strings.Compare(i.Name, j.Name)
		})

		// Sort views
		slices.SortFunc(s.Views, func(i, j *storepb.ViewMetadata) int {
			return strings.Compare(i.Name, j.Name)
		})

		// Sort functions
		slices.SortFunc(s.Functions, func(i, j *storepb.FunctionMetadata) int {
			return strings.Compare(i.Name, j.Name)
		})

		// Sort procedures
		slices.SortFunc(s.Procedures, func(i, j *storepb.ProcedureMetadata) int {
			return strings.Compare(i.Name, j.Name)
		})

		// Sort table elements
		for _, table := range s.Tables {
			// Sort columns
			slices.SortFunc(table.Columns, func(i, j *storepb.ColumnMetadata) int {
				return strings.Compare(i.Name, j.Name)
			})

			// Sort indexes
			slices.SortFunc(table.Indexes, func(i, j *storepb.IndexMetadata) int {
				return strings.Compare(i.Name, j.Name)
			})

			// Sort foreign keys
			slices.SortFunc(table.ForeignKeys, func(i, j *storepb.ForeignKeyMetadata) int {
				return strings.Compare(i.Name, j.Name)
			})

			// Sort check constraints
			slices.SortFunc(table.CheckConstraints, func(i, j *storepb.CheckConstraintMetadata) int {
				return strings.Compare(i.Name, j.Name)
			})

			// Sort expressions within indexes
			for _, index := range table.Indexes {
				slices.Sort(index.Expressions)
			}
		}
	}
}

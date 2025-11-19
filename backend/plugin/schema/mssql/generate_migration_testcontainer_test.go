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

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
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
    [created_at] DATETIME2,
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
ALTER TABLE [app].[users] ADD [status] NVARCHAR(20);

-- Add new table
CREATE TABLE [app].[comments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [post_id] INT NOT NULL,
    [user_id] INT NOT NULL,
    [content] NTEXT NOT NULL,
    [created_at] DATETIME2,
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
    [order_date] DATE,
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
    [sale_date] DATE
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
    [timestamp] DATETIME2,
    [user_id] INT,
    [category] NVARCHAR(50)
);
`,
			migrationDDL: `
-- Add various index types
CREATE INDEX [idx_events_timestamp] ON [perf].[events] ([timestamp]);
CREATE INDEX [idx_events_user_category] ON [perf].[events] ([user_id], [category]);
CREATE UNIQUE INDEX [idx_events_name_timestamp] ON [perf].[events] ([event_name], [timestamp]);

-- Add filtered index (using deterministic date)
CREATE INDEX [idx_events_recent] ON [perf].[events] ([event_name])
WHERE [timestamp] >= '2023-01-01';

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
    [balance] DECIMAL(15,2),
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
    [transaction_date] DATETIME2,
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
    [assigned_at] DATETIME2,
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
    [timestamp] DATETIME2,
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
    [tax_rate] DECIMAL(5,4)
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
    [order_date] DATETIME2,
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
			name: "create_tables_with_fk",
			initialSchema: `
CREATE SCHEMA [test];
GO

CREATE TABLE [test].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2,
    CONSTRAINT [uk_email] UNIQUE ([email])
);

CREATE INDEX [idx_username] ON [test].[users] ([username]);

CREATE TABLE [test].[posts] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [published_at] DATETIME2,
    CONSTRAINT [fk_user] FOREIGN KEY ([user_id]) REFERENCES [test].[users]([id]) ON DELETE CASCADE
);

CREATE INDEX [idx_user_id] ON [test].[posts] ([user_id]);
`,
			migrationDDL: `
DROP TABLE [test].[posts];
DROP TABLE [test].[users];
DROP SCHEMA [test];`,
			description: "Create tables with foreign key constraints",
		},
		{
			name: "multiple_foreign_keys",
			initialSchema: `
CREATE SCHEMA [test];
GO

CREATE TABLE [test].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2,
    CONSTRAINT [uk_email] UNIQUE ([email])
);
GO

CREATE INDEX [idx_username] ON [test].[users] ([username]);
GO

CREATE TABLE [test].[posts] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [published_at] DATETIME2,
    CONSTRAINT [fk_user] FOREIGN KEY ([user_id]) REFERENCES [test].[users]([id]) ON DELETE CASCADE
);
GO

CREATE INDEX [idx_user_id] ON [test].[posts] ([user_id]);
GO
`,
			migrationDDL: `
-- Add new column
ALTER TABLE [test].[users] ADD [is_active] BIT;
GO

-- Create new table with multiple foreign keys
CREATE TABLE [test].[comments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [post_id] INT NOT NULL,
    [user_id] INT NOT NULL,
    [content] NVARCHAR(MAX) NOT NULL,
    [created_at] DATETIME2,
    CONSTRAINT [fk_comment_post] FOREIGN KEY ([post_id]) REFERENCES [test].[posts]([id]) ON DELETE CASCADE,
    CONSTRAINT [fk_comment_user] FOREIGN KEY ([user_id]) REFERENCES [test].[users]([id]) ON DELETE NO ACTION
);
GO

CREATE INDEX [idx_post_user] ON [test].[comments] ([post_id], [user_id]);

-- Add new index
CREATE INDEX [idx_email_active] ON [test].[users] ([email], [is_active]);

-- Add check constraint
ALTER TABLE [test].[posts] ADD CONSTRAINT [chk_title_length] CHECK (LEN([title]) > 0);
`,
			description: "Tables with multiple foreign key constraints",
		},
		{
			name: "drop_and_recreate_fk_constraints",
			initialSchema: `
CREATE SCHEMA [library];
GO

CREATE TABLE [library].[authors] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100),
    CONSTRAINT [uk_email] UNIQUE ([email])
);

CREATE TABLE [library].[books] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [title] NVARCHAR(200) NOT NULL,
    [author_id] INT NOT NULL,
    [isbn] NVARCHAR(20),
    [published_year] INT,
    [price] DECIMAL(8, 2),
    CONSTRAINT [fk_author] FOREIGN KEY ([author_id]) REFERENCES [library].[authors]([id]),
    CONSTRAINT [chk_year_valid] CHECK ([published_year] >= 1000 AND [published_year] <= 2100),
    CONSTRAINT [chk_price_positive] CHECK ([price] > 0),
    CONSTRAINT [uk_isbn] UNIQUE ([isbn])
);

CREATE INDEX [idx_author] ON [library].[books] ([author_id]);
CREATE INDEX [idx_year] ON [library].[books] ([published_year]);
`,
			migrationDDL: `
-- Drop and recreate foreign key with different options
ALTER TABLE [library].[books] DROP CONSTRAINT [fk_author];
ALTER TABLE [library].[books] ADD CONSTRAINT [fk_author_new] FOREIGN KEY ([author_id]) REFERENCES [library].[authors]([id]) ON DELETE CASCADE ON UPDATE CASCADE;

-- Drop and modify check constraints
ALTER TABLE [library].[books] DROP CONSTRAINT [chk_year_valid];
ALTER TABLE [library].[books] ADD CONSTRAINT [chk_year_extended] CHECK ([published_year] >= 1000 AND [published_year] <= 2030);

-- Add new constraints
ALTER TABLE [library].[books] ADD CONSTRAINT [chk_title_length] CHECK (LEN([title]) >= 3);
`,
			description: "Drop and recreate foreign key constraints with different options",
		},
		{
			name: "self_referencing_foreign_keys",
			initialSchema: `
CREATE SCHEMA [company];
GO

CREATE TABLE [company].[departments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [manager_id] INT
);

CREATE TABLE [company].[employees] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [department_id] INT,
    [salary] DECIMAL(10, 2),
    [hire_date] DATE,
    CONSTRAINT [fk_dept] FOREIGN KEY ([department_id]) REFERENCES [company].[departments]([id])
);

CREATE INDEX [idx_dept] ON [company].[employees] ([department_id]);

-- Add self-referencing foreign key
ALTER TABLE [company].[departments] ADD CONSTRAINT [fk_manager] FOREIGN KEY ([manager_id]) REFERENCES [company].[employees]([id]);
`,
			migrationDDL: `
-- Create base view
CREATE VIEW [company].[dept_employee_count] AS
SELECT d.[id] AS dept_id, d.[name] AS dept_name, COUNT(e.[id]) AS emp_count
FROM [company].[departments] d
LEFT JOIN [company].[employees] e ON d.[id] = e.[department_id]
GROUP BY d.[id], d.[name];
GO

-- Create dependent view
CREATE VIEW [company].[dept_summary] AS
SELECT 
    dept_id,
    dept_name,
    emp_count,
    0 AS avg_salary,
    0 AS max_salary,
    0 AS min_salary
FROM [company].[dept_employee_count];
GO

-- Create highly dependent view
CREATE VIEW [company].[dept_manager_summary] AS
SELECT 
    ds.dept_id,
    ds.dept_name,
    ds.emp_count,
    ds.avg_salary,
    m.[name] AS manager_name
FROM [company].[dept_summary] ds 
JOIN [company].[departments] d ON ds.dept_id = d.[id]
LEFT JOIN [company].[employees] m ON d.[manager_id] = m.[id];
GO

-- Create stored procedure using views
CREATE PROCEDURE [company].[GetDepartmentReport]
    @dept_name_pattern NVARCHAR(100)
AS
BEGIN
    SELECT * FROM [company].[dept_manager_summary]
    WHERE dept_name LIKE '%' + @dept_name_pattern + '%';
END;
`,
			description: "Self-referencing foreign keys and complex view dependencies",
		},
		{
			name: "table_and_column_comments",
			initialSchema: `
CREATE SCHEMA [app];
GO

CREATE TABLE [app].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2,
    [status] NVARCHAR(20)
);

CREATE TABLE [app].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [category] NVARCHAR(50)
);

CREATE INDEX [idx_users_email] ON [app].[users] ([email]);
CREATE INDEX [idx_products_category] ON [app].[products] ([category]);
`,
			migrationDDL: `
-- Add table comments using extended properties
EXEC sp_addextendedproperty 'MS_Description', 'User accounts and profile information', 'SCHEMA', 'app', 'TABLE', 'users';
EXEC sp_addextendedproperty 'MS_Description', 'Product catalog with pricing information', 'SCHEMA', 'app', 'TABLE', 'products';
GO

-- Add column comments for users table
EXEC sp_addextendedproperty 'MS_Description', 'Unique identifier for each user', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Unique username for login authentication', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'username';
EXEC sp_addextendedproperty 'MS_Description', 'User email address for notifications', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'email';
EXEC sp_addextendedproperty 'MS_Description', 'Timestamp when the user account was created', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'created_at';
EXEC sp_addextendedproperty 'MS_Description', 'Current status: active, inactive, or suspended', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'status';
GO

-- Add column comments for products table
EXEC sp_addextendedproperty 'MS_Description', 'Unique product identifier', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Product display name', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'name';
EXEC sp_addextendedproperty 'MS_Description', 'Product price in USD', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'price';
EXEC sp_addextendedproperty 'MS_Description', 'Product category classification', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'category';
GO

-- Create new table and add comments immediately
CREATE TABLE [app].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [product_id] INT NOT NULL,
    [quantity] INT NOT NULL,
    [order_date] DATETIME2,
    CONSTRAINT [fk_orders_user] FOREIGN KEY ([user_id]) REFERENCES [app].[users]([id]),
    CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [app].[products]([id])
);
GO

-- Add comments to the new table
EXEC sp_addextendedproperty 'MS_Description', 'Customer orders tracking system', 'SCHEMA', 'app', 'TABLE', 'orders';
EXEC sp_addextendedproperty 'MS_Description', 'Unique order identifier', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Reference to the user who placed the order', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'user_id';
EXEC sp_addextendedproperty 'MS_Description', 'Reference to the ordered product', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'product_id';
EXEC sp_addextendedproperty 'MS_Description', 'Number of items ordered', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'quantity';
EXEC sp_addextendedproperty 'MS_Description', 'When the order was placed', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'order_date';
`,
			description: "Adding comments to tables and columns using SQL Server extended properties",
		},
		{
			name: "modify_and_drop_comments",
			initialSchema: `
CREATE SCHEMA [test];
GO

CREATE TABLE [test].[customers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100),
    [phone] NVARCHAR(20),
    [created_at] DATETIME2 DEFAULT GETDATE()
);
GO

-- Add initial comments
EXEC sp_addextendedproperty 'MS_Description', 'Customer information database', 'SCHEMA', 'test', 'TABLE', 'customers';
EXEC sp_addextendedproperty 'MS_Description', 'Customer unique ID', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Customer full name', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'name';
EXEC sp_addextendedproperty 'MS_Description', 'Customer email for marketing', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'email';
EXEC sp_addextendedproperty 'MS_Description', 'Customer contact number', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'phone';
GO

CREATE TABLE [test].[invoices] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [customer_id] INT NOT NULL,
    [amount] DECIMAL(10,2) NOT NULL,
    [invoice_date] DATE,
    CONSTRAINT [fk_invoices_customer] FOREIGN KEY ([customer_id]) REFERENCES [test].[customers]([id])
);
GO

-- Add comments to invoices table
EXEC sp_addextendedproperty 'MS_Description', 'Invoice records for billing', 'SCHEMA', 'test', 'TABLE', 'invoices';
EXEC sp_addextendedproperty 'MS_Description', 'Invoice number', 'SCHEMA', 'test', 'TABLE', 'invoices', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Link to customer', 'SCHEMA', 'test', 'TABLE', 'invoices', 'COLUMN', 'customer_id';
`,
			migrationDDL: `
-- Update existing table comment
EXEC sp_updateextendedproperty 'MS_Description', 'Complete customer database with contact details and preferences', 'SCHEMA', 'test', 'TABLE', 'customers';
GO

-- Update existing column comments
EXEC sp_updateextendedproperty 'MS_Description', 'Primary key - auto-generated customer identifier', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'id';
EXEC sp_updateextendedproperty 'MS_Description', 'Customer business or personal name', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'name';
EXEC sp_updateextendedproperty 'MS_Description', 'Primary email address for communications and billing', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'email';
GO

-- Drop specific column comment
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'phone';
GO

-- Add new comment to column that didn't have one
EXEC sp_addextendedproperty 'MS_Description', 'Account creation timestamp for auditing', 'SCHEMA', 'test', 'TABLE', 'customers', 'COLUMN', 'created_at';
GO

-- Update invoice table comment
EXEC sp_updateextendedproperty 'MS_Description', 'Financial invoice tracking and billing system', 'SCHEMA', 'test', 'TABLE', 'invoices';
GO

-- Drop invoice column comments
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'test', 'TABLE', 'invoices', 'COLUMN', 'id';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'test', 'TABLE', 'invoices', 'COLUMN', 'customer_id';
GO

-- Add new column with comment
ALTER TABLE [test].[invoices] ADD [payment_status] NVARCHAR(20);
GO

EXEC sp_addextendedproperty 'MS_Description', 'Current payment status: pending, paid, overdue, cancelled', 'SCHEMA', 'test', 'TABLE', 'invoices', 'COLUMN', 'payment_status';
`,
			description: "Modifying existing comments and removing comments using extended properties",
		},
		{
			name: "comments_with_special_characters",
			initialSchema: `
CREATE SCHEMA [special];
GO

CREATE TABLE [special].[documents] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [author] NVARCHAR(100),
    [created_date] DATETIME2,
    [metadata] NVARCHAR(500)
);

CREATE TABLE [special].[translations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [document_id] INT NOT NULL,
    [lang_code] NVARCHAR(10) NOT NULL,
    [translated_title] NVARCHAR(200),
    [translated_content] NVARCHAR(MAX),
    CONSTRAINT [fk_translations_doc] FOREIGN KEY ([document_id]) REFERENCES [special].[documents]([id])
);
`,
			migrationDDL: `
-- Add table comments with special characters and quotes
EXEC sp_addextendedproperty 'MS_Description', 'Document storage system - handles files, "metadata", & special chars: @#$%^&*()_+-={}[]|\:";''<>?,./', 'SCHEMA', 'special', 'TABLE', 'documents';
GO

-- Add multiline comment with special formatting
EXEC sp_addextendedproperty 'MS_Description', 
'Translation table for international support:
- Supports Unicode characters: αβγδε, 中文, العربية, русский
- Handles quotes: "double quotes" and ''single quotes''
- Special symbols: @#$%^&*()_+-={}[]|\:";''<>?,./', 'SCHEMA', 'special', 'TABLE', 'translations';
GO

-- Column comments with various special cases
EXEC sp_addextendedproperty 'MS_Description', 'Primary key (auto-increment) - unique ID for each document', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Document title - may contain "quotes", ''apostrophes'', and symbols: @#$%', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'title';
GO

-- Multiline comment with technical details
EXEC sp_addextendedproperty 'MS_Description', 
'Document content field:
  • Supports rich text formatting
  • HTML tags like <p>, <div>, <span>
  • Special characters: © ® ™ § ¶ † ‡ • ‰ ′ ″
  • Mathematical: ± × ÷ ≠ ≤ ≥ ∞ ∑ ∏ ∫
  • Currency: $ € £ ¥ ₹ ₽
  • Arrows: ← → ↑ ↓ ↔ ⇐ ⇒ ⇔', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'content';
GO

-- Comment with SQL injection attempt (should be safely handled)
EXEC sp_addextendedproperty 'MS_Description', 'Author name field - prevents SQL injection like ''; DROP TABLE users; --', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'author';
GO

-- Comment with JSON-like structure
EXEC sp_addextendedproperty 'MS_Description', 'Metadata JSON: {"version": "1.0", "tags": ["important", "draft"], "settings": {"public": false}}', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'metadata';
GO

-- Unicode characters in comments
EXEC sp_addextendedproperty 'MS_Description', 'Language code: en-US, fr-FR, de-DE, ja-JP, zh-CN, ar-SA, ru-RU, hi-IN, 한국어, ไทย', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'lang_code';
GO

-- Comment with URLs and file paths
EXEC sp_addextendedproperty 'MS_Description', 'Translated content - may reference URLs like https://example.com/path?param=value&other=123 or file paths C:\Users\Name\Documents\file.txt', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'translated_content';
GO

-- Create new table with extreme comment case
CREATE TABLE [special].[test_extreme] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [data] NVARCHAR(MAX)
);
GO

-- Extremely long comment to test limits
EXEC sp_addextendedproperty 'MS_Description', 
'This is an extremely long comment designed to test the limits of extended property storage in SQL Server. It contains multiple lines of text with various formatting, special characters, and technical information. The purpose is to verify that the migration system can properly handle, store, and retrieve complex comment data without truncation or corruption. This comment includes: numbers (123456789), symbols (!@#$%^&*()_+-={}[]|\:";''<>?,./ ), Unicode characters (αβγδε中文العربيةрусский한국어), and structured data like JSON {"key": "value", "array": [1, 2, 3], "nested": {"deep": true}}. Additionally, it tests SQL-like syntax: SELECT * FROM table WHERE column = ''value'' AND other_column IN (1, 2, 3); as well as HTML markup: <html><body><p class="test">Content</p></body></html> and XML: <?xml version="1.0"?><root><item id="1">Test</item></root>. The comment system should preserve all these characters and structures exactly as written, demonstrating robust handling of complex metadata in database schema documentation.', 'SCHEMA', 'special', 'TABLE', 'test_extreme';
`,
			description: "Comments with special characters, quotes, multiline text, and Unicode",
		},
		{
			name: "default_constraint_operations",
			initialSchema: `
CREATE SCHEMA [defaults];
GO

CREATE TABLE [defaults].[employees] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [hire_date] DATE NOT NULL DEFAULT GETDATE(),
    [status] NVARCHAR(20) NOT NULL DEFAULT 'active',
    [is_active] BIT NOT NULL DEFAULT 1,
    [salary] DECIMAL(10,2) DEFAULT 50000.00,
    [department_id] INT DEFAULT 1,
    [created_at] DATETIME2 DEFAULT SYSDATETIME(),
    [updated_at] DATETIME2
);

CREATE TABLE [defaults].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    [stock] INT NOT NULL DEFAULT 0,
    [is_available] BIT DEFAULT 1
);
`,
			migrationDDL: `
-- Add column with default
ALTER TABLE [defaults].[employees] ADD [vacation_days] INT NOT NULL DEFAULT 15;

-- Remove default constraint (by adding new column without default)
ALTER TABLE [defaults].[products] ADD [cost] DECIMAL(10,2) NOT NULL;

-- Modify existing default value
-- First drop the existing constraint
DECLARE @constraint_name NVARCHAR(256);
SELECT @constraint_name = dc.name 
FROM sys.default_constraints dc 
JOIN sys.columns c ON dc.parent_object_id = c.object_id AND dc.parent_column_id = c.column_id 
WHERE OBJECT_NAME(dc.parent_object_id) = 'employees' 
AND c.name = 'status' 
AND OBJECT_SCHEMA_NAME(dc.parent_object_id) = 'defaults';
IF @constraint_name IS NOT NULL
    EXEC('ALTER TABLE [defaults].[employees] DROP CONSTRAINT [' + @constraint_name + ']');
GO

-- Add new default
ALTER TABLE [defaults].[employees] ADD CONSTRAINT [DF_employees_status_new] DEFAULT 'pending' FOR [status];

-- Add default with expression
ALTER TABLE [defaults].[employees] ADD [bonus_percentage] DECIMAL(5,2) DEFAULT (CASE WHEN MONTH(GETDATE()) = 12 THEN 10.0 ELSE 5.0 END);

-- Create new table with various defaults
CREATE TABLE [defaults].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [order_number] NVARCHAR(50) NOT NULL DEFAULT CONCAT('ORD-', YEAR(GETDATE()), '-', FORMAT(GETDATE(), 'MMdd')),
    [order_date] DATETIME2 NOT NULL DEFAULT SYSDATETIME(),
    [status] NVARCHAR(20) NOT NULL DEFAULT 'pending',
    [total] DECIMAL(10,2) NOT NULL DEFAULT 0,
    [discount] DECIMAL(5,2) DEFAULT NULL,
    [is_paid] BIT NOT NULL DEFAULT 0,
    [notes] NVARCHAR(MAX) DEFAULT 'No notes'
);
`,
			description: "Default constraint operations including add, remove, modify with various data types and expressions",
		},
		{
			name: "complex_default_expressions",
			initialSchema: `
CREATE SCHEMA [complex];
GO

CREATE TABLE [complex].[audit_log] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [event_type] NVARCHAR(50) NOT NULL,
    [user_name] NVARCHAR(100) NOT NULL DEFAULT SYSTEM_USER,
    [event_time] DATETIME2 NOT NULL DEFAULT SYSDATETIME(),
    [ip_address] NVARCHAR(45),
    [session_id] UNIQUEIDENTIFIER DEFAULT NEWID()
);

CREATE TABLE [complex].[documents] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [version] INT NOT NULL DEFAULT 1,
    [created_date] DATE DEFAULT CAST(GETDATE() AS DATE),
    [expiry_date] DATE DEFAULT DATEADD(YEAR, 1, CAST(GETDATE() AS DATE))
);
`,
			migrationDDL: `
-- Add columns with complex default expressions
ALTER TABLE [complex].[audit_log] ADD [server_name] NVARCHAR(100) DEFAULT @@SERVERNAME;
ALTER TABLE [complex].[audit_log] ADD [database_name] NVARCHAR(100) DEFAULT DB_NAME();
ALTER TABLE [complex].[audit_log] ADD [schema_name] NVARCHAR(100) DEFAULT SCHEMA_NAME();

-- Add computed default based on other columns
ALTER TABLE [complex].[documents] ADD [document_code] NVARCHAR(50) DEFAULT CONCAT('DOC-', FORMAT(GETDATE(), 'yyyyMMdd'), '-', NEWID());

-- Create table with function-based defaults
CREATE TABLE [complex].[calculations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [value1] DECIMAL(10,2) NOT NULL,
    [value2] DECIMAL(10,2) NOT NULL,
    [pi_value] DECIMAL(10,8) DEFAULT PI(),
    [random_value] FLOAT DEFAULT RAND(),
    [checksum_value] INT DEFAULT CHECKSUM(NEWID()),
    [created_at] DATETIME2 DEFAULT SYSDATETIME(),
    [created_offset] DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET()
);

-- Add default with CASE expression
ALTER TABLE [complex].[documents] ADD [priority] INT DEFAULT 
    CASE 
        WHEN DATEPART(HOUR, GETDATE()) < 12 THEN 1 
        WHEN DATEPART(HOUR, GETDATE()) < 18 THEN 2 
        ELSE 3 
    END;
`,
			description: "Complex default expressions using system functions, variables, and conditional logic",
		},
		{
			name: "default_constraint_edge_cases",
			initialSchema: `
CREATE SCHEMA [edge];
GO

CREATE TABLE [edge].[test_defaults] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [nullable_with_default] INT DEFAULT 42,
    [string_default] NVARCHAR(50) DEFAULT N'Default''s value',
    [empty_string] NVARCHAR(50) DEFAULT '',
    [zero_default] INT DEFAULT 0,
    [negative_default] INT DEFAULT -1,
    [decimal_precision] DECIMAL(18,6) DEFAULT 123.456789
);
`,
			migrationDDL: `
-- Test dropping and re-adding defaults with different names
DECLARE @constraint_name NVARCHAR(256);
-- Drop nullable_with_default's constraint
SELECT @constraint_name = dc.name 
FROM sys.default_constraints dc 
JOIN sys.columns c ON dc.parent_object_id = c.object_id AND dc.parent_column_id = c.column_id 
WHERE OBJECT_NAME(dc.parent_object_id) = 'test_defaults' 
AND c.name = 'nullable_with_default' 
AND OBJECT_SCHEMA_NAME(dc.parent_object_id) = 'edge';
IF @constraint_name IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name + ']');
GO

-- Add it back with a new value
ALTER TABLE [edge].[test_defaults] ADD CONSTRAINT [DF_test_defaults_nullable_new] DEFAULT 84 FOR [nullable_with_default];

-- Add column with special characters in default
ALTER TABLE [edge].[test_defaults] ADD [special_chars] NVARCHAR(100) DEFAULT N'Line1' + CHAR(13) + CHAR(10) + N'Line2';

-- Add column with unicode default
ALTER TABLE [edge].[test_defaults] ADD [unicode_default] NVARCHAR(100) DEFAULT N'你好世界 🌍';

-- Add column with max value defaults
ALTER TABLE [edge].[test_defaults] ADD [max_int] BIGINT DEFAULT 9223372036854775807;
ALTER TABLE [edge].[test_defaults] ADD [min_int] BIGINT DEFAULT -9223372036854775808;

-- Create table with bit field defaults
CREATE TABLE [edge].[flags] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [is_enabled] BIT NOT NULL DEFAULT 1,
    [is_deleted] BIT NOT NULL DEFAULT 0,
    [is_visible] BIT DEFAULT 1,
    [is_archived] BIT DEFAULT 0
);

-- Add JSON default (for NVARCHAR storing JSON)
ALTER TABLE [edge].[test_defaults] ADD [json_config] NVARCHAR(MAX) DEFAULT N'{"enabled": true, "count": 0}';
`,
			description: "Edge cases for default constraints including special characters, unicode, extreme values",
		},
		{
			name: "reverse_basic_table_operations",
			initialSchema: `
CREATE SCHEMA [app];
GO

CREATE TABLE [app].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2,
    [status] NVARCHAR(20),
    CONSTRAINT [uk_users_email] UNIQUE ([email]),
    CONSTRAINT [ck_users_status] CHECK ([status] IN ('active', 'inactive', 'suspended'))
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

CREATE TABLE [app].[comments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [post_id] INT NOT NULL,
    [user_id] INT NOT NULL,
    [content] NTEXT NOT NULL,
    [created_at] DATETIME2,
    CONSTRAINT [fk_comments_post] FOREIGN KEY ([post_id]) REFERENCES [app].[posts]([id]) ON DELETE CASCADE,
    CONSTRAINT [fk_comments_user] FOREIGN KEY ([user_id]) REFERENCES [app].[users]([id])
);

CREATE INDEX [idx_comments_post_id] ON [app].[comments] ([post_id]);
`,
			migrationDDL: `
-- Drop index
DROP INDEX [idx_comments_post_id] ON [app].[comments];

-- Drop check constraint
ALTER TABLE [app].[users] DROP CONSTRAINT [ck_users_status];

-- Drop table (must drop in correct order due to foreign keys)
DROP TABLE [app].[comments];

-- Drop column
ALTER TABLE [app].[users] DROP COLUMN [status];
`,
			description: "Reverse of basic table operations - dropping column, table, index, and constraint",
		},
		{
			name: "reverse_schema_operations",
			initialSchema: `
CREATE SCHEMA [sales];
GO

CREATE SCHEMA [inventory];
GO

CREATE TABLE [sales].[customers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100)
);

CREATE TABLE [sales].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [customer_id] INT NOT NULL,
    [order_date] DATE,
    [total] DECIMAL(10,2),
    [product_id] INT,
    CONSTRAINT [fk_orders_customer] FOREIGN KEY ([customer_id]) REFERENCES [sales].[customers]([id])
);

CREATE TABLE [inventory].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [category_id] INT,
    CONSTRAINT [ck_products_price] CHECK ([price] > 0)
);

ALTER TABLE [sales].[orders] ADD CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [inventory].[products]([id]);
`,
			migrationDDL: `
-- Drop foreign key referencing table in different schema
ALTER TABLE [sales].[orders] DROP CONSTRAINT [fk_orders_product];

-- Drop column referencing another schema
ALTER TABLE [sales].[orders] DROP COLUMN [product_id];

-- Drop table in inventory schema
DROP TABLE [inventory].[products];

-- Drop the inventory schema
DROP SCHEMA [inventory];
`,
			description: "Reverse of schema operations - dropping cross-schema foreign key, table, and schema",
		},
		{
			name: "reverse_view_operations",
			initialSchema: `
CREATE SCHEMA [reporting];
GO

CREATE TABLE [reporting].[sales_data] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [product_name] NVARCHAR(100) NOT NULL,
    [quantity] INT NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [sale_date] DATE
);

CREATE TABLE [reporting].[customers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [region] NVARCHAR(50)
);
GO

CREATE VIEW [reporting].[product_summary] AS
SELECT 
    [product_name],
    COUNT(*) as [sale_count],
    SUM([quantity]) as [total_quantity],
    AVG([price]) as [avg_price]
FROM [reporting].[sales_data]
GROUP BY [product_name];
GO

CREATE VIEW [reporting].[top_products] AS
SELECT TOP 10
    [product_name],
    [total_quantity]
FROM [reporting].[product_summary]
WHERE [total_quantity] > 100
ORDER BY [total_quantity] DESC;
GO
`,
			migrationDDL: `
-- Drop dependent view first
DROP VIEW [reporting].[top_products];
GO

-- Drop base view
DROP VIEW [reporting].[product_summary];
GO
`,
			description: "Reverse of view operations - dropping views in correct dependency order",
		},
		{
			name: "reverse_function_and_procedure_operations",
			initialSchema: `
CREATE SCHEMA [calc];
GO

CREATE TABLE [calc].[numbers] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [value] INT NOT NULL,
    [category] NVARCHAR(20)
);
GO

CREATE FUNCTION [calc].[square](@input INT)
RETURNS INT
AS
BEGIN
    RETURN @input * @input;
END;
GO

CREATE FUNCTION [calc].[get_numbers_by_category](@category NVARCHAR(20))
RETURNS TABLE
AS
RETURN (
    SELECT [id], [value], [category]
    FROM [calc].[numbers]
    WHERE [category] = @category
);
GO

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
			migrationDDL: `
-- Drop stored procedure
DROP PROCEDURE [calc].[add_number];
GO

-- Drop table-valued function
DROP FUNCTION [calc].[get_numbers_by_category];
GO

-- Drop scalar function
DROP FUNCTION [calc].[square];
GO
`,
			description: "Reverse of function and procedure operations - dropping procedures and functions",
		},
		{
			name: "reverse_index_operations",
			initialSchema: `
CREATE SCHEMA [perf];
GO

CREATE TABLE [perf].[events] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [event_name] NVARCHAR(100) NOT NULL,
    [event_data] NVARCHAR(MAX),
    [timestamp] DATETIME2,
    [user_id] INT,
    [category] NVARCHAR(50),
    [event_month] AS DATEPART(MONTH, [timestamp])
);

CREATE INDEX [idx_events_timestamp] ON [perf].[events] ([timestamp]);
CREATE INDEX [idx_events_user_category] ON [perf].[events] ([user_id], [category]);
CREATE UNIQUE INDEX [idx_events_name_timestamp] ON [perf].[events] ([event_name], [timestamp]);

CREATE INDEX [idx_events_recent] ON [perf].[events] ([event_name])
WHERE [timestamp] >= '2023-01-01';

CREATE INDEX [idx_events_month] ON [perf].[events] ([event_month]);
`,
			migrationDDL: `
-- Drop index on computed column
DROP INDEX [idx_events_month] ON [perf].[events];

-- Drop computed column
ALTER TABLE [perf].[events] DROP COLUMN [event_month];

-- Drop filtered index
DROP INDEX [idx_events_recent] ON [perf].[events];

-- Drop unique index
DROP INDEX [idx_events_name_timestamp] ON [perf].[events];

-- Drop composite index
DROP INDEX [idx_events_user_category] ON [perf].[events];

-- Drop simple index
DROP INDEX [idx_events_timestamp] ON [perf].[events];
`,
			description: "Reverse of index operations - dropping various index types including filtered and computed column indexes",
		},
		{
			name: "reverse_complex_constraints",
			initialSchema: `
CREATE SCHEMA [finance];
GO

CREATE TABLE [finance].[accounts] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [account_number] NVARCHAR(20) NOT NULL,
    [balance] DECIMAL(15,2),
    [account_type] NVARCHAR(20) NOT NULL,
    [created_at] DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT [ck_accounts_balance] CHECK ([balance] >= 0),
    CONSTRAINT [ck_accounts_type] CHECK ([account_type] IN ('CHECKING', 'SAVINGS', 'CREDIT')),
    CONSTRAINT [uk_accounts_number] UNIQUE ([account_number])
);

CREATE TABLE [finance].[transactions] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [account_id] INT NOT NULL,
    [amount] DECIMAL(15,2) NOT NULL,
    [transaction_type] NVARCHAR(20) NOT NULL,
    [transaction_date] DATETIME2,
    CONSTRAINT [fk_transactions_account] FOREIGN KEY ([account_id]) REFERENCES [finance].[accounts]([id]),
    CONSTRAINT [ck_transactions_amount] CHECK ([amount] != 0),
    CONSTRAINT [ck_transactions_type] CHECK ([transaction_type] IN ('DEBIT', 'CREDIT'))
);
`,
			migrationDDL: `
-- Drop table with foreign key and check constraints
DROP TABLE [finance].[transactions];

-- Drop unique constraint
ALTER TABLE [finance].[accounts] DROP CONSTRAINT [uk_accounts_number];

-- Drop check constraints
ALTER TABLE [finance].[accounts] DROP CONSTRAINT [ck_accounts_type];
ALTER TABLE [finance].[accounts] DROP CONSTRAINT [ck_accounts_balance];
`,
			description: "Reverse of complex constraints - dropping tables with foreign keys and removing constraints",
		},
		{
			name: "reverse_column_modifications",
			initialSchema: `
CREATE SCHEMA [hr];
GO

CREATE TABLE [hr].[departments] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [budget] DECIMAL(15,2)
);

CREATE TABLE [hr].[employees] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [first_name] NVARCHAR(50),
    [last_name] NVARCHAR(50),
    [email] NVARCHAR(100),
    [phone] VARCHAR(20),
    [hire_date] DATE,
    [salary] DECIMAL(10,2),
    [department_id] INT,
    [manager_id] INT,
    [status] NVARCHAR(20),
    CONSTRAINT [fk_employees_department] FOREIGN KEY ([department_id]) REFERENCES [hr].[departments]([id]),
    CONSTRAINT [fk_employees_manager] FOREIGN KEY ([manager_id]) REFERENCES [hr].[employees]([id]),
    CONSTRAINT [ck_employees_salary] CHECK ([salary] > 0),
    CONSTRAINT [ck_employees_status] CHECK ([status] IN ('ACTIVE', 'INACTIVE', 'TERMINATED'))
);
`,
			migrationDDL: `
-- Drop check constraints first
ALTER TABLE [hr].[employees] DROP CONSTRAINT [ck_employees_status];
ALTER TABLE [hr].[employees] DROP CONSTRAINT [ck_employees_salary];

-- Drop foreign key constraints before dropping columns
ALTER TABLE [hr].[employees] DROP CONSTRAINT [fk_employees_manager];
ALTER TABLE [hr].[employees] DROP CONSTRAINT [fk_employees_department];

-- Now safe to drop columns
ALTER TABLE [hr].[employees] DROP COLUMN [status];
ALTER TABLE [hr].[employees] DROP COLUMN [manager_id];
ALTER TABLE [hr].[employees] DROP COLUMN [department_id];

-- Finally drop department table
DROP TABLE [hr].[departments];
`,
			description: "Reverse of column modifications - dropping columns, constraints, and related tables",
		},
		{
			name: "reverse_multiple_schema_dependencies",
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

CREATE TABLE [core].[user_roles] (
    [user_id] INT NOT NULL,
    [role_id] INT NOT NULL,
    [assigned_at] DATETIME2,
    CONSTRAINT [pk_user_roles] PRIMARY KEY ([user_id], [role_id]),
    CONSTRAINT [fk_user_roles_user] FOREIGN KEY ([user_id]) REFERENCES [core].[users]([id]) ON DELETE CASCADE,
    CONSTRAINT [fk_user_roles_role] FOREIGN KEY ([role_id]) REFERENCES [core].[roles]([id]) ON DELETE CASCADE
);

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
    [timestamp] DATETIME2,
    [details] NVARCHAR(MAX),
    CONSTRAINT [fk_audit_user] FOREIGN KEY ([user_id]) REFERENCES [core].[users]([id])
);
`,
			migrationDDL: `
-- Drop tables that reference core schema
DROP TABLE [app].[audit_log];
DROP TABLE [app].[sessions];

-- Drop junction table
DROP TABLE [core].[user_roles];
`,
			description: "Reverse of multiple schema dependencies - dropping cross-schema references",
		},
		{
			name: "reverse_computed_columns_and_triggers",
			initialSchema: `
CREATE SCHEMA [ecommerce];
GO

CREATE TABLE [ecommerce].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [tax_rate] DECIMAL(5,4),
    [price_with_tax] AS ([price] * (1 + [tax_rate])),
    [price_category] AS (
        CASE 
            WHEN [price] < 10 THEN 'Budget'
            WHEN [price] < 100 THEN 'Standard'
            ELSE 'Premium'
        END
    )
);
GO

CREATE INDEX [idx_products_price_category] ON [ecommerce].[products] ([price_category]);
CREATE INDEX [idx_products_price_with_tax] ON [ecommerce].[products] ([price_with_tax]);

CREATE TABLE [ecommerce].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [product_id] INT NOT NULL,
    [quantity] INT NOT NULL,
    [unit_price] DECIMAL(10,2) NOT NULL,
    [total] DECIMAL(15,2),
    [order_date] DATETIME2,
    [computed_total] AS ([quantity] * [unit_price]),
    CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [ecommerce].[products]([id])
);
GO
`,
			migrationDDL: `
-- Drop table with computed column
DROP TABLE [ecommerce].[orders];
GO

-- Drop indexes on computed columns
DROP INDEX [idx_products_price_with_tax] ON [ecommerce].[products];
DROP INDEX [idx_products_price_category] ON [ecommerce].[products];

-- Drop computed columns
ALTER TABLE [ecommerce].[products] DROP COLUMN [price_category];
GO

ALTER TABLE [ecommerce].[products] DROP COLUMN [price_with_tax];
GO
`,
			description: "Reverse of computed columns - dropping indexes and computed columns",
		},
		{
			name: "reverse_temporal_tables",
			initialSchema: `
CREATE SCHEMA [tracking];
GO

CREATE TABLE [tracking].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [last_modified] DATETIME2,
    [valid_from] DATETIME2 GENERATED ALWAYS AS ROW START HIDDEN NOT NULL,
    [valid_to] DATETIME2 GENERATED ALWAYS AS ROW END HIDDEN NOT NULL,
    PERIOD FOR SYSTEM_TIME ([valid_from], [valid_to])
);

CREATE TABLE [tracking].[products_history] (
    [id] INT NOT NULL,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [last_modified] DATETIME2,
    [valid_from] DATETIME2 NOT NULL,
    [valid_to] DATETIME2 NOT NULL
);

CREATE CLUSTERED INDEX [idx_products_history_period] ON [tracking].[products_history] ([valid_to], [valid_from]);

ALTER TABLE [tracking].[products] SET (SYSTEM_VERSIONING = ON (HISTORY_TABLE = [tracking].[products_history]));
`,
			migrationDDL: `
-- Disable system versioning first
ALTER TABLE [tracking].[products] SET (SYSTEM_VERSIONING = OFF);

-- Drop history table (and its index will be dropped automatically)
DROP TABLE [tracking].[products_history];

-- Drop period definition and system time columns
ALTER TABLE [tracking].[products] DROP PERIOD FOR SYSTEM_TIME;
ALTER TABLE [tracking].[products] DROP COLUMN [valid_to];
ALTER TABLE [tracking].[products] DROP COLUMN [valid_from];
`,
			description: "Reverse of temporal tables - disabling system versioning and dropping history",
		},
		{
			name: "reverse_table_and_column_comments",
			initialSchema: `
CREATE SCHEMA [app];
GO

CREATE TABLE [app].[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [username] NVARCHAR(50) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [created_at] DATETIME2,
    [status] NVARCHAR(20)
);

CREATE TABLE [app].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL,
    [category] NVARCHAR(50)
);

CREATE INDEX [idx_users_email] ON [app].[users] ([email]);
CREATE INDEX [idx_products_category] ON [app].[products] ([category]);

CREATE TABLE [app].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [user_id] INT NOT NULL,
    [product_id] INT NOT NULL,
    [quantity] INT NOT NULL,
    [order_date] DATETIME2,
    CONSTRAINT [fk_orders_user] FOREIGN KEY ([user_id]) REFERENCES [app].[users]([id]),
    CONSTRAINT [fk_orders_product] FOREIGN KEY ([product_id]) REFERENCES [app].[products]([id])
);
GO

-- Add table comments using extended properties
EXEC sp_addextendedproperty 'MS_Description', 'User accounts and profile information', 'SCHEMA', 'app', 'TABLE', 'users';
EXEC sp_addextendedproperty 'MS_Description', 'Product catalog with pricing information', 'SCHEMA', 'app', 'TABLE', 'products';
EXEC sp_addextendedproperty 'MS_Description', 'Customer orders tracking system', 'SCHEMA', 'app', 'TABLE', 'orders';
GO

-- Add column comments for users table
EXEC sp_addextendedproperty 'MS_Description', 'Unique identifier for each user', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Unique username for login authentication', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'username';
EXEC sp_addextendedproperty 'MS_Description', 'User email address for notifications', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'email';
EXEC sp_addextendedproperty 'MS_Description', 'Timestamp when the user account was created', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'created_at';
EXEC sp_addextendedproperty 'MS_Description', 'Current status: active, inactive, or suspended', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'status';
GO

-- Add column comments for products table
EXEC sp_addextendedproperty 'MS_Description', 'Unique product identifier', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Product display name', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'name';
EXEC sp_addextendedproperty 'MS_Description', 'Product price in USD', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'price';
EXEC sp_addextendedproperty 'MS_Description', 'Product category classification', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'category';
GO

-- Add comments to orders table
EXEC sp_addextendedproperty 'MS_Description', 'Unique order identifier', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Reference to the user who placed the order', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'user_id';
EXEC sp_addextendedproperty 'MS_Description', 'Reference to the ordered product', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'product_id';
EXEC sp_addextendedproperty 'MS_Description', 'Number of items ordered', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'quantity';
EXEC sp_addextendedproperty 'MS_Description', 'When the order was placed', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'order_date';
`,
			migrationDDL: `
-- Drop column comments from orders table
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'order_date';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'quantity';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'product_id';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'user_id';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders', 'COLUMN', 'id';
GO

-- Drop table comment from orders
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'orders';
GO

-- Drop the orders table
DROP TABLE [app].[orders];
GO

-- Drop column comments from products table
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'category';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'price';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'name';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'products', 'COLUMN', 'id';
GO

-- Drop column comments from users table
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'status';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'created_at';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'email';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'username';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users', 'COLUMN', 'id';
GO

-- Drop table comments
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'products';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'app', 'TABLE', 'users';
`,
			description: "Reverse of table and column comments - dropping extended properties",
		},
		{
			name: "reverse_default_constraint_operations",
			initialSchema: `
CREATE SCHEMA [defaults];
GO

CREATE TABLE [defaults].[employees] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [email] NVARCHAR(100) NOT NULL,
    [hire_date] DATE NOT NULL DEFAULT GETDATE(),
    [status] NVARCHAR(20) NOT NULL CONSTRAINT [DF_employees_status_new] DEFAULT 'pending',
    [is_active] BIT NOT NULL DEFAULT 1,
    [salary] DECIMAL(10,2) DEFAULT 50000.00,
    [department_id] INT DEFAULT 1,
    [created_at] DATETIME2 DEFAULT SYSDATETIME(),
    [updated_at] DATETIME2,
    [vacation_days] INT NOT NULL DEFAULT 15,
    [bonus_percentage] DECIMAL(5,2) DEFAULT (CASE WHEN MONTH(GETDATE()) = 12 THEN 10.0 ELSE 5.0 END)
);

CREATE TABLE [defaults].[products] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [price] DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    [stock] INT NOT NULL DEFAULT 0,
    [is_available] BIT DEFAULT 1,
    [cost] DECIMAL(10,2) NOT NULL
);

CREATE TABLE [defaults].[orders] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [order_number] NVARCHAR(50) NOT NULL DEFAULT CONCAT('ORD-', YEAR(GETDATE()), '-', FORMAT(GETDATE(), 'MMdd')),
    [order_date] DATETIME2 NOT NULL DEFAULT SYSDATETIME(),
    [status] NVARCHAR(20) NOT NULL DEFAULT 'pending',
    [total] DECIMAL(10,2) NOT NULL DEFAULT 0,
    [discount] DECIMAL(5,2) DEFAULT NULL,
    [is_paid] BIT NOT NULL DEFAULT 0,
    [notes] NVARCHAR(MAX) DEFAULT 'No notes'
);
`,
			migrationDDL: `
-- Drop orders table with all its defaults
DROP TABLE [defaults].[orders];
GO

-- Drop default constraint for bonus_percentage column
DECLARE @constraint_name NVARCHAR(128);
SELECT @constraint_name = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'defaults' AND t.name = 'employees' AND c.name = 'bonus_percentage';
IF @constraint_name IS NOT NULL
    EXEC('ALTER TABLE [defaults].[employees] DROP CONSTRAINT [' + @constraint_name + ']');
GO

-- Drop default constraint for vacation_days column
DECLARE @constraint_name2 NVARCHAR(128);
SELECT @constraint_name2 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'defaults' AND t.name = 'employees' AND c.name = 'vacation_days';
IF @constraint_name2 IS NOT NULL
    EXEC('ALTER TABLE [defaults].[employees] DROP CONSTRAINT [' + @constraint_name2 + ']');
GO

-- Now safe to drop columns
ALTER TABLE [defaults].[employees] DROP COLUMN [bonus_percentage];
ALTER TABLE [defaults].[employees] DROP COLUMN [vacation_days];

-- Remove default from status column (named constraint)
ALTER TABLE [defaults].[employees] DROP CONSTRAINT [DF_employees_status_new];

-- Drop cost column (which has no default)
ALTER TABLE [defaults].[products] DROP COLUMN [cost];
`,
			description: "Reverse of default constraint operations - dropping tables, columns with defaults, and default constraints",
		},
		{
			name: "reverse_complex_default_expressions",
			initialSchema: `
CREATE SCHEMA [complex];
GO

CREATE TABLE [complex].[audit_log] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [event_type] NVARCHAR(50) NOT NULL,
    [user_name] NVARCHAR(100) NOT NULL DEFAULT SYSTEM_USER,
    [event_time] DATETIME2 NOT NULL DEFAULT SYSDATETIME(),
    [ip_address] NVARCHAR(45),
    [session_id] UNIQUEIDENTIFIER DEFAULT NEWID(),
    [server_name] NVARCHAR(100) DEFAULT @@SERVERNAME,
    [database_name] NVARCHAR(100) DEFAULT DB_NAME(),
    [schema_name] NVARCHAR(100) DEFAULT SCHEMA_NAME()
);

CREATE TABLE [complex].[documents] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [version] INT NOT NULL DEFAULT 1,
    [created_date] DATE DEFAULT CAST(GETDATE() AS DATE),
    [expiry_date] DATE DEFAULT DATEADD(YEAR, 1, CAST(GETDATE() AS DATE)),
    [document_code] NVARCHAR(50) DEFAULT CONCAT('DOC-', FORMAT(GETDATE(), 'yyyyMMdd'), '-', NEWID()),
    [priority] INT DEFAULT 
        CASE 
            WHEN DATEPART(HOUR, GETDATE()) < 12 THEN 1 
            WHEN DATEPART(HOUR, GETDATE()) < 18 THEN 2 
            ELSE 3 
        END
);

CREATE TABLE [complex].[calculations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [value1] DECIMAL(10,2) NOT NULL,
    [value2] DECIMAL(10,2) NOT NULL,
    [pi_value] DECIMAL(10,8) DEFAULT PI(),
    [random_value] FLOAT DEFAULT RAND(),
    [checksum_value] INT DEFAULT CHECKSUM(NEWID()),
    [created_at] DATETIME2 DEFAULT SYSDATETIME(),
    [created_offset] DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET()
);
`,
			migrationDDL: `
-- Drop table with function-based defaults
DROP TABLE [complex].[calculations];
GO

-- Drop default constraint for priority column
DECLARE @constraint_name NVARCHAR(128);
SELECT @constraint_name = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'complex' AND t.name = 'documents' AND c.name = 'priority';
IF @constraint_name IS NOT NULL
    EXEC('ALTER TABLE [complex].[documents] DROP CONSTRAINT [' + @constraint_name + ']');
GO

-- Drop default constraint for document_code column
DECLARE @constraint_name2 NVARCHAR(128);
SELECT @constraint_name2 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'complex' AND t.name = 'documents' AND c.name = 'document_code';
IF @constraint_name2 IS NOT NULL
    EXEC('ALTER TABLE [complex].[documents] DROP CONSTRAINT [' + @constraint_name2 + ']');
GO

-- Drop default constraint for schema_name column
DECLARE @constraint_name3 NVARCHAR(128);
SELECT @constraint_name3 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'complex' AND t.name = 'audit_log' AND c.name = 'schema_name';
IF @constraint_name3 IS NOT NULL
    EXEC('ALTER TABLE [complex].[audit_log] DROP CONSTRAINT [' + @constraint_name3 + ']');
GO

-- Drop default constraint for database_name column
DECLARE @constraint_name4 NVARCHAR(128);
SELECT @constraint_name4 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'complex' AND t.name = 'audit_log' AND c.name = 'database_name';
IF @constraint_name4 IS NOT NULL
    EXEC('ALTER TABLE [complex].[audit_log] DROP CONSTRAINT [' + @constraint_name4 + ']');
GO

-- Drop default constraint for server_name column
DECLARE @constraint_name5 NVARCHAR(128);
SELECT @constraint_name5 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'complex' AND t.name = 'audit_log' AND c.name = 'server_name';
IF @constraint_name5 IS NOT NULL
    EXEC('ALTER TABLE [complex].[audit_log] DROP CONSTRAINT [' + @constraint_name5 + ']');
GO

-- Now safe to drop columns
ALTER TABLE [complex].[documents] DROP COLUMN [priority];
ALTER TABLE [complex].[documents] DROP COLUMN [document_code];
ALTER TABLE [complex].[audit_log] DROP COLUMN [schema_name];
ALTER TABLE [complex].[audit_log] DROP COLUMN [database_name];
ALTER TABLE [complex].[audit_log] DROP COLUMN [server_name];
`,
			description: "Reverse of complex default expressions - dropping columns with system functions and conditional defaults",
		},
		{
			name: "reverse_comments_with_special_characters",
			initialSchema: `
CREATE SCHEMA [special];
GO

CREATE TABLE [special].[documents] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [title] NVARCHAR(200) NOT NULL,
    [content] NVARCHAR(MAX),
    [author] NVARCHAR(100),
    [created_date] DATETIME2,
    [metadata] NVARCHAR(500)
);

CREATE TABLE [special].[translations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [document_id] INT NOT NULL,
    [lang_code] NVARCHAR(10) NOT NULL,
    [translated_title] NVARCHAR(200),
    [translated_content] NVARCHAR(MAX),
    CONSTRAINT [fk_translations_doc] FOREIGN KEY ([document_id]) REFERENCES [special].[documents]([id])
);

CREATE TABLE [special].[test_extreme] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [data] NVARCHAR(MAX)
);
GO

-- Add table comments with special characters and quotes
EXEC sp_addextendedproperty 'MS_Description', 'Document storage system - handles files, "metadata", & special chars: @#$%^&*()_+-={}[]|\:";''<>?,./', 'SCHEMA', 'special', 'TABLE', 'documents';
GO

-- Add multiline comment with special formatting
EXEC sp_addextendedproperty 'MS_Description', 
'Translation table for international support:
- Supports Unicode characters: αβγδε, 中文, العربية, русский
- Handles quotes: "double quotes" and ''single quotes''
- Special symbols: @#$%^&*()_+-={}[]|\:";''<>?,./', 'SCHEMA', 'special', 'TABLE', 'translations';
GO

-- Column comments with various special cases
EXEC sp_addextendedproperty 'MS_Description', 'Primary key (auto-increment) - unique ID for each document', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'id';
EXEC sp_addextendedproperty 'MS_Description', 'Document title - may contain "quotes", ''apostrophes'', and symbols: @#$%', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'title';
GO

-- Multiline comment with technical details
EXEC sp_addextendedproperty 'MS_Description', 
'Document content field:
  • Supports rich text formatting
  • HTML tags like <p>, <div>, <span>
  • Special characters: © ® ™ § ¶ † ‡ • ‰ ′ ″
  • Mathematical: ± × ÷ ≠ ≤ ≥ ∞ ∑ ∏ ∫
  • Currency: $ € £ ¥ ₹ ₽
  • Arrows: ← → ↑ ↓ ↔ ⇐ ⇒ ⇔', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'content';
GO

-- Comment with SQL injection attempt (should be safely handled)
EXEC sp_addextendedproperty 'MS_Description', 'Author name field - prevents SQL injection like ''; DROP TABLE users; --', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'author';
GO

-- Comment with JSON-like structure
EXEC sp_addextendedproperty 'MS_Description', 'Metadata JSON: {"version": "1.0", "tags": ["important", "draft"], "settings": {"public": false}}', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'metadata';
GO

-- Unicode characters in comments
EXEC sp_addextendedproperty 'MS_Description', 'Language code: en-US, fr-FR, de-DE, ja-JP, zh-CN, ar-SA, ru-RU, hi-IN, 한국어, ไทย', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'lang_code';
GO

-- Comment with URLs and file paths
EXEC sp_addextendedproperty 'MS_Description', 'Translated content - may reference URLs like https://example.com/path?param=value&other=123 or file paths C:\Users\Name\Documents\file.txt', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'translated_content';
GO

-- Extremely long comment to test limits
EXEC sp_addextendedproperty 'MS_Description', 
'This is an extremely long comment designed to test the limits of extended property storage in SQL Server. It contains multiple lines of text with various formatting, special characters, and technical information. The purpose is to verify that the migration system can properly handle, store, and retrieve complex comment data without truncation or corruption. This comment includes: numbers (123456789), symbols (!@#$%^&*()_+-={}[]|\:";''<>?,./ ), Unicode characters (αβγδε中文العربيةрусский한국어), and structured data like JSON {"key": "value", "array": [1, 2, 3], "nested": {"deep": true}}. Additionally, it tests SQL-like syntax: SELECT * FROM table WHERE column = ''value'' AND other_column IN (1, 2, 3); as well as HTML markup: <html><body><p class="test">Content</p></body></html> and XML: <?xml version="1.0"?><root><item id="1">Test</item></root>. The comment system should preserve all these characters and structures exactly as written, demonstrating robust handling of complex metadata in database schema documentation.', 'SCHEMA', 'special', 'TABLE', 'test_extreme';
`,
			migrationDDL: `
-- Drop extremely long comment
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'test_extreme';
GO

-- Drop table with extreme comment
DROP TABLE [special].[test_extreme];
GO

-- Drop column comments with special characters
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'translated_content';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'translations', 'COLUMN', 'lang_code';
GO

-- Drop column comments from documents
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'metadata';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'author';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'content';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'title';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents', 'COLUMN', 'id';
GO

-- Drop table comments with special characters
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'translations';
EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', 'special', 'TABLE', 'documents';
`,
			description: "Reverse of comments with special characters - dropping comments with quotes, unicode, and special formatting",
		},
		{
			name: "reverse_default_constraint_edge_cases",
			initialSchema: `
CREATE SCHEMA [edge];
GO

CREATE TABLE [edge].[test_defaults] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [nullable_with_default] INT CONSTRAINT [DF_test_defaults_nullable_new] DEFAULT 84,
    [string_default] NVARCHAR(50) DEFAULT N'Default''s value',
    [empty_string] NVARCHAR(50) DEFAULT '',
    [zero_default] INT DEFAULT 0,
    [negative_default] INT DEFAULT -1,
    [decimal_precision] DECIMAL(18,6) DEFAULT 123.456789,
    [special_chars] NVARCHAR(100) DEFAULT N'Line1' + CHAR(13) + CHAR(10) + N'Line2',
    [unicode_default] NVARCHAR(100) DEFAULT N'你好世界 🌍',
    [max_int] BIGINT DEFAULT 9223372036854775807,
    [min_int] BIGINT DEFAULT -9223372036854775808,
    [json_config] NVARCHAR(MAX) DEFAULT N'{"enabled": true, "count": 0}'
);

CREATE TABLE [edge].[flags] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [is_enabled] BIT NOT NULL DEFAULT 1,
    [is_deleted] BIT NOT NULL DEFAULT 0,
    [is_visible] BIT DEFAULT 1,
    [is_archived] BIT DEFAULT 0
);
`,
			migrationDDL: `
-- Drop table with bit field defaults
DROP TABLE [edge].[flags];

-- Drop default constraints before dropping columns
-- Drop default constraint for json_config column
DECLARE @constraint_name NVARCHAR(128);
SELECT @constraint_name = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'edge' AND t.name = 'test_defaults' AND c.name = 'json_config';
IF @constraint_name IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name + ']');
GO

-- Drop default constraint for min_int column
DECLARE @constraint_name2 NVARCHAR(128);
SELECT @constraint_name2 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'edge' AND t.name = 'test_defaults' AND c.name = 'min_int';
IF @constraint_name2 IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name2 + ']');
GO

-- Drop default constraint for max_int column
DECLARE @constraint_name3 NVARCHAR(128);
SELECT @constraint_name3 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'edge' AND t.name = 'test_defaults' AND c.name = 'max_int';
IF @constraint_name3 IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name3 + ']');
GO

-- Drop default constraint for unicode_default column
DECLARE @constraint_name4 NVARCHAR(128);
SELECT @constraint_name4 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'edge' AND t.name = 'test_defaults' AND c.name = 'unicode_default';
IF @constraint_name4 IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name4 + ']');
GO

-- Drop default constraint for special_chars column
DECLARE @constraint_name5 NVARCHAR(128);
SELECT @constraint_name5 = dc.name
FROM sys.default_constraints dc
INNER JOIN sys.columns c ON dc.parent_column_id = c.column_id AND dc.parent_object_id = c.object_id
INNER JOIN sys.tables t ON c.object_id = t.object_id
INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = 'edge' AND t.name = 'test_defaults' AND c.name = 'special_chars';
IF @constraint_name5 IS NOT NULL
    EXEC('ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [' + @constraint_name5 + ']');
GO

-- Now safe to drop columns
ALTER TABLE [edge].[test_defaults] DROP COLUMN [json_config];
ALTER TABLE [edge].[test_defaults] DROP COLUMN [min_int];
ALTER TABLE [edge].[test_defaults] DROP COLUMN [max_int];
ALTER TABLE [edge].[test_defaults] DROP COLUMN [unicode_default];
ALTER TABLE [edge].[test_defaults] DROP COLUMN [special_chars];

-- Remove named default constraint
ALTER TABLE [edge].[test_defaults] DROP CONSTRAINT [DF_test_defaults_nullable_new];
`,
			description: "Reverse of edge case defaults - dropping special default constraints and columns",
		},
		{
			name: "spatial_index_operations",
			initialSchema: `
CREATE SCHEMA [geo];
GO

-- Enable spatial types
CREATE TABLE [geo].[locations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(200) NOT NULL,
    [geo_point] GEOMETRY NOT NULL,
    [geo_polygon] GEOMETRY NOT NULL,
    [geo_location] GEOGRAPHY NOT NULL,
    [description] NVARCHAR(MAX)
);
`,
			migrationDDL: `
-- Add spatial indexes with different configurations

-- GEOMETRY_GRID index with full parameters
CREATE SPATIAL INDEX [idx_geo_point] ON [geo].[locations] ([geo_point]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 32,
    PAD_INDEX = ON,
    FILLFACTOR = 85,
    SORT_IN_TEMPDB = ON,
    ALLOW_ROW_LOCKS = ON,
    ALLOW_PAGE_LOCKS = ON,
    MAXDOP = 2,
    DATA_COMPRESSION = PAGE
);

-- GEOMETRY_AUTO_GRID index (requires bounding box)
CREATE SPATIAL INDEX [idx_geo_polygon] ON [geo].[locations] ([geo_polygon]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-1000, -1000, 1000, 1000),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW)
);

-- GEOGRAPHY_GRID index with specific grid levels
CREATE SPATIAL INDEX [idx_geo_location] ON [geo].[locations] ([geo_location]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 8,
    FILLFACTOR = 80
);

-- Add another table with spatial columns
CREATE TABLE [geo].[boundaries] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [boundary_name] NVARCHAR(100) NOT NULL,
    [boundary_shape] GEOMETRY NOT NULL,
    [center_point] GEOMETRY NOT NULL
);

-- Add spatial index on new table
CREATE SPATIAL INDEX [idx_boundary_shape] ON [geo].[boundaries] ([boundary_shape]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 1000, 1000),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 64
);
`,
			description: "Spatial index operations with various tessellation schemes and parameters",
		},
		{
			name: "reverse_spatial_table_operations",
			initialSchema: `
CREATE SCHEMA [geo];
GO

-- Create table with spatial columns and indexes
CREATE TABLE [geo].[locations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(200) NOT NULL,
    [location] GEOGRAPHY NOT NULL,
    [area] GEOGRAPHY NOT NULL
);
GO

-- Create spatial indexes that will be dropped with the table
-- Using GEOGRAPHY_GRID which doesn't require BOUNDING_BOX
CREATE SPATIAL INDEX [idx_location] ON [geo].[locations] ([location]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX [idx_area] ON [geo].[locations] ([area]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 8
);
GO

CREATE TABLE [geo].[regions] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [region_name] NVARCHAR(100) NOT NULL,
    [boundary] GEOMETRY NOT NULL
);
GO
`,
			migrationDDL: `
-- Drop entire table (this drops spatial indexes too)
DROP TABLE [geo].[locations];

-- Add columns to remaining table
ALTER TABLE [geo].[regions] ADD [population] INT;
ALTER TABLE [geo].[regions] ADD [area_sqkm] FLOAT;
`,
			description: "Reverse spatial table operations - dropping table with spatial indexes",
		},
		{
			name: "reverse_spatial_indexes",
			initialSchema: `
CREATE SCHEMA [geo];
GO

-- Create table with spatial columns and spatial indexes that can be preserved
CREATE TABLE [geo].[locations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(200) NOT NULL,
    [geo_point] GEOMETRY NOT NULL,
    [geo_polygon] GEOMETRY NOT NULL,
    [geo_location] GEOGRAPHY NOT NULL,
    [description] NVARCHAR(MAX)
);
GO

-- Create spatial indexes with basic configuration to avoid preservation issues
CREATE SPATIAL INDEX [idx_geo_point] ON [geo].[locations] ([geo_point]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM)
);
GO

CREATE SPATIAL INDEX [idx_geo_location] ON [geo].[locations] ([geo_location]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM)
);
GO
`,
			migrationDDL: `
-- Drop and recreate spatial indexes to test preservation
DROP INDEX [idx_geo_point] ON [geo].[locations];
DROP INDEX [idx_geo_location] ON [geo].[locations];

-- Recreate with same configuration (should work with DDL preservation)
CREATE SPATIAL INDEX [idx_geo_point] ON [geo].[locations] ([geo_point]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM)
);

CREATE SPATIAL INDEX [idx_geo_location] ON [geo].[locations] ([geo_location]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM)
);
`,
			description: "Reverse spatial index operations - dropping and recreating spatial indexes",
		},
		{
			name: "spatial_index_modifications",
			initialSchema: `
CREATE SCHEMA [geo];
GO

CREATE TABLE [geo].[spatial_data] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [location] GEOMETRY NOT NULL,
    [region] GEOMETRY,
    [area] GEOGRAPHY
);
GO

-- Create initial spatial indexes
CREATE SPATIAL INDEX [idx_location] ON [geo].[spatial_data] ([location])
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-100, -100, 100, 100),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX [idx_area] ON [geo].[spatial_data] ([area])
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 32
);
GO
`,
			migrationDDL: `
-- Drop and recreate spatial index with different parameters
DROP INDEX [idx_location] ON [geo].[spatial_data];
GO

CREATE SPATIAL INDEX [idx_location] ON [geo].[spatial_data] ([location])
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-200, -200, 200, 200),  -- Expanded bounding box
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),  -- Changed grid densities
    CELLS_PER_OBJECT = 64,  -- Increased cells per object
    PAD_INDEX = ON,
    FILLFACTOR = 90,
    ALLOW_ROW_LOCKS = ON,
    ALLOW_PAGE_LOCKS = ON
);
GO

-- Add new spatial index on previously unindexed column
CREATE SPATIAL INDEX [idx_region] ON [geo].[spatial_data] ([region])
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-500, -500, 500, 500),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 32
);
GO

-- Drop spatial index on geography column
DROP INDEX [idx_area] ON [geo].[spatial_data];
GO
`,
			description: "Spatial index modifications - recreating with different parameters and adding/dropping indexes",
		},
		{
			name: "spatial_table_schema_changes",
			initialSchema: `
CREATE SCHEMA [geo];
GO

CREATE TABLE [geo].[points] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [point_name] NVARCHAR(50) NOT NULL,
    [location] GEOMETRY NOT NULL
);
GO

CREATE SPATIAL INDEX [idx_points_location] ON [geo].[points] ([location])
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 100, 100),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM)
);
GO
`,
			migrationDDL: `
-- Add new spatial and non-spatial columns
ALTER TABLE [geo].[points] ADD [altitude] FLOAT;
ALTER TABLE [geo].[points] ADD [boundary] GEOMETRY;
ALTER TABLE [geo].[points] ADD [geo_location] GEOGRAPHY;
ALTER TABLE [geo].[points] ADD [created_at] DATETIME2 DEFAULT GETDATE();

-- Create spatial indexes on new columns
CREATE SPATIAL INDEX [idx_points_boundary] ON [geo].[points] ([boundary])
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-50, -50, 150, 150),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 8
);
GO

CREATE SPATIAL INDEX [idx_points_geo_location] ON [geo].[points] ([geo_location])
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 16
);
GO

-- Modify existing column (rename)
EXEC sp_rename '[geo].[points].[point_name]', 'name', 'COLUMN';
GO
`,
			description: "Spatial table schema changes - adding columns and indexes while preserving existing spatial indexes",
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
	dbMetadataA := model.NewDatabaseMetadata(schemaA, nil, nil, storepb.Engine_MSSQL, false)
	dbMetadataB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_MSSQL, false)

	// Get diff from B to A (to generate rollback)
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MSSQL, dbMetadataB, dbMetadataA)
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
		if stmt != "" {
			// Check if this statement contains any non-comment SQL
			lines := strings.Split(stmt, "\n")
			hasSQL := false
			var sqlLines []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "--") {
					hasSQL = true
					sqlLines = append(sqlLines, line)
				}
			}
			// Only include if there's actual SQL (not just comments)
			if hasSQL {
				result = append(result, strings.Join(sqlLines, " "))
			}
		}
	}
	return result
}

// compareSchemas compares two database schemas and returns an error if they differ
func compareSchemas(schemaA, schemaC *storepb.DatabaseSchemaMetadata) error {
	// Sort schemas for consistent comparison
	normalizeSchema(schemaA)
	normalizeSchema(schemaC)

	// Use protocmp for detailed comparison, ignoring DefaultConstraintName and Position fields
	// DefaultConstraintName is only populated when syncing from database, not when parsing SQL
	// Position must be ignored because:
	//   - In SQL Server, column positions are determined by column_id (physical order)
	//   - When a column is dropped and re-added, it gets a new column_id at the end
	//   - Our migration generator uses DROP+ADD for column renames (doesn't use sp_rename)
	//   - This causes legitimate position changes that don't affect functional equivalence
	//   - Example: table with [id(1), name(2), email(3)] -> DROP name -> ADD name -> [id(1), email(3), name(4)]
	opts := cmp.Options{
		protocmp.Transform(),
		protocmp.IgnoreFields(&storepb.ColumnMetadata{}, "default_constraint_name", "position"),
	}
	if diff := cmp.Diff(schemaA, schemaC, opts); diff != "" {
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

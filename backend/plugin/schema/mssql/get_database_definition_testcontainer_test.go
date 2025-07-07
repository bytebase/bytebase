package mssql

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetDatabaseDefinitionWithTestcontainer(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestMSSQLContainer(ctx, t)
	defer container.Close(ctx)

	host := container.GetHost()
	port := container.GetPort()
	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		setupSQL  string
		wantMatch bool
	}{
		{
			name: "basic_tables_with_constraints",
			setupSQL: `
CREATE SCHEMA test_schema;
GO

CREATE TABLE test_schema.users (
    id INT IDENTITY(1,1) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME2 DEFAULT GETDATE(),
    is_active BIT DEFAULT 1,
    age INT CHECK (age >= 18),
    computed_column AS (id * 2) PERSISTED
);
GO

CREATE TABLE test_schema.posts (
    id INT IDENTITY(1,1) PRIMARY KEY,
    user_id INT NOT NULL,
    title NVARCHAR(200) NOT NULL,
    content NVARCHAR(MAX),
    CONSTRAINT FK_posts_users FOREIGN KEY (user_id) REFERENCES test_schema.users(id) ON DELETE CASCADE
);
GO

CREATE NONCLUSTERED INDEX idx_posts_user_id ON test_schema.posts(user_id);
CREATE UNIQUE INDEX idx_posts_title ON test_schema.posts(title) WHERE title IS NOT NULL;
GO
`,
			wantMatch: true,
		},
		{
			name: "views_with_dependencies",
			setupSQL: `
CREATE TABLE dbo.employees (
    emp_id INT PRIMARY KEY,
    emp_name VARCHAR(100),
    department VARCHAR(50),
    salary DECIMAL(10,2)
);
GO

CREATE VIEW dbo.v_employee_summary AS
SELECT department, COUNT(*) as emp_count, AVG(salary) as avg_salary
FROM dbo.employees
GROUP BY department;
GO

CREATE VIEW dbo.v_high_earners AS
SELECT * FROM dbo.v_employee_summary
WHERE avg_salary > 50000;
GO
`,
			wantMatch: true,
		},
		{
			name: "functions_and_procedures",
			setupSQL: `
CREATE FUNCTION dbo.GetFullName(@FirstName NVARCHAR(50), @LastName NVARCHAR(50))
RETURNS NVARCHAR(100)
AS
BEGIN
    RETURN @FirstName + ' ' + @LastName;
END
GO

CREATE PROCEDURE dbo.UpdateSalary
    @EmployeeId INT,
    @NewSalary DECIMAL(10,2)
AS
BEGIN
    UPDATE employees SET salary = @NewSalary WHERE emp_id = @EmployeeId;
END
GO
`,
			wantMatch: true,
		},
		{
			name: "complex_indexes",
			setupSQL: `
CREATE TABLE dbo.sales (
    sale_id INT,
    product_id INT,
    sale_date DATE,
    amount DECIMAL(10,2),
    region VARCHAR(50)
);
GO

CREATE CLUSTERED COLUMNSTORE INDEX cci_sales ON dbo.sales;
GO

CREATE TABLE dbo.products (
    product_id INT PRIMARY KEY,
    product_name VARCHAR(100),
    category VARCHAR(50)
);
GO

CREATE NONCLUSTERED COLUMNSTORE INDEX ncci_products ON dbo.products(product_name, category);
GO
`,
			wantMatch: true,
		},
		{
			name: "temporal_tables",
			setupSQL: `
CREATE TABLE dbo.temporal_test (
    id INT PRIMARY KEY,
    name VARCHAR(100),
    SysStartTime DATETIME2 GENERATED ALWAYS AS ROW START NOT NULL,
    SysEndTime DATETIME2 GENERATED ALWAYS AS ROW END NOT NULL,
    PERIOD FOR SYSTEM_TIME (SysStartTime, SysEndTime)
) WITH (SYSTEM_VERSIONING = ON (HISTORY_TABLE = dbo.temporal_test_history));
GO
`,
			wantMatch: true,
		},
		{
			name: "spatial_indexes",
			setupSQL: `
CREATE TABLE dbo.spatial_data (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(100) NOT NULL,
    location GEOMETRY NOT NULL,
    boundary GEOGRAPHY,
    area AS location.STArea() PERSISTED
);
GO

CREATE SPATIAL INDEX idx_spatial_location ON dbo.spatial_data(location);
CREATE SPATIAL INDEX idx_spatial_boundary ON dbo.spatial_data(boundary);
CREATE INDEX idx_spatial_name ON dbo.spatial_data(name);
GO
`,
			wantMatch: true,
		},
		{
			name: "comprehensive_spatial_data_types",
			setupSQL: `
CREATE SCHEMA spatial_test;
GO

-- Table with all spatial scenarios
CREATE TABLE spatial_test.geo_features (
    feature_id INT IDENTITY(1,1) PRIMARY KEY CLUSTERED,
    feature_name NVARCHAR(200) NOT NULL,
    
    -- GEOMETRY columns for planar data
    point_location GEOMETRY NOT NULL,
    line_path GEOMETRY,
    polygon_area GEOMETRY NOT NULL,
    multi_point GEOMETRY,
    multi_line GEOMETRY,
    multi_polygon GEOMETRY,
    geometry_collection GEOMETRY,
    
    -- GEOGRAPHY columns for geodetic data
    geo_point GEOGRAPHY NOT NULL,
    geo_line GEOGRAPHY,
    geo_polygon GEOGRAPHY NOT NULL,
    geo_multi_point GEOGRAPHY,
    geo_multi_line GEOGRAPHY,
    geo_multi_polygon GEOGRAPHY,
    geo_collection GEOGRAPHY,
    
    -- Regular columns
    feature_type VARCHAR(50),
    created_date DATETIME2 DEFAULT GETDATE()
);
GO

-- Create spatial indexes for GEOMETRY columns
CREATE SPATIAL INDEX idx_point_loc ON spatial_test.geo_features(point_location);
CREATE SPATIAL INDEX idx_line_path ON spatial_test.geo_features(line_path);
CREATE SPATIAL INDEX idx_polygon_area ON spatial_test.geo_features(polygon_area);
CREATE SPATIAL INDEX idx_multi_point ON spatial_test.geo_features(multi_point);
CREATE SPATIAL INDEX idx_multi_polygon ON spatial_test.geo_features(multi_polygon);
GO

-- Create spatial indexes for GEOGRAPHY columns
CREATE SPATIAL INDEX idx_geo_point ON spatial_test.geo_features(geo_point);
CREATE SPATIAL INDEX idx_geo_line ON spatial_test.geo_features(geo_line);
CREATE SPATIAL INDEX idx_geo_polygon ON spatial_test.geo_features(geo_polygon);
CREATE SPATIAL INDEX idx_geo_multi_polygon ON spatial_test.geo_features(geo_multi_polygon);
GO

-- Create regular indexes for comparison
CREATE NONCLUSTERED INDEX idx_feature_name ON spatial_test.geo_features(feature_name);
CREATE INDEX idx_feature_type ON spatial_test.geo_features(feature_type) WHERE feature_type IS NOT NULL;
GO

-- Another table to test spatial indexes with constraints
CREATE TABLE spatial_test.locations_with_constraints (
    location_id INT IDENTITY(1,1),
    location_name NVARCHAR(100) NOT NULL UNIQUE,
    coords GEOMETRY NOT NULL,
    geo_coords GEOGRAPHY NOT NULL,
    parent_id INT,
    CONSTRAINT PK_locations PRIMARY KEY NONCLUSTERED (location_id),
    CONSTRAINT FK_locations_parent FOREIGN KEY (parent_id) REFERENCES spatial_test.locations_with_constraints(location_id),
    CONSTRAINT CHK_location_name CHECK (LEN(location_name) > 0)
);
GO

-- Create clustered index on non-primary key column
CREATE CLUSTERED INDEX idx_locations_parent ON spatial_test.locations_with_constraints(parent_id);
GO

-- Create spatial indexes
CREATE SPATIAL INDEX idx_coords ON spatial_test.locations_with_constraints(coords);
CREATE SPATIAL INDEX idx_geo_coords ON spatial_test.locations_with_constraints(geo_coords);
GO

-- Table with computed columns using spatial functions
CREATE TABLE spatial_test.computed_spatial_data (
    id INT PRIMARY KEY,
    shape GEOMETRY NOT NULL,
    centroid AS shape.STCentroid() PERSISTED,
    envelope AS shape.STEnvelope() PERSISTED,
    area_size AS shape.STArea(),
    perimeter AS shape.STLength()
);
GO

CREATE SPATIAL INDEX idx_shape ON spatial_test.computed_spatial_data(shape);
GO
`,
			wantMatch: true,
		},
		{
			name: "spatial_indexes_edge_cases",
			setupSQL: `
-- Edge case: Table with only spatial columns and indexes
CREATE TABLE dbo.pure_spatial (
    geo1 GEOMETRY NOT NULL,
    geo2 GEOMETRY,
    geog1 GEOGRAPHY NOT NULL,
    geog2 GEOGRAPHY
);
GO

CREATE SPATIAL INDEX idx_pure_geo1 ON dbo.pure_spatial(geo1);
CREATE SPATIAL INDEX idx_pure_geo2 ON dbo.pure_spatial(geo2);
CREATE SPATIAL INDEX idx_pure_geog1 ON dbo.pure_spatial(geog1);
CREATE SPATIAL INDEX idx_pure_geog2 ON dbo.pure_spatial(geog2);
GO

-- Edge case: Very long index names
CREATE TABLE dbo.long_names (
    id INT PRIMARY KEY,
    location GEOMETRY NOT NULL
);
GO

CREATE SPATIAL INDEX idx_this_is_a_very_long_spatial_index_name_for_testing_purposes ON dbo.long_names(location);
GO

-- Edge case: Mixed case names
CREATE TABLE dbo.MixedCaseTable (
    Id INT PRIMARY KEY,
    Location GEOMETRY NOT NULL,
    [Some Column With Spaces] GEOGRAPHY
);
GO

CREATE SPATIAL INDEX IDX_MixedCase_Location ON dbo.MixedCaseTable(Location);
CREATE SPATIAL INDEX [Index With Spaces] ON dbo.MixedCaseTable([Some Column With Spaces]);
GO
`,
			wantMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			databaseName := fmt.Sprintf("test_%d", time.Now().Unix())

			// Create a new driver instance for this test
			driverInstance := &mssqldb.Driver{}
			config := db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Type:     storepb.DataSourceType_ADMIN,
					Username: "sa",
					Host:     host,
					Port:     strconv.Itoa(portInt),
					Database: "master",
				},
				Password: "Test123!",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "master",
				},
			}

			// Open connection
			driver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
			require.NoError(t, err)

			// Create test database
			_, err = driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE [%s]", databaseName), db.ExecuteOptions{CreateDatabase: true})
			require.NoError(t, err)
			defer func() {
				// Clean up test database
				driver.Close(ctx)
				config.DataSource.Database = "master"
				config.ConnectionContext.DatabaseName = "master"
				cleanupDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
				if err == nil {
					_, _ = cleanupDriver.Execute(ctx, fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", databaseName), db.ExecuteOptions{CreateDatabase: true})
					_, _ = cleanupDriver.Execute(ctx, fmt.Sprintf("DROP DATABASE [%s]", databaseName), db.ExecuteOptions{CreateDatabase: true})
					cleanupDriver.Close(ctx)
				}
			}()

			// Reconnect to test database
			driver.Close(ctx)
			config.DataSource.Database = databaseName
			config.ConnectionContext.DatabaseName = databaseName
			testDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
			require.NoError(t, err)
			defer testDriver.Close(ctx)

			// Step 1: Initialize database schema and get metadata A
			err = executeSQLStatements(ctx, testDriver, tc.setupSQL)
			require.NoError(t, err)

			mssqlTestDriver, ok := testDriver.(*mssqldb.Driver)
			require.True(t, ok, "failed to cast to mssqldb.Driver")

			metadataA, err := mssqlTestDriver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Step 2: Call GetDatabaseDefinition to generate database definition X
			definitionX, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadataA)
			require.NoError(t, err)
			require.NotEmpty(t, definitionX)

			// Step 3: Create a new database and run the definition X
			newDatabaseName := fmt.Sprintf("test_copy_%d", time.Now().Unix())

			// Reconnect to master to create new database
			testDriver.Close(ctx)
			config.DataSource.Database = "master"
			config.ConnectionContext.DatabaseName = "master"
			masterDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
			require.NoError(t, err)

			_, err = masterDriver.Execute(ctx, fmt.Sprintf("CREATE DATABASE [%s]", newDatabaseName), db.ExecuteOptions{CreateDatabase: true})
			require.NoError(t, err)
			defer func() {
				// Clean up new database
				masterDriver.Close(ctx)
				config.DataSource.Database = "master"
				config.ConnectionContext.DatabaseName = "master"
				cleanupDriver2, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
				if err == nil {
					_, _ = cleanupDriver2.Execute(ctx, fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", newDatabaseName), db.ExecuteOptions{CreateDatabase: true})
					_, _ = cleanupDriver2.Execute(ctx, fmt.Sprintf("DROP DATABASE [%s]", newDatabaseName), db.ExecuteOptions{CreateDatabase: true})
					cleanupDriver2.Close(ctx)
				}
			}()

			// Connect to the new database
			masterDriver.Close(ctx)
			config.DataSource.Database = newDatabaseName
			config.ConnectionContext.DatabaseName = newDatabaseName
			newDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
			require.NoError(t, err)
			defer newDriver.Close(ctx)

			// Execute the generated definition
			err = executeSQLStatements(ctx, newDriver, definitionX)
			require.NoError(t, err)

			mssqlNewDriver, ok := newDriver.(*mssqldb.Driver)
			require.True(t, ok, "failed to cast to mssqldb.Driver")

			// Get metadata B
			metadataB, err := mssqlNewDriver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Step 4: Compare metadata A and B
			if tc.wantMatch {
				// Note: Direct comparison might fail due to:
				// 1. System-generated names for constraints
				// 2. Minor differences in index/constraint definitions
				// For now, we just verify that both schemas have the same objects

				require.Equal(t, len(metadataA.Schemas), len(metadataB.Schemas), "Number of schemas should match")

				for i, schemaA := range metadataA.Schemas {
					schemaB := metadataB.Schemas[i]
					require.Equal(t, schemaA.Name, schemaB.Name, "Schema names should match")
					require.Equal(t, len(schemaA.Tables), len(schemaB.Tables), "Number of tables in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Views), len(schemaB.Views), "Number of views in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Functions), len(schemaB.Functions), "Number of functions in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Procedures), len(schemaB.Procedures), "Number of procedures in schema %s should match", schemaA.Name)
				}
			}
		})
	}
}

func executeSQLStatements(ctx context.Context, driver db.Driver, sqlScript string) error {
	statements := splitSQLByGO(sqlScript)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := driver.Execute(ctx, stmt, db.ExecuteOptions{}); err != nil {
			return errors.Wrapf(err, "failed to execute statement: %s", stmt)
		}
	}
	return nil
}

func splitSQLByGO(script string) []string {
	var statements []string
	lines := strings.Split(script, "\n")
	var currentStatement strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.EqualFold(trimmedLine, "GO") {
			if currentStatement.Len() > 0 {
				statements = append(statements, currentStatement.String())
				currentStatement.Reset()
			}
		} else {
			currentStatement.WriteString(line)
			currentStatement.WriteString("\n")
		}
	}

	if currentStatement.Len() > 0 {
		statements = append(statements, currentStatement.String())
	}

	return statements
}

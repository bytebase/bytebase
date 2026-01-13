package mssql

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

//nolint:tparallel
func TestGetDatabaseDefinitionWithTestcontainer(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestMSSQLContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

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
			name: "spatial_indexes_geography",
			setupSQL: `
CREATE SCHEMA spatial_schema;
GO

CREATE TABLE spatial_schema.locations (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(200) NOT NULL,
    location_point GEOGRAPHY NOT NULL,
    boundary_polygon GEOGRAPHY NOT NULL,
    route_line GEOGRAPHY,
    description NVARCHAR(MAX),
    created_at DATETIME2 DEFAULT GETDATE()
);
GO

-- Create spatial indexes with various configurations
CREATE SPATIAL INDEX idx_location_point ON spatial_schema.locations(location_point) 
USING GEOGRAPHY_GRID 
WITH (
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
GO

CREATE SPATIAL INDEX idx_boundary_polygon ON spatial_schema.locations(boundary_polygon) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16,
    FILLFACTOR = 80
);
GO

CREATE SPATIAL INDEX idx_route_line ON spatial_schema.locations(route_line) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 64
);
GO

-- Create regular indexes on the same table
CREATE INDEX idx_name ON spatial_schema.locations(name);
CREATE INDEX idx_created_at ON spatial_schema.locations(created_at);
GO
`,
			wantMatch: true,
		},
		{
			name: "spatial_indexes_geometry",
			setupSQL: `
CREATE SCHEMA geom_schema;
GO

CREATE TABLE geom_schema.shapes (
    shape_id INT IDENTITY(1,1) PRIMARY KEY,
    shape_name NVARCHAR(100) NOT NULL,
    shape_point GEOMETRY NOT NULL,
    shape_polygon GEOMETRY NOT NULL,
    shape_line GEOMETRY,
    area_size AS shape_polygon.STArea() PERSISTED,
    center_x AS shape_point.STX PERSISTED,
    center_y AS shape_point.STY PERSISTED
);
GO

-- Create GEOMETRY spatial indexes with bounding boxes
CREATE SPATIAL INDEX idx_shape_point ON geom_schema.shapes(shape_point) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 100, 100),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 16,
    PAD_INDEX = ON,
    FILLFACTOR = 90
);
GO

CREATE SPATIAL INDEX idx_shape_polygon ON geom_schema.shapes(shape_polygon) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 64
);
GO

CREATE SPATIAL INDEX idx_shape_line ON geom_schema.shapes(shape_line) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-1000, -1000, 1000, 1000),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = MEDIUM, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8
);
GO

-- Create unique index on shape_name
CREATE UNIQUE INDEX idx_shape_name ON geom_schema.shapes(shape_name);
GO
`,
			wantMatch: true,
		},
		{
			name: "spatial_indexes_mixed_with_constraints",
			setupSQL: `
CREATE TABLE dbo.spatial_mixed (
    id INT IDENTITY(1,1) PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    geo_location GEOGRAPHY NOT NULL,
    geom_shape GEOMETRY NOT NULL,
    status NVARCHAR(20) CHECK (status IN ('active', 'inactive', 'pending')),
    parent_id INT,
    CONSTRAINT FK_spatial_mixed_parent FOREIGN KEY (parent_id) REFERENCES dbo.spatial_mixed(id)
);
GO

-- Create both GEOGRAPHY and GEOMETRY spatial indexes
CREATE SPATIAL INDEX idx_geo_location_mixed ON dbo.spatial_mixed(geo_location) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX idx_geom_shape_mixed ON dbo.spatial_mixed(geom_shape) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-100, -100, 100, 100),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = LOW, LEVEL_3 = HIGH, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 32
);
GO

-- Create regular indexes
CREATE INDEX idx_status ON dbo.spatial_mixed(status);
CREATE INDEX idx_parent_id ON dbo.spatial_mixed(parent_id);
GO

-- Create a view using spatial functions
CREATE VIEW dbo.v_spatial_summary AS
SELECT 
    id,
    code,
    geo_location.STAsText() AS geo_location_wkt,
    geom_shape.STAsText() AS geom_shape_wkt,
    geom_shape.STArea() AS shape_area,
    status
FROM dbo.spatial_mixed
WHERE status = 'active';
GO
`,
			wantMatch: true,
		},
		{
			name: "spatial_indexes_advanced_configurations",
			setupSQL: `
CREATE SCHEMA advanced_spatial;
GO

-- Table with multiple spatial columns
CREATE TABLE advanced_spatial.geo_data (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(255) NOT NULL,
    -- Geography columns
    current_location GEOGRAPHY NOT NULL,
    delivery_route GEOGRAPHY,
    service_area GEOGRAPHY NOT NULL,
    -- Geometry columns  
    building_footprint GEOMETRY,
    floor_plan GEOMETRY,
    -- Regular columns
    category VARCHAR(100),
    is_active BIT DEFAULT 1,
    last_updated DATETIME2 DEFAULT GETDATE()
);
GO

-- Create spatial indexes with different tessellation schemes
CREATE SPATIAL INDEX idx_current_location ON advanced_spatial.geo_data(current_location) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 1,
    PAD_INDEX = OFF,
    FILLFACTOR = 100,
    ALLOW_ROW_LOCKS = OFF,
    ALLOW_PAGE_LOCKS = OFF
);
GO

CREATE SPATIAL INDEX idx_delivery_route ON advanced_spatial.geo_data(delivery_route) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8192,
    PAD_INDEX = ON,
    FILLFACTOR = 50
);
GO

CREATE SPATIAL INDEX idx_service_area ON advanced_spatial.geo_data(service_area) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = LOW, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 256
);
GO

CREATE SPATIAL INDEX idx_building_footprint ON advanced_spatial.geo_data(building_footprint) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-10000, -10000, 10000, 10000),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 128
);
GO

CREATE SPATIAL INDEX idx_floor_plan ON advanced_spatial.geo_data(floor_plan) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 1000, 1000),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = MEDIUM, LEVEL_3 = LOW, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 512
);
GO

-- Create filtered indexes
CREATE INDEX idx_active_category ON advanced_spatial.geo_data(category) 
WHERE is_active = 1;
GO

-- Create columnstore index
CREATE NONCLUSTERED COLUMNSTORE INDEX ncci_geo_data 
ON advanced_spatial.geo_data(id, name, category, is_active, last_updated);
GO
`,
			wantMatch: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Safe to parallelize - shared container, unique databases per test
			// Use test name for database name - each test case has a unique name
			databaseName := fmt.Sprintf("test_%s", strings.ReplaceAll(tc.name, " ", "_"))

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
			newDatabaseName := fmt.Sprintf("test_copy_%s", strings.ReplaceAll(tc.name, " ", "_"))

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

				require.Equal(t, len(metadataA.Schemas), len(metadataB.Schemas), "Number of schemas should match")

				for i, schemaA := range metadataA.Schemas {
					schemaB := metadataB.Schemas[i]
					require.Equal(t, schemaA.Name, schemaB.Name, "Schema names should match")
					require.Equal(t, len(schemaA.Tables), len(schemaB.Tables), "Number of tables in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Views), len(schemaB.Views), "Number of views in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Functions), len(schemaB.Functions), "Number of functions in schema %s should match", schemaA.Name)
					require.Equal(t, len(schemaA.Procedures), len(schemaB.Procedures), "Number of procedures in schema %s should match", schemaA.Name)

					// Compare tables in detail
					for j, tableA := range schemaA.Tables {
						tableB := schemaB.Tables[j]
						require.Equal(t, tableA.Name, tableB.Name, "Table names should match")
						require.Equal(t, len(tableA.Columns), len(tableB.Columns), "Number of columns in table %s.%s should match", schemaA.Name, tableA.Name)
						require.Equal(t, len(tableA.Indexes), len(tableB.Indexes), "Number of indexes in table %s.%s should match", schemaA.Name, tableA.Name)

						// Compare spatial indexes specifically
						spatialIndexesA := make(map[string]*storepb.IndexMetadata)
						spatialIndexesB := make(map[string]*storepb.IndexMetadata)

						for _, idx := range tableA.Indexes {
							if idx.Type == "SPATIAL" {
								spatialIndexesA[idx.Name] = idx
							}
						}

						for _, idx := range tableB.Indexes {
							if idx.Type == "SPATIAL" {
								spatialIndexesB[idx.Name] = idx
							}
						}

						require.Equal(t, len(spatialIndexesA), len(spatialIndexesB), "Number of spatial indexes in table %s.%s should match", schemaA.Name, tableA.Name)

						// Compare each spatial index in detail
						for name, idxA := range spatialIndexesA {
							idxB, exists := spatialIndexesB[name]
							require.True(t, exists, "Spatial index %s should exist in both schemas", name)

							// Compare spatial index properties
							require.Equal(t, idxA.Type, idxB.Type, "Index type should match for %s", name)
							require.Equal(t, idxA.Unique, idxB.Unique, "Unique property should match for %s", name)
							require.Equal(t, idxA.Primary, idxB.Primary, "Primary property should match for %s", name)
							require.Equal(t, idxA.Expressions, idxB.Expressions, "Expressions should match for %s", name)

							// Compare spatial configuration
							if idxA.SpatialConfig != nil {
								require.NotNil(t, idxB.SpatialConfig, "Spatial config should exist for %s in schema B", name)

								// Compare method
								require.Equal(t, idxA.SpatialConfig.Method, idxB.SpatialConfig.Method, "Spatial method should match for %s", name)

								// Compare tessellation
								if idxA.SpatialConfig.Tessellation != nil {
									require.NotNil(t, idxB.SpatialConfig.Tessellation, "Tessellation config should exist for %s in schema B", name)
									require.Equal(t, idxA.SpatialConfig.Tessellation.Scheme, idxB.SpatialConfig.Tessellation.Scheme, "Tessellation scheme should match for %s", name)
									require.Equal(t, idxA.SpatialConfig.Tessellation.CellsPerObject, idxB.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should match for %s", name)

									// Compare grid levels
									require.Equal(t, len(idxA.SpatialConfig.Tessellation.GridLevels), len(idxB.SpatialConfig.Tessellation.GridLevels), "Number of grid levels should match for %s", name)
									for k, levelA := range idxA.SpatialConfig.Tessellation.GridLevels {
										levelB := idxB.SpatialConfig.Tessellation.GridLevels[k]
										require.Equal(t, levelA.Level, levelB.Level, "Grid level should match for %s", name)
										require.Equal(t, levelA.Density, levelB.Density, "Grid density should match for %s at level %d", name, levelA.Level)
									}

									// Compare bounding box for GEOMETRY indexes
									if idxA.SpatialConfig.Tessellation.BoundingBox != nil {
										require.NotNil(t, idxB.SpatialConfig.Tessellation.BoundingBox, "Bounding box should exist for %s in schema B", name)
										require.Equal(t, idxA.SpatialConfig.Tessellation.BoundingBox.Xmin, idxB.SpatialConfig.Tessellation.BoundingBox.Xmin, "Xmin should match for %s", name)
										require.Equal(t, idxA.SpatialConfig.Tessellation.BoundingBox.Ymin, idxB.SpatialConfig.Tessellation.BoundingBox.Ymin, "Ymin should match for %s", name)
										require.Equal(t, idxA.SpatialConfig.Tessellation.BoundingBox.Xmax, idxB.SpatialConfig.Tessellation.BoundingBox.Xmax, "Xmax should match for %s", name)
										require.Equal(t, idxA.SpatialConfig.Tessellation.BoundingBox.Ymax, idxB.SpatialConfig.Tessellation.BoundingBox.Ymax, "Ymax should match for %s", name)
									}
								}

								// Compare storage configuration
								if idxA.SpatialConfig.Storage != nil {
									require.NotNil(t, idxB.SpatialConfig.Storage, "Storage config should exist for %s in schema B", name)
									require.Equal(t, idxA.SpatialConfig.Storage.Fillfactor, idxB.SpatialConfig.Storage.Fillfactor, "Fillfactor should match for %s", name)
									require.Equal(t, idxA.SpatialConfig.Storage.PadIndex, idxB.SpatialConfig.Storage.PadIndex, "PadIndex should match for %s", name)
									require.Equal(t, idxA.SpatialConfig.Storage.AllowRowLocks, idxB.SpatialConfig.Storage.AllowRowLocks, "AllowRowLocks should match for %s", name)
									require.Equal(t, idxA.SpatialConfig.Storage.AllowPageLocks, idxB.SpatialConfig.Storage.AllowPageLocks, "AllowPageLocks should match for %s", name)
								}

								// Compare dimensional configuration
								if idxA.SpatialConfig.Dimensional != nil {
									require.NotNil(t, idxB.SpatialConfig.Dimensional, "Dimensional config should exist for %s in schema B", name)
									require.Equal(t, idxA.SpatialConfig.Dimensional.DataType, idxB.SpatialConfig.Dimensional.DataType, "Spatial data type should match for %s", name)
									require.Equal(t, idxA.SpatialConfig.Dimensional.Dimensions, idxB.SpatialConfig.Dimensional.Dimensions, "Dimensions should match for %s", name)
								}
							}
						}
					}
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

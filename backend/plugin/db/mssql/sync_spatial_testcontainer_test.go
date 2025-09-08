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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// syncDBSchemaWithRetry attempts to sync the database schema with retry logic for transient errors
func syncDBSchemaWithRetry(ctx context.Context, driver *Driver) (*storepb.DatabaseSchemaMetadata, error) {
	const maxRetries = 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		metadata, err := driver.SyncDBSchema(ctx)
		if err == nil {
			return metadata, nil
		}

		// Check if it's a transient error that should be retried
		errorStr := strings.ToLower(err.Error())
		isTransientError := strings.Contains(errorStr, "eof") ||
			strings.Contains(errorStr, "i/o timeout") ||
			strings.Contains(errorStr, "connection") ||
			strings.Contains(errorStr, "broken pipe") ||
			strings.Contains(errorStr, "network") ||
			strings.Contains(errorStr, "timeout")

		if isTransientError {
			lastErr = err
			if i < maxRetries-1 {
				// Wait before retry with exponential backoff
				waitTime := time.Duration(i+1) * 500 * time.Millisecond
				time.Sleep(waitTime)
				continue
			}
		}
		// For non-transient errors, return immediately
		return nil, err
	}
	return nil, errors.Wrapf(lastErr, "failed after %d retries", maxRetries)
}

func TestSyncSpatialIndexWithTestcontainer(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestMSSQLContainer(ctx, t)
	defer container.Close(ctx)

	host := container.GetHost()
	port := container.GetPort()
	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		setupSQL string
		validate func(*testing.T, *Driver, string)
	}{
		{
			name: "sync_spatial_indexes_with_full_configuration",
			setupSQL: `
CREATE SCHEMA [geo];
GO

-- Create table with spatial columns
CREATE TABLE [geo].[locations] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(200) NOT NULL,
    [location_point] GEOGRAPHY NOT NULL,
    [boundary_polygon] GEOGRAPHY NOT NULL,
    [area_shape] GEOGRAPHY NOT NULL,
    [description] NVARCHAR(MAX)
);
GO

-- Create GEOGRAPHY spatial indexes with comprehensive configuration
CREATE SPATIAL INDEX [idx_location_point] ON [geo].[locations] ([location_point]) 
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

CREATE SPATIAL INDEX [idx_boundary_polygon] ON [geo].[locations] ([boundary_polygon]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16,
    FILLFACTOR = 80
);
GO

CREATE SPATIAL INDEX [idx_area_shape] ON [geo].[locations] ([area_shape]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 64
);
GO
`,
			validate: func(t *testing.T, driver *Driver, _ string) {
				// Add delay to allow metadata propagation
				time.Sleep(1 * time.Second)

				// Sync database schema with retry logic
				metadata, err := syncDBSchemaWithRetry(ctx, driver)
				require.NoError(t, err)
				require.NotNil(t, metadata)

				// Find the geo schema
				var geoSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "geo" {
						geoSchema = schema
						break
					}
				}
				require.NotNil(t, geoSchema, "geo schema should exist")

				// Find the locations table
				require.Len(t, geoSchema.Tables, 1)
				table := geoSchema.Tables[0]
				require.Equal(t, "locations", table.Name)

				// Verify spatial columns exist
				spatialColumns := make(map[string]*storepb.ColumnMetadata)
				for _, col := range table.Columns {
					if strings.Contains(strings.ToLower(col.Type), "geography") {
						spatialColumns[col.Name] = col
					}
				}
				require.Contains(t, spatialColumns, "location_point")
				require.Contains(t, spatialColumns, "boundary_polygon")
				require.Contains(t, spatialColumns, "area_shape")

				// Verify spatial indexes
				spatialIndexes := make(map[string]*storepb.IndexMetadata)
				for _, idx := range table.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexes[idx.Name] = idx
					}
				}

				// Verify specific spatial indexes and their configurations
				// Helper function to validate spatial index metadata
				validateSpatialIndex := func(t *testing.T, idx *storepb.IndexMetadata, expectedName string, expectedColumn string, expectedDataType string) {
					t.Helper()

					// Basic index properties
					require.Equal(t, "SPATIAL", idx.Type, "Index type should be SPATIAL")
					require.Equal(t, expectedName, idx.Name, "Index name should match")
					require.False(t, idx.Unique, "Spatial indexes should not be unique")
					require.False(t, idx.Primary, "Spatial indexes should not be primary")
					require.Len(t, idx.Expressions, 1, "Should have exactly one expression")
					require.Equal(t, expectedColumn, idx.Expressions[0], "Column name should match")
					require.Len(t, idx.Descending, 1, "Should have descending flag")
					require.False(t, idx.Descending[0], "Spatial indexes don't support descending")

					// Spatial configuration
					require.NotNil(t, idx.SpatialConfig, "Spatial config should not be nil")
					require.Equal(t, "SPATIAL", idx.SpatialConfig.Method, "Method should be SPATIAL")

					// Tessellation configuration
					require.NotNil(t, idx.SpatialConfig.Tessellation, "Tessellation config should exist")
					if idx.SpatialConfig.Tessellation.Scheme != "UNKNOWN" {
						// Full SQL Server with spatial metadata
						require.Equal(t, expectedDataType+"_GRID", idx.SpatialConfig.Tessellation.Scheme)

						// Grid levels (if available)
						if len(idx.SpatialConfig.Tessellation.GridLevels) > 0 {
							for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
								require.GreaterOrEqual(t, level.Level, int32(1), "Level should be >= 1")
								require.LessOrEqual(t, level.Level, int32(4), "Level should be <= 4")
								require.Contains(t, []string{"LOW", "MEDIUM", "HIGH"}, level.Density, "Density should be valid")
							}
						}

						// Cells per object
						if idx.SpatialConfig.Tessellation.CellsPerObject > 0 {
							require.GreaterOrEqual(t, idx.SpatialConfig.Tessellation.CellsPerObject, int32(1))
							require.LessOrEqual(t, idx.SpatialConfig.Tessellation.CellsPerObject, int32(8192))
						}

						// Bounding box (only for GEOMETRY indexes)
						if expectedDataType == "GEOMETRY" && idx.SpatialConfig.Tessellation.BoundingBox != nil {
							bbox := idx.SpatialConfig.Tessellation.BoundingBox
							require.Less(t, bbox.Xmin, bbox.Xmax, "Xmin should be less than Xmax")
							require.Less(t, bbox.Ymin, bbox.Ymax, "Ymin should be less than Ymax")
						}
					}

					// Storage configuration
					require.NotNil(t, idx.SpatialConfig.Storage, "Storage config should exist")
					if idx.SpatialConfig.Storage.Fillfactor > 0 {
						require.GreaterOrEqual(t, idx.SpatialConfig.Storage.Fillfactor, int32(1))
						require.LessOrEqual(t, idx.SpatialConfig.Storage.Fillfactor, int32(100))
					}

					// Dimensional configuration
					require.NotNil(t, idx.SpatialConfig.Dimensional, "Dimensional config should exist")
					require.Equal(t, expectedDataType, idx.SpatialConfig.Dimensional.DataType)
					require.Equal(t, int32(2), idx.SpatialConfig.Dimensional.Dimensions, "SQL Server spatial is always 2D")
				}

				// Validate each spatial index with specific expected values
				if idx, exists := spatialIndexes["idx_location_point"]; exists {
					validateSpatialIndex(t, idx, "idx_location_point", "location_point", "GEOGRAPHY")

					// Verify specific configuration values from CREATE INDEX statement
					require.Equal(t, int32(32), idx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 32")
					require.Equal(t, int32(85), idx.SpatialConfig.Storage.Fillfactor, "Fillfactor should be 85")
					require.True(t, idx.SpatialConfig.Storage.PadIndex, "PadIndex should be ON")
					require.True(t, idx.SpatialConfig.Storage.AllowRowLocks, "AllowRowLocks should be ON")
					require.True(t, idx.SpatialConfig.Storage.AllowPageLocks, "AllowPageLocks should be ON")
					// Note: MAXDOP, SORT_IN_TEMPDB, and DATA_COMPRESSION might not be captured by sys.spatial_indexes

					// Verify grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "MEDIUM", gridMap[1], "Level 1 should be MEDIUM")
					require.Equal(t, "HIGH", gridMap[2], "Level 2 should be HIGH")
					require.Equal(t, "MEDIUM", gridMap[3], "Level 3 should be MEDIUM")
					require.Equal(t, "LOW", gridMap[4], "Level 4 should be LOW")
				} else {
					t.Error("Expected spatial index idx_location_point not found")
				}

				if idx, exists := spatialIndexes["idx_boundary_polygon"]; exists {
					validateSpatialIndex(t, idx, "idx_boundary_polygon", "boundary_polygon", "GEOGRAPHY")

					// Verify specific configuration values
					require.Equal(t, int32(16), idx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 16")
					require.Equal(t, int32(80), idx.SpatialConfig.Storage.Fillfactor, "Fillfactor should be 80")

					// Verify grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "LOW", gridMap[1], "Level 1 should be LOW")
					require.Equal(t, "MEDIUM", gridMap[2], "Level 2 should be MEDIUM")
					require.Equal(t, "HIGH", gridMap[3], "Level 3 should be HIGH")
					require.Equal(t, "MEDIUM", gridMap[4], "Level 4 should be MEDIUM")
				} else {
					t.Error("Expected spatial index idx_boundary_polygon not found")
				}

				if idx, exists := spatialIndexes["idx_area_shape"]; exists {
					validateSpatialIndex(t, idx, "idx_area_shape", "area_shape", "GEOGRAPHY")

					// Verify specific configuration values
					require.Equal(t, int32(64), idx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 64")
					// No fillfactor specified, so it should be 0 or default

					// Verify grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "HIGH", gridMap[1], "Level 1 should be HIGH")
					require.Equal(t, "HIGH", gridMap[2], "Level 2 should be HIGH")
					require.Equal(t, "MEDIUM", gridMap[3], "Level 3 should be MEDIUM")
					require.Equal(t, "LOW", gridMap[4], "Level 4 should be LOW")
				} else {
					t.Error("Expected spatial index idx_area_shape not found")
				}

				// Verify that we have at least the expected number of spatial indexes
				// (Even if configuration details are missing in SQL Server Express)
				require.GreaterOrEqual(t, len(spatialIndexes), 3, "Should have at least 3 spatial indexes")
			},
		},
		{
			name: "sync_spatial_indexes_with_ddl_preservation",
			setupSQL: `
CREATE SCHEMA [spatial_test];
GO

-- Create table with mixed spatial data types
CREATE TABLE [spatial_test].[mixed_spatial] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [geo_point] GEOGRAPHY NOT NULL,
    [geo_line] GEOGRAPHY NOT NULL,
    [created_at] DATETIME2 DEFAULT GETDATE()
);
GO

-- Create spatial indexes that should be preserved
CREATE SPATIAL INDEX [idx_geo_point] ON [spatial_test].[mixed_spatial] ([geo_point]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX [idx_geo_line] ON [spatial_test].[mixed_spatial] ([geo_line]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = MEDIUM, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8
);
GO
`,
			validate: func(t *testing.T, driver *Driver, _ string) {
				// Add delay to allow metadata propagation
				time.Sleep(1 * time.Second)

				// First sync to capture initial state with retry logic
				metadata1, err := syncDBSchemaWithRetry(ctx, driver)
				require.NoError(t, err)
				require.NotNil(t, metadata1)

				// Find the spatial_test schema
				var spatialSchema *storepb.SchemaMetadata
				for _, schema := range metadata1.Schemas {
					if schema.Name == "spatial_test" {
						spatialSchema = schema
						break
					}
				}
				require.NotNil(t, spatialSchema, "spatial_test schema should exist")

				// Find the mixed_spatial table
				require.Len(t, spatialSchema.Tables, 1)
				table := spatialSchema.Tables[0]
				require.Equal(t, "mixed_spatial", table.Name)

				// Count spatial indexes
				spatialIndexCount := 0
				var spatialIndexes []*storepb.IndexMetadata
				for _, idx := range table.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexCount++
						spatialIndexes = append(spatialIndexes, idx)
					}
				}

				require.GreaterOrEqual(t, spatialIndexCount, 2, "Should have at least 2 spatial indexes")

				// Verify specific indexes exist and have spatial config
				indexNames := make(map[string]*storepb.IndexMetadata)
				for _, idx := range spatialIndexes {
					indexNames[idx.Name] = idx

					// Spatial config should be available through DDL preservation
					require.NotNil(t, idx.SpatialConfig, "Spatial config should be preserved for index %s", idx.Name)
				}

				// Verify specific configuration values for idx_geo_point
				if idx, exists := indexNames["idx_geo_point"]; exists {
					require.Equal(t, int32(16), idx.SpatialConfig.Tessellation.CellsPerObject, "idx_geo_point CellsPerObject should be 16")
					require.Equal(t, "GEOGRAPHY_GRID", idx.SpatialConfig.Tessellation.Scheme, "Should use GEOGRAPHY_GRID tessellation")

					// Verify grid levels for idx_geo_point
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "MEDIUM", gridMap[1], "Level 1 should be MEDIUM")
					require.Equal(t, "HIGH", gridMap[2], "Level 2 should be HIGH")
					require.Equal(t, "MEDIUM", gridMap[3], "Level 3 should be MEDIUM")
					require.Equal(t, "LOW", gridMap[4], "Level 4 should be LOW")
				}

				// Verify specific configuration values for idx_geo_line
				if idx, exists := indexNames["idx_geo_line"]; exists {
					require.Equal(t, int32(8), idx.SpatialConfig.Tessellation.CellsPerObject, "idx_geo_line CellsPerObject should be 8")
					require.Equal(t, "GEOGRAPHY_GRID", idx.SpatialConfig.Tessellation.Scheme, "Should use GEOGRAPHY_GRID tessellation")

					// Verify grid levels for idx_geo_line
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "LOW", gridMap[1], "Level 1 should be LOW")
					require.Equal(t, "LOW", gridMap[2], "Level 2 should be LOW")
					require.Equal(t, "MEDIUM", gridMap[3], "Level 3 should be MEDIUM")
					require.Equal(t, "HIGH", gridMap[4], "Level 4 should be HIGH")
				}

				// Verify expected indexes are present
				expectedIndexes := []string{"idx_geo_point", "idx_geo_line"}
				for _, expectedName := range expectedIndexes {
					if _, exists := indexNames[expectedName]; !exists {
						// In case of naming variations, check for partial matches
						found := false
						for actualName := range indexNames {
							if strings.Contains(actualName, "geo_point") || strings.Contains(actualName, "geo_line") {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("No spatial index found matching pattern for %s", expectedName)
						}
					}
				}

				// Perform second sync to test consistency with retry logic
				metadata2, err := syncDBSchemaWithRetry(ctx, driver)
				require.NoError(t, err)
				require.NotNil(t, metadata2)

				// Find the spatial_test schema in second sync
				var spatialSchema2 *storepb.SchemaMetadata
				for _, schema := range metadata2.Schemas {
					if schema.Name == "spatial_test" {
						spatialSchema2 = schema
						break
					}
				}
				require.NotNil(t, spatialSchema2, "spatial_test schema should exist in second sync")

				// Verify consistency between syncs
				table2 := spatialSchema2.Tables[0]
				spatialIndexCount2 := 0
				for _, idx := range table2.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexCount2++
					}
				}

				require.Equal(t, spatialIndexCount, spatialIndexCount2, "Spatial index count should be consistent between syncs")
			},
		},
		{
			name: "sync_geometry_spatial_indexes",
			setupSQL: `
CREATE SCHEMA [geom_test];
GO

-- Create table with GEOMETRY spatial columns
CREATE TABLE [geom_test].[spatial_data] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [location] GEOMETRY NOT NULL,
    [boundary] GEOMETRY NOT NULL
);
GO

-- Create GEOMETRY spatial indexes with bounding box
CREATE SPATIAL INDEX [idx_location_geom] ON [geom_test].[spatial_data] ([location]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 100, 100),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 16,
    PAD_INDEX = ON,
    FILLFACTOR = 90
);
GO

CREATE SPATIAL INDEX [idx_boundary_geom] ON [geom_test].[spatial_data] ([boundary]) 
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 64
);
GO
`,
			validate: func(t *testing.T, driver *Driver, _ string) {
				// Add delay to allow metadata propagation
				time.Sleep(1 * time.Second)

				// Sync database schema with retry logic
				metadata, err := syncDBSchemaWithRetry(ctx, driver)
				require.NoError(t, err)
				require.NotNil(t, metadata)

				// Find the geom_test schema
				var geomSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "geom_test" {
						geomSchema = schema
						break
					}
				}
				require.NotNil(t, geomSchema, "geom_test schema should exist")

				// Find the spatial_data table
				require.Len(t, geomSchema.Tables, 1)
				table := geomSchema.Tables[0]
				require.Equal(t, "spatial_data", table.Name)

				// Extract spatial indexes
				spatialIndexes := make(map[string]*storepb.IndexMetadata)
				for _, idx := range table.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexes[idx.Name] = idx
					}
				}

				// Use the same validation function
				validateSpatialIndex := func(t *testing.T, idx *storepb.IndexMetadata, expectedName string, expectedColumn string, expectedDataType string) {
					t.Helper()

					// Basic index properties
					require.Equal(t, "SPATIAL", idx.Type, "Index type should be SPATIAL")
					require.Equal(t, expectedName, idx.Name, "Index name should match")
					require.False(t, idx.Unique, "Spatial indexes should not be unique")
					require.False(t, idx.Primary, "Spatial indexes should not be primary")
					require.Len(t, idx.Expressions, 1, "Should have exactly one expression")
					require.Equal(t, expectedColumn, idx.Expressions[0], "Column name should match")
					require.Len(t, idx.Descending, 1, "Should have descending flag")
					require.False(t, idx.Descending[0], "Spatial indexes don't support descending")

					// Spatial configuration
					require.NotNil(t, idx.SpatialConfig, "Spatial config should not be nil")
					require.Equal(t, "SPATIAL", idx.SpatialConfig.Method, "Method should be SPATIAL")

					// Tessellation configuration
					require.NotNil(t, idx.SpatialConfig.Tessellation, "Tessellation config should exist")

					// Storage configuration
					require.NotNil(t, idx.SpatialConfig.Storage, "Storage config should exist")

					// Dimensional configuration
					require.NotNil(t, idx.SpatialConfig.Dimensional, "Dimensional config should exist")
					require.Equal(t, expectedDataType, idx.SpatialConfig.Dimensional.DataType)
					require.Equal(t, int32(2), idx.SpatialConfig.Dimensional.Dimensions, "SQL Server spatial is always 2D")
				}

				// Validate each GEOMETRY spatial index with specific values
				if idx, exists := spatialIndexes["idx_location_geom"]; exists {
					validateSpatialIndex(t, idx, "idx_location_geom", "location", "GEOMETRY")

					// Verify specific configuration values
					require.Equal(t, int32(16), idx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 16")
					require.Equal(t, int32(90), idx.SpatialConfig.Storage.Fillfactor, "Fillfactor should be 90")
					require.True(t, idx.SpatialConfig.Storage.PadIndex, "PadIndex should be ON")

					// Verify bounding box for GEOMETRY indexes
					require.NotNil(t, idx.SpatialConfig.Tessellation.BoundingBox, "GEOMETRY index should have bounding box")
					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(0), bbox.Xmin, "Xmin should be 0")
					require.Equal(t, float64(0), bbox.Ymin, "Ymin should be 0")
					require.Equal(t, float64(100), bbox.Xmax, "Xmax should be 100")
					require.Equal(t, float64(100), bbox.Ymax, "Ymax should be 100")

					// Verify grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "MEDIUM", gridMap[1], "Level 1 should be MEDIUM")
					require.Equal(t, "HIGH", gridMap[2], "Level 2 should be HIGH")
					require.Equal(t, "LOW", gridMap[3], "Level 3 should be LOW")
					require.Equal(t, "LOW", gridMap[4], "Level 4 should be LOW")
				} else {
					t.Error("Expected spatial index idx_location_geom not found")
				}

				if idx, exists := spatialIndexes["idx_boundary_geom"]; exists {
					validateSpatialIndex(t, idx, "idx_boundary_geom", "boundary", "GEOMETRY")

					// Verify specific configuration values
					require.Equal(t, int32(64), idx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 64")

					// Verify bounding box
					require.NotNil(t, idx.SpatialConfig.Tessellation.BoundingBox, "GEOMETRY index should have bounding box")
					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(-180), bbox.Xmin, "Xmin should be -180")
					require.Equal(t, float64(-90), bbox.Ymin, "Ymin should be -90")
					require.Equal(t, float64(180), bbox.Xmax, "Xmax should be 180")
					require.Equal(t, float64(90), bbox.Ymax, "Ymax should be 90")

					// Verify grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "HIGH", gridMap[1], "Level 1 should be HIGH")
					require.Equal(t, "HIGH", gridMap[2], "Level 2 should be HIGH")
					require.Equal(t, "MEDIUM", gridMap[3], "Level 3 should be MEDIUM")
					require.Equal(t, "LOW", gridMap[4], "Level 4 should be LOW")
				} else {
					t.Error("Expected spatial index idx_boundary_geom not found")
				}

				require.GreaterOrEqual(t, len(spatialIndexes), 2, "Should have at least 2 GEOMETRY spatial indexes")
			},
		},
		{
			name: "sync_spatial_indexes_with_regular_indexes",
			setupSQL: `
CREATE SCHEMA [mixed_test];
GO

-- Create table with both spatial and regular columns
CREATE TABLE [mixed_test].[data_table] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] NVARCHAR(100) NOT NULL,
    [code] VARCHAR(50) UNIQUE,
    [geo_location] GEOGRAPHY NOT NULL,
    [status] NVARCHAR(20),
    [created_at] DATETIME2
);
GO

-- Create regular indexes
CREATE INDEX [idx_name] ON [mixed_test].[data_table] ([name]);
GO

CREATE INDEX [idx_status_created] ON [mixed_test].[data_table] ([status], [created_at]);
GO

-- Create spatial index
CREATE SPATIAL INDEX [idx_geo_location] ON [mixed_test].[data_table] ([geo_location]) 
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16
);
GO
`,
			validate: func(t *testing.T, driver *Driver, _ string) {
				// Sync database schema with retry logic for transient errors
				metadata, err := syncDBSchemaWithRetry(ctx, driver)
				require.NoError(t, err)
				require.NotNil(t, metadata)

				// Find the mixed_test schema
				var mixedSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "mixed_test" {
						mixedSchema = schema
						break
					}
				}
				require.NotNil(t, mixedSchema, "mixed_test schema should exist")

				// Find the data_table
				require.Len(t, mixedSchema.Tables, 1)
				table := mixedSchema.Tables[0]
				require.Equal(t, "data_table", table.Name)

				// Categorize indexes
				var spatialIndexes []*storepb.IndexMetadata
				var regularIndexes []*storepb.IndexMetadata
				var uniqueIndexes []*storepb.IndexMetadata
				var primaryKeys []*storepb.IndexMetadata

				for _, idx := range table.Indexes {
					switch {
					case idx.Type == "SPATIAL":
						spatialIndexes = append(spatialIndexes, idx)
					case idx.Primary:
						primaryKeys = append(primaryKeys, idx)
					case idx.Unique:
						uniqueIndexes = append(uniqueIndexes, idx)
					default:
						regularIndexes = append(regularIndexes, idx)
					}
				}

				// Verify we have the expected types of indexes
				require.Equal(t, 1, len(primaryKeys), "Should have 1 primary key")
				require.Equal(t, 1, len(uniqueIndexes), "Should have 1 unique index (code)")
				require.GreaterOrEqual(t, len(regularIndexes), 2, "Should have at least 2 regular indexes")
				require.GreaterOrEqual(t, len(spatialIndexes), 1, "Should have at least 1 spatial index")

				// Verify spatial index details
				if len(spatialIndexes) > 0 {
					spatialIdx := spatialIndexes[0]
					require.Equal(t, "SPATIAL", spatialIdx.Type)
					require.False(t, spatialIdx.Unique)
					require.False(t, spatialIdx.Primary)
					require.Len(t, spatialIdx.Expressions, 1)
					require.Equal(t, "geo_location", spatialIdx.Expressions[0])

					// Spatial config should be available through DDL preservation
					require.NotNil(t, spatialIdx.SpatialConfig, "Spatial config should be preserved for geo_location index")
					require.NotNil(t, spatialIdx.SpatialConfig.Tessellation, "Tessellation config should exist")

					// Verify specific configuration values from CREATE INDEX statement
					require.Equal(t, int32(16), spatialIdx.SpatialConfig.Tessellation.CellsPerObject, "CellsPerObject should be 16")

					// Verify grid levels - all set to MEDIUM
					require.Len(t, spatialIdx.SpatialConfig.Tessellation.GridLevels, 4, "Should have 4 grid levels")
					for _, level := range spatialIdx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "MEDIUM", level.Density, fmt.Sprintf("Level %d should be MEDIUM", level.Level))
					}

					// Verify tessellation scheme
					require.Equal(t, "GEOGRAPHY_GRID", spatialIdx.SpatialConfig.Tessellation.Scheme, "Should use GEOGRAPHY_GRID tessellation")

					// Verify dimensional configuration
					require.NotNil(t, spatialIdx.SpatialConfig.Dimensional, "Dimensional config should exist")
					require.Equal(t, "GEOGRAPHY", spatialIdx.SpatialConfig.Dimensional.DataType)
					require.Equal(t, int32(2), spatialIdx.SpatialConfig.Dimensional.Dimensions, "SQL Server spatial is always 2D")
				}

				// Verify regular index details
				regularIndexNames := make(map[string]bool)
				for _, idx := range regularIndexes {
					regularIndexNames[idx.Name] = true
				}

				// Check for expected regular indexes (allowing for name variations)
				hasNameIndex := false
				hasStatusIndex := false
				for name := range regularIndexNames {
					if strings.Contains(name, "name") {
						hasNameIndex = true
					}
					if strings.Contains(name, "status") {
						hasStatusIndex = true
					}
				}
				require.True(t, hasNameIndex, "Should have an index on name column")
				require.True(t, hasStatusIndex, "Should have an index involving status column")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create unique database name for this test
			databaseName := fmt.Sprintf("test_%s_%d", tc.name, time.Now().Unix())

			// Create driver instance
			driverInstance := &Driver{}
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
			defer driver.Close(ctx)

			// Create test database
			_, err = driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE [%s]", databaseName), db.ExecuteOptions{CreateDatabase: true})
			require.NoError(t, err)

			// Clean up database after test
			defer func() {
				// Close current connection first
				driver.Close(ctx)

				// Reconnect to master to drop the test database
				config.DataSource.Database = "master"
				config.ConnectionContext.DatabaseName = "master"
				cleanupDriver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
				if err == nil {
					// Add retry logic for database cleanup
					for i := 0; i < 3; i++ {
						// First, force close all connections to the test database
						_, _ = cleanupDriver.Execute(ctx, fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", databaseName), db.ExecuteOptions{CreateDatabase: true})

						// Then drop the database
						_, err := cleanupDriver.Execute(ctx, fmt.Sprintf("DROP DATABASE [%s]", databaseName), db.ExecuteOptions{CreateDatabase: true})
						if err == nil {
							break
						}

						// Wait before retry
						if i < 2 {
							time.Sleep(500 * time.Millisecond)
						}
					}
					cleanupDriver.Close(ctx)
				}
			}()

			// Connect to the test database
			driver.Close(ctx)
			config.DataSource.Database = databaseName
			config.ConnectionContext.DatabaseName = databaseName
			driver, err = driverInstance.Open(ctx, storepb.Engine_MSSQL, config)
			require.NoError(t, err)

			// Execute setup SQL with retry logic for transient errors
			statements := splitSQLStatements(tc.setupSQL)
			for _, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}

				// Retry statement execution for transient errors
				var lastErr error
				for retry := 0; retry < 3; retry++ {
					_, err = driver.Execute(ctx, stmt, db.ExecuteOptions{})
					if err == nil {
						break
					}

					// Check if it's a transient error
					if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "i/o timeout") || strings.Contains(err.Error(), "connection") {
						lastErr = err
						if retry < 2 {
							time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
							continue
						}
					}
					// For non-transient errors, fail immediately
					require.NoError(t, err)
				}
				if lastErr != nil {
					require.NoError(t, lastErr)
				}
			}

			// Cast to *Driver to access SyncDBSchema
			mssqlDriver, ok := driver.(*Driver)
			require.True(t, ok)

			// Add delay to allow metadata propagation for all test cases
			time.Sleep(1 * time.Second)

			// Run validation
			tc.validate(t, mssqlDriver, databaseName)
		})
	}
}

func splitSQLStatements(script string) []string {
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

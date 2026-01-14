package mssql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

//nolint:tparallel
func TestGetDatabaseMetadataWithTestcontainer(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestMSSQLContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	host := container.GetHost()
	port := container.GetPort()

	testCases := []struct {
		name     string
		setupSQL string
		validate func(*testing.T, *storepb.DatabaseSchemaMetadata)
	}{
		{
			name: "spatial_indexes_comprehensive",
			setupSQL: `
CREATE SCHEMA geo;
GO

CREATE TABLE geo.locations (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(200) NOT NULL,
    location_point GEOMETRY NOT NULL,
    boundary_polygon GEOMETRY,
    geo_location GEOGRAPHY NOT NULL,
    route_line GEOGRAPHY
);
GO

-- Create GEOMETRY spatial indexes with full configuration
CREATE SPATIAL INDEX idx_location_point ON geo.locations(location_point)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-180, -90, 180, 90),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 32,
    PAD_INDEX = ON,
    FILLFACTOR = 90,
    ALLOW_ROW_LOCKS = ON,
    ALLOW_PAGE_LOCKS = ON
);
GO

CREATE SPATIAL INDEX idx_boundary_polygon ON geo.locations(boundary_polygon)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 1000, 1000),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 64
);
GO

-- Create GEOGRAPHY spatial indexes with different configurations
CREATE SPATIAL INDEX idx_geo_location ON geo.locations(geo_location)
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16,
    FILLFACTOR = 85,
    PAD_INDEX = OFF,
    ALLOW_ROW_LOCKS = OFF,
    ALLOW_PAGE_LOCKS = OFF
);
GO

CREATE SPATIAL INDEX idx_route_line ON geo.locations(route_line)
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 128
);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
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

				// Verify columns
				require.Len(t, table.Columns, 6)
				columnNames := make(map[string]*storepb.ColumnMetadata)
				for _, col := range table.Columns {
					columnNames[col.Name] = col
				}

				// Check spatial columns
				require.Contains(t, columnNames, "location_point")
				require.Equal(t, "geometry", strings.ToLower(columnNames["location_point"].Type))

				require.Contains(t, columnNames, "boundary_polygon")
				require.Equal(t, "geometry", strings.ToLower(columnNames["boundary_polygon"].Type))

				require.Contains(t, columnNames, "geo_location")
				require.Equal(t, "geography", strings.ToLower(columnNames["geo_location"].Type))

				require.Contains(t, columnNames, "route_line")
				require.Equal(t, "geography", strings.ToLower(columnNames["route_line"].Type))

				// Verify spatial indexes with full configuration validation
				spatialIndexes := make(map[string]*storepb.IndexMetadata)
				for _, idx := range table.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexes[idx.Name] = idx
					}
				}

				require.Len(t, spatialIndexes, 4, "Should have exactly 4 spatial indexes")

				// Validate idx_location_point
				if idx, exists := spatialIndexes["idx_location_point"]; exists {
					require.NotNil(t, idx.SpatialConfig, "Spatial config should exist for idx_location_point")
					require.Equal(t, "SPATIAL", idx.SpatialConfig.Method)

					// Validate tessellation
					require.NotNil(t, idx.SpatialConfig.Tessellation)
					require.Equal(t, "GEOMETRY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(32), idx.SpatialConfig.Tessellation.CellsPerObject)

					// Validate bounding box
					require.NotNil(t, idx.SpatialConfig.Tessellation.BoundingBox)
					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(-180), bbox.Xmin)
					require.Equal(t, float64(-90), bbox.Ymin)
					require.Equal(t, float64(180), bbox.Xmax)
					require.Equal(t, float64(90), bbox.Ymax)

					// Validate grid levels
					require.Len(t, idx.SpatialConfig.Tessellation.GridLevels, 4)
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "MEDIUM", gridMap[1])
					require.Equal(t, "HIGH", gridMap[2])
					require.Equal(t, "MEDIUM", gridMap[3])
					require.Equal(t, "LOW", gridMap[4])

					// Validate storage configuration
					require.NotNil(t, idx.SpatialConfig.Storage)
					require.Equal(t, int32(90), idx.SpatialConfig.Storage.Fillfactor)
					require.True(t, idx.SpatialConfig.Storage.PadIndex)
					require.True(t, idx.SpatialConfig.Storage.AllowRowLocks)
					require.True(t, idx.SpatialConfig.Storage.AllowPageLocks)

					// Validate dimensional configuration
					require.NotNil(t, idx.SpatialConfig.Dimensional)
					require.Equal(t, "GEOMETRY", idx.SpatialConfig.Dimensional.DataType)
					require.Equal(t, int32(2), idx.SpatialConfig.Dimensional.Dimensions)
				} else {
					t.Error("idx_location_point not found")
				}

				// Validate idx_boundary_polygon
				if idx, exists := spatialIndexes["idx_boundary_polygon"]; exists {
					require.NotNil(t, idx.SpatialConfig, "Spatial config should exist for idx_boundary_polygon")
					require.Equal(t, "GEOMETRY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(64), idx.SpatialConfig.Tessellation.CellsPerObject)

					// Validate bounding box
					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(0), bbox.Xmin)
					require.Equal(t, float64(0), bbox.Ymin)
					require.Equal(t, float64(1000), bbox.Xmax)
					require.Equal(t, float64(1000), bbox.Ymax)

					// Validate grid levels
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "LOW", gridMap[1])
					require.Equal(t, "LOW", gridMap[2])
					require.Equal(t, "HIGH", gridMap[3])
					require.Equal(t, "HIGH", gridMap[4])
				} else {
					t.Error("idx_boundary_polygon not found")
				}

				// Validate idx_geo_location
				if idx, exists := spatialIndexes["idx_geo_location"]; exists {
					require.NotNil(t, idx.SpatialConfig, "Spatial config should exist for idx_geo_location")
					require.Equal(t, "GEOGRAPHY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(16), idx.SpatialConfig.Tessellation.CellsPerObject)

					// GEOGRAPHY indexes don't have bounding box
					require.Nil(t, idx.SpatialConfig.Tessellation.BoundingBox)

					// Validate grid levels
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "LOW", gridMap[1])
					require.Equal(t, "MEDIUM", gridMap[2])
					require.Equal(t, "HIGH", gridMap[3])
					require.Equal(t, "MEDIUM", gridMap[4])

					// Validate storage configuration
					require.NotNil(t, idx.SpatialConfig.Storage)
					require.Equal(t, int32(85), idx.SpatialConfig.Storage.Fillfactor)
					require.False(t, idx.SpatialConfig.Storage.PadIndex)
					require.False(t, idx.SpatialConfig.Storage.AllowRowLocks)
					require.False(t, idx.SpatialConfig.Storage.AllowPageLocks)

					// Validate dimensional configuration
					require.Equal(t, "GEOGRAPHY", idx.SpatialConfig.Dimensional.DataType)
					require.Equal(t, int32(2), idx.SpatialConfig.Dimensional.Dimensions)
				} else {
					t.Error("idx_geo_location not found")
				}

				// Validate idx_route_line
				if idx, exists := spatialIndexes["idx_route_line"]; exists {
					require.NotNil(t, idx.SpatialConfig, "Spatial config should exist for idx_route_line")
					require.Equal(t, "GEOGRAPHY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(128), idx.SpatialConfig.Tessellation.CellsPerObject)

					// Validate grid levels
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "HIGH", gridMap[1])
					require.Equal(t, "HIGH", gridMap[2])
					require.Equal(t, "HIGH", gridMap[3])
					require.Equal(t, "HIGH", gridMap[4])
				} else {
					t.Error("idx_route_line not found")
				}
			},
		},
		{
			name: "spatial_indexes_edge_cases",
			setupSQL: `
CREATE TABLE dbo.spatial_edge_cases (
    id INT IDENTITY(1,1) PRIMARY KEY,
    min_config_geom GEOMETRY NOT NULL,
    max_config_geom GEOMETRY NOT NULL,
    min_config_geog GEOGRAPHY NOT NULL,
    max_config_geog GEOGRAPHY NOT NULL
);
GO

-- Minimal configuration spatial index
CREATE SPATIAL INDEX idx_min_geom ON dbo.spatial_edge_cases(min_config_geom)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (0, 0, 1, 1),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 1
);
GO

-- Maximum configuration spatial index
CREATE SPATIAL INDEX idx_max_geom ON dbo.spatial_edge_cases(max_config_geom)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-1000000, -1000000, 1000000, 1000000),
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8192,
    PAD_INDEX = ON,
    FILLFACTOR = 100,
    ALLOW_ROW_LOCKS = ON,
    ALLOW_PAGE_LOCKS = ON,
    MAXDOP = 4,
    DATA_COMPRESSION = PAGE
);
GO

-- Minimal GEOGRAPHY index
CREATE SPATIAL INDEX idx_min_geog ON dbo.spatial_edge_cases(min_config_geog)
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 1
);
GO

-- Maximum GEOGRAPHY index
CREATE SPATIAL INDEX idx_max_geog ON dbo.spatial_edge_cases(max_config_geog)
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = HIGH, LEVEL_3 = HIGH, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8192,
    FILLFACTOR = 50
);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				// Find dbo schema
				var dboSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "dbo" {
						dboSchema = schema
						break
					}
				}
				require.NotNil(t, dboSchema, "dbo schema should exist")

				// Find spatial_edge_cases table
				var spatialTable *storepb.TableMetadata
				for _, table := range dboSchema.Tables {
					if table.Name == "spatial_edge_cases" {
						spatialTable = table
						break
					}
				}
				require.NotNil(t, spatialTable, "spatial_edge_cases table should exist")

				// Collect spatial indexes
				spatialIndexes := make(map[string]*storepb.IndexMetadata)
				for _, idx := range spatialTable.Indexes {
					if idx.Type == "SPATIAL" {
						spatialIndexes[idx.Name] = idx
					}
				}

				require.Len(t, spatialIndexes, 4, "Should have exactly 4 spatial indexes")

				// Validate minimal GEOMETRY index
				if idx, exists := spatialIndexes["idx_min_geom"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, int32(1), idx.SpatialConfig.Tessellation.CellsPerObject)

					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(0), bbox.Xmin)
					require.Equal(t, float64(0), bbox.Ymin)
					require.Equal(t, float64(1), bbox.Xmax)
					require.Equal(t, float64(1), bbox.Ymax)

					// All grid levels should be LOW
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "LOW", level.Density)
					}
				} else {
					t.Error("idx_min_geom not found")
				}

				// Validate maximum GEOMETRY index
				if idx, exists := spatialIndexes["idx_max_geom"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, int32(8192), idx.SpatialConfig.Tessellation.CellsPerObject)

					bbox := idx.SpatialConfig.Tessellation.BoundingBox
					require.Equal(t, float64(-1000000), bbox.Xmin)
					require.Equal(t, float64(-1000000), bbox.Ymin)
					require.Equal(t, float64(1000000), bbox.Xmax)
					require.Equal(t, float64(1000000), bbox.Ymax)

					// All grid levels should be HIGH
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "HIGH", level.Density)
					}

					// Check storage options
					require.Equal(t, int32(100), idx.SpatialConfig.Storage.Fillfactor)
					require.True(t, idx.SpatialConfig.Storage.PadIndex)
					require.True(t, idx.SpatialConfig.Storage.AllowRowLocks)
					require.True(t, idx.SpatialConfig.Storage.AllowPageLocks)
				} else {
					t.Error("idx_max_geom not found")
				}

				// Validate minimal GEOGRAPHY index
				if idx, exists := spatialIndexes["idx_min_geog"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, int32(1), idx.SpatialConfig.Tessellation.CellsPerObject)
					require.Nil(t, idx.SpatialConfig.Tessellation.BoundingBox) // GEOGRAPHY doesn't have bounding box

					// All grid levels should be LOW
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "LOW", level.Density)
					}
				} else {
					t.Error("idx_min_geog not found")
				}

				// Validate maximum GEOGRAPHY index
				if idx, exists := spatialIndexes["idx_max_geog"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, int32(8192), idx.SpatialConfig.Tessellation.CellsPerObject)

					// All grid levels should be HIGH
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "HIGH", level.Density)
					}

					require.Equal(t, int32(50), idx.SpatialConfig.Storage.Fillfactor)
				} else {
					t.Error("idx_max_geog not found")
				}
			},
		},
		{
			name: "mixed_spatial_and_regular_indexes",
			setupSQL: `
CREATE TABLE dbo.mixed_indexes (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE,
    location GEOMETRY NOT NULL,
    area GEOGRAPHY,
    status INT,
    created_at DATETIME2 DEFAULT GETDATE()
);
GO

-- Create regular indexes
CREATE INDEX idx_name ON dbo.mixed_indexes(name);
CREATE INDEX idx_status_created ON dbo.mixed_indexes(status, created_at);
CREATE UNIQUE INDEX idx_name_status ON dbo.mixed_indexes(name, status) WHERE status IS NOT NULL;
GO

-- Create spatial indexes
CREATE SPATIAL INDEX idx_location ON dbo.mixed_indexes(location)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-100, -100, 100, 100),
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = MEDIUM, LEVEL_3 = MEDIUM, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX idx_area ON dbo.mixed_indexes(area)
USING GEOGRAPHY_GRID 
WITH (
    GRIDS = (LEVEL_1 = HIGH, LEVEL_2 = LOW, LEVEL_3 = HIGH, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 32
);
GO

-- Create columnstore index
CREATE NONCLUSTERED COLUMNSTORE INDEX idx_columnstore ON dbo.mixed_indexes(id, name, status);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				// Find dbo schema
				var dboSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "dbo" {
						dboSchema = schema
						break
					}
				}
				require.NotNil(t, dboSchema, "dbo schema should exist")

				// Find mixed_indexes table
				var mixedTable *storepb.TableMetadata
				for _, table := range dboSchema.Tables {
					if table.Name == "mixed_indexes" {
						mixedTable = table
						break
					}
				}
				require.NotNil(t, mixedTable, "mixed_indexes table should exist")

				// Count index types
				indexTypes := make(map[string]int)
				spatialIndexes := make(map[string]*storepb.IndexMetadata)

				for _, idx := range mixedTable.Indexes {
					if idx.Primary {
						indexTypes["PRIMARY"]++
					} else if idx.Type == "SPATIAL" {
						indexTypes["SPATIAL"]++
						spatialIndexes[idx.Name] = idx
					} else if idx.Type == "NONCLUSTERED COLUMNSTORE" {
						indexTypes["COLUMNSTORE"]++
					} else if idx.Unique {
						indexTypes["UNIQUE"]++
					} else {
						indexTypes["REGULAR"]++
					}
				}

				require.Equal(t, 1, indexTypes["PRIMARY"], "Should have 1 primary key")
				require.GreaterOrEqual(t, indexTypes["UNIQUE"], 1, "Should have at least 1 unique index")
				require.GreaterOrEqual(t, indexTypes["REGULAR"], 2, "Should have at least 2 regular indexes")
				require.Equal(t, 2, indexTypes["SPATIAL"], "Should have 2 spatial indexes")
				require.Equal(t, 1, indexTypes["COLUMNSTORE"], "Should have 1 columnstore index")

				// Validate spatial indexes
				if idx, exists := spatialIndexes["idx_location"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, "GEOMETRY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(16), idx.SpatialConfig.Tessellation.CellsPerObject)

					// All grid levels should be MEDIUM
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						require.Equal(t, "MEDIUM", level.Density)
					}
				}

				if idx, exists := spatialIndexes["idx_area"]; exists {
					require.NotNil(t, idx.SpatialConfig)
					require.Equal(t, "GEOGRAPHY_GRID", idx.SpatialConfig.Tessellation.Scheme)
					require.Equal(t, int32(32), idx.SpatialConfig.Tessellation.CellsPerObject)

					// Check alternating grid levels
					gridMap := make(map[int32]string)
					for _, level := range idx.SpatialConfig.Tessellation.GridLevels {
						gridMap[level.Level] = level.Density
					}
					require.Equal(t, "HIGH", gridMap[1])
					require.Equal(t, "LOW", gridMap[2])
					require.Equal(t, "HIGH", gridMap[3])
					require.Equal(t, "LOW", gridMap[4])
				}
			},
		},
		{
			name: "spatial_indexes_with_computed_columns",
			setupSQL: `
CREATE TABLE dbo.spatial_computed (
    id INT IDENTITY(1,1) PRIMARY KEY,
    shape GEOMETRY NOT NULL,
    -- Computed columns based on spatial data
    area AS shape.STArea() PERSISTED,
    perimeter AS shape.STLength() PERSISTED,
    center_x AS shape.STCentroid().STX PERSISTED,
    center_y AS shape.STCentroid().STY PERSISTED,
    as_text AS shape.STAsText() PERSISTED
);
GO

-- Create spatial index on the base column
CREATE SPATIAL INDEX idx_shape ON dbo.spatial_computed(shape)
USING GEOMETRY_GRID 
WITH (
    BOUNDING_BOX = (-500, -500, 500, 500),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 24,
    FILLFACTOR = 75
);
GO

-- Create regular indexes on computed columns
CREATE INDEX idx_area ON dbo.spatial_computed(area);
CREATE INDEX idx_center ON dbo.spatial_computed(center_x, center_y);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				// Find dbo schema
				var dboSchema *storepb.SchemaMetadata
				for _, schema := range metadata.Schemas {
					if schema.Name == "dbo" {
						dboSchema = schema
						break
					}
				}
				require.NotNil(t, dboSchema, "dbo schema should exist")

				// Find spatial_computed table
				var computedTable *storepb.TableMetadata
				for _, table := range dboSchema.Tables {
					if table.Name == "spatial_computed" {
						computedTable = table
						break
					}
				}
				require.NotNil(t, computedTable, "spatial_computed table should exist")

				// Verify computed columns exist
				columnNames := make(map[string]*storepb.ColumnMetadata)
				for _, col := range computedTable.Columns {
					columnNames[col.Name] = col
				}

				// Check that computed columns are present
				require.Contains(t, columnNames, "area")
				require.Contains(t, columnNames, "perimeter")
				require.Contains(t, columnNames, "center_x")
				require.Contains(t, columnNames, "center_y")
				require.Contains(t, columnNames, "as_text")

				// Find spatial index
				var spatialIndex *storepb.IndexMetadata
				for _, idx := range computedTable.Indexes {
					if idx.Type == "SPATIAL" && idx.Name == "idx_shape" {
						spatialIndex = idx
						break
					}
				}

				require.NotNil(t, spatialIndex, "idx_shape spatial index should exist")
				require.NotNil(t, spatialIndex.SpatialConfig, "Spatial config should exist")

				// Validate spatial index configuration
				require.Equal(t, "GEOMETRY_GRID", spatialIndex.SpatialConfig.Tessellation.Scheme)
				require.Equal(t, int32(24), spatialIndex.SpatialConfig.Tessellation.CellsPerObject)
				require.Equal(t, int32(75), spatialIndex.SpatialConfig.Storage.Fillfactor)

				// Validate bounding box
				bbox := spatialIndex.SpatialConfig.Tessellation.BoundingBox
				require.Equal(t, float64(-500), bbox.Xmin)
				require.Equal(t, float64(-500), bbox.Ymin)
				require.Equal(t, float64(500), bbox.Xmax)
				require.Equal(t, float64(500), bbox.Ymax)

				// Validate grid levels
				gridMap := make(map[int32]string)
				for _, level := range spatialIndex.SpatialConfig.Tessellation.GridLevels {
					gridMap[level.Level] = level.Density
				}
				require.Equal(t, "LOW", gridMap[1])
				require.Equal(t, "MEDIUM", gridMap[2])
				require.Equal(t, "HIGH", gridMap[3])
				require.Equal(t, "MEDIUM", gridMap[4])

				// Verify regular indexes on computed columns exist
				regularIndexCount := 0
				for _, idx := range computedTable.Indexes {
					if idx.Type != "SPATIAL" && !idx.Primary && !idx.Unique {
						regularIndexCount++
					}
				}
				require.GreaterOrEqual(t, regularIndexCount, 2, "Should have at least 2 regular indexes on computed columns")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Safe to parallelize - shared container, unique databases per test
			databaseName := fmt.Sprintf("test_%s", strings.ReplaceAll(tc.name, " ", "_"))

			// Create test database using container's master connection
			_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE [%s]", databaseName))
			require.NoError(t, err)

			// Connect to the test database
			driver, err := createMSSQLDriver(ctx, host, port, databaseName)
			require.NoError(t, err)
			defer driver.Close(ctx)

			// Execute setup SQL
			err = executeSQL(ctx, driver, tc.setupSQL)
			require.NoError(t, err)

			// Sync database schema to get full metadata
			metadata, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)

			// Validate the metadata
			tc.validate(t, metadata)
		})
	}
}

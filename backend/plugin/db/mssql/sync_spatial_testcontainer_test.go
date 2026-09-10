package mssql

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestSyncSpatialIndex pins what SyncDBSchema reports for SQL Server spatial
// indexes: tessellation scheme and grid levels, bounding box, storage options
// and the dimensional descriptor. sys.spatial_indexes exposes these through
// several catalog views, so a sync regression shows up here rather than in the
// parser tests under backend/plugin/schema/mssql.
//
//nolint:tparallel
func TestSyncSpatialIndex(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)

	testCases := []struct {
		name     string
		setupSQL string
		validate func(*testing.T, *storepb.DatabaseSchemaMetadata)
	}{
		{
			name: "geometry_and_geography_indexes",
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
				table := requireTable(t, metadata, "geo", "locations")

				require.Len(t, table.Columns, 6)
				columns := make(map[string]*storepb.ColumnMetadata)
				for _, column := range table.Columns {
					columns[column.Name] = column
				}
				require.Equal(t, "geometry", strings.ToLower(columns["location_point"].Type))
				require.Equal(t, "geometry", strings.ToLower(columns["boundary_polygon"].Type))
				require.Equal(t, "geography", strings.ToLower(columns["geo_location"].Type))
				require.Equal(t, "geography", strings.ToLower(columns["route_line"].Type))

				require.Len(t, spatialIndexes(table), 4)

				locationPoint := requireSpatialIndex(t, table, "idx_location_point", "location_point", "GEOMETRY")
				require.Equal(t, int32(32), locationPoint.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, locationPoint, -180, -90, 180, 90)
				requireGridLevels(t, locationPoint, "MEDIUM", "HIGH", "MEDIUM", "LOW")
				require.Equal(t, int32(90), locationPoint.SpatialConfig.Storage.Fillfactor)
				require.True(t, locationPoint.SpatialConfig.Storage.PadIndex)
				require.True(t, locationPoint.SpatialConfig.Storage.AllowRowLocks)
				require.True(t, locationPoint.SpatialConfig.Storage.AllowPageLocks)

				boundaryPolygon := requireSpatialIndex(t, table, "idx_boundary_polygon", "boundary_polygon", "GEOMETRY")
				require.Equal(t, int32(64), boundaryPolygon.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, boundaryPolygon, 0, 0, 1000, 1000)
				requireGridLevels(t, boundaryPolygon, "LOW", "LOW", "HIGH", "HIGH")

				geoLocation := requireSpatialIndex(t, table, "idx_geo_location", "geo_location", "GEOGRAPHY")
				require.Equal(t, int32(16), geoLocation.SpatialConfig.Tessellation.CellsPerObject)
				requireGridLevels(t, geoLocation, "LOW", "MEDIUM", "HIGH", "MEDIUM")
				require.Equal(t, int32(85), geoLocation.SpatialConfig.Storage.Fillfactor)
				require.False(t, geoLocation.SpatialConfig.Storage.PadIndex)
				require.False(t, geoLocation.SpatialConfig.Storage.AllowRowLocks)
				require.False(t, geoLocation.SpatialConfig.Storage.AllowPageLocks)

				routeLine := requireSpatialIndex(t, table, "idx_route_line", "route_line", "GEOGRAPHY")
				require.Equal(t, int32(128), routeLine.SpatialConfig.Tessellation.CellsPerObject)
				requireGridLevels(t, routeLine, "HIGH", "HIGH", "HIGH", "HIGH")
			},
		},
		{
			name: "configuration_bounds",
			setupSQL: `
CREATE TABLE dbo.spatial_edge_cases (
    id INT IDENTITY(1,1) PRIMARY KEY,
    min_config_geom GEOMETRY NOT NULL,
    max_config_geom GEOMETRY NOT NULL,
    min_config_geog GEOGRAPHY NOT NULL,
    max_config_geog GEOGRAPHY NOT NULL
);
GO

CREATE SPATIAL INDEX idx_min_geom ON dbo.spatial_edge_cases(min_config_geom)
USING GEOMETRY_GRID
WITH (
    BOUNDING_BOX = (0, 0, 1, 1),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 1
);
GO

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

CREATE SPATIAL INDEX idx_min_geog ON dbo.spatial_edge_cases(min_config_geog)
USING GEOGRAPHY_GRID
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = LOW, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 1
);
GO

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
				table := requireTable(t, metadata, "dbo", "spatial_edge_cases")
				require.Len(t, spatialIndexes(table), 4)

				minGeom := requireSpatialIndex(t, table, "idx_min_geom", "min_config_geom", "GEOMETRY")
				require.Equal(t, int32(1), minGeom.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, minGeom, 0, 0, 1, 1)
				requireGridLevels(t, minGeom, "LOW", "LOW", "LOW", "LOW")

				maxGeom := requireSpatialIndex(t, table, "idx_max_geom", "max_config_geom", "GEOMETRY")
				require.Equal(t, int32(8192), maxGeom.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, maxGeom, -1000000, -1000000, 1000000, 1000000)
				requireGridLevels(t, maxGeom, "HIGH", "HIGH", "HIGH", "HIGH")
				require.Equal(t, int32(100), maxGeom.SpatialConfig.Storage.Fillfactor)
				require.True(t, maxGeom.SpatialConfig.Storage.PadIndex)
				require.True(t, maxGeom.SpatialConfig.Storage.AllowRowLocks)
				require.True(t, maxGeom.SpatialConfig.Storage.AllowPageLocks)

				minGeog := requireSpatialIndex(t, table, "idx_min_geog", "min_config_geog", "GEOGRAPHY")
				require.Equal(t, int32(1), minGeog.SpatialConfig.Tessellation.CellsPerObject)
				requireGridLevels(t, minGeog, "LOW", "LOW", "LOW", "LOW")

				maxGeog := requireSpatialIndex(t, table, "idx_max_geog", "max_config_geog", "GEOGRAPHY")
				require.Equal(t, int32(8192), maxGeog.SpatialConfig.Tessellation.CellsPerObject)
				requireGridLevels(t, maxGeog, "HIGH", "HIGH", "HIGH", "HIGH")
				require.Equal(t, int32(50), maxGeog.SpatialConfig.Storage.Fillfactor)
			},
		},
		{
			name: "mixed_index_types",
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

CREATE INDEX idx_name ON dbo.mixed_indexes(name);
CREATE INDEX idx_status_created ON dbo.mixed_indexes(status, created_at);
CREATE UNIQUE INDEX idx_name_status ON dbo.mixed_indexes(name, status) WHERE status IS NOT NULL;
GO

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

CREATE NONCLUSTERED COLUMNSTORE INDEX idx_columnstore ON dbo.mixed_indexes(id, name, status);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				table := requireTable(t, metadata, "dbo", "mixed_indexes")

				// Spatial indexes must not disturb how sync classifies the
				// other index kinds on the same table.
				counts := make(map[string]int)
				for _, index := range table.Indexes {
					switch {
					case index.Primary:
						counts["PRIMARY"]++
					case index.Type == "SPATIAL":
						counts["SPATIAL"]++
					case index.Type == "NONCLUSTERED COLUMNSTORE":
						counts["COLUMNSTORE"]++
					case index.Unique:
						counts["UNIQUE"]++
					default:
						counts["REGULAR"]++
					}
				}
				require.Equal(t, map[string]int{
					"PRIMARY":     1,
					"UNIQUE":      2,
					"REGULAR":     2,
					"SPATIAL":     2,
					"COLUMNSTORE": 1,
				}, counts)

				location := requireSpatialIndex(t, table, "idx_location", "location", "GEOMETRY")
				require.Equal(t, int32(16), location.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, location, -100, -100, 100, 100)
				requireGridLevels(t, location, "MEDIUM", "MEDIUM", "MEDIUM", "MEDIUM")

				area := requireSpatialIndex(t, table, "idx_area", "area", "GEOGRAPHY")
				require.Equal(t, int32(32), area.SpatialConfig.Tessellation.CellsPerObject)
				requireGridLevels(t, area, "HIGH", "LOW", "HIGH", "LOW")
			},
		},
		{
			name: "computed_columns",
			setupSQL: `
CREATE TABLE dbo.spatial_computed (
    id INT IDENTITY(1,1) PRIMARY KEY,
    shape GEOMETRY NOT NULL,
    area AS shape.STArea() PERSISTED,
    perimeter AS shape.STLength() PERSISTED,
    center_x AS shape.STCentroid().STX PERSISTED,
    center_y AS shape.STCentroid().STY PERSISTED,
    as_text AS shape.STAsText() PERSISTED
);
GO

CREATE SPATIAL INDEX idx_shape ON dbo.spatial_computed(shape)
USING GEOMETRY_GRID
WITH (
    BOUNDING_BOX = (-500, -500, 500, 500),
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = MEDIUM, LEVEL_3 = HIGH, LEVEL_4 = MEDIUM),
    CELLS_PER_OBJECT = 24,
    FILLFACTOR = 75
);
GO

CREATE INDEX idx_area ON dbo.spatial_computed(area);
CREATE INDEX idx_center ON dbo.spatial_computed(center_x, center_y);
GO
`,
			validate: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				table := requireTable(t, metadata, "dbo", "spatial_computed")

				columns := make(map[string]*storepb.ColumnMetadata)
				for _, column := range table.Columns {
					columns[column.Name] = column
				}
				for _, name := range []string{"area", "perimeter", "center_x", "center_y", "as_text"} {
					require.Contains(t, columns, name)
				}

				shape := requireSpatialIndex(t, table, "idx_shape", "shape", "GEOMETRY")
				require.Equal(t, int32(24), shape.SpatialConfig.Tessellation.CellsPerObject)
				requireBoundingBox(t, shape, -500, -500, 500, 500)
				requireGridLevels(t, shape, "LOW", "MEDIUM", "HIGH", "MEDIUM")
				require.Equal(t, int32(75), shape.SpatialConfig.Storage.Fillfactor)

				regular := 0
				for _, index := range table.Indexes {
					if index.Type != "SPATIAL" && !index.Primary && !index.Unique {
						regular++
					}
				}
				require.Equal(t, 2, regular, "indexes over computed columns should survive the sync")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			driver := newSyncTestDatabase(ctx, t, container)
			executeBatches(ctx, t, driver, tc.setupSQL)

			metadata, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err)

			tc.validate(t, metadata)
		})
	}
}

// TestSyncSpatialIndexIsRepeatable guards the spatial branch of the sync against
// per-connection state: the second sync of an unchanged database must report the
// same indexes as the first.
func TestSyncSpatialIndexIsRepeatable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)

	driver := newSyncTestDatabase(ctx, t, container)
	executeBatches(ctx, t, driver, `
CREATE SCHEMA spatial_test;
GO

CREATE TABLE spatial_test.mixed_spatial (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(100) NOT NULL,
    geo_point GEOGRAPHY NOT NULL,
    geo_line GEOGRAPHY NOT NULL,
    created_at DATETIME2 DEFAULT GETDATE()
);
GO

CREATE SPATIAL INDEX idx_geo_point ON spatial_test.mixed_spatial(geo_point)
USING GEOGRAPHY_GRID
WITH (
    GRIDS = (LEVEL_1 = MEDIUM, LEVEL_2 = HIGH, LEVEL_3 = MEDIUM, LEVEL_4 = LOW),
    CELLS_PER_OBJECT = 16
);
GO

CREATE SPATIAL INDEX idx_geo_line ON spatial_test.mixed_spatial(geo_line)
USING GEOGRAPHY_GRID
WITH (
    GRIDS = (LEVEL_1 = LOW, LEVEL_2 = LOW, LEVEL_3 = MEDIUM, LEVEL_4 = HIGH),
    CELLS_PER_OBJECT = 8
);
GO
`)

	first, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	table := requireTable(t, first, "spatial_test", "mixed_spatial")
	require.Len(t, spatialIndexes(table), 2)

	geoPoint := requireSpatialIndex(t, table, "idx_geo_point", "geo_point", "GEOGRAPHY")
	require.Equal(t, int32(16), geoPoint.SpatialConfig.Tessellation.CellsPerObject)
	requireGridLevels(t, geoPoint, "MEDIUM", "HIGH", "MEDIUM", "LOW")

	geoLine := requireSpatialIndex(t, table, "idx_geo_line", "geo_line", "GEOGRAPHY")
	require.Equal(t, int32(8), geoLine.SpatialConfig.Tessellation.CellsPerObject)
	requireGridLevels(t, geoLine, "LOW", "LOW", "MEDIUM", "HIGH")

	second, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	// Keyed by name because sync assembles indexes out of a map, so their
	// order within a table is not itself part of the contract.
	byName := func(metadata *storepb.DatabaseSchemaMetadata) map[string]*storepb.IndexMetadata {
		indexes := make(map[string]*storepb.IndexMetadata)
		for _, index := range spatialIndexes(requireTable(t, metadata, "spatial_test", "mixed_spatial")) {
			indexes[index.Name] = index
		}
		return indexes
	}
	require.Empty(t, cmp.Diff(byName(first), byName(second), protocmp.Transform()))
}

func spatialIndexes(table *storepb.TableMetadata) []*storepb.IndexMetadata {
	var indexes []*storepb.IndexMetadata
	for _, index := range table.Indexes {
		if index.Type == "SPATIAL" {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

// requireSpatialIndex asserts the properties every synced spatial index carries
// and returns it for the case-specific configuration checks. dataType is
// GEOMETRY or GEOGRAPHY; only GEOMETRY indexes carry a bounding box.
func requireSpatialIndex(t *testing.T, table *storepb.TableMetadata, name, column, dataType string) *storepb.IndexMetadata {
	t.Helper()

	var index *storepb.IndexMetadata
	for _, candidate := range table.Indexes {
		if candidate.Name == name {
			index = candidate
			break
		}
	}
	require.NotNilf(t, index, "spatial index %s not synced", name)

	require.Equal(t, "SPATIAL", index.Type)
	require.False(t, index.Unique)
	require.False(t, index.Primary)
	require.Equal(t, []string{column}, index.Expressions)
	require.Equal(t, []bool{false}, index.Descending, "spatial indexes have no direction")

	require.NotNil(t, index.SpatialConfig)
	require.Equal(t, "SPATIAL", index.SpatialConfig.Method)
	require.NotNil(t, index.SpatialConfig.Tessellation)
	require.Equal(t, dataType+"_GRID", index.SpatialConfig.Tessellation.Scheme)
	require.NotNil(t, index.SpatialConfig.Storage)
	require.NotNil(t, index.SpatialConfig.Dimensional)
	require.Equal(t, dataType, index.SpatialConfig.Dimensional.DataType)
	require.Equal(t, int32(2), index.SpatialConfig.Dimensional.Dimensions, "SQL Server spatial is always 2D")

	if dataType == "GEOGRAPHY" {
		require.Nil(t, index.SpatialConfig.Tessellation.BoundingBox, "GEOGRAPHY indexes carry no bounding box")
	}
	return index
}

func requireBoundingBox(t *testing.T, index *storepb.IndexMetadata, xmin, ymin, xmax, ymax float64) {
	t.Helper()

	bbox := index.SpatialConfig.Tessellation.BoundingBox
	require.NotNil(t, bbox)
	require.Equal(t, xmin, bbox.Xmin)
	require.Equal(t, ymin, bbox.Ymin)
	require.Equal(t, xmax, bbox.Xmax)
	require.Equal(t, ymax, bbox.Ymax)
}

func requireGridLevels(t *testing.T, index *storepb.IndexMetadata, level1, level2, level3, level4 string) {
	t.Helper()

	densities := make(map[int32]string)
	for _, level := range index.SpatialConfig.Tessellation.GridLevels {
		densities[level.Level] = level.Density
	}
	require.Equal(t, map[int32]string{1: level1, 2: level2, 3: level3, 4: level4}, densities)
}

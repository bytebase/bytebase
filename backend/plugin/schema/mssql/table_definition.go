package mssql

import (
	"fmt"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
)

// GetTableDefinition renders one table as the T-SQL that recreates it.
//
// This is the legacy Bytebase renderer, not omni's. SQL Server has no
// single-table definition in omni's catalog at the pinned version: catalog
// exports DatabaseDefinition and the view, function and procedure renderers,
// but no table renderer. GetDatabaseDefinition no longer uses this code — it
// delegates to the catalog — so the table renderer and the spatial-index
// helpers below are the only rendering this package still owns.
func GetTableDefinition(schemaName string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	writeTable(&buf, schemaName, table)
	return buf.String(), nil
}

func writeTable(out *strings.Builder, schemaName string, table *metadatapb.TableMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE TABLE [%s].[%s] (\n", schemaName, table.Name)
	for i, column := range table.Columns {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		writeColumn(out, column)
	}

	for _, key := range table.Indexes {
		if !key.IsConstraint {
			continue
		}

		_, _ = out.WriteString(",\n")
		writeKey(out, key)
	}

	for _, fk := range table.ForeignKeys {
		_, _ = out.WriteString(",\n")
		writeForeignKey(out, fk)
	}

	for _, check := range table.CheckConstraints {
		_, _ = out.WriteString(",\n")
		writeCheck(out, check)
	}
	_, _ = fmt.Fprint(out, "\n);\n\n")

	for _, index := range table.Indexes {
		if index.IsConstraint {
			continue
		}
		writeIndex(out, schemaName, table.Name, index)
	}
}

func writeClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *metadatapb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE CLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s];\n\n", index.Name, schemaName, tableName)
}

func writeNonClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *metadatapb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE NONCLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeSpatialIndex(out *strings.Builder, schemaName string, tableName string, index *metadatapb.IndexMetadata) {
	// Use the enhanced spatial index DDL generation
	spatialDDL := generateSpatialIndexDefinition(index, schemaName, tableName)
	_, _ = out.WriteString(spatialDDL)
	_, _ = out.WriteString(";\n\n")
}

func writeNormalIndex(out *strings.Builder, schemaName string, tableName string, index *metadatapb.IndexMetadata) {
	_, _ = out.WriteString("CREATE")
	if index.Unique {
		_, _ = out.WriteString(" UNIQUE")
	}
	if index.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", index.Type)
	}
	_, _ = fmt.Fprintf(out, " INDEX [%s] ON\n[%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
		if i < len(index.Descending) && index.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeIndex(out *strings.Builder, schemaName string, tableName string, index *metadatapb.IndexMetadata) {
	switch strings.ToUpper(index.Type) {
	case "CLUSTERED COLUMNSTORE":
		writeClusteredColumnStoreIndex(out, schemaName, tableName, index)
	case "NONCLUSTERED COLUMNSTORE":
		writeNonClusteredColumnStoreIndex(out, schemaName, tableName, index)
	case "SPATIAL":
		writeSpatialIndex(out, schemaName, tableName, index)
	default:
		writeNormalIndex(out, schemaName, tableName, index)
	}
}

func writeCheck(out *strings.Builder, check *metadatapb.CheckConstraintMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] CHECK %s", check.Name, check.Expression)
}

func writeForeignKey(out *strings.Builder, fk *metadatapb.ForeignKeyMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] FOREIGN KEY (", fk.Name)
	for i, column := range fk.Columns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = fmt.Fprintf(out, ") REFERENCES [%s].[%s] (", fk.ReferencedSchema, fk.ReferencedTable)
	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = out.WriteString(")")
	if fk.OnDelete != "" {
		_, _ = fmt.Fprintf(out, " ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		_, _ = fmt.Fprintf(out, " ON UPDATE %s", fk.OnUpdate)
	}
}

func writeKey(out *strings.Builder, key *metadatapb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s]", key.Name)
	if key.Primary {
		_, _ = out.WriteString(" PRIMARY KEY")
	} else if key.Unique {
		_, _ = out.WriteString(" UNIQUE")
	}

	if key.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", key.Type)
	}
	_, _ = out.WriteString(" (")
	for i, column := range key.Expressions {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
		if i < len(key.Descending) && key.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString(")")
}

func writeColumn(out *strings.Builder, column *metadatapb.ColumnMetadata) {
	_, _ = fmt.Fprintf(out, "    [%s] %s", column.Name, column.Type)
	if column.IsIdentity {
		_, _ = fmt.Fprintf(out, " IDENTITY(%d,%d)", column.IdentitySeed, column.IdentityIncrement)
	}
	if column.Collation != "" {
		_, _ = fmt.Fprintf(out, " COLLATE %s", column.Collation)
	}
	if column.GetDefault() != "" {
		_, _ = fmt.Fprintf(out, " DEFAULT %s", column.GetDefault())
	}
	if !column.Nullable {
		_, _ = out.WriteString(" NOT NULL")
	}
}

func generateSpatialIndexDefinition(index *metadatapb.IndexMetadata, schemaName string, tableName string) string {
	var buf strings.Builder

	// Build the CREATE SPATIAL INDEX statement
	buf.WriteString("CREATE SPATIAL INDEX [")
	buf.WriteString(index.Name)
	buf.WriteString("] ON [")
	buf.WriteString(schemaName)
	buf.WriteString("].[")
	buf.WriteString(tableName)
	buf.WriteString("] (")

	// Add column expressions
	for i, expr := range index.Expressions {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("[")
		buf.WriteString(expr)
		buf.WriteString("]")
	}
	buf.WriteString(")")

	// Check if spatial configuration exists
	if index.SpatialConfig == nil || index.SpatialConfig.Tessellation == nil {
		return buf.String()
	}

	tessellation := index.SpatialConfig.Tessellation

	// Add USING clause for tessellation scheme
	if tessellation.Scheme != "" {
		buf.WriteString("\nUSING ")
		buf.WriteString(tessellation.Scheme)
	}

	// Build WITH clause parameters
	withParams := []string{}

	// Add tessellation parameters
	withParams = append(withParams, buildTessellationParams(tessellation)...)

	// Add storage parameters
	if index.SpatialConfig.Storage != nil {
		withParams = append(withParams, buildStorageParams(index.SpatialConfig.Storage)...)
	}

	// Add WITH clause if we have parameters
	if len(withParams) > 0 {
		buf.WriteString("\nWITH (\n    ")
		buf.WriteString(strings.Join(withParams, ",\n    "))
		buf.WriteString("\n)")
	}

	return buf.String()
}

func buildTessellationParams(tessellation *metadatapb.TessellationConfig) []string {
	params := []string{}

	// BOUNDING_BOX for GEOMETRY indexes
	if tessellation.BoundingBox != nil {
		bbox := tessellation.BoundingBox
		params = append(params, fmt.Sprintf("BOUNDING_BOX = (%g, %g, %g, %g)",
			bbox.Xmin, bbox.Ymin, bbox.Xmax, bbox.Ymax))
	}

	// GRIDS configuration
	if len(tessellation.GridLevels) > 0 {
		gridParts := []string{}
		for _, level := range tessellation.GridLevels {
			gridParts = append(gridParts, fmt.Sprintf("LEVEL_%d = %s", level.Level, level.Density))
		}
		if len(gridParts) > 0 {
			params = append(params, fmt.Sprintf("GRIDS = (%s)", strings.Join(gridParts, ", ")))
		}
	}

	// CELLS_PER_OBJECT
	if tessellation.CellsPerObject > 0 {
		params = append(params, fmt.Sprintf("CELLS_PER_OBJECT = %d", tessellation.CellsPerObject))
	}

	return params
}

func buildStorageParams(storage *metadatapb.StorageConfig) []string {
	params := []string{}

	// PAD_INDEX (defaults to OFF, so only output when ON)
	if storage.PadIndex {
		params = append(params, "PAD_INDEX = ON")
	}

	// FILLFACTOR
	if storage.Fillfactor > 0 {
		params = append(params, fmt.Sprintf("FILLFACTOR = %d", storage.Fillfactor))
	}

	// SORT_IN_TEMPDB
	if storage.SortInTempdb != "" {
		params = append(params, fmt.Sprintf("SORT_IN_TEMPDB = %s", storage.SortInTempdb))
	}

	// DROP_EXISTING
	if storage.DropExisting {
		params = append(params, "DROP_EXISTING = ON")
	}

	// ONLINE
	if storage.Online {
		params = append(params, "ONLINE = ON")
	}

	// ALLOW_ROW_LOCKS and ALLOW_PAGE_LOCKS (default to ON for spatial indexes)
	// Only output when they differ from the default (ON)
	if !storage.AllowRowLocks {
		params = append(params, "ALLOW_ROW_LOCKS = OFF")
	}

	if !storage.AllowPageLocks {
		params = append(params, "ALLOW_PAGE_LOCKS = OFF")
	}

	// MAXDOP
	if storage.Maxdop > 0 {
		params = append(params, fmt.Sprintf("MAXDOP = %d", storage.Maxdop))
	}

	// DATA_COMPRESSION
	if storage.DataCompression != "" && storage.DataCompression != "NONE" {
		params = append(params, fmt.Sprintf("DATA_COMPRESSION = %s", storage.DataCompression))
	}

	return params
}

# Unified Metadata+Config Design

## Overview

Unify the parallel Config and Metadata hierarchies into a single hierarchy where each level holds both metadata (proto) and config (catalog). This creates consistency with the existing `DatabaseMetadata` pattern and simplifies the API.

## Goals

Create a parallel structure at all levels:
- `DatabaseMetadata` → holds `DatabaseSchemaMetadata` + `DatabaseConfig`
- `SchemaMetadata` → holds `SchemaMetadata` + `SchemaCatalog`
- `TableMetadata` → holds `TableMetadata` + `TableCatalog`
- `ColumnMetadata` → holds `ColumnMetadata` + `ColumnCatalog`

## Structural Changes

### 1. SchemaMetadata (merged from SchemaMetadata + SchemaConfig)

**Before:**
- `SchemaMetadata`: holds metadata + separate table maps
- `SchemaConfig`: holds only `TableConfig` map

**After:**
```go
type SchemaMetadata struct {
    isObjectCaseSensitive    bool
    isDetailCaseSensitive    bool
    internalTables           map[string]*TableMetadata  // Single unified map
    internalExternalTable    map[string]*ExternalTableMetadata
    internalViews            map[string]*storepb.ViewMetadata
    internalMaterializedView map[string]*storepb.MaterializedViewMetadata
    internalProcedures       map[string]*storepb.ProcedureMetadata
    internalSequences        map[string]*storepb.SequenceMetadata
    internalPackages         map[string]*storepb.PackageMetadata

    proto  *storepb.SchemaMetadata
    config *storepb.SchemaCatalog  // Can be nil if no config
}
```

**Methods:**
- Remove: `GetTableConfig(name)` - functionality merged into `TableMetadata`
- Keep: `GetTable(name)`, `CreateTable()`, `DropTable()`, `RenameTable()`, all other existing methods
- New: `GetCatalog()` - returns `*storepb.SchemaCatalog` (may be nil)

### 2. TableMetadata (merged from TableMetadata + TableConfig)

**Before:**
- `TableMetadata`: holds metadata + `storepb.ColumnMetadata` map
- `TableConfig`: holds `Classification` + `ColumnCatalog` map

**After:**
```go
type TableMetadata struct {
    partitionOf           *TableMetadata
    isDetailCaseSensitive bool
    internalColumn        map[string]*ColumnMetadata  // Now wrapper type
    internalIndexes       map[string]*IndexMetadata

    proto  *storepb.TableMetadata
    config *storepb.TableCatalog  // Can be nil if no config
}
```

**Methods:**
- Remove: `GetColumnConfig(name)` - functionality merged into `ColumnMetadata`
- Modify: `GetColumn(name)` - returns `*ColumnMetadata` (wrapper, not proto directly)
- Modify: `CreateColumn()` - now takes optional `ColumnCatalog`, creates `ColumnMetadata` wrapper
- Keep: `DropColumn()`, `RenameColumn()`, all other existing methods
- New: `GetCatalog()` - returns `*storepb.TableCatalog` (may be nil)

### 3. ColumnMetadata (new wrapper)

**Before:** Using `*storepb.ColumnMetadata` directly everywhere

**After:**
```go
type ColumnMetadata struct {
    proto  *storepb.ColumnMetadata
    config *storepb.ColumnCatalog  // Can be nil if no config
}
```

**Methods:**
- `GetProto()` - returns `*storepb.ColumnMetadata`
- `GetCatalog()` - returns `*storepb.ColumnCatalog` (may be nil)

Callers use `col.GetProto().Name`, `col.GetCatalog().SemanticType`, etc.

### 4. DatabaseMetadata Changes

**Remove:**
- `configInternal map[string]*SchemaConfig` field
- `GetSchemaConfig(name) *SchemaConfig` method

**Modify:**
- `NewDatabaseMetadata()` - build unified `SchemaMetadata` with embedded catalogs
- `BuildDatabaseConfig()` - walk `SchemaMetadata` hierarchy to extract catalogs

### 5. Removed Types

- `SchemaConfig` struct - merged into `SchemaMetadata`
- `TableConfig` struct - merged into `TableMetadata`

## Constructor Changes

### NewDatabaseMetadata

**Logic:**
1. Loop through `metadata.Schemas` to build `SchemaMetadata`
2. For each schema, look up matching `config.Schemas[schemaName]` to get `SchemaCatalog`
3. For each table in schema, look up matching `TableCatalog` from `SchemaCatalog`
4. For each column in table, create `ColumnMetadata` wrapper with matching `ColumnCatalog`
5. Store everything in single unified maps

**Changes:**
- No more separate `configInternal` map building
- Build `SchemaMetadata` with embedded `SchemaCatalog`
- Build `TableMetadata` with embedded `TableCatalog`
- Build `ColumnMetadata` wrappers with embedded `ColumnCatalog`

## Nil Config Handling

All catalog fields can be nil when no config exists:
- `SchemaMetadata.config *storepb.SchemaCatalog` - can be nil
- `TableMetadata.config *storepb.TableCatalog` - can be nil
- `ColumnMetadata.config *storepb.ColumnCatalog` - can be nil

`GetCatalog()` methods return nil directly - callers must check:
```go
if catalog := table.GetCatalog(); catalog != nil {
    classification := catalog.Classification
}
```

## Migration Impact

### Files to Update

**Core implementation:**
- `backend/store/model/database.go` - main changes

**Callers to migrate:**
- All files calling `GetSchemaConfig()` → change to `schema.GetCatalog()`
- All files calling `GetTableConfig()` → change to `table.GetCatalog()`
- All files calling `GetColumnConfig()` → change to `column.GetCatalog()`
- All files directly accessing `*storepb.ColumnMetadata` → use `ColumnMetadata` wrapper or call `.GetProto()`

### Migration Pattern

**Before:**
```go
schemaConfig := dbMeta.GetSchemaConfig(schemaName)
tableConfig := schemaConfig.GetTableConfig(tableName)
columnConfig := tableConfig.GetColumnConfig(columnName)
semanticType := columnConfig.SemanticType
```

**After:**
```go
schema := dbMeta.GetSchemaMetadata(schemaName)
table := schema.GetTable(tableName)
column := table.GetColumn(columnName)
if catalog := column.GetCatalog(); catalog != nil {
    semanticType := catalog.SemanticType
}
```

## Benefits

1. **Consistency**: All levels follow the same pattern (metadata + config wrapper)
2. **Simplicity**: Single map lookup instead of parallel map lookups
3. **Type safety**: `ColumnMetadata` wrapper prevents direct proto mutation
4. **Clarity**: Clear separation between proto data and config data within each type
5. **Matches existing pattern**: Extends the `DatabaseMetadata` pattern downward

## Proto Impact

No proto changes needed. We keep existing proto message names:
- `SchemaCatalog` (not `SchemaConfig`)
- `TableCatalog` (not `TableConfig`)
- `ColumnCatalog` (not `ColumnConfig`)

This avoids breaking changes in the proto layer.

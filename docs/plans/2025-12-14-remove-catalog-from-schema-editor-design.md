# Remove Catalog from Schema Editor

## Context

Database catalog stores metadata annotations (semantic types, labels, classifications) for database objects. Investigation revealed that catalog is not used in Schema Editor's core DDL generation functionality:

- The `DiffMetadata` API only accepts `DatabaseMetadata`, not catalog
- `SchemaEditorDrawer.vue` fetches catalog but discards it when generating DDL
- Recent commit `06b8b885a4` already removed catalog fields from `DiffMetadataRequest`

## Goal

Complete removal of catalog-related code from Schema Editor to simplify the codebase and eliminate unused data fetching.

## Design

### 1. Core Type Changes

**File:** `frontend/src/components/SchemaEditorLite/types.ts`

Remove catalog from `EditTarget`:
```typescript
export type EditTarget = {
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  baselineMetadata: DatabaseMetadata;
  // REMOVE: catalog: DatabaseCatalog;
  // REMOVE: baselineCatalog: DatabaseCatalog;
};
```

Remove `DatabaseCatalog` import.

### 2. Context and State Management

**File:** `frontend/src/components/SchemaEditorLite/context/config.ts`

Remove catalog reactive maps and CRUD methods:
- `databaseCatalog`, `tableCatalog`, `columnCatalog` maps
- `getColumnCatalog()`, `upsertColumnCatalog()`, `removeColumnCatalog()`
- `getTableCatalog()`, `upsertTableCatalog()`, `removeTableCatalog()`
- `getSchemaCatalog()`, `upsertSchemaCatalog()`, `removeSchemaCatalog()`
- `getDatabaseCatalog()`, `upsertDatabaseCatalog()`

**File:** `frontend/src/components/SchemaEditorLite/algorithm/diff-merge.ts`

Remove catalog merging and diffing:
- `mergeSchemaCatalog()`, `mergeTableCatalog()`, `mergeColumnCatalog()`, `diffColumnCatalog()`
- Catalog-based "updated" status detection logic

**File:** `frontend/src/components/SchemaEditorLite/algorithm/apply.ts`

Remove catalog from return value:
```typescript
// Before: return { metadata, catalog }
// After: return { metadata }
```

### 3. UI Component Removal

**Delete components:**
- `frontend/src/components/SchemaEditorLite/Panels/TableColumnEditor/components/SemanticTypeCell.vue`
- `frontend/src/components/SchemaEditorLite/Panels/TableColumnEditor/components/LabelsCell.vue`

**File:** `frontend/src/components/SchemaEditorLite/Panels/TableColumnEditor/TableColumnEditor.vue`

Remove:
- Catalog update logic when column names change (lines 302-318)
- Semantic type and labels column definitions
- Catalog-related imports

### 4. Drawer and Integration Points

**File:** `frontend/src/components/Plan/components/StatementSection/SchemaEditorDrawer.vue`

Simplify `prepareDatabaseMetadata()`:
```typescript
// Before:
const [metadata, catalog] = await Promise.all([
  dbSchemaStore.getOrFetchDatabaseMetadata(...),
  dbCatalogStore.getOrFetchDatabaseCatalog(...),
]);

state.targets = [{
  database,
  metadata: cloneDeep(metadata),
  baselineMetadata: metadata,
  catalog: cloneDeep(catalog),
  baselineCatalog: catalog,
}];

// After:
const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata(...);

state.targets = [{
  database,
  metadata: cloneDeep(metadata),
  baselineMetadata: metadata,
}];
```

Simplify `handleInsertSQL()`:
```typescript
// Before:
const { metadata, catalog } = applyMetadataEdit(...)

// After:
const { metadata } = applyMetadataEdit(...)
```

Remove import of `useDatabaseCatalogV1Store`.

### 5. Testing and Verification

**Manual testing:**
- Open Schema Editor drawer
- Create/modify/delete tables and columns
- Generate DDL and verify correctness
- Test with multiple database engines

**What should still work:**
- All metadata-based schema editing
- DDL generation via `generateDiffDDL()`
- Diff-merge algorithm for change detection
- Multi-database scenarios

**What will be removed:**
- Semantic type editing in table editor
- Labels editing in table editor
- Catalog state management

**Automated checks:**
```bash
pnpm --dir frontend biome:check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

## Impact

**Benefits:**
- Simpler codebase with less state to manage
- Faster Schema Editor loading (one less API call)
- Clearer separation: catalog is not part of schema structure

**Trade-offs:**
- Cannot edit semantic types or labels in Schema Editor
- These fields may still be editable elsewhere if catalog is used in other parts of the system

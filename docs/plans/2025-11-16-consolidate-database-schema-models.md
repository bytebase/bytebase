# Consolidate Database Schema Models Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate `DatabaseState` and `DatabaseSchema` by adding mutation methods directly to `model.DatabaseMetadata`, eliminating the need for a separate mutable wrapper and allowing all code to use a unified schema API.

**Architecture:** Add mutation methods (CreateTable, DropTable, CreateColumn, etc.) directly to `model.DatabaseMetadata` and related types. Advisors will use `proto.Clone` to create separate origin (immutable) and final (mutable) instances of `DatabaseMetadata`. No separate `DatabaseState` wrapper needed - walk-through code mutates `DatabaseMetadata` directly.

**Tech Stack:** Go, Protocol Buffers (storepb), ANTLR parsers (MySQL, PostgreSQL, TiDB)

---

## Task 1: Add CreateTable Method to SchemaMetadata

**Files:**
- Modify: `backend/store/model/database.go` (add to SchemaMetadata)
- Test: `backend/store/model/database_test.go`

**Step 1: Write the failing test**

Create or append to `backend/store/model/database_test.go`:

```go
package model

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/stretchr/testify/require"
)

func TestSchemaMetadata_CreateTable(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")

	// Create a new table
	tableMeta, err := schemaMeta.CreateTable("products")

	require.Nil(t, err)
	require.NotNil(t, tableMeta)
	require.Equal(t, "products", tableMeta.GetProto().Name)

	// Verify table is now accessible via GetTable
	retrieved := schemaMeta.GetTable("products")
	require.NotNil(t, retrieved)
	require.Equal(t, "products", retrieved.GetProto().Name)
}

func TestSchemaMetadata_CreateTable_AlreadyExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")

	// Try to create table that already exists
	_, err := schemaMeta.CreateTable("users")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "already exists")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestSchemaMetadata_CreateTable"`

Expected: FAIL with "schemaMeta.CreateTable undefined"

**Step 3: Add CreateTable method to SchemaMetadata**

Add to `backend/store/model/database.go` in the SchemaMetadata section:

```go
// CreateTable creates a new table in the schema.
// Returns the created TableMetadata or an error if the table already exists.
func (s *SchemaMetadata) CreateTable(tableName string) (*TableMetadata, error) {
	// Check if table already exists
	if s.GetTable(tableName) != nil {
		return nil, fmt.Errorf("table %q already exists in schema %q", tableName, s.proto.Name)
	}

	// Create new table proto
	newTableProto := &storepb.TableMetadata{
		Name:    tableName,
		Columns: []*storepb.ColumnMetadata{},
		Indexes: []*storepb.IndexMetadata{},
	}

	// Add to proto's table list
	s.proto.Tables = append(s.proto.Tables, newTableProto)

	// Create TableMetadata wrapper
	tableMeta := &TableMetadata{
		isDetailCaseSensitive: s.isDetailCaseSensitive,
		internalColumn:        make(map[string]*storepb.ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		columns:               []*storepb.ColumnMetadata{},
		proto:                 newTableProto,
	}

	// Add to internal map
	var tableID string
	if s.isObjectCaseSensitive {
		tableID = tableName
	} else {
		tableID = strings.ToLower(tableName)
	}
	s.internalTables[tableID] = tableMeta

	return tableMeta, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestSchemaMetadata_CreateTable"`

Expected: PASS

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/model/database.go`

Expected: No issues

**Step 6: Commit**

```bash
git add backend/store/model/database.go backend/store/model/database_test.go
git commit -m "feat: add CreateTable method to SchemaMetadata"
```

---

## Task 2: Add DropTable Method to SchemaMetadata

**Files:**
- Modify: `backend/store/model/database.go`
- Test: `backend/store/model/database_test.go`

**Step 1: Write the failing test**

Add to `backend/store/model/database_test.go`:

```go
func TestSchemaMetadata_DropTable(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
					{Name: "products"},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")

	// Drop table
	err := schemaMeta.DropTable("users")

	require.Nil(t, err)

	// Verify table is gone
	retrieved := schemaMeta.GetTable("users")
	require.Nil(t, retrieved)

	// Verify other table still exists
	retrieved = schemaMeta.GetTable("products")
	require.NotNil(t, retrieved)
}

func TestSchemaMetadata_DropTable_NotExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")

	// Try to drop non-existent table
	err := schemaMeta.DropTable("nonexistent")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "does not exist")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestSchemaMetadata_DropTable"`

Expected: FAIL with "schemaMeta.DropTable undefined"

**Step 3: Add DropTable method**

Add to `backend/store/model/database.go`:

```go
// DropTable drops a table from the schema.
// Returns an error if the table does not exist.
func (s *SchemaMetadata) DropTable(tableName string) error {
	// Check if table exists
	if s.GetTable(tableName) == nil {
		return fmt.Errorf("table %q does not exist in schema %q", tableName, s.proto.Name)
	}

	// Remove from internal map
	var tableID string
	if s.isObjectCaseSensitive {
		tableID = tableName
	} else {
		tableID = strings.ToLower(tableName)
	}
	delete(s.internalTables, tableID)

	// Remove from proto's table list
	newTables := make([]*storepb.TableMetadata, 0, len(s.proto.Tables)-1)
	for _, table := range s.proto.Tables {
		if s.isObjectCaseSensitive {
			if table.Name != tableName {
				newTables = append(newTables, table)
			}
		} else {
			if !strings.EqualFold(table.Name, tableName) {
				newTables = append(newTables, table)
			}
		}
	}
	s.proto.Tables = newTables

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestSchemaMetadata_DropTable"`

Expected: PASS

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/model/database.go`

Expected: No issues

**Step 6: Commit**

```bash
git add backend/store/model/database.go backend/store/model/database_test.go
git commit -m "feat: add DropTable method to SchemaMetadata"
```

---

## Task 3: Add CreateColumn Method to TableMetadata

**Files:**
- Modify: `backend/store/model/database.go`
- Test: `backend/store/model/database_test.go`

**Step 1: Write the failing test**

Add to `backend/store/model/database_test.go`:

```go
func TestTableMetadata_CreateColumn(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")
	tableMeta := schemaMeta.GetTable("users")

	// Create a new column
	columnProto := &storepb.ColumnMetadata{
		Name:     "email",
		Type:     "varchar",
		Nullable: true,
	}
	err := tableMeta.CreateColumn(columnProto)

	require.Nil(t, err)

	// Verify column is now accessible
	retrieved := tableMeta.GetColumn("email")
	require.NotNil(t, retrieved)
	require.Equal(t, "email", retrieved.Name)
	require.Equal(t, "varchar", retrieved.Type)
}

func TestTableMetadata_CreateColumn_AlreadyExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")
	tableMeta := schemaMeta.GetTable("users")

	// Try to create column that already exists
	columnProto := &storepb.ColumnMetadata{
		Name: "id",
		Type: "bigint",
	}
	err := tableMeta.CreateColumn(columnProto)

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "already exists")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestTableMetadata_CreateColumn"`

Expected: FAIL with "tableMeta.CreateColumn undefined"

**Step 3: Add CreateColumn method**

Add to `backend/store/model/database.go`:

```go
// CreateColumn creates a new column in the table.
// Returns an error if the column already exists.
func (t *TableMetadata) CreateColumn(columnProto *storepb.ColumnMetadata) error {
	// Check if column already exists
	if t.GetColumn(columnProto.Name) != nil {
		return fmt.Errorf("column %q already exists in table %q", columnProto.Name, t.proto.Name)
	}

	// Add to proto's column list
	t.proto.Columns = append(t.proto.Columns, columnProto)

	// Add to internal map
	var columnID string
	if t.isDetailCaseSensitive {
		columnID = columnProto.Name
	} else {
		columnID = strings.ToLower(columnProto.Name)
	}
	t.internalColumn[columnID] = columnProto

	// Add to columns slice
	t.columns = append(t.columns, columnProto)

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestTableMetadata_CreateColumn"`

Expected: PASS

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/model/database.go`

Expected: No issues

**Step 6: Commit**

```bash
git add backend/store/model/database.go backend/store/model/database_test.go
git commit -m "feat: add CreateColumn method to TableMetadata"
```

---

## Task 4: Add DropColumn Method to TableMetadata

**Files:**
- Modify: `backend/store/model/database.go`
- Test: `backend/store/model/database_test.go`

**Step 1: Write the failing test**

Add to `backend/store/model/database_test.go`:

```go
func TestTableMetadata_DropColumn(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "email", Type: "varchar"},
							{Name: "name", Type: "varchar"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")
	tableMeta := schemaMeta.GetTable("users")

	// Drop column
	err := tableMeta.DropColumn("email")

	require.Nil(t, err)

	// Verify column is gone
	retrieved := tableMeta.GetColumn("email")
	require.Nil(t, retrieved)

	// Verify other columns still exist
	require.NotNil(t, tableMeta.GetColumn("id"))
	require.NotNil(t, tableMeta.GetColumn("name"))
}

func TestTableMetadata_DropColumn_NotExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	schema := NewDatabaseSchema(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	dbMeta := schema.GetDatabaseMetadata()
	schemaMeta := dbMeta.GetSchema("public")
	tableMeta := schemaMeta.GetTable("users")

	// Try to drop non-existent column
	err := tableMeta.DropColumn("nonexistent")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "does not exist")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestTableMetadata_DropColumn"`

Expected: FAIL with "tableMeta.DropColumn undefined"

**Step 3: Add DropColumn method**

Add to `backend/store/model/database.go`:

```go
// DropColumn drops a column from the table.
// Returns an error if the column does not exist.
func (t *TableMetadata) DropColumn(columnName string) error {
	// Check if column exists
	if t.GetColumn(columnName) == nil {
		return fmt.Errorf("column %q does not exist in table %q", columnName, t.proto.Name)
	}

	// Remove from internal map
	var columnID string
	if t.isDetailCaseSensitive {
		columnID = columnName
	} else {
		columnID = strings.ToLower(columnName)
	}
	delete(t.internalColumn, columnID)

	// Remove from proto's column list
	newColumns := make([]*storepb.ColumnMetadata, 0, len(t.proto.Columns)-1)
	for _, column := range t.proto.Columns {
		if t.isDetailCaseSensitive {
			if column.Name != columnName {
				newColumns = append(newColumns, column)
			}
		} else {
			if !strings.EqualFold(column.Name, columnName) {
				newColumns = append(newColumns, column)
			}
		}
	}
	t.proto.Columns = newColumns

	// Rebuild columns slice
	t.columns = newColumns

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/model -run "TestTableMetadata_DropColumn"`

Expected: PASS

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/model/database.go`

Expected: No issues

**Step 6: Commit**

```bash
git add backend/store/model/database.go backend/store/model/database_test.go
git commit -m "feat: add DropColumn method to TableMetadata"
```

---

## Task 5: Update catalog.NewCatalog to Use Cloning

**Files:**
- Modify: `backend/plugin/advisor/catalog/catalog.go`
- Test: `backend/plugin/advisor/catalog/catalog_test.go` (create if needed)

**Step 1: Write the failing test**

Create `backend/plugin/advisor/catalog/catalog_test.go`:

```go
package catalog

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/stretchr/testify/require"
)

func TestNewCatalogWithMetadata(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	origin, final, err := NewCatalogWithMetadata(metadata, storepb.Engine_POSTGRES, true)

	require.Nil(t, err)
	require.NotNil(t, origin)
	require.NotNil(t, final)

	// Verify both have the table initially
	require.NotNil(t, origin.GetSchema("public").GetTable("users"))
	require.NotNil(t, final.GetSchema("public").GetTable("users"))

	// Mutate final
	err = final.GetSchema("public").CreateTable("products")
	require.Nil(t, err)

	// Verify origin is unchanged
	require.Nil(t, origin.GetSchema("public").GetTable("products"))

	// Verify final has the new table
	require.NotNil(t, final.GetSchema("public").GetTable("products"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/catalog -run "TestNewCatalogWithMetadata"`

Expected: FAIL with "undefined: NewCatalogWithMetadata"

**Step 3: Add NewCatalogWithMetadata function**

Modify `backend/plugin/advisor/catalog/catalog.go`:

```go
// NewCatalogWithMetadata creates origin and final database metadata from schema metadata.
// Uses proto cloning to create independent copies.
func NewCatalogWithMetadata(metadata *storepb.DatabaseSchemaMetadata, engineType storepb.Engine, isCaseSensitive bool) (origin *model.DatabaseMetadata, final *model.DatabaseMetadata, err error) {
	// Create origin from original metadata
	originSchema := model.NewDatabaseSchema(metadata, nil, nil, engineType, isCaseSensitive)
	origin = originSchema.GetDatabaseMetadata()

	// Clone metadata for final
	clonedMetadata := &storepb.DatabaseSchemaMetadata{}
	if err := clonedMetadata.UnmarshalVT(metadata.MarshalVT()); err != nil {
		return nil, nil, err
	}

	finalSchema := model.NewDatabaseSchema(clonedMetadata, nil, nil, engineType, isCaseSensitive)
	final = finalSchema.GetDatabaseMetadata()

	return origin, final, nil
}
```

Update existing `NewCatalog` to use `NewCatalogWithMetadata`:

```go
// NewCatalog creates origin and final database catalog states.
func NewCatalog(ctx context.Context, s *store.Store, instanceID, databaseName string, engineType storepb.Engine, isCaseSensitive bool, overrideDatabaseMetadata *storepb.DatabaseSchemaMetadata) (origin *model.DatabaseMetadata, final *model.DatabaseMetadata, err error) {
	var metadata *storepb.DatabaseSchemaMetadata

	if overrideDatabaseMetadata != nil {
		metadata = overrideDatabaseMetadata
	} else {
		databaseMeta, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return nil, nil, err
		}
		if databaseMeta == nil {
			return nil, nil, nil
		}
		metadata = databaseMeta.GetMetadata()
	}

	return NewCatalogWithMetadata(metadata, engineType, isCaseSensitive)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/catalog -run "TestNewCatalogWithMetadata"`

Expected: PASS

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/plugin/advisor/catalog/`

Expected: No issues

**Step 6: Commit**

```bash
git add backend/plugin/advisor/catalog/catalog.go backend/plugin/advisor/catalog/catalog_test.go
git commit -m "refactor: use DatabaseMetadata with cloning in NewCatalog"
```

---

## Task 6: Update Advisor Context to Use DatabaseMetadata

**Files:**
- Modify: `backend/plugin/advisor/advisor.go`
- Test: Verify with existing advisor tests

**Step 1: Update Context struct**

In `backend/plugin/advisor/advisor.go`, change the Context struct:

```go
// Context is the context for advisor.
type Context struct {
	DBSchema              *storepb.DatabaseSchemaMetadata
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	EnablePriorBackup     bool
	ClassificationConfig  *storepb.DataClassificationSetting_DataClassificationConfig
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// SQL review rule special fields.
	AST any
	Rule *storepb.SQLReviewRule

	// CHANGED: Use model.DatabaseMetadata instead of catalog.DatabaseState
	OriginCatalog *model.DatabaseMetadata
	FinalCatalog  *model.DatabaseMetadata

	Driver *sql.DB
}
```

**Step 2: Update all references in advisor code**

This will require searching for all usages of `OriginCatalog` and `FinalCatalog` throughout the advisor package and updating them to use `DatabaseMetadata` methods instead of `DatabaseState` methods.

**Step 3: Run advisor tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/...`

Expected: Tests may fail initially - we'll fix in next steps

**Step 4: Commit**

```bash
git add backend/plugin/advisor/advisor.go
git commit -m "refactor: change advisor Context to use DatabaseMetadata"
```

---

## Task 7: Migrate Walk-Through Code to Use DatabaseMetadata

**Files:**
- Modify: `backend/plugin/advisor/catalog/walk_through_for_mysql.go`
- Modify: `backend/plugin/advisor/catalog/walk_through_for_pg.go`
- Modify: `backend/plugin/advisor/catalog/walk_through_for_tidb.go`
- Modify: `backend/plugin/advisor/catalog/walk_through.go`

**Step 1: Update MySQLWalkThrough signature**

Change walk-through functions to accept `*model.DatabaseMetadata`:

```go
// MySQLWalkThrough walks through MySQL AST and updates the database metadata.
func MySQLWalkThrough(d *model.DatabaseMetadata, ast any) error {
	// Implementation...
}
```

**Step 2: Update walk-through listener to use DatabaseMetadata**

```go
type mysqlListener struct {
	*mysql.BaseMySQLParserListener

	baseLine         int
	lineNumber       int
	text             string
	databaseMetadata *model.DatabaseMetadata
	err              error
}
```

**Step 3: Replace DatabaseState method calls with DatabaseMetadata methods**

For example, change:
```go
schema.CreateTable(tableName)
```
To use the new methods we added to `SchemaMetadata`.

**Step 4: Run walk-through tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/catalog -run ".*WalkThrough"`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/plugin/advisor/catalog/walk_through*.go
git commit -m "refactor: migrate walk-through code to use DatabaseMetadata"
```

---

## Task 8: Remove DatabaseState

**Files:**
- Delete: `backend/plugin/advisor/catalog/state.go`
- Update imports throughout advisor package

**Step 1: Verify no references to DatabaseState remain**

Run: `grep -r "DatabaseState" backend/plugin/advisor/`

Expected: No results (or only in comments)

**Step 2: Delete state.go**

Run: `git rm backend/plugin/advisor/catalog/state.go`

**Step 3: Run all tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`

Expected: All tests PASS

**Step 4: Commit**

```bash
git commit -m "refactor: remove DatabaseState (replaced by DatabaseMetadata)"
```

---

## Task 9: Document the New Architecture

**Files:**
- Create: `backend/store/model/README.md`
- Update: `backend/plugin/advisor/catalog/README.md`

**Step 1: Write model documentation**

Create `backend/store/model/README.md`:

```markdown
# Database Model Architecture

## Overview

The `model` package provides the unified database schema representation used throughout Bytebase. Both read-only operations (querying schema) and mutation operations (walk-through simulation) use the same `DatabaseMetadata` API.

## Core Types

### DatabaseSchema

Immutable wrapper around `DatabaseSchemaMetadata` proto.

- Contains: metadata (proto), schema (raw dump), config
- Created from storage via `store.GetDBSchema()`
- Provides access to `DatabaseMetadata` via `GetDatabaseMetadata()`

### DatabaseMetadata

The main schema catalog with both read and mutation methods.

**Read methods:**
- `GetSchema(name)` - Get schema by name
- `SearchTable(searchPath, name)` - Search with PostgreSQL search path
- `SearchView(searchPath, name)` - Search view with search path
- `GetOwner()`, `GetSearchPath()` - Database-level getters

**Mutation methods:**
(Added for advisor walk-through simulation)
- Schema-level: Via `SchemaMetadata`
- Table-level: Via `TableMetadata`

### SchemaMetadata

Schema-level catalog with table operations.

**Read methods:**
- `GetTable(name)` - Get table by name
- `GetView(name)` - Get view by name
- `ListTableNames()` - List all tables

**Mutation methods:**
- `CreateTable(name)` - Create new table
- `DropTable(name)` - Drop existing table
- `RenameTable(old, new)` - Rename table

### TableMetadata

Table-level catalog with column/index operations.

**Read methods:**
- `GetColumn(name)` - Get column by name
- `GetIndex(name)` - Get index by name
- `GetColumns()` - List all columns

**Mutation methods:**
- `CreateColumn(proto)` - Create new column
- `DropColumn(name)` - Drop existing column
- `CreateIndex(proto)` - Create new index
- `DropIndex(name)` - Drop existing index

## Usage Patterns

### Read-Only (Query Schema)

```go
schema, _ := store.GetDBSchema(ctx, ...)
dbMeta := schema.GetDatabaseMetadata()

// Query schema
table := dbMeta.GetSchema("public").GetTable("users")
column := table.GetColumn("email")
```

### Mutation (Advisor Walk-Through)

```go
// Get base metadata
metadata := baseSchema.GetMetadata()

// Create two independent copies using proto.Clone
origin, final, _ := catalog.NewCatalogWithMetadata(metadata, engine, caseSensitive)

// origin = immutable snapshot
// final = will be mutated during walk-through

// Mutate final during SQL analysis
final.GetSchema("public").CreateTable("products")
final.GetSchema("public").DropTable("old_table")

// Compare origin vs final to generate advisor recommendations
```

## Benefits

1. **Unified API**: Same types for read and mutation operations
2. **No duplication**: Single source of truth for schema structure
3. **Simple cloning**: Use `proto.Clone` for independent copies
4. **Type safety**: All methods strongly typed, errors returned explicitly
5. **Maintainable**: One place to add new schema object types

## Migration from DatabaseState

Previously, advisors used `catalog.DatabaseState` (a separate mutable wrapper). This has been removed in favor of adding mutation methods directly to `DatabaseMetadata`.

**Old:**
```go
origin := catalog.NewDatabaseState(metadata, ...)
final := catalog.NewDatabaseState(metadata, ...)
```

**New:**
```go
origin, final, _ := catalog.NewCatalogWithMetadata(metadata, ...)
```
```

**Step 2: Update advisor catalog README**

Update `backend/plugin/advisor/catalog/README.md`:

```markdown
# Advisor Catalog

## Overview

The catalog package provides helper functions for creating database metadata catalogs for SQL advisors.

## Key Functions

### NewCatalog

Creates origin and final database metadata from store.

```go
origin, final, err := catalog.NewCatalog(ctx, store, instanceID, dbName, engine, caseSensitive, override)
```

- Fetches current schema from storage (or uses override)
- Returns two independent `DatabaseMetadata` instances
- `origin` = immutable snapshot of current state
- `final` = mutable copy for walk-through simulation

### NewCatalogWithMetadata

Creates origin and final database metadata from existing metadata proto.

```go
origin, final, err := catalog.NewCatalogWithMetadata(metadata, engine, caseSensitive)
```

Uses `proto.Clone` to create independent copies.

## Walk-Through Functions

### MySQLWalkThrough / PgWalkThrough / TiDBWalkThrough

Walks through parsed SQL AST and mutates the final database metadata.

```go
err := catalog.MySQLWalkThrough(final, parsedAST)
```

These functions call mutation methods on `model.DatabaseMetadata` to simulate schema changes.

## Architecture

All catalog functionality delegates to `backend/store/model` types. This package provides advisor-specific helpers but does not define its own schema model types.

See `backend/store/model/README.md` for details on the unified schema model.
```

**Step 3: Commit documentation**

```bash
git add backend/store/model/README.md backend/plugin/advisor/catalog/README.md
git commit -m "docs: document unified DatabaseMetadata architecture"
```

---

## Task 10: Build and Verify

**Files:**
- Build: All backend code

**Step 1: Build the backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

Expected: SUCCESS (no build errors)

**Step 2: Run full test suite**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`

Expected: All tests PASS

**Step 3: Run final linter check**

Run: `golangci-lint run --allow-parallel-runners`

Keep running until no issues remain (max-issues limit may require multiple runs)

Expected: Clean output

**Step 4: Final commit**

```bash
git add .
git commit -m "chore: final cleanup and verification"
```

---

## Summary

This plan consolidates database schema models by:

1. **Adding mutation methods to `model.DatabaseMetadata`** and related types
2. **Using `proto.Clone`** to create independent origin/final catalogs
3. **Removing `catalog.DatabaseState`** entirely
4. **Updating advisors** to work directly with `DatabaseMetadata`
5. **Migrating walk-through code** to use the new mutation methods

The result: A single unified schema model (`DatabaseMetadata`) used by all code, with both read and mutation capabilities.

**Key architectural principles:**
- DRY: Single schema model, not duplicated
- Simple: proto.Clone for independent copies
- Unified: Same API for read and mutation
- Type-safe: Explicit error returns

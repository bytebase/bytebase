# PostgreSQL SDL Code Refactoring Design

## Overview

Refactor the PostgreSQL SDL (Schema Definition Language) codebase to improve readability and extensibility by creating a dedicated `pg/sdl/` subdirectory with files organized by object type.

## Goals

1. **Readability** - Break down the 8000+ line `get_sdl_diff.go` into focused, navigable files
2. **Extensibility** - Establish a pattern that makes adding new object types straightforward
3. **Future Template** - Create a structure that can be adopted by other database engines

## Current State

```
backend/plugin/schema/pg/
├── get_sdl_diff.go              # 8008 lines, 120+ functions
├── generate_migration.go        # 5080 lines
├── get_database_definition.go
├── get_database_metadata.go
├── *_sdl_diff_test.go           # Object-specific tests (already split)
├── *_test.go                    # ~50 test files
└── sdl_testdata/                # 105 test case files
```

### Problems

- `get_sdl_diff.go` handles 13+ object types in one file
- Hard to navigate and understand
- Adding new objects requires understanding the entire file
- Test files already split by object, but implementation is monolithic

## Target Structure

```
backend/plugin/schema/pg/
├── sdl/                              # New subdirectory
│   ├── diff.go                       # Entry point GetDiff (~300 lines)
│   ├── chunk.go                      # ChunkText + sdlChunkExtractor (~500 lines)
│   ├── common.go                     # Shared utilities, types (~400 lines)
│   │
│   ├── table.go                      # Table diff + drift + helpers
│   ├── column.go                     # Column processing (split from table)
│   ├── constraint.go                 # FK, PK, Check, Unique, Exclude constraints
│   ├── index.go                      # Standalone index processing
│   ├── view.go                       # View diff + drift
│   ├── materialized_view.go          # Materialized view diff + drift
│   ├── function.go                   # Function diff + drift
│   ├── procedure.go                  # Procedure diff + drift
│   ├── sequence.go                   # Sequence diff + drift
│   ├── trigger.go                    # Trigger diff + drift
│   ├── enum_type.go                  # Enum type diff + drift
│   ├── extension.go                  # Extension diff + drift
│   ├── schema.go                     # Schema diff + implicit creation
│   ├── comment.go                    # Comment processing (cross-object)
│   │
│   ├── migration.go                  # SQL generation (from generate_migration.go)
│   │
│   ├── testdata/                     # Moved from pg/sdl_testdata/
│   │
│   └── *_test.go                     # All SDL-related tests
│
├── register.go                       # Register SDL functions to schema registry
├── get_database_definition.go        # Stays (SDL output, not diff)
├── get_database_metadata.go          # Stays (database introspection)
├── function_comparer.go              # Stays (utility, not SDL core)
├── index_comparer.go                 # Stays
├── view_comparer.go                  # Stays
├── trigger_differ.go                 # Stays
├── walk_through.go                   # Stays
└── *_testcontainer_test.go           # Stays (db definition/metadata tests)
```

## Design Decisions

### Package Name

- Package: `sdl`
- Import path: `github.com/bytebase/bytebase/backend/plugin/schema/pg/sdl`

### Exported Functions

| Current | New |
|---------|-----|
| `pg.GetSDLDiff()` | `sdl.GetDiff()` |
| `pg.ChunkSDLText()` | `sdl.ChunkText()` |
| `pg.WriteMigrationSQL()` | `sdl.WriteMigrationSQL()` |

### Registration

A small `register.go` file in `pg/` will register SDL functions:

```go
package pg

import (
    "github.com/bytebase/bytebase/backend/plugin/schema/pg/sdl"
    // ...
)

func init() {
    schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, sdl.GetDiff)
    schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, sdl.GetDiff)
}
```

### File Organization per Object Type

Each object file (e.g., `view.go`) contains:

1. `processViewChanges()` - Detect changes between SDL versions
2. `applyViewChangesToChunks()` - Handle drift synchronization
3. Helper functions specific to that object type
4. Any extractors/generators for that object

### Test Organization

- Tests move with code into `sdl/`
- Test files consolidated by object type where appropriate
- Testcontainer tests for `get_database_definition.go` and `get_database_metadata.go` stay in `pg/`

### Files Staying in `pg/`

| File | Reason |
|------|--------|
| `get_database_definition.go` | SDL output generation, not diff logic |
| `get_database_metadata.go` | Database introspection, independent of SDL |
| `function_comparer.go` | Utility for comparing functions |
| `index_comparer.go` | Utility for comparing indexes |
| `view_comparer.go` | Utility for comparing views |
| `trigger_differ.go` | Utility for comparing triggers |
| `walk_through.go` | Integration test framework |
| `*_testcontainer_test.go` | Tests for non-SDL functionality |

## Migration Plan

### Phase 1: Create Structure

1. Create `pg/sdl/` directory
2. Create `diff.go` with entry point
3. Create `chunk.go` with chunk extraction
4. Create `common.go` with shared utilities

### Phase 2: Extract Object Types

For each object type (table, view, function, etc.):

1. Create `{object}.go`
2. Move `process{Object}Changes()` function
3. Move `apply{Object}ChangesToChunks()` function
4. Move related helper functions
5. Move corresponding test file

### Phase 3: Extract Migration

1. Move `generate_migration.go` to `sdl/migration.go`
2. Update package and imports

### Phase 4: Cleanup

1. Create `pg/register.go` for registration
2. Update all import paths
3. Move `sdl_testdata/` to `sdl/testdata/`
4. Remove empty `get_sdl_diff.go`
5. Run tests and fix any issues

## Estimated File Sizes

| File | Estimated Lines |
|------|-----------------|
| `diff.go` | ~300 |
| `chunk.go` | ~500 |
| `common.go` | ~400 |
| `table.go` | ~800 |
| `column.go` | ~600 |
| `constraint.go` | ~800 |
| `index.go` | ~300 |
| `view.go` | ~400 |
| `materialized_view.go` | ~400 |
| `function.go` | ~400 |
| `procedure.go` | ~200 |
| `sequence.go` | ~600 |
| `trigger.go` | ~400 |
| `enum_type.go` | ~300 |
| `extension.go` | ~300 |
| `schema.go` | ~200 |
| `comment.go` | ~400 |
| `migration.go` | ~5000 |

## Success Criteria

1. All existing tests pass
2. No functional changes to SDL behavior
3. Each file < 1000 lines (except migration.go)
4. Clear separation of concerns by object type
5. Easy to add new object types by following existing patterns

## Future Work

- Apply similar structure to other database engines (MySQL, TiDB, etc.)
- Consider splitting `migration.go` further if needed
- Add documentation for adding new object types to SDL

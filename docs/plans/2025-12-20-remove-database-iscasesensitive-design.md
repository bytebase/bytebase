# Remove IsCaseSensitive from Database Store Lookups

**Date:** 2025-12-20
**Status:** Proposed

## Problem Statement

The `IsCaseSensitive` field in `FindDatabaseMessage` adds unnecessary complexity to database store lookups:

**Current behavior:**
- When `IsCaseSensitive=true`: Uses `db.name = ?` (exact match)
- When `IsCaseSensitive=false`: Uses `LOWER(db.name) = LOWER(?)` (case-insensitive)

**Why it exists:**
- Originally intended to handle database engines where object names are case-insensitive (MySQL with `lower_case_table_names`, TiDB, MSSQL)
- Assumption: User might provide "mydb" when database is stored as "MyDB"

**Why it's unnecessary:**

1. **Database names are synced exactly** - The sync pipeline stores database names exactly as returned by the engine with no normalization (`mysql/sync.go:102`, `tidb/sync.go:94`, `pg/pg.go:305`)

2. **Clients use exact names** - API clients get resource names from our responses (e.g., `GET /v1/instances/prod/databases` returns `{name: "instances/prod/databases/MyDB"}`) and reuse them verbatim in subsequent requests

3. **Internal references are exact** - Tasks, rollouts, and other internal components store the exact database name from database records

4. **Performance cost** - `LOWER(db.name) = LOWER(?)` prevents efficient index usage on the `UNIQUE(instance, name)` constraint

## Proposed Solution

Remove the `IsCaseSensitive` field entirely and always use exact matching in store layer queries.

**Code change:**

```go
// Before (database.go:147-151)
if find.IsCaseSensitive {
    where.And("db.name = ?", *v)
} else {
    where.And("LOWER(db.name) = LOWER(?)", *v)
}

// After
where.And("db.name = ?", *v)
```

**Scope of changes:**

1. **Store layer (remove case-insensitive logic):**
   - `backend/store/database.go` - Remove `IsCaseSensitive` field from `FindDatabaseMessage` (line 71)
   - Simplify query logic (lines 147-151) to always use exact match
   - Remove ~15 call sites that set `IsCaseSensitive: store.IsObjectCaseSensitive(instance)`:
     - `backend/api/v1/database_service.go`
     - `backend/api/v1/database_service_changelog.go`
     - `backend/api/v1/common.go`
     - `backend/api/v1/rollout_service.go`
     - `backend/component/export/resources.go`
     - `backend/component/sampleinstance/manager.go`
     - `backend/runner/approval/runner.go`
     - `backend/runner/taskrun/database_migrate_executor.go`

2. **Query/LSP layer (keep existing logic):**
   - **No changes** to `IsObjectCaseSensitive()` function in `backend/store/instance.go:348`
   - This function is still needed for Query/LSP identifier matching
   - When users write SQL like `SELECT * FROM mydb.users`, the Query/LSP layer needs to match "mydb" against stored metadata using case-sensitivity rules
   - Continues using `IsObjectCaseSensitive(instance)` for auto-completion and span analysis

3. **Parser contexts (keep existing):**
   - `backend/plugin/parser/base/` - Keep `IsCaseSensitive` field in parser contexts
   - These are unrelated to store lookups, used for SQL parsing rules

## Migration Strategy

- **No data migration needed** - Only changing query behavior
- **No API changes** - This is internal to the store layer
- **Backward compatible** - Exact matching is stricter but represents correct behavior

## Edge Case Handling

If a database name mismatch occurs (e.g., from manual DB manipulation or API bug):
- **Before**: Case-insensitive match might succeed incorrectly
- **After**: Returns "not found" - correct fail-fast behavior

## Validation

Before implementing the change:

1. **Verify no case duplicates exist:**
   ```sql
   SELECT instance, name FROM db
   GROUP BY instance, LOWER(name)
   HAVING COUNT(*) > 1;
   ```
   Expected: Zero rows (enforced by `UNIQUE(instance, name)` constraint)

2. **Grep for manual database name construction:**
   ```bash
   grep -r "DatabaseName.*=" backend/ | grep -v "database.DatabaseName"
   ```
   Verify callers get names from database records, not constructing them

## Testing Strategy

1. **Existing tests should pass** - If tests break, they were relying on incorrect fuzzy matching

2. **Add explicit test for exact matching:**
   ```go
   func TestGetDatabase_ExactMatchRequired(t *testing.T) {
       // Create database "MyDatabase"
       // Query with "mydatabase" should return nil (not found)
       // Query with "MyDatabase" should succeed
   }
   ```

3. **Integration test**: Sync a real MySQL/TiDB instance, verify databases are found with their exact synced names

## Benefits

1. **Performance** - Exact match enables index usage on `UNIQUE(instance, name)` constraint
2. **Simplicity** - Removes conditional query logic and boolean flag
3. **Correctness** - Fail fast on mismatches rather than fuzzy matching
4. **Clarity** - Separates concerns: store lookups use exact match, Query/LSP uses case rules

## Implementation Order

1. Run validation queries to verify no case-mismatch scenarios exist
2. Remove `IsCaseSensitive` field from `FindDatabaseMessage` struct
3. Simplify query logic to always use exact match
4. Remove `IsCaseSensitive` assignments from all ~15 call sites
5. Run full test suite
6. Manual testing with MySQL (case-insensitive) and PostgreSQL (case-sensitive) instances

## Rollout Safety

- **Low risk** - Makes queries stricter, not looser
- **Observable** - Issues manifest as "database not found" errors (easy to spot in logs)
- **Reversible** - Can restore `IsCaseSensitive` logic if needed (though it shouldn't be)

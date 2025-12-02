# DDL/DML Consolidation for Plan Checks and SQL Review

## Overview

Consolidate DDL/DML distinction in plan checks and SQL review. The backend no longer distinguishes between DDL and DML changes, so the plan check and advisor systems should reflect this.

## Current State

### Proto Enum
```proto
enum ChangeDatabaseType {
  CHANGE_DATABASE_TYPE_UNSPECIFIED = 0;
  DDL = 1;
  DML = 2;
  SDL = 3;
  DDL_GHOST = 4;
}
```

### Advisor Rules Affected
- `statement.disallow-mix-in-ddl` - runs for DDL/SDL/DDL_GHOST
- `statement.disallow-mix-in-dml` - runs for DML (never triggers - DML not set)
- `table.disallow-dml` - runs for DML (never triggers)
- `statement.dml-dry-run` - runs for DML (never triggers)
- `builtin.prior-backup-check` - runs for DML (never triggers)
- `advice.online-migration` - checks for DDL_GHOST specifically

## Design

### 1. Proto Changes

**File:** `proto/store/store/plan_check_run.proto`

```proto
enum ChangeDatabaseType {
  CHANGE_DATABASE_TYPE_UNSPECIFIED = 0;
  CHANGE_DATABASE = 1;
  SDL = 2;
}
```

### 2. Advisor Context Changes

**File:** `backend/plugin/advisor/advisor.go`

Add `EnableGhost bool` field to Context struct:
```go
type Context struct {
    // ... existing fields ...
    ChangeType   storepb.PlanCheckRunConfig_ChangeDatabaseType
    EnableGhost  bool  // NEW: whether gh-ost online migration is enabled
    // ... rest of fields ...
}
```

**File:** `backend/runner/plancheck/statement_advise_executor.go`

Extract `EnableGhost` from plan spec config and pass to advisor Context.

### 3. Rules to Remove

Delete completely (files + registrations):

| Rule | Files |
|------|-------|
| `statement.disallow-mix-in-ddl` | mysql, pg, mssql, oracle, tidb |
| `statement.disallow-mix-in-dml` | mysql, pg, mssql, oracle, tidb |

Also remove from `backend/plugin/advisor/sql_review.go` rule type constants.

### 4. Rules to Update (Run Unconditionally)

Remove ChangeType guards:

| Rule | Files | Change |
|------|-------|--------|
| `table.disallow-dml` | mysql, mssql | Remove `if ChangeType == DML` guard |
| `statement.dml-dry-run` | pg | Remove `if ChangeType == DML` guard |
| `builtin.prior-backup-check` | mysql, pg, oracle, mssql, tidb | Remove `if ChangeType == DML` guard, keep `if EnablePriorBackup` |

### 5. Online Migration Rule Update

**File:** `backend/plugin/advisor/mysql/rule_online_migration.go`

Change from checking `ChangeType == DDL_GHOST` to checking `EnableGhost == true`.

### 6. Type Conversion Update

**File:** `backend/api/v1/plan_service_plan_check.go`

```go
func convertToChangeDatabaseType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) storepb.PlanCheckRunConfig_ChangeDatabaseType {
    switch t {
    case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
        return storepb.PlanCheckRunConfig_CHANGE_DATABASE
    case storepb.PlanConfig_ChangeDatabaseConfig_SDL:
        return storepb.PlanCheckRunConfig_SDL
    default:
        return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
    }
}
```

### 7. Database Migrations

**Migration A: plan_check_run.config**

```sql
-- Update changeDatabaseType in JSONB config
UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"CHANGE_DATABASE"')
WHERE config->>'changeDatabaseType' IN ('DDL', 'DML', 'DDL_GHOST');

UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"SDL"')
WHERE config->>'changeDatabaseType' = 'SDL';
```

**Migration B: review_config.payload**

```sql
-- Remove disallow-mix rules from SQL review configs
UPDATE review_config
SET payload = jsonb_set(
  payload,
  '{rules}',
  (SELECT jsonb_agg(r)
   FROM jsonb_array_elements(payload->'rules') r
   WHERE r->>'type' NOT IN (
     'statement.disallow-mix-in-ddl',
     'statement.disallow-mix-in-dml'
   ))
);
```

## Summary Table

| Area | Change |
|------|--------|
| Proto enum | `DDL/DML/SDL/DDL_GHOST` → `CHANGE_DATABASE/SDL` |
| Advisor Context | Add `EnableGhost bool` field |
| Remove rules | `statement.disallow-mix-in-ddl`, `statement.disallow-mix-in-dml` (10 files) |
| Update rules | `table.disallow-dml`, `dml-dry-run`, `prior-backup-check` - remove ChangeType guards |
| Update rule | `online-migration` - check `EnableGhost` instead of `DDL_GHOST` |
| Type conversion | Remove `enableGhost` param, always return `CHANGE_DATABASE` or `SDL` |
| DB migration | Update `plan_check_run.config` values, remove mix rules from `review_config` |

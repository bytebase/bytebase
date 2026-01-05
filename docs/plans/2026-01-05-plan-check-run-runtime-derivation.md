# Plan Check Run Runtime Derivation

## Overview

Refactor plan check run to derive configuration from the associated plan at runtime, rather than storing a self-contained copy of the configuration.

## Motivation

1. **Simplify code** — Remove the config copying logic in `getPlanCheckRunFromPlan()`
2. **Reduce storage** — Avoid storing redundant data that duplicates the plan
3. **Prevent staleness** — Plan check run config can't get out of sync with plan

## Current State

Plan check run stores duplicated fields from the plan:

| Field | In Plan | In Plan Check Run |
|-------|---------|-------------------|
| `sheet_sha256` | `ChangeDatabaseConfig` | `CheckTarget` |
| `enable_prior_backup` | `ChangeDatabaseConfig` | `CheckTarget` |
| `enable_ghost` | `ChangeDatabaseConfig` | `CheckTarget` |
| `ghost_flags` | `ChangeDatabaseConfig` | `CheckTarget` |
| `targets` | Spec level | Expanded per `CheckTarget` |

The `config` column stores `PlanCheckRunConfig` which contains these duplicated fields. The `payload` column is unused.

## Design

### Key Insight

There is only one plan check run per plan (unique constraint on `plan_id`). When plan checks are rerun, the old run is replaced. This means:
- No need to preserve historical config
- Results are self-documenting (contain target info)
- Config can be derived from current plan state

### Data Model Changes

**Remove from `plan_check_run` table:**
- `config` column (JSONB storing `PlanCheckRunConfig`)
- `payload` column (unused, reserved)

**Keep in `plan_check_run` table:**
- `id` (primary key)
- `created_at`, `updated_at`
- `plan_id` (foreign key to plan)
- `status` (RUNNING/DONE/FAILED/CANCELED)
- `result` (JSONB storing `PlanCheckRunResult`)

**Proto changes:**
- Remove `PlanCheckRunConfig` message from `proto/store/store/plan_check_run.proto`
- Remove `CheckTarget` message (only used in config)
- Update API proto if config is exposed to clients

**Store layer:**
- Remove `Config` field from `PlanCheckRunMessage`
- Update CRUD operations to exclude config/payload columns

### Runtime Derivation

**`getPlanCheckRunFromPlan()` becomes a pure derivation function:**

Instead of creating a persisted config, it returns an in-memory struct:

```go
type DerivedCheckTarget struct {
    Target            string   // database resource name
    SheetSHA256       string   // from plan spec
    EnablePriorBackup bool
    EnableGhost       bool
    GhostFlags        map[string]string
    Types             []storepb.PlanCheckType
}
```

**Executor flow:**
1. Fetch plan check run (has `plan_id`, `status`)
2. Fetch plan by `plan_id`
3. Call derivation function to get targets and config
4. Run checks against each target using derived config
5. Store results in `result` field

### Migration

```sql
ALTER TABLE plan_check_run DROP COLUMN config;
ALTER TABLE plan_check_run DROP COLUMN payload;
```

## Files to Change

| Layer | File(s) | Change |
|-------|---------|--------|
| Proto (store) | `proto/store/store/plan_check_run.proto` | Remove `PlanCheckRunConfig`, `CheckTarget` |
| Proto (API) | `proto/v1/plan_service.proto` | Remove config from `PlanCheckRun` if exposed |
| Migration | `backend/migrator/migration/LATEST.sql` + new | Drop columns |
| Store | `backend/store/plan_check_run.go` | Remove `Config` field, update CRUD |
| API | `backend/api/v1/plan_service.go` | Simplify creation, extract derivation |
| Runner | `backend/runner/plancheck/runner.go` | Fetch plan, derive targets at runtime |
| Executors | `backend/runner/plancheck/*.go` | Receive derived config |

## What Stays the Same

- `PlanCheckRunResult` proto (stores actual results per target)
- Plan check run status lifecycle (RUNNING -> DONE/FAILED/CANCELED)
- The derivation logic itself (database group expansion, CI sampling)
- One plan check run per plan constraint

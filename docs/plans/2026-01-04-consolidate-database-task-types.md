# Consolidate DATABASE_SDL and DATABASE_MIGRATE Task Types Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the DATABASE_SDL task type by using release/sheet type to determine execution strategy

**Architecture:** The system currently has two separate task types (DATABASE_SDL and DATABASE_MIGRATE) with separate executors. This creates redundancy since the execution strategy can be determined from the release type (for release-based tasks) or inferred from sheet content (for sheet-based tasks). The refactoring consolidates to a single DATABASE_MIGRATE task type, with execution logic branching based on release.payload.type (VERSIONED vs DECLARATIVE).

**Tech Stack:** Go, Protocol Buffers, PostgreSQL, TypeScript/Vue.js

---

## Task 1: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/3.14/0019##merge_database_sdl_to_migrate.sql`

**Step 1: Write the migration SQL**

Create the migration file that converts all existing DATABASE_SDL tasks to DATABASE_MIGRATE:

```sql
-- Consolidate DATABASE_SDL into DATABASE_MIGRATE
-- The execution strategy will be determined by release type or sheet analysis

-- Convert all DATABASE_SDL tasks to DATABASE_MIGRATE
UPDATE task
SET type = 'DATABASE_MIGRATE'
WHERE type = 'DATABASE_SDL';
```

**Step 2: Commit**

```bash
git add backend/migrator/migration/3.14/0019##merge_database_sdl_to_migrate.sql
git commit -m "feat: add migration to consolidate DATABASE_SDL into DATABASE_MIGRATE

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Update Task Proto Definition

**Files:**
- Modify: `proto/store/store/task.proto:11-22`

**Step 1: Remove DATABASE_SDL from proto**

Replace the enum section to completely remove DATABASE_SDL:

```protobuf
  // Type represents the type of database operation to perform.
  enum Type {
    TASK_TYPE_UNSPECIFIED = 0;
    // Create a new database.
    DATABASE_CREATE = 1;
    // Apply schema/data migrations to an existing database.
    // Execution strategy is determined by release type (VERSIONED/DECLARATIVE)
    // or sheet content for non-release tasks.
    DATABASE_MIGRATE = 2;
    // Export data from a database.
    DATABASE_EXPORT = 3;
  }
```

**Step 2: Generate proto code**

Run: `cd proto && buf generate`
Expected: Code generation succeeds, generated files updated

**Step 3: Format proto**

Run: `buf format -w proto`
Expected: Proto file formatted

**Step 4: Commit**

```bash
git add proto/store/store/task.proto backend/generated-go/store/task.pb.go
git commit -m "refactor!: remove DATABASE_SDL task type from proto

BREAKING CHANGE: DATABASE_SDL task type removed. Use DATABASE_MIGRATE
with release type to determine execution strategy.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Remove DATABASE_SDL from V1 API Proto

**Files:**
- Modify: `proto/v1/v1/rollout_service.proto:363-381`

**Step 1: Remove DATABASE_SDL from enum**

Update the enum to completely remove DATABASE_SDL:

```protobuf
  enum Type {
    // Unspecified task type.
    TYPE_UNSPECIFIED = 0;
    // General task for miscellaneous operations.
    GENERAL = 1;
    // Database creation task that creates a new database.
    // Use payload DatabaseCreate.
    DATABASE_CREATE = 2;
    // Database migration task that applies schema/data changes.
    // Use payload DatabaseUpdate.
    DATABASE_MIGRATE = 3;
    // Database export task that exports query results or table data.
    // Use payload DatabaseDataExport.
    DATABASE_EXPORT = 4;
  }
```

**Step 2: Generate proto code**

Run: `cd proto && buf generate`
Expected: Code generation succeeds

**Step 3: Format proto**

Run: `buf format -w proto`
Expected: Proto file formatted

**Step 4: Commit**

```bash
git add proto/v1/v1/rollout_service.proto backend/generated-go/v1/rollout_service.pb.go frontend/src/types/proto-es/v1/rollout_service_pb.d.ts frontend/src/types/proto-es/v1/rollout_service_pb.js
git commit -m "refactor!: remove DATABASE_SDL from V1 API

BREAKING CHANGE: DATABASE_SDL task type removed from API.
Use DATABASE_MIGRATE with DatabaseChangeType instead.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Remove SDL Executor Workaround in DatabaseMigrateExecutor

**Files:**
- Modify: `backend/runner/taskrun/database_migrate_executor.go:241-420`

**Step 1: Remove temporary task type switching**

In the `runReleaseTask` method, locate the DECLARATIVE case (lines 332-410) and remove the workaround (lines 397-405):

Remove these lines:
```go
// Temporarily change task type to DATABASE_SDL so revision is created with correct type
originalTaskType := task.Type
task.Type = storepb.Task_DATABASE_SDL
```

And:
```go
// Restore original task type
task.Type = originalTaskType
```

Keep the execFunc and the call to runMigrationWithFunc, just remove the task type manipulation.

**Step 2: Update executor.go revision logic**

Modify: `backend/runner/taskrun/executor.go:432-441`

Update the revision type determination to rely solely on releaseType:

```go
if isDone {
    // if isDone, record in revision
    if mc.version != "" {
        // Determine revision type from release type
        revisionType := storepb.SchemaChangeType_VERSIONED
        if mc.releaseType != storepb.SchemaChangeType_SCHEMA_CHANGE_TYPE_UNSPECIFIED {
            // Use release type for release-based tasks
            revisionType = mc.releaseType
        }
        // Note: Sheet-based tasks always use VERSIONED (default)

        r := &store.RevisionMessage{
            // ... rest of revision creation ...
            Payload: &storepb.RevisionPayload{
                Release:     mc.release.release,
                File:        mc.release.file,
                SheetSha256: mc.sheet.Sha256,
                TaskRun:     mc.taskRunName,
                Type:        revisionType,
            },
        }
        // ... rest unchanged ...
    }
}
```

Remove the entire block that checks for `task.Type == storepb.Task_DATABASE_SDL`.

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/taskrun/`
Expected: No errors

**Step 4: Commit**

```bash
git add backend/runner/taskrun/database_migrate_executor.go backend/runner/taskrun/executor.go
git commit -m "refactor: remove DATABASE_SDL task type workaround

Use release type directly instead of temporarily changing task type.
Sheet-based tasks always use VERSIONED revision type.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Remove SchemaDeclareExecutor

**Files:**
- Delete: `backend/runner/taskrun/schema_update_sdl_executor.go`
- Modify: `backend/server/server.go:192-194`

**Step 1: Remove executor registration**

In `backend/server/server.go`, remove line 194 completely:

```go
s.taskScheduler.Register(storepb.Task_DATABASE_CREATE, taskrun.NewDatabaseCreateExecutor(stores, s.dbFactory, s.schemaSyncer))
s.taskScheduler.Register(storepb.Task_DATABASE_MIGRATE, taskrun.NewDatabaseMigrateExecutor(stores, s.dbFactory, s.bus, s.schemaSyncer, profile))
s.taskScheduler.Register(storepb.Task_DATABASE_EXPORT, taskrun.NewDataExportExecutor(stores, s.dbFactory, s.licenseService))
```

Delete the entire line:
```go
s.taskScheduler.Register(storepb.Task_DATABASE_SDL, taskrun.NewSchemaDeclareExecutor(stores, s.dbFactory, s.bus, s.schemaSyncer, profile))
```

**Step 2: Delete the SDL executor file**

Run: `rm backend/runner/taskrun/schema_update_sdl_executor.go`
Expected: File deleted

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/server/`
Expected: No errors

**Step 4: Commit**

```bash
git add backend/runner/taskrun/schema_update_sdl_executor.go backend/server/server.go
git commit -m "refactor: remove SchemaDeclareExecutor

DATABASE_SDL tasks are now handled by DatabaseMigrateExecutor
using release type to determine execution strategy.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Update Task Creation Logic

**Files:**
- Modify: `backend/api/v1/rollout_service_task.go:181-241`

**Step 1: Remove task type selection logic**

In the `getTaskCreatesFromChangeDatabaseConfig` function, replace lines 199-203 with:

```go
// All change database tasks are DATABASE_MIGRATE
// Execution strategy is determined by:
// - Release-based: release.payload.type (VERSIONED or DECLARATIVE)
// - Sheet-based: always imperative (VERSIONED)
taskType := storepb.Task_DATABASE_MIGRATE
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/api/v1/rollout_service_task.go
git commit -m "refactor: simplify task creation to use DATABASE_MIGRATE only

Removed task type selection logic. All database change tasks now use
DATABASE_MIGRATE with execution strategy from release type.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Update API Converter

**Files:**
- Modify: `backend/api/v1/rollout_service_converter.go:319-371`

**Step 1: Update convertToTaskFromSchemaUpdate function**

Replace the DatabaseChangeType determination logic (lines 324-333) with simplified logic since we only have DATABASE_MIGRATE now:

```go
// Determine DatabaseChangeType based on source
var databaseChangeType v1pb.DatabaseChangeType
if releaseName := task.Payload.GetRelease(); releaseName != "" {
    // For release-based tasks, fetch the release to determine type
    _, releaseUID, err := common.GetProjectReleaseUID(releaseName)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to parse release name %q", releaseName)
    }
    release, err := convertToTaskFromSchemaUpdateStore.GetReleaseByUID(ctx, releaseUID)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get release %d", releaseUID)
    }
    if release != nil && release.Payload != nil && release.Payload.Type == storepb.SchemaChangeType_DECLARATIVE {
        databaseChangeType = v1pb.DatabaseChangeType_SDL
    } else {
        databaseChangeType = v1pb.DatabaseChangeType_MIGRATE
    }
} else {
    // Sheet-based tasks are always MIGRATE (imperative)
    databaseChangeType = v1pb.DatabaseChangeType_MIGRATE
}
```

Note: This requires access to the store. Since the converter needs context, we'll need to pass the store as a parameter to this function. Check the function signature.

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/api/v1/rollout_service_converter.go
git commit -m "refactor: update task converter to infer DatabaseChangeType

DatabaseChangeType is now inferred from release type (for release-based tasks)
or defaulted to MIGRATE for sheet-based tasks.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Update Rollout Filter

**Files:**
- Modify: `backend/store/rollout_filter.go:109-124`

**Step 1: Remove DATABASE_SDL case**

Update `convertV1ToStoreTaskType` function:

```go
func convertV1ToStoreTaskType(taskType v1pb.Task_Type) storepb.Task_Type {
    switch taskType {
    case v1pb.Task_DATABASE_CREATE:
        return storepb.Task_DATABASE_CREATE
    case v1pb.Task_DATABASE_MIGRATE:
        return storepb.Task_DATABASE_MIGRATE
    case v1pb.Task_DATABASE_EXPORT:
        return storepb.Task_DATABASE_EXPORT
    case v1pb.Task_TYPE_UNSPECIFIED, v1pb.Task_GENERAL:
        return storepb.Task_TASK_TYPE_UNSPECIFIED
    default:
        return storepb.Task_TASK_TYPE_UNSPECIFIED
    }
}
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/store/rollout_filter.go
git commit -m "refactor: remove DATABASE_SDL from rollout filter

DATABASE_SDL task type no longer exists.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update running_scheduler.go

**Files:**
- Modify: `backend/runner/taskrun/running_scheduler.go:271-276`

**Step 1: Remove DATABASE_SDL from sequential task check**

Find the code that checks for DATABASE_SDL and remove it:

```go
// Only DATABASE_MIGRATE is sequential
func isSequentialTask(taskType storepb.Task_Type) bool {
    return taskType == storepb.Task_DATABASE_MIGRATE
}
```

Or if it's inline logic, remove the `|| taskType == storepb.Task_DATABASE_SDL` part.

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/taskrun/`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/runner/taskrun/running_scheduler.go
git commit -m "refactor: remove DATABASE_SDL from sequential task check

All database migrations now use DATABASE_MIGRATE task type.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Update Frontend Code

**Files:**
- All frontend files that reference DATABASE_SDL

**Step 1: Search for DATABASE_SDL usage in frontend**

Run: `grep -r "DATABASE_SDL" frontend/src/ --include="*.ts" --include="*.vue"`
Expected: List of files using DATABASE_SDL

**Step 2: Remove DATABASE_SDL references**

For each file found, update the code to remove `DATABASE_SDL` references completely:

Example pattern:
```typescript
// Before:
if (task.type === Task_Type.DATABASE_SDL) {
  // SDL-specific logic
}

// After:
if (task.type === Task_Type.DATABASE_MIGRATE) {
  // Use DatabaseChangeType to distinguish if needed
  // Check task.databaseUpdate?.databaseChangeType === DatabaseChangeType.SDL
}
```

Or for checks like:
```typescript
// Before:
if (task.type === Task_Type.DATABASE_MIGRATE || task.type === Task_Type.DATABASE_SDL) {
  // ...
}

// After:
if (task.type === Task_Type.DATABASE_MIGRATE) {
  // ...
}
```

**Step 3: Run frontend linter**

Run: `pnpm --dir frontend lint --fix`
Expected: No errors

**Step 4: Run frontend type check**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 5: Format frontend code**

Run: `pnpm --dir frontend biome:format`
Expected: Files formatted

**Step 6: Commit**

```bash
git add frontend/src/
git commit -m "refactor: remove DATABASE_SDL from frontend

DATABASE_SDL task type no longer exists. Use DATABASE_MIGRATE with
DatabaseChangeType to distinguish between imperative and declarative.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Run Database Migration

**Files:**
- N/A (runtime migration)

**Step 1: Build the backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run migration in dev environment**

Run: `PG_URL=postgresql://bbdev@localhost/bbdev ./bytebase-build/bytebase --port 8080 --data . --debug`
Expected: Server starts, migration runs successfully

**Step 3: Verify migration**

Run: `psql -U bbdev bbdev -c "SELECT type, COUNT(*) FROM task GROUP BY type;"`
Expected: No DATABASE_SDL tasks, all converted to DATABASE_MIGRATE

**Step 4: Stop server**

Stop the server after verification.

---

## Task 12: Update Tests

**Files:**
- Search and update test files as needed

**Step 1: Find tests referencing DATABASE_SDL**

Run: `grep -r "DATABASE_SDL" backend/ --include="*.go"`
Expected: List of files (if any) that need updates

**Step 2: Update tests to remove DATABASE_SDL**

For each file found, update to remove `DATABASE_SDL` references.

Example:
```go
// Before:
task := &store.TaskMessage{
    Type: storepb.Task_DATABASE_SDL,
    // ...
}

// After:
task := &store.TaskMessage{
    Type: storepb.Task_DATABASE_MIGRATE,
    // ... (release with DECLARATIVE type will trigger SDL execution)
}
```

**Step 3: Run affected tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/tests -run TestName`
Expected: All tests pass

**Step 4: Run all backend tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`
Expected: All tests pass

**Step 5: Commit**

```bash
git add backend/
git commit -m "test: remove DATABASE_SDL from tests

Tests now use DATABASE_MIGRATE with appropriate release types.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 13: Verification and Final Testing

**Files:**
- N/A (testing only)

**Step 1: Run full backend test suite**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`
Expected: All tests pass

**Step 2: Run frontend tests**

Run: `pnpm --dir frontend test`
Expected: All tests pass

**Step 3: Run golangci-lint on entire backend**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No linter errors

**Step 4: Build frontend**

Run: `pnpm --dir frontend build`
Expected: Build succeeds

**Step 5: Manual smoke test**

1. Start the application
2. Create a new plan with sheet-based migration (should create DATABASE_MIGRATE task)
3. Create a new plan with release-based VERSIONED migration (should create DATABASE_MIGRATE task, execute as imperative)
4. Create a new plan with release-based DECLARATIVE migration (should create DATABASE_MIGRATE task, execute with diff)
5. Verify tasks execute correctly

---

## Notes

**Breaking Changes:**
- Task type `DATABASE_SDL` is completely removed from both store and V1 API protos
- Existing DATABASE_SDL tasks in the database are migrated to DATABASE_MIGRATE
- All code references to DATABASE_SDL are removed

**Execution Strategy Determination:**
- **Release-based tasks**: Determined by `release.payload.type` (VERSIONED or DECLARATIVE)
- **Sheet-based tasks**: Always use imperative execution (VERSIONED)

**Benefits:**
- Eliminates redundant task type and executor
- Simplifies task creation logic
- Removes the awkward task type switching workaround
- Better aligns with the release-level type architecture
- Cleaner codebase without deprecated enum values

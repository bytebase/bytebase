# Surface baseline changelog sync failures in task run logs

**Status:** Design
**Date:** 2026-05-15
**Area:** backend / task runner

## Problem

When a deploy task fails before any SQL statement runs — typically because the target database doesn't actually exist on the instance — the failure originates inside `ensureBaselineChangelog` → `schemaSyncer.SyncDatabaseSchemaToHistory`, which fails to connect. The error is wrapped and returned, ending up only on the `task_run.detail` field.

In the UI, this produces two bad outcomes:

1. The right-side "Detail" panel of the plan detail page shows a **"LATEST LOGS" section that is empty**, because `TaskRunLogViewer` returns `null` when no `task_run_log` rows exist (`frontend/src/react/components/task-run-log/TaskRunLogViewer.tsx:112-114`).
2. The only place the failure reason is rendered is the truncated "Detail" cell of the History table, which is reached via an `EllipsisText` hover tooltip. The tooltip itself is unstyled for long text and stretches across the full viewport.

The user-facing effect is: the most important information about why the task failed is hidden behind a hover-only tooltip that renders incorrectly. Statement execution failures don't have this problem — they emit `COMMAND_EXECUTE` / `COMMAND_RESPONSE` log entries that appear cleanly under "LATEST LOGS" with the error inline.

## Goal

Pre-execution sync failures inside the database migrate executor should appear in "LATEST LOGS" the same way statement execution errors do — as a structured log section with the error string visible when expanded — so the user does not have to discover the error by hovering a truncated table cell.

## Non-goals

- **`EllipsisText` tooltip overflow.** This is a separate UI bug at `frontend/src/react/components/ui/ellipsis-text.tsx:54` (`whitespace-nowrap` popup with no `max-width`). It affects every consumer of `EllipsisText`, not just this view. Out of scope here; tracked as a follow-up.
- **Other pre-execution failure paths** in `RunOnce` (instance/database/project lookups, sheet/release fetches). These are internal store calls that don't realistically fail in normal user flow; not worth logging entries for. If a real-world failure mode emerges we can extend.
- **Proto schema changes.** The existing `DATABASE_SYNC_START` / `DATABASE_SYNC_END` entry types already carry an `error` string field and are already rendered by the frontend log viewer.
- **Frontend changes.** The existing `TaskRunLogViewer` rendering of `DATABASE_SYNC_END` entries is sufficient.

## Approach

Mirror the existing `ExecuteOptions.LogDatabaseSyncStart` / `LogDatabaseSyncEnd` pattern (`backend/plugin/db/driver.go:232-258`) inside `ensureBaselineChangelog`. Emit a `DATABASE_SYNC_START` entry before calling `SyncDatabaseSchemaToHistory`, then emit a `DATABASE_SYNC_END` entry on completion, populating its `Error` field with `err.Error()` when the sync fails.

Use `store.CreateTaskRunLogS` directly (the safe wrapper that logs-and-continues on failure to write). This is the same approach already used elsewhere in this file for `PRIOR_BACKUP_*` entries (`backend/runner/taskrun/database_migrate_executor.go:185-200`). We don't need an `ExecuteOptions` instance here, and we don't want to construct one just for this — `ExecuteOptions` is the driver-execution wiring, and baseline sync runs outside that path.

## Implementation

Single file: `backend/runner/taskrun/database_migrate_executor.go`.

### 1. Change `ensureBaselineChangelog` signature

Current (line 130):

```go
func (exec *DatabaseMigrateExecutor) ensureBaselineChangelog(
    ctx context.Context,
    database *store.DatabaseMessage,
    _ *store.InstanceMessage,
) error
```

New:

```go
func (exec *DatabaseMigrateExecutor) ensureBaselineChangelog(
    ctx context.Context,
    database *store.DatabaseMessage,
    _ *store.InstanceMessage,
    taskRunUID int64,
) error
```

(The unused `*store.InstanceMessage` parameter stays as-is — that's an existing shape we're not cleaning up here.)

`database.ProjectID` is already accessible from the `database` parameter, so we don't need to add a `projectID` argument.

### 2. Update the call site

Line 79 in `RunOnce`:

```go
if err := exec.ensureBaselineChangelog(ctx, database, instance, taskRunUID); err != nil {
    return nil, errors.Wrap(err, "failed to ensure baseline changelog")
}
```

`taskRunUID` is already a parameter of `RunOnce`.

### 3. Wrap the sync call inside `ensureBaselineChangelog`

Replace lines 142-146 (the existing `len(existingChangelogs) == 0` branch's sync call):

```go
if len(existingChangelogs) == 0 {
    exec.store.CreateTaskRunLogS(ctx, database.ProjectID, taskRunUID, time.Now(), exec.profile.ReplicaID, &storepb.TaskRunLog{
        Type:              storepb.TaskRunLog_DATABASE_SYNC_START,
        DatabaseSyncStart: &storepb.TaskRunLog_DatabaseSyncStart{},
    })

    baselineSyncHistory, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
    if err != nil {
        exec.store.CreateTaskRunLogS(ctx, database.ProjectID, taskRunUID, time.Now(), exec.profile.ReplicaID, &storepb.TaskRunLog{
            Type: storepb.TaskRunLog_DATABASE_SYNC_END,
            DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{
                Error: err.Error(),
            },
        })
        return errors.Wrapf(err, "failed to sync database schema for baseline")
    }
    exec.store.CreateTaskRunLogS(ctx, database.ProjectID, taskRunUID, time.Now(), exec.profile.ReplicaID, &storepb.TaskRunLog{
        Type:            storepb.TaskRunLog_DATABASE_SYNC_END,
        DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{},
    })

    // ... existing CreateChangelog call unchanged ...
}
```

Notes:
- `CreateTaskRunLogS` is the swallow-error variant — it logs an `slog.Warn` on its own write failure rather than bubbling. That matches what we want here; a failure to record a log entry should not mask the actual sync error we're trying to surface.
- On the success path we still emit a `DATABASE_SYNC_END` with no error, matching the pattern used by the driver execution path so the section reads cleanly in the UI.
- The wrapped error returned from `ensureBaselineChangelog` is unchanged, so `task_run.detail` still gets the same text it has today. The History table's "Detail" column keeps working.

## What the user will see after this change

For a task that fails because the target database doesn't exist:

- Right side "LATEST LOGS" tab is no longer empty.
- One section labeled with the localized "Syncing" / `task-run.log-detail.syncing` string appears.
- When the user expands the section, the connection error string (e.g. `failed to get databases: failed to connect to ...: tls error: server refused`) is visible inline, with red error styling that the existing renderer already provides for `DatabaseSyncEnd.error`.
- The History table "Detail" column still shows the wrapped error string (unchanged behavior).

## Testing

This code path requires a live failing database sync to exercise end-to-end. Manual verification:

1. Create a database resource in Bytebase whose `database_name` does not exist on the underlying instance (the screenshot scenario).
2. Create a plan/issue with a simple migration (`select 123`) targeting that database.
3. Trigger the rollout. The task should fail.
4. Open the plan detail page, navigate to the failed task. Confirm "LATEST LOGS" now contains a single sync section with the connection error visible when expanded.

No unit test changes — the change is wiring two `CreateTaskRunLogS` calls around an existing call that has no return-value-affecting branches for the happy path.

## Risk

Very low.
- No proto changes.
- No schema changes.
- No frontend changes.
- Uses existing log entry types and the existing `CreateTaskRunLogS` safe wrapper.
- Pattern already proven in the same file for `PRIOR_BACKUP_*` (lines 185-200).
- The only signature change is internal to this file (single caller).

## Out of scope / follow-ups

- Fix `EllipsisText` tooltip width (`frontend/src/react/components/ui/ellipsis-text.tsx:54`) — apply `max-w-*` and allow wrapping so long errors no longer overflow the viewport when this or any other truncated text is hovered.
- Consider whether the right-side "Detail" panel should also render a small inline error indicator near the task title when the latest task run failed, so the failure reason doesn't require expanding a logs section. Worth doing only if the current logs-section presentation still feels buried in practice.

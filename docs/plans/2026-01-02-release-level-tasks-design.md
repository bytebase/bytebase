# Release-Level Tasks for GitOps

## Summary

Change GitOps tasks from (database, sheet) granularity to (database, release) granularity to simplify rollout management and reduce task count.

## Motivation

**Current State:**
- GitOps releases contain multiple migration files (e.g., V0001.sql, V0002.sql, V0003.sql)
- For N databases and M files, we create N×M tasks
- Each task runs one sheet against one database
- This creates many tasks and complicates rollout management

**Problem:**
- Too many tasks for large releases across many databases
- Harder to track "which databases have completed the release"
- Task explosion: 10 databases × 20 files = 200 tasks

**Goal:**
- Create N tasks (one per database) for GitOps releases
- Each task executes all unapplied files from the release
- Resume from failure: revision table already tracks which versions succeeded

## Design

### 1. Proto Changes

**File:** `proto/store/task.proto`

**Before:**
```proto
message Task {
  string sheet_sha256 = 4;
  string schema_version = 10;
  TaskReleaseSource task_release_source = 13;
  // ... other fields
}
```

**After:**
```proto
message Task {
  oneof source {
    string sheet_sha256 = 10;  // For regular (non-release) tasks
    string release = 13;       // For release-based tasks: projects/{project}/releases/{release}
  }
  // ... other fields
  // Removed: schema_version (field 10, replaced by sheet_sha256)
  // Removed: task_release_source (field 13, replaced by release)
}
```

**Key changes:**
- Use `oneof source` to make sheet vs release mutually exclusive
- Reuse field 10 for `sheet_sha256` (moved from field 4)
- Reuse field 13 for `release` (replaces `task_release_source`)
- Remove `schema_version` (was only used for release tasks, now tracked per file)

### 2. Task Creation Changes

**File:** `backend/api/v1/rollout_service_task.go`

**Function:** `getTaskCreatesFromChangeDatabaseConfigWithRelease()` (line 352)

**Before:**
```go
// Creates N×M tasks
for _, database := range databases {
    for _, file := range release.Payload.Files {
        // Filter by applied versions
        if alreadyApplied(file.Version) {
            continue
        }
        // Create task(database, file)
        task := &store.TaskMessage{
            Payload: &storepb.Task{
                SheetSha256: file.SheetSha256,
                SchemaVersion: file.Version,
                TaskReleaseSource: &storepb.TaskReleaseSource{
                    File: formatReleaseFile(file.Id),
                },
            },
        }
        taskCreates = append(taskCreates, task)
    }
}
```

**After:**
```go
// Creates N tasks
for _, database := range databases {
    // Create task(database, release)
    task := &store.TaskMessage{
        InstanceID:   database.InstanceID,
        DatabaseName: &database.DatabaseName,
        Environment:  database.EffectiveEnvironmentID,
        Type:         storepb.Task_DATABASE_MIGRATE,
        Payload: &storepb.Task{
            SpecId:  spec.Id,
            Release: c.Release,  // Store release name, not individual files
            // Remove SheetSha256, SchemaVersion, TaskReleaseSource
        },
    }
    taskCreates = append(taskCreates, task)
}
```

**Benefits:**
- Simpler logic: no nested file loop
- No version filtering at task creation time (deferred to execution)
- One task per database regardless of file count

### 3. Task Execution Changes

**File:** `backend/runner/taskrun/database_migrate_executor.go`

**Current execution:**
```go
func (exec *DatabaseMigrateExecutor) RunOnce(ctx context.Context, ...) {
    sheet := getSheet(task.Payload.GetSheetSha256())
    runMigration(ctx, ..., sheet, task.Payload.GetSchemaVersion())
}
```

**New execution for release tasks:**
```go
func (exec *DatabaseMigrateExecutor) RunOnce(ctx context.Context, ...) {
    // Check if this is a release-based task
    if releaseName := task.Payload.GetRelease(); releaseName != "" {
        return exec.runReleaseTask(ctx, task, taskRunUID, releaseName)
    }

    // Fall back to single-sheet execution
    sheet := getSheet(task.Payload.GetSheetSha256())
    runMigration(ctx, ..., sheet, "")
}

func (exec *DatabaseMigrateExecutor) runReleaseTask(
    ctx context.Context,
    task *store.TaskMessage,
    taskRunUID int,
    releaseName string,
) error {
    // 1. Fetch release
    _, releaseUID, err := common.GetProjectReleaseUID(releaseName)
    release, err := exec.store.GetReleaseByUID(ctx, releaseUID)

    // 2. Get existing revisions for this database
    revisions, err := exec.store.ListRevisions(ctx, &store.FindRevisionMessage{
        InstanceID:   &task.InstanceID,
        DatabaseName: task.DatabaseName,
    })

    // 3. Build map of applied versions
    appliedVersions := make(map[string]bool)
    for _, revision := range revisions {
        if revision.Payload.Type == storepb.SchemaChangeType_VERSIONED {
            appliedVersions[revision.Version] = true
        }
    }

    // 4. Execute unapplied files in order
    for _, file := range release.Payload.Files {
        if file.Type != storepb.SchemaChangeType_VERSIONED {
            continue  // Skip declarative for now
        }

        // Skip if already applied
        if appliedVersions[file.Version] {
            continue
        }

        // Fetch sheet and execute
        sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{
            Sha256: &file.SheetSha256,
        })

        // Run this file's migration
        _, result, err := runMigration(
            ctx,
            ...,
            sheet,
            file.Version,  // Pass version from file, not task
        )
        if err != nil {
            return err  // Stop on first failure
        }

        // Migration succeeded, revision already recorded by runMigration
    }

    return nil
}
```

**Key points:**
- Check `task.Payload.GetRelease()` to detect release tasks
- Reuse existing revision-checking logic (lines 378-405 from task creation)
- Execute files sequentially, stop on first failure
- Natural resume: next task run will skip succeeded versions

### 4. Database Migration

**File:** `backend/migrator/migration/<version>/XXXXX##release_level_tasks.sql`

```sql
-- Update task payload JSONB structure
-- Remove schema_version and task_release_source fields
-- Existing tasks remain unchanged (backward compatible at DB level)
-- New tasks use the new structure

-- No migration needed since:
-- 1. We're reusing existing field numbers in proto
-- 2. JSONB is flexible - old tasks keep old structure
-- 3. New tasks will have new structure
-- 4. Execution logic checks which fields are present
```

**No actual migration needed** - proto field reuse and JSONB flexibility handle this.

### 5. Store Layer Changes

**File:** `backend/store/task.go`

**Update TaskPatch:**
```go
type TaskPatch struct {
    // ... existing fields

    // Remove or deprecate:
    // SchemaVersion *string

    // Add:
    Release *string  // For setting release name on task
}
```

**Update payload update logic (line 406-410):**
```go
// Remove:
// if v := patch.SchemaVersion; v != nil {
//     payloadParts.Join(" || ", "jsonb_build_object('schemaVersion', ?::TEXT)", *v)
// }

// Add:
if v := patch.Release; v != nil {
    payloadParts.Join(" || ", "jsonb_build_object('release', ?::TEXT)", *v)
}
```

### 6. API Converter Changes

**File:** `backend/api/v1/rollout_service_converter.go`

**Update task conversion (line 346-350):**
```go
// Before:
DatabaseUpdate: &v1pb.Task_DatabaseUpdate{
    Sheet:              common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256()),
    SchemaVersion:      task.Payload.GetSchemaVersion(),
    DatabaseChangeType: databaseChangeType,
}

// After:
databaseUpdate := &v1pb.Task_DatabaseUpdate{
    DatabaseChangeType: databaseChangeType,
}

// Set either sheet or release
if releaseName := task.Payload.GetRelease(); releaseName != "" {
    databaseUpdate.Release = releaseName
} else {
    databaseUpdate.Sheet = common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256())
}

DatabaseUpdate: databaseUpdate
```

**Update v1 proto if needed:**
Check if `v1pb.Task_DatabaseUpdate` needs a `release` field for API responses.

### 7. Frontend Changes

**Files to update:**
- `frontend/src/utils/v1/issue/rollout.ts:187` - `extractSchemaVersionFromTask()`
- `frontend/src/components/RolloutV1/components/TaskView.vue`
- `frontend/src/components/RolloutV1/components/TaskTable.vue`
- `frontend/src/components/IssueV1/components/TaskListSection/TaskCard.vue`

**Changes:**
1. Display release name instead of schema version for release tasks
2. Show "Release: v1.2.3" instead of "Version: 00001"
3. Update task detail view to show release files
4. Consider showing progress: "3/5 files applied" for release tasks

## Implementation Steps

1. **Proto changes** - Update task.proto and regenerate
2. **Backend - Task creation** - Modify `getTaskCreatesFromChangeDatabaseConfigWithRelease()`
3. **Backend - Task execution** - Add `runReleaseTask()` to executor
4. **Backend - Store** - Update TaskPatch and payload serialization
5. **Backend - Converters** - Update API response converters
6. **Frontend** - Update task display components
7. **Testing** - Verify release tasks work end-to-end

## Testing Plan

1. **Unit tests:**
   - Task creation produces N tasks (not N×M)
   - Task execution skips applied versions
   - Resume after failure works correctly

2. **Integration tests:**
   - Create release with 5 files, 3 databases
   - Verify 3 tasks created (not 15)
   - Fail task on file 3, verify retry skips files 1-2
   - Verify revisions recorded correctly

3. **Backward compatibility:**
   - Existing sheet-based tasks still execute
   - Old completed tasks still display correctly

## Metrics

**Before:**
- 10 databases × 20 files = 200 tasks
- Each task tracks one (database, file) pair

**After:**
- 10 databases = 10 tasks
- Each task handles all files for one database
- 95% reduction in task count for large releases

## Migration Strategy

1. **No data migration needed** - JSONB flexibility handles old vs new structure
2. **New releases** automatically use new task structure
3. **Old tasks** continue working with existing logic
4. **Gradual rollout** - new releases see benefits immediately

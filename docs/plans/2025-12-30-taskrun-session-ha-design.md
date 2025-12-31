# TaskRun Session Monitoring - HA-Compatible Design

## Overview

Replace in-memory `TaskRunConnectionID` state with database-native connection identification using `application_name`. This makes session monitoring work in High Availability (HA) deployments where multiple Bytebase replicas run simultaneously.

## Problem

The current implementation stores connection IDs in `state.TaskRunConnectionID` (sync.Map), which breaks in HA:
- Replica A executes a task and stores connection ID in its local memory
- User requests session info, load balancer routes to Replica B
- Replica B has no connection ID in its memory → request fails

## Solution

Use PostgreSQL's `application_name` parameter to make connections self-identifying. Query `pg_stat_activity` by `application_name` instead of connection ID.

### Application Name Format

```
bytebase-taskrun-{taskRunUID}
```

Examples:
- Task run execution: `bytebase-taskrun-12345`
- Regular connections: `bytebase` (existing behavior)

### Why This Works

- PostgreSQL exposes `application_name` in `pg_stat_activity` view
- No coordination needed between replicas - database is the source of truth
- Auto-cleanup when connection closes
- Fully stateless

## Architecture Changes

### 1. ConnectionContext Enhancement

Add `TaskRunUID` to connection context (`backend/plugin/db/driver.go`):

```go
type ConnectionContext struct {
    EnvironmentID string
    InstanceID    string
    EngineVersion string
    TenantMode    bool
    DatabaseName  string
    DataShare     bool
    ReadOnly      bool
    MessageBuffer []*v1pb.QueryResult_Message
    TaskRunUID    *int  // NEW: Set when executing a task run
}
```

### 2. PostgreSQL Driver Update

Modify `backend/plugin/db/pg/pg.go` (line 79):

```go
// Current:
pgxConnConfig.RuntimeParams["application_name"] = "bytebase"

// New:
appName := "bytebase"
if config.ConnectionContext.TaskRunUID != nil {
    appName = fmt.Sprintf("bytebase-taskrun-%d", *config.ConnectionContext.TaskRunUID)
}
pgxConnConfig.RuntimeParams["application_name"] = appName
```

### 3. Task Executor Changes

Pass `TaskRunUID` when creating driver (`backend/runner/taskrun/executor.go`):

```go
driver, err := mc.dbFactory.GetAdminDatabaseDriver(ctx, mc.instance, mc.database, db.ConnectionContext{
    EnvironmentID: mc.database.EnvironmentID,
    InstanceID:    mc.instance.ResourceID,
    DatabaseName:  mc.database.DatabaseName,
    TaskRunUID:    &mc.taskRunUID,  // NEW
})
```

### 4. GetTaskRunSession API Refactoring

Update `backend/api/v1/rollout_service.go`:

**Before:**
1. Load connection ID from `stateCfg.TaskRunConnectionID.Load(taskRunUID)`
2. Query `pg_stat_activity WHERE pid = $1`

**After:**
1. Construct `appName = fmt.Sprintf("bytebase-taskrun-%d", taskRunUID)`
2. Query `pg_stat_activity WHERE application_name = $1`

**Query Changes:**

```sql
-- Find main session and blocking/blocked sessions
SELECT
    pid,
    pg_blocking_pids(pid) AS blocked_by_pids,
    query,
    state,
    wait_event_type,
    wait_event,
    datname,
    usename,
    application_name,
    client_addr,
    client_port,
    backend_start,
    xact_start,
    query_start
FROM
    pg_catalog.pg_stat_activity
WHERE application_name = $1
OR pid = ANY(pg_blocking_pids((SELECT pid FROM pg_stat_activity WHERE application_name = $1 LIMIT 1)))
OR (SELECT pid FROM pg_stat_activity WHERE application_name = $1 LIMIT 1) = ANY(pg_blocking_pids(pid))
ORDER BY pid
```

Identify main session by comparing `application_name` instead of `pid`.

### 5. Cleanup - Remove In-Memory State

**Delete from `backend/component/state/state.go`:**
```go
TaskRunConnectionID sync.Map // map[taskRunID]string
```

**Delete from `backend/plugin/db/driver.go` ExecuteOptions:**
```go
SetConnectionID    func(id string)
DeleteConnectionID func()
```

**Delete from `backend/runner/taskrun/executor.go`:**
```go
opts.SetConnectionID = func(id string) {
    stateCfg.TaskRunConnectionID.Store(mc.taskRunUID, id)
}
opts.DeleteConnectionID = func() {
    stateCfg.TaskRunConnectionID.Delete(mc.taskRunUID)
}
```

**Delete from `backend/plugin/db/pg/pg.go` (2 occurrences):**
```go
if opts.SetConnectionID != nil {
    var pid string
    if err := conn.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
        return 0, errors.Wrapf(err, "failed to get connection id")
    }
    opts.SetConnectionID(pid)

    if opts.DeleteConnectionID != nil {
        defer opts.DeleteConnectionID()
    }
}
```

## Engine Support

### Supported
- **PostgreSQL**: Native `application_name` support
- **CockroachDB**: PostgreSQL-compatible, uses same driver

### Unsupported (for now)
- MySQL, TiDB, MariaDB, OceanBase
- Oracle
- MSSQL (has `app name` but deferred to future work)
- All other engines

For unsupported engines, `GetTaskRunSession()` returns:
```
"session monitoring is only supported for PostgreSQL and CockroachDB"
```

## Implementation Plan

### Files to Modify

| File | Changes |
|------|---------|
| `backend/plugin/db/driver.go` | Add `TaskRunUID *int` to `ConnectionContext` |
| `backend/plugin/db/pg/pg.go` | Set `application_name` based on `TaskRunUID` |
| `backend/runner/taskrun/executor.go` | Pass `TaskRunUID` in `ConnectionContext` |
| `backend/api/v1/rollout_service.go` | Query by `application_name` instead of loading from state |
| `backend/component/state/state.go` | Remove `TaskRunConnectionID` field |

### Implementation Order

1. Add `TaskRunUID` to `ConnectionContext`
2. Update PostgreSQL driver to set `application_name`
3. Update task executor to pass `TaskRunUID`
4. Update `GetTaskRunSession()` API to query by `application_name`
5. Remove all in-memory state tracking code
6. Test with PostgreSQL task execution

### Testing Strategy

**Manual Testing:**
1. Execute task on PostgreSQL database
2. Call `GetTaskRunSession()` during execution from any replica
3. Verify session info shows correct query, state, blocking sessions
4. Execute task on MySQL database
5. Verify `GetTaskRunSession()` returns "not supported" error

**Automated Testing:**
1. Unit test: Verify `application_name` format matches `bytebase-taskrun-{uid}`
2. Integration test: Execute task, query `pg_stat_activity`, verify app name appears

## Benefits

### HA Compatibility
- No shared state between replicas
- Any replica can serve `GetTaskRunSession()` requests
- Database is the single source of truth

### Simplification
- Removes 5 code locations maintaining connection ID state
- No cleanup logic needed (auto-cleanup when connection closes)
- Fewer potential bugs (no stale connection IDs)

### Observability
- Connection purpose visible in `pg_stat_activity` without API calls
- DBAs can identify Bytebase task executions directly
- Easier debugging of long-running migrations

## Future Work

- Add MSSQL support using `app name` parameter and `sys.dm_exec_sessions`
- Explore MySQL alternatives (CONNECTION_ID() with session variables)
- Add Oracle support if equivalent feature exists

## Compatibility

No breaking changes:
- New `TaskRunUID` field is optional (pointer)
- Existing connections continue to use `application_name = "bytebase"`
- Only affects PostgreSQL/CockroachDB task run connections
- Other engines get clear "not supported" message

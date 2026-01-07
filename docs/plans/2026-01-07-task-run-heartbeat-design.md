# Task Run Heartbeat for HA

## Problem

If a Bytebase replica crashes while executing a task run, the task stays `RUNNING` forever, blocking other tasks (especially `DATABASE_MIGRATE` which has mutual exclusion).

## Solution

- Each Bytebase replica sends periodic heartbeats to a `replica_heartbeat` table
- Task runs record which replica claimed them (`replica_id` column)
- Data cleaner detects stale tasks (replica missing or heartbeat too old) and marks them `FAILED`

## Components

| Component | Responsibility |
|-----------|----------------|
| Heartbeat runner | Sends heartbeat every 10s using `deployId` |
| Data cleaner | Detects stale RUNNING tasks (30s interval), cleans up old heartbeats (1h interval) |
| `replica_heartbeat` table | Tracks replica liveness |
| `task_run.replica_id` column | Links task to owning replica |

## Schema Changes

### New table

```sql
CREATE TABLE replica_heartbeat (
    replica_id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMPTZ NOT NULL
);
```

### Modify task_run table

```sql
ALTER TABLE task_run ADD COLUMN replica_id TEXT;
```

### Migration

```sql
UPDATE task_run
SET status = 'FAILED',
    result = '{"detail": "Marked as failed during heartbeat migration"}'
WHERE status = 'RUNNING';
```

## Heartbeat Runner

**Location:** `backend/runner/heartbeat/runner.go`

**Behavior:**
- Uses `profile.DeployID` as replica identifier
- Every 10 seconds, UPSERTs into `replica_heartbeat`:

```sql
INSERT INTO replica_heartbeat (replica_id, last_heartbeat)
VALUES ($1, NOW())
ON CONFLICT (replica_id)
DO UPDATE SET last_heartbeat = NOW();
```

## Task Claiming Changes

In `ClaimAvailableTaskRuns()`, set `replica_id` when claiming:

```sql
UPDATE task_run SET status = 'RUNNING', replica_id = $1
WHERE status = 'AVAILABLE' ...
RETURNING ...
```

## Stale Task Detection

**Location:** Data cleaner runner

**Interval:** 30 seconds

**Detection query:**

```sql
UPDATE task_run
SET status = 'FAILED',
    result = '{"detail": "Task run abandoned: owning replica stopped responding"}'
WHERE status = 'RUNNING'
  AND (
    replica_id NOT IN (SELECT replica_id FROM replica_heartbeat)
    OR replica_id IN (
      SELECT replica_id FROM replica_heartbeat
      WHERE last_heartbeat < NOW() - INTERVAL '1 minute'
    )
  )
RETURNING id;
```

## Heartbeat Cleanup

**Interval:** 1 hour

```sql
DELETE FROM replica_heartbeat
WHERE last_heartbeat < NOW() - INTERVAL '1 hour';
```

## Constants

| Constant | Value |
|----------|-------|
| Heartbeat interval | 10 seconds |
| Staleness threshold | 1 minute |
| Stale detection interval | 30 seconds |
| Heartbeat cleanup interval | 1 hour |

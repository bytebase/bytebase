# Task Scheduler AVAILABLE Status Refactoring

## Overview

Refactor the task scheduler to introduce a dedicated `AVAILABLE` status between `PENDING` and `RUNNING`. This separates "ready to execute" from "actively executing", preparing the foundation for High Availability (HA) scheduler instances.

## Status Model

### New Status Flow

```
PENDING → AVAILABLE → RUNNING → DONE/FAILED/CANCELED
```

### Status Definitions

| Status | Meaning |
|--------|---------|
| PENDING | Waiting for constraints: RunAt time, version ordering, database locks, parallel limits |
| AVAILABLE | All constraints satisfied, ready for immediate execution by any scheduler instance |
| RUNNING | Actively executing |
| DONE/FAILED/CANCELED | Terminal states |

### Transition Rules

- `PENDING → AVAILABLE`: pending_scheduler promotes when ALL gating checks pass
- `AVAILABLE → RUNNING`: running_scheduler atomically claims via optimistic locking
- `RUNNING → terminal`: running_scheduler updates after execution completes

## Scheduler Architecture

### pending_scheduler.go (Enhanced)

Responsibility: PENDING → AVAILABLE transitions with all gating logic.

```
Every 5 seconds:
1. Query task_runs WHERE status = 'PENDING'
2. Track in-memory for this round:
   - availableDBs: databases already promoted this round
   - rolloutCounts: tasks promoted per rollout this round
3. For each pending task:
   a. Check RunAt time
   b. Check version ordering (no smaller versions pending/available/running on same DB)
   c. Check database mutual exclusion (sequential tasks only):
      - Skip if availableDBs[database_id] is set
      - Query: no RUNNING or AVAILABLE on same database
   d. Check parallel limit:
      - currentCount = COUNT(*) WHERE rollout_id = ? AND status IN ('RUNNING', 'AVAILABLE')
      - Skip if currentCount + rolloutCounts[rollout_id] >= limit
   e. If all pass:
      - UPDATE status = 'AVAILABLE'
      - availableDBs[database_id] = true
      - rolloutCounts[rollout_id]++
```

### running_scheduler.go (Simplified)

Responsibility: Claim AVAILABLE tasks and execute. No gating logic.

```
Every 5 seconds (or when tickled):
1. Query task_runs WHERE status = 'AVAILABLE'
2. For each available task:
   a. Attempt atomic claim:
      UPDATE task_run SET status = 'RUNNING', started_at = NOW()
      WHERE id = ? AND status = 'AVAILABLE'
   b. If claim succeeds (rows affected = 1):
      - Spawn goroutine to execute task
   c. If claim fails (rows affected = 0):
      - Another instance claimed it, skip
3. Re-execute orphaned RUNNING tasks on startup (maintains current behavior)
```

### Key Simplification

running_scheduler no longer maintains:
- `RunningDatabaseMigration` map
- Database mutual exclusion checks
- Parallel limit checks
- Version ordering checks

All gating logic is centralized in pending_scheduler.

## Gating Logic Details

### Checks Performed (PENDING → AVAILABLE)

1. **RunAt time**: `task_run.run_at <= NOW()`
2. **Version ordering**: No blocking tasks with smaller versions on same database in PENDING/AVAILABLE/RUNNING
3. **Database mutual exclusion** (for sequential task types DDL/SDL):
   - No RUNNING or AVAILABLE tasks on same database
   - Plus in-memory tracking within the loop
4. **Parallel task limit per rollout**:
   - `COUNT(*) WHERE rollout_id = ? AND status IN ('RUNNING', 'AVAILABLE') < limit`
   - Plus in-memory tracking within the loop

### In-Memory Loop Tracking

Within a single pending_scheduler iteration, track locally:

```go
availableDBs := map[int]bool{}     // database_id -> has AVAILABLE this round
rolloutCounts := map[int]int{}     // rollout_id -> count promoted this round
```

This prevents marking multiple tasks AVAILABLE for the same database or exceeding rollout limits within one loop.

### Future HA Consideration

For HA with multiple scheduler instances, atomic check-and-update may be needed:

```sql
UPDATE task_run
SET status = 'AVAILABLE'
WHERE id = ?
  AND status = 'PENDING'
  AND (SELECT COUNT(*) FROM task_run
       WHERE rollout_id = ? AND status IN ('RUNNING', 'AVAILABLE')) < ?
```

This is future work; current implementation assumes single pending_scheduler instance.

## Database Schema Changes

### Migration

```sql
-- Add AVAILABLE to CHECK constraint
ALTER TABLE task_run
DROP CONSTRAINT task_run_status_check,
ADD CONSTRAINT task_run_status_check
CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Update partial index for active statuses
DROP INDEX idx_task_run_active_status_id;
CREATE INDEX idx_task_run_active_status_id ON task_run (status, id)
WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');
```

### Proto Changes

`proto/store/task_run.proto`:

```proto
enum Status {
  STATUS_UNSPECIFIED = 0;
  PENDING = 1;
  RUNNING = 2;
  DONE = 3;
  FAILED = 4;
  CANCELED = 5;
  NOT_STARTED = 6;
  SKIPPED = 7;
  AVAILABLE = 8;  // NEW: Ready for immediate execution
}
```

## Frontend Changes

### Status Classification

`frontend/src/components/RolloutV1/components/utils/taskStatus.ts`:

```typescript
const ACTIONABLE_STATUSES = [
  "NOT_STARTED", "PENDING", "AVAILABLE", "RUNNING", "FAILED", "CANCELED"
];

const TERMINAL_STATUSES = ["DONE", "SKIPPED"];
```

### Visual Representation

- AVAILABLE gets distinct color/icon (ready indicator)
- Status display order: NOT_STARTED → PENDING → AVAILABLE → RUNNING → terminal

### Locale Strings

Add translations for "Available" status in `frontend/src/locales/`.

### SchedulerInfo Display

- Only show waiting causes for PENDING tasks
- AVAILABLE tasks have no waiting info (they're ready)

## Implementation Plan

### Files to Modify

| File | Changes |
|------|---------|
| `proto/store/task_run.proto` | Add `AVAILABLE = 8` to Status enum |
| `backend/migrator/migration/XXX/` | New migration for schema changes |
| `backend/migrator/migration/LATEST.sql` | Update CHECK constraint and index |
| `backend/runner/taskrun/pending_scheduler.go` | Add all gating logic, promote to AVAILABLE |
| `backend/runner/taskrun/running_scheduler.go` | Simplify to: claim AVAILABLE → execute |
| `backend/store/task_run.go` | Add AVAILABLE constant, update queries |
| `frontend/src/components/RolloutV1/.../taskStatus.ts` | Add AVAILABLE to actionable statuses |
| `frontend/src/locales/*.json` | Add "available" translation |
| Frontend status display components | Add AVAILABLE visual styling |

### Implementation Order

1. Proto + generate
2. Database migration
3. Store layer updates
4. pending_scheduler (add gating, PENDING → AVAILABLE)
5. running_scheduler (simplify to claim + execute)
6. Frontend status handling
7. Testing

### Compatibility

No breaking changes. Existing PENDING/RUNNING tasks continue to work. New tasks will use the AVAILABLE intermediate state.

## HA Preparation

This refactoring prepares for HA by:

1. **Clear state separation**: AVAILABLE = ready, RUNNING = executing
2. **Optimistic locking**: Multiple running_scheduler instances can safely compete for AVAILABLE tasks
3. **Database as source of truth**: No in-memory state coordination needed between instances
4. **Future lease-based claiming**: Status model supports adding claimed_by/expiry fields later

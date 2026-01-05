# Plan Check Run Scheduler HA Compatibility

## Overview

Make the plan check run scheduler HA (High Availability) compatible by following patterns established in the task run scheduler.

## Current State

The plan check scheduler is **not HA-compatible**:
- Uses in-memory `sync.Map` to track running checks
- Simple polling loop every 5 seconds
- No database locking - relies on single-instance assumption
- Status: `RUNNING → DONE/FAILED/CANCELED`

## Target State

Follow the task run scheduler HA pattern:
- Use `FOR UPDATE SKIP LOCKED` to atomically claim work
- Status state machine: `AVAILABLE → RUNNING → DONE/FAILED/CANCELED`
- Multiple instances compete fairly via database queries
- No leader election - peer-to-peer work queue pattern

## Design

### Schema Changes

**Migration file:** `backend/migrator/migration/3.X/XXXX##plan_check_run_ha.sql`

```sql
-- Add AVAILABLE to status check constraint
ALTER TABLE plan_check_run
  DROP CONSTRAINT plan_check_run_status_check,
  ADD CONSTRAINT plan_check_run_status_check
    CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Convert existing RUNNING to AVAILABLE (will be re-claimed)
UPDATE plan_check_run SET status = 'AVAILABLE' WHERE status = 'RUNNING';

-- Update index to include AVAILABLE for efficient claiming
DROP INDEX IF EXISTS idx_plan_check_run_active_status;
CREATE INDEX idx_plan_check_run_active_status
  ON plan_check_run(status, id)
  WHERE status IN ('AVAILABLE', 'RUNNING');
```

### Store Layer Changes

**File:** `backend/store/plan_check_run.go`

Add claiming function:

```go
// ClaimAvailablePlanCheckRuns atomically claims all AVAILABLE plan check runs.
// Uses FOR UPDATE SKIP LOCKED for HA-safe concurrent claiming.
func (s *Store) ClaimAvailablePlanCheckRuns(ctx context.Context) ([]*PlanCheckRunMessage, error) {
    query := `
        UPDATE plan_check_run
        SET status = 'RUNNING', updated_at = now()
        WHERE id IN (
            SELECT id FROM plan_check_run
            WHERE status = 'AVAILABLE'
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, plan_id
    `
    // Execute and return claimed runs
}
```

Modify `CreatePlanCheckRun`: Change default status from `RUNNING` to `AVAILABLE`.

### Scheduler Changes

**File:** `backend/runner/plancheck/scheduler.go`

1. **Remove in-memory tracking** - Delete `RunningPlanChecks` and `RunningPlanCheckRunsCancelFunc` sync.Maps

2. **Update `runOnce()`** - Use atomic claiming instead of querying RUNNING status:
   ```go
   func (s *Scheduler) runOnce(ctx context.Context) {
       claimed, err := s.store.ClaimAvailablePlanCheckRuns(ctx)
       for _, planCheckRun := range claimed {
           go s.runPlanCheckRun(ctx, planCheckRun)
       }
   }
   ```

3. **Simplify cancellation** - Update database status to `CANCELED` directly

### API Changes

No changes needed - store layer handles status transition.

## Files to Modify

| File | Change |
|------|--------|
| `backend/migrator/migration/3.X/XXXX##plan_check_run_ha.sql` | New migration |
| `backend/migrator/migration/LATEST.sql` | Update schema |
| `backend/store/plan_check_run.go` | Add claiming, change default status |
| `backend/runner/plancheck/scheduler.go` | Remove sync.Map, use claiming |

## Migration Strategy

Existing `RUNNING` plan check runs are converted to `AVAILABLE` during migration. They will be re-claimed and re-executed by the scheduler after deployment.

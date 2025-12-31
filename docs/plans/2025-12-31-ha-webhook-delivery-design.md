# HA-Compatible Webhook Delivery Design

**Date:** 2025-12-31
**Status:** Design
**Author:** Danny

## Overview

This design makes pipeline webhook notifications (PIPELINE_FAILED and PIPELINE_COMPLETED) work correctly in High Availability (HA) deployments where multiple Bytebase replicas run simultaneously. It replaces the in-memory `TaskSkippedOrDoneChan` channel with a database-backed idempotent delivery system.

## Problem

The current webhook notification system uses an in-memory channel (`state.TaskSkippedOrDoneChan`) which breaks in HA:

```go
// backend/component/state/state.go
type State struct {
    TaskSkippedOrDoneChan chan int  // In-memory, not shared across replicas
}

// backend/runner/taskrun/running_scheduler.go
s.stateCfg.TaskSkippedOrDoneChan <- task.ID  // Send notification

// backend/runner/taskrun/scheduler.go
case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:  // Receive notification
    s.checkPlanCompletion(ctx, task.PlanID)
```

**HA Issues:**
- Replica A marks task as done, sends to its local channel
- Replica B receives user request, doesn't see anything in its channel
- Both replicas might process same task failure → duplicate webhooks
- Or neither replica sends webhook → missed notification

## Requirements

1. **No spam**: Send PIPELINE_FAILED only once when first task fails
2. **Retry support**: Send new PIPELINE_FAILED when user clicks "Retry Tasks"
3. **Completion**: Send PIPELINE_COMPLETED once when all tasks succeed
4. **HA-safe**: Multiple replicas shouldn't send duplicate webhooks
5. **Simple**: Avoid complex aggregation windows or polling

## Solution: Database-Backed Delivery Log

Use a single database table with PRIMARY KEY constraint to ensure exactly-once delivery across HA replicas.

### Key Insight

- `PIPELINE_FAILED` and `PIPELINE_COMPLETED` are **mutually exclusive** per plan
- User clicking "Retry Tasks" is an explicit action that resets notification state
- Database PRIMARY KEY constraint provides atomic claim mechanism

### Schema

```sql
-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE webhook_delivery_log (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Design choices:**
- `plan_id` as PRIMARY KEY enforces uniqueness automatically
- Keep `event_type` for audit trail (debugging which event was sent)
- No CHECK constraint - application code controls values, comments document them
- No separate indexes needed - PRIMARY KEY provides all we need

## Architecture

### 1. When Task Run Fails

Location: `backend/runner/taskrun/running_scheduler.go`

```go
// Add after task run status is marked as FAILED
func (s *Scheduler) handleTaskRunFailure(ctx context.Context, taskRun *store.TaskRunMessage) {
    task, err := s.store.GetTaskByID(ctx, taskRun.TaskID)
    if err != nil {
        slog.Error("failed to get task for failure notification", log.BBError(err))
        return
    }

    // Try to claim notification (only one replica succeeds)
    claimed, err := s.store.ClaimPipelineFailureNotification(ctx, task.PlanID)
    if err != nil {
        slog.Error("failed to claim pipeline failure notification", log.BBError(err))
        return
    }
    if !claimed {
        // Already sent by this or another replica
        return
    }

    // Get all failed tasks for this plan
    failures, err := s.getFailedTaskRuns(ctx, task.PlanID)
    if err != nil {
        slog.Error("failed to get failed task runs", log.BBError(err))
        return
    }

    // Get plan/project context
    plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
    if err != nil || plan == nil {
        slog.Error("failed to get plan for failure webhook", log.BBError(err))
        return
    }

    project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
    if err != nil || project == nil {
        slog.Error("failed to get project for failure webhook", log.BBError(err))
        return
    }

    // Send webhook
    s.webhookManager.CreateEvent(ctx, &webhook.Event{
        Type:    storepb.Activity_PIPELINE_FAILED,
        Project: webhook.NewProject(project),
        PipelineFailed: &webhook.EventPipelineFailed{
            Rollout:     webhook.NewRollout(plan),
            FailedTasks: failures,
        },
    })
}
```

### 2. When Pipeline Completes

Location: `backend/runner/taskrun/scheduler.go` (existing `checkPlanCompletion` function)

```go
func (s *Scheduler) checkPlanCompletion(ctx context.Context, planID int64) {
    // ... existing task completion check logic ...

    if !allComplete {
        return
    }

    if hasFailures {
        return  // Don't send completion if there were failures
    }

    // Try to claim completion notification (HA-safe)
    claimed, err := s.store.ClaimPipelineCompletionNotification(ctx, planID)
    if err != nil {
        slog.Error("failed to claim pipeline completion notification", log.BBError(err))
        return
    }
    if !claimed {
        return  // Already sent
    }

    // ... existing webhook send code ...
    s.webhookManager.CreateEvent(ctx, &webhook.Event{
        Type:    storepb.Activity_PIPELINE_COMPLETED,
        Project: webhook.NewProject(project),
        RolloutCompleted: &webhook.EventRolloutCompleted{
            Rollout: webhook.NewRollout(plan),
        },
    })
}
```

### 3. When User Clicks "Retry Tasks"

Location: `backend/api/v1/rollout_service.go` (existing `BatchRunTasks` function)

```go
func (s *RolloutService) BatchRunTasks(ctx context.Context, req *connect.Request[v1pb.BatchRunTasksRequest]) (*connect.Response[v1pb.BatchRunTasksResponse], error) {
    // ... existing validation code ...

    // Reset notification state so user gets fresh feedback on retry
    if err := s.store.ResetPipelineNotification(ctx, planID); err != nil {
        slog.Error("failed to reset pipeline notification", log.BBError(err))
        // Don't fail the request - notification is non-critical
    }

    // ... existing CreatePendingTaskRuns call ...

    return connect.NewResponse(&v1pb.BatchRunTasksResponse{}), nil
}
```

## Store Methods

Location: `backend/store/webhook_delivery_log.go` (new file)

```go
package store

import (
    "context"
    "database/sql"
)

// ResetPipelineNotification deletes the notification log for a plan.
// Called when user clicks BatchRunTasks to enable new notifications on retry.
func (s *Store) ResetPipelineNotification(ctx context.Context, planID int64) error {
    query := `DELETE FROM webhook_delivery_log WHERE plan_id = $1`
    _, err := s.db.ExecContext(ctx, query, planID)
    return err
}

// ClaimPipelineFailureNotification attempts to claim the right to send PIPELINE_FAILED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineFailureNotification(ctx context.Context, planID int64) (bool, error) {
    query := `
        INSERT INTO webhook_delivery_log (plan_id, event_type)
        VALUES ($1, 'PIPELINE_FAILED')
        ON CONFLICT (plan_id) DO NOTHING
        RETURNING plan_id
    `

    var id int64
    err := s.db.QueryRowContext(ctx, query, planID).Scan(&id)
    if err == sql.ErrNoRows {
        return false, nil  // Already exists
    }
    return err == nil, err
}

// ClaimPipelineCompletionNotification attempts to claim the right to send PIPELINE_COMPLETED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineCompletionNotification(ctx context.Context, planID int64) (bool, error) {
    query := `
        INSERT INTO webhook_delivery_log (plan_id, event_type)
        VALUES ($1, 'PIPELINE_COMPLETED')
        ON CONFLICT (plan_id) DO NOTHING
        RETURNING plan_id
    `

    var id int64
    err := s.db.QueryRowContext(ctx, query, planID).Scan(&id)
    if err == sql.ErrNoRows {
        return false, nil  // Already exists
    }
    return err == nil, err
}
```

## Example Flow

### Scenario 1: Initial Deployment with Failures

```
Initial deployment (all tasks attempt 0):
→ Task A fails
  → Replica 1: ClaimPipelineFailureNotification(plan=123)
  → INSERT succeeds → Send webhook: "Pipeline failed - Task A failed"

→ Task B fails
  → Replica 2: ClaimPipelineFailureNotification(plan=123)
  → ON CONFLICT (row already exists) → Skip, no duplicate webhook

Result: One webhook sent, no spam ✅
Database: (plan_id=123, event_type='PIPELINE_FAILED')
```

### Scenario 2: User Retries Failed Tasks

```
User clicks "Retry Tasks":
→ BatchRunTasks API called
  → DELETE FROM webhook_delivery_log WHERE plan_id=123
  → CreatePendingTaskRuns (creates attempt=1 for failed tasks)

Database: (empty - row deleted)

→ Task A succeeds (attempt=1)
→ Task B fails (attempt=1)
  → Replica 1: ClaimPipelineFailureNotification(plan=123)
  → INSERT succeeds (no conflict, row was deleted)
  → Send webhook: "Pipeline failed on retry - Task B failed"

Result: User gets feedback on retry ✅
Database: (plan_id=123, event_type='PIPELINE_FAILED')
```

### Scenario 3: Successful Completion

```
User retries again:
→ BatchRunTasks called
  → DELETE FROM webhook_delivery_log WHERE plan_id=123

→ All tasks succeed
  → checkPlanCompletion detects all tasks done, no failures
  → Replica 2: ClaimPipelineCompletionNotification(plan=123)
  → INSERT succeeds
  → Send webhook: "Pipeline completed successfully"

Result: Completion notification sent ✅
Database: (plan_id=123, event_type='PIPELINE_COMPLETED')
```

### Scenario 4: HA Race Condition Handling

```
Two replicas detect same task failure simultaneously:
→ Replica A: ClaimPipelineFailureNotification(plan=123)
  → INSERT INTO webhook_delivery_log VALUES (123, 'PIPELINE_FAILED')
  → Returns plan_id=123 → claimed=true → Sends webhook

→ Replica B: ClaimPipelineFailureNotification(plan=123) (100ms later)
  → INSERT INTO webhook_delivery_log VALUES (123, 'PIPELINE_FAILED')
  → ON CONFLICT (plan_id) DO NOTHING
  → Returns no rows → claimed=false → Skips webhook

Result: Only one webhook sent across HA replicas ✅
```

## Migration

Create migration file: `backend/migrator/migration/<<version>>/<<sequence>>##webhook_delivery_log.sql`

```sql
CREATE TABLE webhook_delivery_log (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Cleanup

After implementation, remove in-memory channel:

1. **Remove from state:**
   ```go
   // backend/component/state/state.go
   // DELETE: TaskSkippedOrDoneChan chan int
   ```

2. **Remove senders:**
   ```go
   // backend/api/v1/rollout_service.go
   // DELETE: s.stateCfg.TaskSkippedOrDoneChan <- task.ID

   // backend/runner/taskrun/running_scheduler.go
   // DELETE: s.stateCfg.TaskSkippedOrDoneChan <- task.ID
   ```

3. **Remove receiver:**
   ```go
   // backend/runner/taskrun/scheduler.go
   // DELETE: case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
   ```

## Benefits

✅ **HA-Compatible**: Database PRIMARY KEY prevents duplicate webhooks across replicas
✅ **No Spam**: One notification per failure/completion phase
✅ **Retry Support**: DELETE on BatchRunTasks resets state for fresh notifications
✅ **Immediate Delivery**: No polling or aggregation windows needed
✅ **Simple Schema**: Single table with PRIMARY KEY, no complex indexes
✅ **Audit Trail**: `event_type` shows which notification was sent
✅ **Follows Existing Pattern**: Same approach as TaskRun session monitoring (commit 564f1e5618)

## Comparison to Alternatives

### Alternative 1: 5-Minute Aggregation Windows
❌ Complex: Needs polling worker, window management
❌ Delayed: 5-minute wait before notification
❌ Race-prone: Multiple replicas might process same window

### Alternative 2: PostgreSQL Advisory Locks
❌ Lock held during webhook HTTP call (can timeout)
❌ More complex than simple INSERT/DELETE

### Alternative 3: In-Memory Channel (Current)
❌ Breaks in HA - not shared across replicas
❌ No persistence - lost on restart

### This Design (Database-Backed Log)
✅ Simple INSERT/DELETE operations
✅ Immediate notifications
✅ HA-safe with database constraints
✅ Stateless - any replica can handle requests

## Future Enhancements

- Add `retry_count` column to track how many times plan was retried
- Add `failed_task_count` to show aggregated metrics
- TTL cleanup for old delivery logs (e.g., delete after 30 days)

## References

- Similar pattern: TaskRun session monitoring HA design (commit 564f1e5618)
- Related: Webhook events redesign (docs/plans/2025-12-30-webhook-events-redesign.md)

# HA-Compatible Webhook Delivery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement database-backed webhook delivery to make PIPELINE_FAILED and PIPELINE_COMPLETED notifications work correctly in HA deployments

**Architecture:** Replace in-memory `TaskSkippedOrDoneChan` channel with `plan_webhook_delivery` table using PRIMARY KEY constraint for atomic claiming across replicas

**Tech Stack:** Go, PostgreSQL, existing webhook infrastructure

---

## Phase 1: Database Schema

### Task 1: Create Migration for plan_webhook_delivery Table

**Files:**
- Create: `backend/migrator/migration/3.14/0016##plan_webhook_delivery.sql`

**Step 1: Create migration file**

Create `backend/migrator/migration/3.14/0016##plan_webhook_delivery.sql`:

```sql
-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE plan_webhook_delivery (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Step 2: Update LATEST.sql**

Add to `backend/migrator/migration/LATEST.sql` after the `plan_check_run` table (around line 231):

```sql
-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE plan_webhook_delivery (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER SEQUENCE plan_webhook_delivery_plan_id_seq RESTART WITH 101;
```

**Step 3: Verify migration files**

Run: `ls backend/migrator/migration/3.14/`
Expected: See `0016##plan_webhook_delivery.sql`

**Step 4: Commit migration**

```bash
git add backend/migrator/migration/3.14/0016##plan_webhook_delivery.sql backend/migrator/migration/LATEST.sql
git commit -m "feat: add plan_webhook_delivery table for HA webhook deduplication

Table provides atomic claim mechanism for webhook delivery across HA replicas
using PRIMARY KEY constraint. Deleted on BatchRunTasks to reset state.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: Store Methods

### Task 2: Implement Store Methods for Webhook Delivery

**Files:**
- Create: `backend/store/plan_webhook_delivery.go`

**Step 1: Create store methods file**

Create `backend/store/plan_webhook_delivery.go`:

```go
package store

import (
	"context"
	"database/sql"
)

// ResetPlanWebhookDelivery deletes the webhook delivery record for a plan.
// Called when user clicks BatchRunTasks to enable new notifications on retry.
func (s *Store) ResetPlanWebhookDelivery(ctx context.Context, planID int64) error {
	query := `DELETE FROM plan_webhook_delivery WHERE plan_id = $1`
	_, err := s.db.ExecContext(ctx, query, planID)
	return err
}

// ClaimPipelineFailureNotification attempts to claim the right to send PIPELINE_FAILED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineFailureNotification(ctx context.Context, planID int64) (bool, error) {
	query := `
		INSERT INTO plan_webhook_delivery (plan_id, event_type)
		VALUES ($1, 'PIPELINE_FAILED')
		ON CONFLICT (plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.db.QueryRowContext(ctx, query, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}

// ClaimPipelineCompletionNotification attempts to claim the right to send PIPELINE_COMPLETED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineCompletionNotification(ctx context.Context, planID int64) (bool, error) {
	query := `
		INSERT INTO plan_webhook_delivery (plan_id, event_type)
		VALUES ($1, 'PIPELINE_COMPLETED')
		ON CONFLICT (plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.db.QueryRowContext(ctx, query, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}
```

**Step 2: Build to verify**

Run: `go build ./backend/store/...`
Expected: Build succeeds

**Step 3: Commit store methods**

```bash
git add backend/store/plan_webhook_delivery.go
git commit -m "feat: add store methods for plan webhook delivery

Implement atomic claim methods for PIPELINE_FAILED and PIPELINE_COMPLETED
webhooks using INSERT...ON CONFLICT for HA safety. Add reset method for
BatchRunTasks.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 3: Integrate into BatchRunTasks

### Task 3: Reset Webhook State on BatchRunTasks

**Files:**
- Modify: `backend/api/v1/rollout_service.go:627-752`

**Step 1: Add reset call to BatchRunTasks**

In `backend/api/v1/rollout_service.go`, add after getting the plan (around line 655):

```go
// Reset notification state so user gets fresh feedback on retry
if err := s.store.ResetPlanWebhookDelivery(ctx, planID); err != nil {
	slog.Error("failed to reset plan webhook delivery", log.BBError(err))
	// Don't fail the request - notification is non-critical
}
```

Full context - the function should look like:

```go
func (s *RolloutService) BatchRunTasks(ctx context.Context, req *connect.Request[v1pb.BatchRunTasksRequest]) (*connect.Response[v1pb.BatchRunTasksResponse], error) {
	request := req.Msg
	// ... existing validation code ...

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		ProjectID: &projectID,
		UID:       &planID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find plan for rollout"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
	}

	// Reset notification state so user gets fresh feedback on retry
	if err := s.store.ResetPlanWebhookDelivery(ctx, planID); err != nil {
		slog.Error("failed to reset plan webhook delivery", log.BBError(err))
		// Don't fail the request - notification is non-critical
	}

	// ... rest of existing code ...
}
```

**Step 2: Build to verify**

Run: `go build ./backend/api/v1/...`
Expected: Build succeeds

**Step 3: Commit BatchRunTasks integration**

```bash
git add backend/api/v1/rollout_service.go
git commit -m "feat: reset webhook delivery state on BatchRunTasks

Delete plan_webhook_delivery row when user retries tasks to enable fresh
notifications on retry attempts.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 4: Send PIPELINE_FAILED on Task Failure

### Task 4: Add Helper to Get Failed Task Runs

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go`

**Step 1: Add helper method to get failed task runs**

Add near the end of `backend/runner/taskrun/scheduler.go`:

```go
// getFailedTaskRuns returns all failed task runs for a plan to include in webhook payload.
func (s *Scheduler) getFailedTaskRuns(ctx context.Context, planID int64) []webhook.FailedTask {
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		slog.Error("failed to list tasks for failed webhook", log.BBError(err))
		return nil
	}

	var failures []webhook.FailedTask
	for _, task := range tasks {
		if task.LatestTaskRunStatus != storepb.TaskRun_FAILED {
			continue
		}

		// Get the latest failed task run
		taskRuns, err := s.store.ListTaskRuns(ctx, &store.TaskRunFind{TaskID: &task.ID})
		if err != nil {
			slog.Error("failed to list task runs", log.BBError(err))
			continue
		}

		// Find the latest failed run
		var latestFailed *store.TaskRunMessage
		for _, tr := range taskRuns {
			if tr.Status == storepb.TaskRun_FAILED {
				if latestFailed == nil || tr.UpdatedAt.After(*latestFailed.UpdatedAt) {
					latestFailed = tr
				}
			}
		}

		if latestFailed == nil {
			continue
		}

		errorMsg := ""
		if latestFailed.Result != nil && latestFailed.Result.Error != "" {
			errorMsg = latestFailed.Result.Error
		}

		failures = append(failures, webhook.FailedTask{
			TaskID:       int64(task.ID),
			TaskName:     task.Name,
			DatabaseName: task.DatabaseName,
			InstanceName: task.Instance.ResourceID,
			ErrorMessage: errorMsg,
			FailedAt:     *latestFailed.UpdatedAt,
		})
	}

	return failures
}
```

**Step 2: Build to verify**

Run: `go build ./backend/runner/taskrun/...`
Expected: Build succeeds

**Step 3: Commit helper method**

```bash
git add backend/runner/taskrun/scheduler.go
git commit -m "feat: add helper to get failed task runs for webhook

Collects all failed tasks in a plan with error messages for webhook payload.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Send PIPELINE_FAILED Webhook on Task Failure

**Files:**
- Modify: `backend/runner/taskrun/running_scheduler.go`

**Step 1: Check where task is marked as FAILED**

Read `backend/runner/taskrun/running_scheduler.go` around line 200-240 to find where task status is set to FAILED.

Expected: Find code like:
```go
if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
    ID:     taskRun.ID,
    Status: storepb.TaskRun_FAILED,
    ...
}); err != nil {
    ...
}
```

**Step 2: Add webhook notification after marking task as FAILED**

After the task is marked as FAILED and before the return statement, add:

```go
// Send PIPELINE_FAILED webhook (HA-safe atomic claim)
s.sendPipelineFailureWebhook(ctx, task)
```

**Step 3: Implement sendPipelineFailureWebhook method**

Add to `backend/runner/taskrun/running_scheduler.go`:

```go
// sendPipelineFailureWebhook attempts to send PIPELINE_FAILED webhook.
// Uses atomic claim to prevent duplicate sends in HA deployments.
func (s *Scheduler) sendPipelineFailureWebhook(ctx context.Context, task *store.TaskMessage) {
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

	// Get plan context
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

	// Get all failed tasks
	failures := s.getFailedTaskRuns(ctx, task.PlanID)
	if len(failures) == 0 {
		slog.Warn("no failed tasks found for pipeline failure webhook", slog.Int64("plan_id", task.PlanID))
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

**Step 4: Build to verify**

Run: `go build ./backend/runner/taskrun/...`
Expected: Build succeeds

**Step 5: Commit pipeline failure webhook**

```bash
git add backend/runner/taskrun/running_scheduler.go
git commit -m "feat: send PIPELINE_FAILED webhook on task failure

Add atomic claim-based webhook delivery when task fails. Only first failure
in a plan triggers notification, preventing spam in HA deployments.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 5: Send PIPELINE_COMPLETED on Success

### Task 6: Update checkPlanCompletion to Send Webhook

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go:112-210`

**Step 1: Add claim check before sending completion webhook**

In `backend/runner/taskrun/scheduler.go`, find the `checkPlanCompletion` function (around line 112).

After the `if hasFailures { return }` check (around line 167), add:

```go
// Try to claim completion notification (HA-safe)
claimed, err := s.store.ClaimPipelineCompletionNotification(ctx, planID)
if err != nil {
	slog.Error("failed to claim pipeline completion notification", log.BBError(err))
	return
}
if !claimed {
	return // Already sent
}
```

Full context - the section should look like:

```go
// Not all tasks complete yet
if !allComplete {
	return
}

// Always clear the failure window when plan completes
s.pipelineEvents.Clear(planID)

// Only send completion webhook if there were no failures
if hasFailures {
	return
}

// Try to claim completion notification (HA-safe)
claimed, err := s.store.ClaimPipelineCompletionNotification(ctx, planID)
if err != nil {
	slog.Error("failed to claim pipeline completion notification", log.BBError(err))
	return
}
if !claimed {
	return // Already sent
}

// ... existing webhook send code continues ...
```

**Step 2: Build to verify**

Run: `go build ./backend/runner/taskrun/...`
Expected: Build succeeds

**Step 3: Commit pipeline completion webhook**

```bash
git add backend/runner/taskrun/scheduler.go
git commit -m "feat: add atomic claim for PIPELINE_COMPLETED webhook

Prevent duplicate completion webhooks in HA deployments using database-backed
claim mechanism.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 6: Cleanup In-Memory Channel

### Task 7: Remove TaskSkippedOrDoneChan Channel

**Files:**
- Modify: `backend/component/state/state.go`
- Modify: `backend/api/v1/rollout_service.go`
- Modify: `backend/runner/taskrun/running_scheduler.go`
- Modify: `backend/runner/taskrun/scheduler.go`

**Step 1: Remove channel from state**

In `backend/component/state/state.go`, remove these lines (around line 23-24):

```go
// DELETE these lines:
// TaskSkippedOrDoneChan is the channel for notifying the task is skipped or done.
TaskSkippedOrDoneChan chan int
```

And in the `New()` function (around line 37):

```go
// DELETE this line:
TaskSkippedOrDoneChan: make(chan int, 1000),
```

**Step 2: Remove sender in rollout_service.go**

In `backend/api/v1/rollout_service.go`, find and remove (around line 836):

```go
// DELETE these lines:
for _, task := range tasksToSkip {
	s.stateCfg.TaskSkippedOrDoneChan <- task.ID
}
```

**Step 3: Remove sender in running_scheduler.go**

In `backend/runner/taskrun/running_scheduler.go`, find and remove (around line 232):

```go
// DELETE this line:
s.stateCfg.TaskSkippedOrDoneChan <- task.ID
```

**Step 4: Remove receiver in scheduler.go**

In `backend/runner/taskrun/scheduler.go`, find the `runTaskCompletionListener` function (around line 82-110).

Remove the entire case statement:

```go
// DELETE this entire case block:
case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
	if err := func() error {
		task, err := s.store.GetTaskByID(ctx, taskUID)
		if err != nil {
			return errors.Wrapf(err, "failed to get task")
		}

		// Check if entire plan is complete and handle webhooks
		s.checkPlanCompletion(ctx, task.PlanID)

		return nil
	}(); err != nil {
		slog.Error("failed to handle task completion", log.BBError(err))
	}
```

**Step 5: Build to verify**

Run: `go build ./backend/...`
Expected: Build succeeds

**Step 6: Commit channel removal**

```bash
git add backend/component/state/state.go backend/api/v1/rollout_service.go backend/runner/taskrun/running_scheduler.go backend/runner/taskrun/scheduler.go
git commit -m "refactor: remove TaskSkippedOrDoneChan in-memory channel

Remove in-memory channel replaced by database-backed webhook delivery.
TaskSkippedOrDoneChan is no longer needed as webhooks are triggered directly
in task failure and completion handlers.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 7: Testing and Verification

### Task 8: Manual Testing

**Step 1: Start local Bytebase**

Run: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
Expected: Server starts, migration runs successfully

**Step 2: Verify table created**

Run: `psql -U bbdev bbdev -c "\d plan_webhook_delivery"`
Expected: See table schema with plan_id PRIMARY KEY

**Step 3: Test PIPELINE_FAILED webhook**

1. Create a plan with tasks
2. Make a task fail
3. Check webhook sent
4. Query: `SELECT * FROM plan_webhook_delivery;`
   Expected: One row with event_type='PIPELINE_FAILED'

**Step 4: Test retry doesn't send duplicate**

1. Fail another task in same plan
2. Check no duplicate webhook
3. Query: `SELECT COUNT(*) FROM plan_webhook_delivery WHERE plan_id = ?;`
   Expected: Still 1 row

**Step 5: Test BatchRunTasks resets state**

1. Click "Retry Tasks"
2. Query: `SELECT * FROM plan_webhook_delivery WHERE plan_id = ?;`
   Expected: 0 rows (deleted)
3. Fail task again
4. Check webhook sent again
5. Query: `SELECT COUNT(*) FROM plan_webhook_delivery WHERE plan_id = ?;`
   Expected: 1 row

**Step 6: Test PIPELINE_COMPLETED**

1. Retry and make all tasks succeed
2. Check PIPELINE_COMPLETED webhook sent
3. Query: `SELECT * FROM plan_webhook_delivery WHERE plan_id = ?;`
   Expected: One row with event_type='PIPELINE_COMPLETED'

**Step 7: Document test results**

Create test notes in commit message for final commit.

---

## Phase 8: Final Commit and Integration

### Task 9: Update Migrator Test

**Files:**
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Find TestLatestVersion**

In `backend/migrator/migrator_test.go`, find `TestLatestVersion` function.

**Step 2: Update latest version**

Find the line that looks like:
```go
latestSchemaVersion := "3.14.15"
```

Increment the last number:
```go
latestSchemaVersion := "3.14.16"
```

**Step 3: Run test**

Run: `go test -v ./backend/migrator/ -run TestLatestVersion`
Expected: Test passes

**Step 4: Commit migrator test update**

```bash
git add backend/migrator/migrator_test.go
git commit -m "test: update latest schema version to 3.14.16

Update for plan_webhook_delivery table migration.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Run Full Build and Lint

**Step 1: Format Go code**

Run: `gofmt -w backend/store/plan_webhook_delivery.go backend/api/v1/rollout_service.go backend/runner/taskrun/scheduler.go backend/runner/taskrun/running_scheduler.go backend/component/state/state.go`
Expected: Files formatted

**Step 2: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors (run multiple times until clean)

**Step 3: Auto-fix lint issues**

Run: `golangci-lint run --fix --allow-parallel-runners`
Expected: Auto-fixable issues resolved

**Step 4: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 5: Commit any lint fixes**

```bash
git add -A
git commit -m "chore: fix linting issues

Auto-fix and manual lint issue resolution.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Completion Checklist

- [ ] Migration created for `plan_webhook_delivery` table
- [ ] LATEST.sql updated with new table
- [ ] Store methods implemented (Claim, Reset)
- [ ] BatchRunTasks resets webhook state
- [ ] PIPELINE_FAILED webhook sent on task failure
- [ ] PIPELINE_COMPLETED webhook sent on plan completion
- [ ] In-memory `TaskSkippedOrDoneChan` removed
- [ ] Migrator test updated
- [ ] Code formatted and linted
- [ ] Manual testing completed
- [ ] All commits have proper messages

## Notes

- Each webhook is sent exactly once per "phase" (initial run, each retry)
- HA-safe: PRIMARY KEY constraint prevents duplicates across replicas
- User action (BatchRunTasks) explicitly resets state
- No polling or background workers needed
- Immediate notification delivery

# Task Scheduler AVAILABLE Status Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add AVAILABLE status to task runs, separating "ready to execute" from "actively executing" to prepare for HA.

**Architecture:** PENDING → AVAILABLE → RUNNING flow. pending_scheduler handles all gating (RunAt, version ordering, database locks, parallel limits) and promotes to AVAILABLE. running_scheduler simply claims AVAILABLE tasks and executes.

**Tech Stack:** Go, PostgreSQL, Protocol Buffers, TypeScript/Vue

---

## Task 1: Add AVAILABLE to Proto

**Files:**
- Modify: `proto/store/store/task_run.proto:13-29`

**Step 1: Add AVAILABLE enum value**

In `proto/store/store/task_run.proto`, add AVAILABLE after SKIPPED:

```proto
  enum Status {
    STATUS_UNSPECIFIED = 0;
    // Task run is queued and waiting to execute.
    PENDING = 1;
    // Task run is currently executing.
    RUNNING = 2;
    // Task run completed successfully.
    DONE = 3;
    // Task run encountered an error and failed.
    FAILED = 4;
    // Task run was canceled by user or system.
    CANCELED = 5;
    // Task run has not started yet.
    NOT_STARTED = 6;
    // Task run was skipped and will not execute.
    SKIPPED = 7;
    // Task run is ready for immediate execution.
    AVAILABLE = 8;
  }
```

**Step 2: Generate proto files**

Run:
```bash
cd proto && buf generate
```

Expected: New generated files in `backend/generated-go/` and `frontend/src/types/proto-es/`

**Step 3: Commit**

```bash
but commit task-scheduler-available-status -m "proto: add AVAILABLE status to TaskRun"
```

---

## Task 2: Database Migration

**Files:**
- Create: `backend/migrator/migration/3.14/0014##task_run_available_status.sql`
- Modify: `backend/migrator/migration/LATEST.sql:257,273`
- Modify: `backend/migrator/migrator_test.go` (update TestLatestVersion)

**Step 1: Create migration file**

Create `backend/migrator/migration/3.14/0014##task_run_available_status.sql`:

```sql
-- Add AVAILABLE status to task_run CHECK constraint
ALTER TABLE task_run DROP CONSTRAINT task_run_status_check;
ALTER TABLE task_run ADD CONSTRAINT task_run_status_check
    CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Update partial index to include AVAILABLE
DROP INDEX idx_task_run_active_status_id;
CREATE INDEX idx_task_run_active_status_id ON task_run(status, id)
    WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');
```

**Step 2: Update LATEST.sql**

In `backend/migrator/migration/LATEST.sql`, line 257, change:

```sql
status text NOT NULL CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
```

And line 273, change:

```sql
CREATE INDEX idx_task_run_active_status_id ON task_run(status, id) WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');
```

**Step 3: Update migrator test**

In `backend/migrator/migrator_test.go`, update TestLatestVersion to expect:
- Version: `3.14.14`
- Path: `migration/3.14/0014##task_run_available_status.sql`

**Step 4: Commit**

```bash
but commit task-scheduler-available-status -m "migration: add AVAILABLE status to task_run"
```

---

## Task 3: Update Store Layer

**Files:**
- Modify: `backend/store/task_run.go:297-304`

**Step 1: Update CreatePendingTaskRuns to check AVAILABLE**

In `backend/store/task_run.go`, the `CreatePendingTaskRuns` function checks for existing active task runs. Update line 303-304 to include AVAILABLE:

```go
		storepb.TaskRun_PENDING.String(), storepb.TaskRun_AVAILABLE.String(), storepb.TaskRun_RUNNING.String(), storepb.TaskRun_DONE.String())
```

**Step 2: Run tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/store -run ^TestTaskRun
```

**Step 3: Commit**

```bash
but commit task-scheduler-available-status -m "store: include AVAILABLE in active task run check"
```

---

## Task 4: Update FindBlockingTaskByVersion

**Files:**
- Modify: `backend/store/task.go:107-160`

**Step 1: Update query to check for AVAILABLE status**

The `FindBlockingTaskByVersion` function checks if there are blocking tasks. It currently considers any task whose latest status is not DONE as blocking. This is correct - AVAILABLE tasks should also block.

No change needed - the query already blocks on anything that's not DONE.

**Step 2: Verify logic**

Run:
```bash
go build ./backend/...
```

---

## Task 5: Refactor pending_scheduler - Add Gating Logic

**Files:**
- Modify: `backend/runner/taskrun/pending_scheduler.go`

**Step 1: Add helper function for database mutual exclusion check**

Add after line 16:

```go
// checkDatabaseMutualExclusion checks if there's already an AVAILABLE or RUNNING task on the database.
// Returns true if the task can proceed (no conflict), false otherwise.
func (s *Scheduler) checkDatabaseMutualExclusion(ctx context.Context, task *store.TaskMessage, availableDBs map[string]bool) (bool, *int, error) {
	if task.DatabaseName == nil {
		return true, nil, nil
	}
	if !isSequentialTask(task) {
		return true, nil, nil
	}

	databaseKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)

	// Check in-memory tracking first (tasks promoted this round)
	if availableDBs[databaseKey] {
		return false, nil, nil
	}

	// Check database for AVAILABLE or RUNNING tasks on the same database
	activeTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to list active task runs")
	}

	for _, tr := range activeTaskRuns {
		activeTask, err := s.store.GetTaskByID(ctx, tr.TaskUID)
		if err != nil {
			return false, nil, errors.Wrapf(err, "failed to get task")
		}
		if activeTask.DatabaseName == nil {
			continue
		}
		if !isSequentialTask(activeTask) {
			continue
		}
		if getDatabaseKey(activeTask.InstanceID, *activeTask.DatabaseName) == databaseKey {
			return false, &activeTask.ID, nil
		}
	}

	return true, nil, nil
}
```

**Step 2: Add helper function for parallel limit check**

Add after the previous function:

```go
// checkParallelLimit checks if promoting this task would exceed the parallel task limit.
// Returns true if the task can proceed, false otherwise.
func (s *Scheduler) checkParallelLimit(ctx context.Context, task *store.TaskMessage, rolloutCounts map[int64]int) (bool, error) {
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return false, errors.Errorf("plan %v not found", task.PlanID)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return false, errors.Errorf("project %v not found", plan.ProjectID)
	}

	maxParallel := int(project.Setting.GetParallelTasksPerRollout())
	if maxParallel <= 0 {
		// No limit
		return true, nil
	}

	// Count current AVAILABLE + RUNNING for this rollout
	activeTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		PlanUID: &task.PlanID,
		Status:  &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list active task runs")
	}

	currentCount := len(activeTaskRuns) + rolloutCounts[task.PlanID]
	return currentCount < maxParallel, nil
}
```

**Step 3: Rewrite schedulePendingTaskRuns with in-memory tracking**

Replace the `schedulePendingTaskRuns` function (lines 41-55):

```go
func (s *Scheduler) schedulePendingTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_PENDING},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending tasks")
	}

	// Track what we've promoted this round to avoid over-committing
	availableDBs := map[string]bool{}   // database key -> has AVAILABLE this round
	rolloutCounts := map[int64]int{}    // plan_id -> count promoted this round

	for _, taskRun := range taskRuns {
		promoted, err := s.schedulePendingTaskRun(ctx, taskRun, availableDBs, rolloutCounts)
		if err != nil {
			slog.Error("failed to schedule pending task run", log.BBError(err))
			continue
		}
		if promoted {
			task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
			if err != nil {
				slog.Error("failed to get task after promotion", log.BBError(err))
				continue
			}
			if task.DatabaseName != nil && isSequentialTask(task) {
				availableDBs[getDatabaseKey(task.InstanceID, *task.DatabaseName)] = true
			}
			rolloutCounts[task.PlanID]++
		}
	}

	return nil
}
```

**Step 4: Rewrite schedulePendingTaskRun with all gating logic**

Replace the `schedulePendingTaskRun` function (lines 57-129):

```go
func (s *Scheduler) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, availableDBs map[string]bool, rolloutCounts map[int64]int) (bool, error) {
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get task")
	}

	// Check 1: RunAt time
	if taskRun.RunAt != nil && time.Now().Before(*taskRun.RunAt) {
		return false, nil
	}

	// Check 2: Version ordering (blocking tasks with smaller versions)
	if task.DatabaseName != nil {
		schemaVersion := task.Payload.GetSchemaVersion()
		if schemaVersion != "" {
			maybeTaskID, err := s.store.FindBlockingTaskByVersion(ctx, task.PlanID, task.InstanceID, *task.DatabaseName, schemaVersion)
			if err != nil {
				return false, errors.Wrapf(err, "failed to find blocking versioned tasks")
			}
			if maybeTaskID != nil {
				s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
					ReportTime: timestamppb.Now(),
					WaitingCause: &storepb.SchedulerInfo_WaitingCause{
						Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
							TaskUid: int32(*maybeTaskID),
						},
					},
				})
				return false, nil
			}
		}
	}

	// Check 3: Database mutual exclusion (for sequential tasks)
	canProceed, blockingTaskID, err := s.checkDatabaseMutualExclusion(ctx, task, availableDBs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check database mutual exclusion")
	}
	if !canProceed {
		if blockingTaskID != nil {
			s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
				ReportTime: timestamppb.Now(),
				WaitingCause: &storepb.SchedulerInfo_WaitingCause{
					Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
						TaskUid: int32(*blockingTaskID),
					},
				},
			})
		}
		return false, nil
	}

	// Check 4: Parallel task limit per rollout
	withinLimit, err := s.checkParallelLimit(ctx, task, rolloutCounts)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check parallel limit")
	}
	if !withinLimit {
		s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
			ReportTime: timestamppb.Now(),
			WaitingCause: &storepb.SchedulerInfo_WaitingCause{
				Cause: &storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit{
					ParallelTasksLimit: true,
				},
			},
		})
		return false, nil
	}

	// All checks passed - promote to AVAILABLE
	s.stateCfg.TaskRunSchedulerInfo.Delete(taskRun.ID)
	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:      taskRun.ID,
		Updater: common.SystemBotEmail,
		Status:  storepb.TaskRun_AVAILABLE,
	}); err != nil {
		return false, errors.Wrapf(err, "failed to update task run status to available")
	}

	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_WAITING,
		},
	})

	// Tickle the running scheduler
	select {
	case s.stateCfg.TaskRunTickleChan <- 0:
	default:
	}

	return true, nil
}
```

**Step 5: Add import for isSequentialTask**

The `isSequentialTask` function is in `running_scheduler.go`. Move it to a shared location or add the import. Since it's in the same package, no import needed.

**Step 6: Build and verify**

```bash
go build ./backend/runner/taskrun/...
```

**Step 7: Commit**

```bash
but commit task-scheduler-available-status -m "pending_scheduler: add all gating logic for PENDING->AVAILABLE"
```

---

## Task 6: Simplify running_scheduler

**Files:**
- Modify: `backend/runner/taskrun/running_scheduler.go`

**Step 1: Rewrite scheduleRunningTaskRuns to query AVAILABLE**

Replace `scheduleRunningTaskRuns` function (lines 49-90):

```go
func (s *Scheduler) scheduleRunningTaskRuns(ctx context.Context) error {
	// Query AVAILABLE tasks (ready for execution)
	availableTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list available task runs")
	}

	for _, taskRun := range availableTaskRuns {
		if err := s.claimAndExecuteTaskRun(ctx, taskRun); err != nil {
			slog.Error("failed to claim and execute task run", log.BBError(err))
		}
	}

	// Also re-execute orphaned RUNNING tasks (for restart recovery)
	runningTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list running task runs")
	}

	for _, taskRun := range runningTaskRuns {
		// Skip if already executing in this instance
		if _, ok := s.stateCfg.RunningTaskRuns.Load(taskRun.ID); ok {
			continue
		}
		// Re-execute orphaned RUNNING task
		if err := s.executeTaskRun(ctx, taskRun); err != nil {
			slog.Error("failed to re-execute orphaned task run", log.BBError(err))
		}
	}

	return nil
}
```

**Step 2: Add claimAndExecuteTaskRun function**

Add new function after scheduleRunningTaskRuns:

```go
// claimAndExecuteTaskRun attempts to atomically claim an AVAILABLE task and execute it.
func (s *Scheduler) claimAndExecuteTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// Optimistic locking: attempt to claim by updating AVAILABLE -> RUNNING
	claimed, err := s.store.ClaimAvailableTaskRun(ctx, taskRun.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to claim task run")
	}
	if !claimed {
		// Another instance claimed it
		return nil
	}

	return s.executeTaskRun(ctx, taskRun)
}

// executeTaskRun executes a task run that is already in RUNNING status.
func (s *Scheduler) executeTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return errors.Wrapf(err, "failed to get instance")
	}
	if instance.Deleted {
		return errors.Errorf("instance %v is deleted", task.InstanceID)
	}

	executor, ok := s.executorMap[task.Type]
	if !ok {
		return errors.Errorf("executor not found for task type: %v", task.Type)
	}

	// Update started_at
	if err := s.store.UpdateTaskRunStartAt(ctx, taskRun.ID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}

	// Register as running
	s.stateCfg.RunningTaskRuns.Store(taskRun.ID, true)

	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_RUNNING,
		},
	})

	go s.runTaskRunOnce(ctx, taskRun, task, executor)
	return nil
}
```

**Step 3: Remove old scheduleRunningTaskRun function**

Delete the old `scheduleRunningTaskRun` function (lines 92-209) - it's replaced by the simplified logic above.

**Step 4: Update runTaskRunOnce to remove in-memory state tracking**

In `runTaskRunOnce` function, remove the database migration tracking since it's no longer needed. Update the defer block (lines 221-228):

```go
	defer func() {
		s.stateCfg.RunningTaskRunsCancelFunc.Delete(taskRun.ID)
	}()
```

Remove references to:
- `s.stateCfg.RunningDatabaseMigration`
- `s.stateCfg.RolloutOutstandingTasks`

**Step 5: Build and verify**

```bash
go build ./backend/runner/taskrun/...
```

**Step 6: Commit**

```bash
but commit task-scheduler-available-status -m "running_scheduler: simplify to claim AVAILABLE and execute"
```

---

## Task 7: Add ClaimAvailableTaskRun to Store

**Files:**
- Modify: `backend/store/task_run.go`

**Step 1: Add ClaimAvailableTaskRun function**

Add after `UpdateTaskRunStatus`:

```go
// ClaimAvailableTaskRun attempts to atomically claim an AVAILABLE task run by updating it to RUNNING.
// Returns true if the claim succeeded, false if another process claimed it first.
func (s *Store) ClaimAvailableTaskRun(ctx context.Context, taskRunID int) (bool, error) {
	q := qb.Q().Space(`
		UPDATE task_run
		SET status = ?, updated_at = now()
		WHERE id = ? AND status = ?
	`, storepb.TaskRun_RUNNING.String(), taskRunID, storepb.TaskRun_AVAILABLE.String())

	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to claim task run")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to get rows affected")
	}

	return rowsAffected == 1, nil
}
```

**Step 2: Build and verify**

```bash
go build ./backend/store/...
```

**Step 3: Commit**

```bash
but commit task-scheduler-available-status -m "store: add ClaimAvailableTaskRun for optimistic locking"
```

---

## Task 8: Remove Unused State Fields

**Files:**
- Modify: `backend/component/state/state.go`

**Step 1: Remove RunningDatabaseMigration field**

This field is no longer needed since gating is done in pending_scheduler via DB queries. Remove line 22-23:

```go
	// RunningDatabaseMigration is the taskUID of the running migration on the database.
	RunningDatabaseMigration sync.Map // map[databaseKey]taskUID
```

**Step 2: Remove RolloutOutstandingTasks field**

This is no longer needed. Remove lines 31-32:

```go
	// RolloutOutstandingTasks is the maximum number of tasks per rollout.
	RolloutOutstandingTasks *resourceLimiter
```

Also update the `New()` function to remove initialization.

**Step 3: Build and verify all usages are removed**

```bash
go build ./backend/...
```

**Step 4: Commit**

```bash
but commit task-scheduler-available-status -m "state: remove unused RunningDatabaseMigration and RolloutOutstandingTasks"
```

---

## Task 9: Update BatchCancelTaskRuns

**Files:**
- Modify: `backend/store/task_run.go:369-389`

**Step 1: Update to also cancel AVAILABLE tasks**

The batch cancel should work for AVAILABLE tasks too. No change needed - it updates by ID regardless of status.

**Step 2: Verify the rollout service uses correct statuses**

Check `backend/api/v1/rollout_service.go` for any status checks that need AVAILABLE added.

---

## Task 10: Frontend - Add AVAILABLE Status

**Files:**
- Modify: `frontend/src/components/RolloutV1/constants/task.ts`
- Modify: `frontend/src/components/Plan/constants/task.ts`
- Modify: `frontend/src/components/RolloutV1/components/utils/taskStatus.ts`

**Step 1: Update TASK_STATUS_FILTERS in RolloutV1**

In `frontend/src/components/RolloutV1/constants/task.ts`:

```typescript
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.RUNNING,
  Task_Status.AVAILABLE,
  Task_Status.FAILED,
  Task_Status.PENDING,
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];
```

**Step 2: Update TASK_STATUS_FILTERS in Plan**

In `frontend/src/components/Plan/constants/task.ts`:

```typescript
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.RUNNING,
  Task_Status.AVAILABLE,
  Task_Status.FAILED,
  Task_Status.PENDING,
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];
```

**Step 3: Update taskStatus.ts**

In `frontend/src/components/RolloutV1/components/utils/taskStatus.ts`, add AVAILABLE to ACTIONABLE_STATUSES:

```typescript
const ACTIONABLE_STATUSES = new Set<Task_Status>([
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.AVAILABLE,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
]);
```

**Step 4: Commit**

```bash
but commit task-scheduler-available-status -m "frontend: add AVAILABLE status to task filters"
```

---

## Task 11: Frontend - Add Locale Strings

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/es-ES.json`

**Step 1: Add translations**

Find the task status section in each locale file and add AVAILABLE translation.

For `en-US.json`:
```json
"available": "Available"
```

For `zh-CN.json`:
```json
"available": "就绪"
```

**Step 2: Commit**

```bash
but commit task-scheduler-available-status -m "i18n: add AVAILABLE status translations"
```

---

## Task 12: Lint and Test

**Step 1: Run Go linter**

```bash
golangci-lint run --allow-parallel-runners
```

Fix any issues. Run repeatedly until clean.

**Step 2: Run frontend checks**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend biome:check
```

**Step 3: Run backend build**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 4: Commit any fixes**

```bash
but commit task-scheduler-available-status -m "chore: fix lint issues"
```

---

## Task 13: Final Review and Push

**Step 1: Review all changes**

```bash
but status
git diff main...HEAD
```

**Step 2: Push branch**

```bash
but push task-scheduler-available-status
```

**Step 3: Create PR**

```bash
gh pr create --base main --head task-scheduler-available-status \
  --title "refactor: add AVAILABLE status to task scheduler" \
  --body "$(cat <<'EOF'
## Summary
- Add AVAILABLE status between PENDING and RUNNING for task runs
- Centralize all gating logic in pending_scheduler (RunAt, version ordering, database locks, parallel limits)
- Simplify running_scheduler to just claim AVAILABLE tasks and execute
- Prepare foundation for HA with optimistic locking

## Test plan
- [ ] Verify PENDING tasks transition to AVAILABLE when all constraints are met
- [ ] Verify AVAILABLE tasks are claimed and executed
- [ ] Verify database mutual exclusion works (only one DDL/SDL per database)
- [ ] Verify parallel limit is respected
- [ ] Verify frontend shows AVAILABLE status correctly

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

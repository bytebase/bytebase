# Plan Check Run HA Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the plan check run scheduler HA-compatible using database-level atomic claiming.

**Architecture:** Add `AVAILABLE` status to plan check runs. Scheduler atomically claims `AVAILABLE` runs using `FOR UPDATE SKIP LOCKED`, transitioning them to `RUNNING`. Remove in-memory tracking from bus.

**Tech Stack:** Go, PostgreSQL, protobuf

---

## Task 1: Add AVAILABLE status constant to store

**Files:**
- Modify: `backend/store/plan_check_run.go:18-27`

**Step 1: Add the new status constant**

In `backend/store/plan_check_run.go`, add `PlanCheckRunStatusAvailable` after line 18:

```go
const (
	// PlanCheckRunStatusAvailable is the plan check status for AVAILABLE.
	PlanCheckRunStatusAvailable PlanCheckRunStatus = "AVAILABLE"
	// PlanCheckRunStatusRunning is the plan check status for RUNNING.
	PlanCheckRunStatusRunning PlanCheckRunStatus = "RUNNING"
	// PlanCheckRunStatusDone is the plan check status for DONE.
	PlanCheckRunStatusDone PlanCheckRunStatus = "DONE"
	// PlanCheckRunStatusFailed is the plan check status for FAILED.
	PlanCheckRunStatusFailed PlanCheckRunStatus = "FAILED"
	// PlanCheckRunStatusCanceled is the plan check status for CANCELED.
	PlanCheckRunStatusCanceled PlanCheckRunStatus = "CANCELED"
)
```

**Step 2: Verify build**

Run: `go build ./backend/store/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
but commit plan-check-ha -m "feat(store): add AVAILABLE status for plan check runs"
```

---

## Task 2: Add ClaimAvailablePlanCheckRuns function to store

**Files:**
- Modify: `backend/store/plan_check_run.go`

**Step 1: Add ClaimedPlanCheckRun struct and claiming function**

Add after `BatchCancelPlanCheckRuns` function (after line 189):

```go
// ClaimedPlanCheckRun represents a plan check run that was atomically claimed.
type ClaimedPlanCheckRun struct {
	UID     int
	PlanUID int64
}

// ClaimAvailablePlanCheckRuns atomically claims all AVAILABLE plan check runs by updating them to RUNNING
// and returns the claimed UIDs. Uses FOR UPDATE SKIP LOCKED to allow concurrent schedulers to claim different runs.
func (s *Store) ClaimAvailablePlanCheckRuns(ctx context.Context) ([]*ClaimedPlanCheckRun, error) {
	q := qb.Q().Space(`
		UPDATE plan_check_run
		SET status = ?, updated_at = now()
		WHERE id IN (
			SELECT id FROM plan_check_run
			WHERE status = ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, plan_id
	`, PlanCheckRunStatusRunning, PlanCheckRunStatusAvailable)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim plan check runs")
	}
	defer rows.Close()

	var claimed []*ClaimedPlanCheckRun
	for rows.Next() {
		var c ClaimedPlanCheckRun
		if err := rows.Scan(&c.UID, &c.PlanUID); err != nil {
			return nil, err
		}
		claimed = append(claimed, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return claimed, nil
}
```

**Step 2: Verify build**

Run: `go build ./backend/store/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
but commit plan-check-ha -m "feat(store): add ClaimAvailablePlanCheckRuns for HA scheduling"
```

---

## Task 3: Update CreatePlanCheckRun to use AVAILABLE status

**Files:**
- Modify: `backend/store/plan_check_run.go:50-68`

**Step 1: Modify CreatePlanCheckRun to always use AVAILABLE**

Replace the `CreatePlanCheckRun` function to ignore the passed status and always use AVAILABLE:

```go
// CreatePlanCheckRun creates or replaces the plan check run for a plan.
// Always creates with AVAILABLE status for HA-safe scheduling.
func (s *Store) CreatePlanCheckRun(ctx context.Context, create *PlanCheckRunMessage) error {
	result, err := protojson.Marshal(create.Result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal result")
	}

	query := `
		INSERT INTO plan_check_run (plan_id, status, result)
		VALUES ($1, $2, $3)
		ON CONFLICT (plan_id) DO UPDATE SET
			status = EXCLUDED.status,
			result = EXCLUDED.result,
			updated_at = now()
	`
	if _, err := s.GetDB().ExecContext(ctx, query, create.PlanUID, PlanCheckRunStatusAvailable, result); err != nil {
		return errors.Wrapf(err, "failed to upsert plan check run")
	}
	return nil
}
```

**Step 2: Verify build**

Run: `go build ./backend/store/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
but commit plan-check-ha -m "feat(store): CreatePlanCheckRun always uses AVAILABLE status"
```

---

## Task 4: Remove in-memory plan check tracking from bus

**Files:**
- Modify: `backend/component/bus/bus.go:19-22`

**Step 1: Remove RunningPlanChecks and RunningPlanCheckRunsCancelFunc**

Remove these two lines from the Bus struct:

```go
// RunningPlanChecks is the set of running plan checks.
RunningPlanChecks sync.Map
// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc
```

The Bus struct should now look like:

```go
// Bus is the message bus for all in-memory communication within the server.
type Bus struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan int64 // issue UID

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan int64

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan int64
}
```

**Step 2: Verify build fails (expected - scheduler still references these)**

Run: `go build ./backend/...`
Expected: Build fails with references to deleted fields

**Step 3: Commit (partial - will fix in next task)**

```bash
but commit plan-check-ha -m "refactor(bus): remove in-memory plan check tracking"
```

---

## Task 5: Update scheduler to use atomic claiming

**Files:**
- Modify: `backend/runner/plancheck/scheduler.go`

**Step 1: Update runOnce to use claiming**

Replace the `runOnce` function:

```go
func (s *Scheduler) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Plan check scheduler PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	claimed, err := s.store.ClaimAvailablePlanCheckRuns(ctx)
	if err != nil {
		slog.Error("failed to claim available plan check runs", log.BBError(err))
		return
	}

	for _, c := range claimed {
		go s.runPlanCheckRun(ctx, c.UID, c.PlanUID)
	}
}
```

**Step 2: Update runPlanCheckRun signature and remove in-memory tracking**

Replace the `runPlanCheckRun` function:

```go
func (s *Scheduler) runPlanCheckRun(ctx context.Context, uid int, planUID int64) {
	// Fetch plan to derive check targets at runtime
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planUID})
	if err != nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, err.Error())
		return
	}
	if plan == nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, "plan not found")
		return
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, err.Error())
		return
	}
	if project == nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, "project not found")
		return
	}

	// Get database group if needed (for spec expansion)
	databaseGroup, err := s.getDatabaseGroupForPlan(ctx, plan)
	if err != nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, err.Error())
		return
	}

	// Derive check targets from plan
	targets, err := DeriveCheckTargets(project, plan, databaseGroup)
	if err != nil {
		s.markPlanCheckRunFailed(ctx, uid, planUID, err.Error())
		return
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, target := range targets {
		targetResults, targetErr := s.executor.RunForTarget(ctx, target)
		if targetErr != nil {
			err = targetErr
			break
		}
		results = append(results, targetResults...)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.markPlanCheckRunCanceled(ctx, uid, planUID, err.Error())
		} else {
			s.markPlanCheckRunFailed(ctx, uid, planUID, err.Error())
		}
	} else {
		s.markPlanCheckRunDone(ctx, uid, planUID, results)
	}
}
```

**Step 3: Update helper functions to use uid and planUID parameters**

Replace the three mark functions:

```go
func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, uid int, planUID int64, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results: results,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusDone,
		result,
		uid,
	); err != nil {
		slog.Error("failed to mark plan check run done", log.BBError(err))
		return
	}

	// Auto-create rollout if plan checks pass
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planUID})
	if err != nil {
		slog.Error("failed to get issue for approval check after plan check",
			slog.Int("plan_id", int(planUID)),
			log.BBError(err))
		return
	}
	if issue != nil && issue.PlanUID != nil {
		// Trigger approval finding
		s.bus.ApprovalCheckChan <- int64(issue.UID)
		// Trigger rollout creation (existing behavior)
		s.bus.RolloutCreationChan <- planUID
	}
}

func (s *Scheduler) markPlanCheckRunFailed(ctx context.Context, uid int, planUID int64, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error: reason,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusFailed,
		result,
		uid,
	); err != nil {
		slog.Error("failed to mark plan check run failed", log.BBError(err))
	}
}

func (s *Scheduler) markPlanCheckRunCanceled(ctx context.Context, uid int, planUID int64, reason string) {
	result := &storepb.PlanCheckRunResult{
		Error: reason,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusCanceled,
		result,
		uid,
	); err != nil {
		slog.Error("failed to mark plan check run canceled", log.BBError(err))
	}
}
```

**Step 4: Verify build**

Run: `go build ./backend/...`
Expected: Build succeeds

**Step 5: Commit**

```bash
but commit plan-check-ha -m "refactor(plancheck): use atomic claiming instead of in-memory tracking"
```

---

## Task 6: Add database migration

**Files:**
- Create: `backend/migrator/migration/3.14/0022##plan_check_run_ha.sql`
- Modify: `backend/migrator/migration/LATEST.sql:218,225`

**Step 1: Create migration file**

Create `backend/migrator/migration/3.14/0022##plan_check_run_ha.sql`:

```sql
-- Add AVAILABLE status for HA-compatible plan check scheduling.
-- Uses FOR UPDATE SKIP LOCKED pattern for atomic claiming.

-- Update status constraint to include AVAILABLE
ALTER TABLE plan_check_run
    DROP CONSTRAINT plan_check_run_status_check,
    ADD CONSTRAINT plan_check_run_status_check
        CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Convert existing RUNNING to AVAILABLE (will be re-claimed after deployment)
UPDATE plan_check_run SET status = 'AVAILABLE' WHERE status = 'RUNNING';

-- Update index to include AVAILABLE for efficient claiming
DROP INDEX IF EXISTS idx_plan_check_run_active_status;
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');
```

**Step 2: Update LATEST.sql constraint**

In `backend/migrator/migration/LATEST.sql`, change line 218 from:

```sql
    status text NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
```

to:

```sql
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
```

**Step 3: Update LATEST.sql index**

In `backend/migrator/migration/LATEST.sql`, change line 225 from:

```sql
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status = 'RUNNING';
```

to:

```sql
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');
```

**Step 4: Commit**

```bash
but commit plan-check-ha -m "chore(migration): add AVAILABLE status for plan check runs"
```

---

## Task 7: Update migrator test version

**Files:**
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Find and update TestLatestVersion**

Search for `TestLatestVersion` and update the migration count to include the new migration file.

Run: `grep -n "TestLatestVersion\|3.14" backend/migrator/migrator_test.go`

Update the `3.14` entry to include the new migration count (should be 22 now).

**Step 2: Verify test**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$`
Expected: Test passes

**Step 3: Commit**

```bash
but commit plan-check-ha -m "test(migrator): update version for plan check run HA migration"
```

---

## Task 8: Run linter and fix issues

**Step 1: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners`

**Step 2: Fix any issues reported**

Common issues to watch for:
- Unused parameters (prefix with `_`)
- Import ordering

**Step 3: Run lint again until clean**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No issues

**Step 4: Commit if any fixes**

```bash
but commit plan-check-ha -m "fix: address linter issues"
```

---

## Task 9: Build and verify

**Step 1: Full backend build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run related tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run PlanCheck`
Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/runner/plancheck/...`

**Step 3: Commit if any fixes needed**

---

## Task 10: Create PR

**Step 1: Push branch**

```bash
but push plan-check-ha
```

**Step 2: Create PR**

```bash
gh pr create --base main --head plan-check-ha \
  --title "feat: make plan check run scheduler HA compatible" \
  --body "$(cat <<'EOF'
## Summary
- Add `AVAILABLE` status to plan check runs
- Implement atomic claiming with `FOR UPDATE SKIP LOCKED`
- Remove in-memory tracking from bus component
- Follows the same HA pattern as task run scheduler

## Test plan
- [ ] Verify plan checks still execute correctly in single-instance mode
- [ ] Verify migration applies cleanly
- [ ] Verify existing RUNNING plan checks are re-executed after deployment

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

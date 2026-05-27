# Task Run Log Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add consistent structured task-run log context for backend task-run execution logs.

**Architecture:** Add one small helper in `backend/runner/taskrun` that returns the standard `slog` attributes: `project`, `task_run_id`, and `replica_id`. Use scoped `slog.Logger` values at task-run scheduler and execution boundaries, without storing loggers in `context.Context` and without changing the executor interface.

**Tech Stack:** Go, `log/slog`, existing Bytebase task-run scheduler, `stretchr/testify` for focused unit tests.

---

## File Structure

- Create `backend/runner/taskrun/log_context.go`
  - Owns the reusable task-run log attribute helper.
- Create `backend/runner/taskrun/log_context_test.go`
  - Verifies helper output and field names.
- Modify `backend/runner/taskrun/running_scheduler.go`
  - Replaces ambiguous task-run `id` log attributes with scoped loggers carrying `project`, `task_run_id`, and `replica_id`.
  - Keeps event-specific fields out unless needed for that event.

No database, proto, frontend, or executor interface files should change.

### Task 1: Add Task-Run Log Attribute Helper

**Files:**
- Create: `backend/runner/taskrun/log_context_test.go`
- Create: `backend/runner/taskrun/log_context.go`

- [ ] **Step 1: Write the failing helper test**

Create `backend/runner/taskrun/log_context_test.go`:

```go
package taskrun

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskRunLogAttrs(t *testing.T) {
	attrs := taskRunLogAttrs("project-a", 123, "replica-1")

	require.Equal(t, []slog.Attr{
		slog.String("project", "project-a"),
		slog.Int64("task_run_id", 123),
		slog.String("replica_id", "replica-1"),
	}, attrs)
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test -v -count=1 ./backend/runner/taskrun -run ^TestTaskRunLogAttrs$
```

Expected: FAIL with `undefined: taskRunLogAttrs`.

- [ ] **Step 3: Implement the helper**

Create `backend/runner/taskrun/log_context.go`:

```go
package taskrun

import "log/slog"

func taskRunLogAttrs(projectID string, taskRunUID int64, replicaID string) []slog.Attr {
	return []slog.Attr{
		slog.String("project", projectID),
		slog.Int64("task_run_id", taskRunUID),
		slog.String("replica_id", replicaID),
	}
}
```

- [ ] **Step 4: Run the focused test and verify it passes**

Run:

```bash
go test -v -count=1 ./backend/runner/taskrun -run ^TestTaskRunLogAttrs$
```

Expected: PASS.

- [ ] **Step 5: Commit the helper**

Run:

```bash
gofmt -w backend/runner/taskrun/log_context.go backend/runner/taskrun/log_context_test.go
git add backend/runner/taskrun/log_context.go backend/runner/taskrun/log_context_test.go
git commit -m "chore: add task run log context helper"
```

### Task 2: Use Scoped Loggers in Running Task-Run Scheduler

**Files:**
- Modify: `backend/runner/taskrun/running_scheduler.go`

- [ ] **Step 1: Update claim execution failure logging**

In `scheduleRunningTaskRuns`, replace the loop body:

```go
for _, c := range claimed {
	if err := s.executeTaskRun(ctx, c.ProjectID, c.TaskRunUID, c.TaskUID); err != nil {
		slog.Error("failed to execute task run", slog.Int64("id", c.TaskRunUID), log.BBError(err))
	}
}
```

with:

```go
for _, c := range claimed {
	logger := slog.With(taskRunLogAttrs(c.ProjectID, c.TaskRunUID, s.profile.ReplicaID)...)
	if err := s.executeTaskRun(ctx, c.ProjectID, c.TaskRunUID, c.TaskUID); err != nil {
		logger.Error("failed to execute task run", log.BBError(err))
	}
}
```

- [ ] **Step 2: Add a scoped logger in `executeTaskRun`**

In `executeTaskRun`, add this as the first statement in the function body:

```go
logger := slog.With(taskRunLogAttrs(projectID, taskRunUID, s.profile.ReplicaID)...)
```

Then replace the drift validation warning:

```go
slog.Warn("task run blocked by drift validation",
	slog.Int64("id", task.ID),
	slog.String("type", task.Type.String()),
	log.BBError(err),
)
```

with:

```go
logger.Warn("task run blocked by drift validation", log.BBError(err))
```

- [ ] **Step 3: Add a scoped logger in `runTaskRunOnce`**

In `runTaskRunOnce`, add this as the first statement in the function body:

```go
logger := slog.With(taskRunLogAttrs(task.ProjectID, taskRunUID, s.profile.ReplicaID)...)
```

Then replace the panic recovery log:

```go
slog.Error("Task scheduler V2 runTaskRunOnce PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
```

with:

```go
logger.Error("Task scheduler V2 runTaskRunOnce PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
```

- [ ] **Step 4: Replace cancellation and failure logs**

Replace:

```go
slog.Warn("task run is canceled",
	slog.Int64("id", task.ID),
	slog.String("type", task.Type.String()),
	log.BBError(err),
)
```

with:

```go
logger.Warn("task run is canceled", log.BBError(err))
```

Replace:

```go
slog.Error("Failed to mark task as CANCELED",
	slog.Int64("id", task.ID),
	log.BBError(err),
)
```

with:

```go
logger.Error("Failed to mark task as CANCELED", log.BBError(err))
```

Replace:

```go
slog.Warn("task run failed",
	slog.Int64("id", task.ID),
	slog.String("type", task.Type.String()),
	log.BBError(err),
)
```

with:

```go
logger.Warn("task run failed", log.BBError(err))
```

Replace:

```go
slog.Error("Failed to mark task as FAILED",
	slog.Int64("id", task.ID),
	log.BBError(err),
)
```

with:

```go
logger.Error("Failed to mark task as FAILED", log.BBError(err))
```

- [ ] **Step 5: Replace webhook follow-up failure logs**

Inside the failed task-run branch, replace:

```go
slog.Error("failed to claim pipeline failure notification", log.BBError(err))
```

with:

```go
logger.Error("failed to claim pipeline failure notification", log.BBError(err))
```

Replace:

```go
slog.Error("failed to get plan for failure webhook", log.BBError(err))
```

with:

```go
logger.Error("failed to get plan for failure webhook", log.BBError(err))
```

Replace:

```go
slog.Error("failed to get project for failure webhook", log.BBError(err))
```

with:

```go
logger.Error("failed to get project for failure webhook", log.BBError(err))
```

- [ ] **Step 6: Replace success status failure log**

Replace:

```go
slog.Error("Failed to mark task as DONE",
	slog.Int64("id", task.ID),
	log.BBError(err),
)
```

with:

```go
logger.Error("Failed to mark task as DONE", log.BBError(err))
```

- [ ] **Step 7: Format and run focused task-run tests**

Run:

```bash
gofmt -w backend/runner/taskrun/running_scheduler.go
go test -v -count=1 ./backend/runner/taskrun -run '^(TestTaskRunLogAttrs|TestCheckTaskDrift|TestValidateTaskFreshness_DatabaseCreateSkipsValidation)$'
```

Expected: PASS.

- [ ] **Step 8: Commit scheduler log context usage**

Run:

```bash
git add backend/runner/taskrun/running_scheduler.go
git commit -m "chore: add task run context to scheduler logs"
```

### Task 3: Run Repository Verification

**Files:**
- Validate: `backend/runner/taskrun/log_context.go`
- Validate: `backend/runner/taskrun/log_context_test.go`
- Validate: `backend/runner/taskrun/running_scheduler.go`

- [ ] **Step 1: Run Go formatting on all modified Go files**

Run:

```bash
gofmt -w backend/runner/taskrun/log_context.go backend/runner/taskrun/log_context_test.go backend/runner/taskrun/running_scheduler.go
```

Expected: no output.

- [ ] **Step 2: Run the full task-run package tests**

Run:

```bash
go test -v -count=1 ./backend/runner/taskrun
```

Expected: PASS.

- [ ] **Step 3: Run backend lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: no issues. If issues are reported, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Repeat until `golangci-lint run --allow-parallel-runners` reports no issues.

- [ ] **Step 4: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: build exits successfully and writes `./bytebase-build/bytebase`.

- [ ] **Step 5: Commit any verification-driven fixes**

If lint or build required code changes, run:

```bash
git add backend/runner/taskrun/log_context.go backend/runner/taskrun/log_context_test.go backend/runner/taskrun/running_scheduler.go
git commit -m "chore: fix task run log context verification"
```

If no code changed during verification, do not create an empty commit.

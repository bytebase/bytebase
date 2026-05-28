# Task Run Log Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add consistent structured task-run log context for backend task-run execution logs.

**Architecture:** Store task-run log fields as metadata in `context.Context`, and wrap the configured `slog.Handler` so `slog.*Context` calls append those fields. Task-run code derives a task-run context with `project` and `task_run_id`; gh-ost receives the same context through its existing migration context path.

**Tech Stack:** Go, `log/slog`, existing Bytebase task-run scheduler, `stretchr/testify` for focused unit tests.

---

## File Structure

- Create `backend/common/log/context.go`
  - Owns generic context-carried slog attributes and a handler wrapper.
- Create `backend/common/log/context_test.go`
  - Verifies `InfoContext` records include context attributes.
- Modify `backend/bin/server/cmd/root.go`
  - Wraps the configured text or JSON slog handler with `log.NewContextHandler`.
- Modify `backend/runner/taskrun/log_context.go`
  - Owns task-run log attributes and `taskRunLogContext`.
- Modify `backend/runner/taskrun/running_scheduler.go`
  - Derives task-run contexts at scheduling and execution boundaries.
- Modify task-run executors in `backend/runner/taskrun`
  - Uses `slog.*Context` instead of passing scoped loggers.
- Modify `backend/component/ghost`
  - Makes the gh-ost logger adapter hold `context.Context` and log through `slog.*Context`.
- Modify `backend/runner/plancheck/ghost_sync_executor.go`
  - Uses the simplified gh-ost migration context signature.

No database, proto, frontend, or persisted execution state changes.

### Task 1: Add Context-Aware Slog Handler

**Files:**
- Create: `backend/common/log/context_test.go`
- Create: `backend/common/log/context.go`
- Modify: `backend/bin/server/cmd/root.go`

- [ ] **Step 1: Write failing tests for context log attributes**

Add tests that create a logger with `log.NewContextHandler`, add `project` and `task_run_id` through `log.WithAttrs`, and assert `logger.InfoContext(ctx, ...)` emits those fields.

- [ ] **Step 2: Verify the tests fail**

Run:

```bash
go test -v -count=1 ./backend/common/log -run '^(TestContextHandlerAddsAttrsFromContext|TestContextHandlerSkipsEmptyContext)$'
```

Expected: FAIL because `NewContextHandler` and `WithAttrs` do not exist.

- [ ] **Step 3: Implement context attrs and handler wrapper**

Implement `WithAttrs(ctx, attrs...)` and `NewContextHandler(handler)` in `backend/common/log/context.go`. `Handle` should append attributes from context to each slog record.

- [ ] **Step 4: Wire the handler at startup**

In `backend/bin/server/cmd/root.go`, create the text or JSON handler as before, then call:

```go
slog.SetDefault(slog.New(log.NewContextHandler(handler)))
```

### Task 2: Move Task-Run Log Context Into Context

**Files:**
- Modify: `backend/runner/taskrun/log_context.go`
- Modify: `backend/runner/taskrun/running_scheduler.go`
- Modify: `backend/runner/taskrun/executor.go`
- Modify: task-run executor files that log task-run events

- [ ] **Step 1: Write failing tests for task-run context**

Add a test that derives `ctx := taskRunLogContext(context.Background(), "project-a", 123)` and verifies a context-aware slog call emits `project=project-a` and `task_run_id=123`.

- [ ] **Step 2: Implement `taskRunLogContext`**

Keep `taskRunLogAttrs`, and add:

```go
func taskRunLogContext(ctx context.Context, projectID string, taskRunUID int64) context.Context {
	return log.WithAttrs(ctx, taskRunLogAttrs(projectID, taskRunUID)...)
}
```

- [ ] **Step 3: Replace scoped logger plumbing**

Derive task-run context at scheduler and executor boundaries, then replace task-run-scoped `logger.Warn/Error/Info/Debug` calls with `slog.WarnContext`, `slog.ErrorContext`, `slog.InfoContext`, or `slog.DebugContext`.

### Task 3: Make gh-ost Use Context Log Fields

**Files:**
- Modify: `backend/component/ghost/logger.go`
- Modify: `backend/component/ghost/config.go`
- Modify: `backend/runner/taskrun/database_migrate_executor.go`
- Modify: `backend/runner/plancheck/ghost_sync_executor.go`

- [ ] **Step 1: Write failing gh-ost context tests**

Add tests that install a context-aware default slog handler, pass a context with `project` and `task_run_id`, and assert gh-ost logs include those fields and format messages without `!BADKEY`.

- [ ] **Step 2: Update gh-ost logger adapter**

Change `ghostLogger` to hold `context.Context` and call `slog.*Context`.

- [ ] **Step 3: Simplify `NewMigrationContext`**

Remove the `*slog.Logger` parameter. Use the received context for both `migrationContext.Log` and gh-ost configuration logs.

### Task 4: Verify

**Files:**
- Validate all modified Go files and docs.

- [ ] **Step 1: Format**

Run `gofmt -w` on modified Go files.

- [ ] **Step 2: Run focused tests**

Run:

```bash
go test -v -count=1 ./backend/common/log ./backend/component/ghost ./backend/runner/taskrun ./backend/runner/plancheck
```

- [ ] **Step 3: Run lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

- [ ] **Step 4: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

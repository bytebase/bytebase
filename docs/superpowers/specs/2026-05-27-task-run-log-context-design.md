# Task Run Log Context Design

## Goal

Make backend task-run execution easier to trace in Bytebase server logs by adding consistent structured log fields around task-run scheduling and execution.

The first phase focuses on tracing one task run across backend logs. End-to-end request-to-async execution tracing is valid future work, but it is out of scope for this design.

## Current Context

Bytebase backend logging already uses Go `log/slog`, configured at server startup with text or JSON handlers. Existing logs often attach ad hoc fields, but task-run execution logs use ambiguous keys such as `id`, where the value may be a task ID in one message and a task-run ID in another.

Task-run execution flows through `backend/runner/taskrun/running_scheduler.go`:

- `scheduleRunningTaskRuns` claims available task runs.
- `executeTaskRun` loads the task, validates freshness, updates `started_at`, and starts asynchronous execution.
- `runTaskRunOnce` runs the executor and updates the final task-run status.

Executors receive `task`, `taskRunUID`, and have access to profile state such as `replica_id`, so they can attach the same log context without changing persisted state.

## Approach

Use scoped `slog.Logger` values at task-run execution boundaries.

Add a small helper in `backend/runner/taskrun` that builds task-run log attributes from the available execution data. The standard field set is:

- `project`
- `task_run_id`
- `task_id`
- `plan_id`
- `task_type`
- `replica_id`

At scheduler boundaries, create a scoped logger with:

```go
logger := slog.With(taskRunLogAttrs(task, taskRunUID, replicaID)...)
```

Then use `logger.Warn`, `logger.Error`, and related methods for task-run scoped events. Logs that occur before the full task is loaded should use a smaller claimed-task-run context with `project`, `task_run_id`, `task_id`, and `replica_id`.

This avoids putting the logger into `context.Context`. Context remains reserved for cancellation, deadlines, and future cross-cutting values such as `request_id` or `trace_id`. Task-run IDs are domain fields already available at the call site, so explicit scoped loggers are clearer and easier to audit.

## Scope

Update task-run scheduling and execution logs in `backend/runner/taskrun`, especially:

- claim and execute failures
- task freshness validation failures
- panic recovery
- cancellation
- executor failure
- task-run status update failures
- webhook follow-up failures after task-run failure

Where existing logs use ambiguous `id`, replace it with explicit `task_id` or `task_run_id`.

Executor-internal logs may use the same helper where the relevant data is already available. The first implementation should keep this opportunistic and focused, without changing the executor interface.

## Non-Goals

- No database schema changes.
- No proto changes.
- No persisted execution correlation ID.
- No OpenTelemetry integration.
- No request ID propagation.
- No logger stored in `context.Context`.
- No broad logging abstraction rewrite.

## Future Work

Later request-level tracing can add a request or trace ID through HTTP/gRPC middleware and context propagation. That work should coexist with the task-run fields rather than replace them. A future log line may include both `request_id` and `task_run_id` when the relationship is available.

If many executors need richer shared execution state, introduce a small execution context struct, for example a struct containing `TaskRunUID` and `Logger`. Do not add that interface churn until repeated executor changes justify it.

## Testing

Run normal Go formatting and validation for backend changes:

- `gofmt -w` on modified Go files
- `golangci-lint run --allow-parallel-runners`
- relevant task-run tests
- backend build

Because the change is observability-only, most verification should be compile, lint, and focused task-run test coverage. Add unit tests only if the helper logic becomes non-trivial.

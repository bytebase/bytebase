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

Executors receive `context.Context` and `taskRunUID`, so they can attach the same log context without changing persisted state.

## Approach

Store task-run log attributes in `context.Context`, and wrap the configured `slog.Handler` so context-aware slog calls append those attributes.

Add a small helper in `backend/common/log` for context-carried slog attributes:

- `WithAttrs(ctx, attrs...)`
- `NewContextHandler(handler)`

`NewContextHandler` wraps the text or JSON handler configured at startup. Logs must use `slog.InfoContext`, `slog.WarnContext`, `slog.ErrorContext`, and related context methods to receive the context attributes.

Add a small helper in `backend/runner/taskrun` that stores task-run log attributes from the available execution data. The standard field set is:

- `project`
- `task_run_id`

At scheduler boundaries, derive a task-run context:

```go
ctx := taskRunLogContext(ctx, projectID, taskRunUID)
```

Then use context-aware slog calls for task-run scoped events:

```go
slog.WarnContext(ctx, "task run blocked by drift validation", log.BBError(err))
```

Do not put a logger in `context.Context`. The context stores metadata only. This keeps task-run fields compatible with future request or trace IDs and avoids passing `*slog.Logger` through every executor boundary.

For gh-ost, keep its logger adapter but make it hold the task-run context and call `slog.InfoContext`, `slog.WarnContext`, and `slog.ErrorContext`. `ghost.NewMigrationContext` already receives `ctx`, so gh-ost logs can inherit task-run fields without changing gh-ost itself.

## Scope

Update task-run scheduling and execution logs in `backend/runner/taskrun`, especially:

- claim and execute failures
- task freshness validation failures
- panic recovery
- cancellation
- executor failure
- task-run status update failures
- webhook follow-up failures after task-run failure

Where existing logs use ambiguous `id` for the task run, replace it with explicit `task_run_id`. Avoid adding extra task metadata unless a specific log event needs it.

Executor-internal logs should use context-aware slog calls when the task-run context is available. The executor interface remains unchanged.

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

If many executors need richer shared execution state, introduce a small execution context struct. Do not add that interface churn until repeated executor changes justify it.

## Testing

Run normal Go formatting and validation for backend changes:

- `gofmt -w` on modified Go files
- `golangci-lint run --allow-parallel-runners`
- relevant task-run tests
- backend build

Because the change is observability-only, most verification should be compile, lint, and focused task-run test coverage. Add unit tests only if the helper logic becomes non-trivial.

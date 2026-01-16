# Task Run Payload for HA Scheduler State

## Problem

`TaskRunSchedulerInfo` is stored in-memory in `Bus.sync.Map`. This breaks HA:
- Lost on replica crash
- Not shared across replicas
- UI can't show why a task is waiting if the tracking replica dies

## Solution

Add a `payload` JSONB column to `task_run` table, storing a `TaskRunPayload` proto that contains `SchedulerInfo`. Future fields can be added to the proto without schema changes.

## Design

### Proto

`proto/store/store/task_run.proto`:
```protobuf
message TaskRunPayload {
  SchedulerInfo scheduler_info = 1;
  // Future fields added here without schema changes
}
```

### Schema

Migration `backend/migrator/migration/3.14/0032##task_run_payload.sql`:
```sql
ALTER TABLE task_run ADD COLUMN payload jsonb NOT NULL DEFAULT '{}';
```

### Store Layer

`backend/store/task_run.go`:
- Add `PayloadProto *storepb.TaskRunPayload` to `TaskRunMessage`
- Read: Add `payload` to SELECT, unmarshal with `protojson`
- Write: `UpdateTaskRunPayload(ctx, taskRunID, payload)` method

### Scheduler

`backend/runner/taskrun/pending_scheduler.go`:
- `storeParallelLimitCause()`: Write `TaskRunPayload{SchedulerInfo: ...}` to DB
- `promoteTaskRun()`: Write empty `TaskRunPayload{}` to clear

### API

`backend/api/v1/rollout_service_converter.go`:
- Remove `bus *bus.Bus` parameter
- Read `SchedulerInfo` from `taskRun.PayloadProto.SchedulerInfo`

### Cleanup

`backend/component/bus/bus.go`:
- Remove `TaskRunSchedulerInfo sync.Map`

## Files to Modify

1. `proto/store/store/task_run.proto` - Add `TaskRunPayload`
2. `backend/migrator/migration/3.14/0032##task_run_payload.sql` - New
3. `backend/migrator/migration/LATEST.sql` - Add column
4. `backend/migrator/migrator_test.go` - Update version
5. `backend/store/task_run.go` - Add field and methods
6. `backend/runner/taskrun/pending_scheduler.go` - Use store
7. `backend/api/v1/rollout_service_converter.go` - Remove Bus
8. `backend/api/v1/rollout_service.go` - Update converter calls
9. `backend/component/bus/bus.go` - Remove field

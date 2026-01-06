# Remove Redundant Task Proto Fields Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove redundant fields from task proto messages and retrieve them from plan specs instead

**Architecture:** Currently, task payload duplicates information already stored in plan specs. We'll remove these duplicate fields from `storepb.Task` and `v1pb.Task.DatabaseUpdate`, and modify converters and executors to retrieve this data from the plan spec using the `spec_id` field.

**Tech Stack:** Protocol Buffers, Go, PostgreSQL JSONB

---

## Overview

### Fields to Remove

**From `proto/store/store/task.proto` (storepb.Task):**
- `environment_id` (line 32)
- `database_name` (line 34)
- `table_name` (line 36)
- `character_set` (line 38)
- `collation` (line 40)
- `password` (line 58)
- `format` (line 60)

**From `proto/v1/v1/rollout_service.proto` (v1pb.Task.DatabaseUpdate):**
- `schema_version` (line 427)

**Keep in `proto/v1/v1/rollout_service.proto`:**
- `v1pb.Task.DatabaseCreate` message (lines 398-415) - part of public API
- `v1pb.Task.DatabaseDataExport` message (lines 430-443) - part of public API

### Architecture Decision

Tasks are linked to plan specs via `task.payload.spec_id`. Each spec contains the authoritative configuration:
- `PlanConfig_Spec.CreateDatabaseConfig` has database, table, character_set, collation, environment
- `PlanConfig_Spec.ExportDataConfig` has format, password

The converters and executors will look up the spec and extract these fields when needed.

---

## Task 1: Update proto definitions

**Files:**
- Modify: `proto/store/store/task.proto`
- Modify: `proto/v1/v1/rollout_service.proto`

**Step 1: Remove fields from storepb.Task**

Edit `proto/store/store/task.proto` and remove lines 31-40 and 57-60:

```protobuf
// REMOVE these lines (31-40):
  // The environment where the database will be created.
  string environment_id = 5;
  // Name of the database to create.
  string database_name = 6;
  // Optional table name to create (required for some databases like MongoDB).
  string table_name = 7;
  // Character set for the new database.
  string character_set = 8;
  // Collation for the new database.
  string collation = 9;

// REMOVE these lines (57-60):
  // Password to encrypt the exported data archive.
  string password = 14;
  // Format of the exported data (SQL, CSV, JSON, etc).
  ExportFormat format = 15;
```

**Step 2: Remove schema_version from v1pb.Task.DatabaseUpdate**

Edit `proto/v1/v1/rollout_service.proto` and remove lines 426-427:

```protobuf
  message DatabaseUpdate {
    oneof source {
      string sheet = 1;
      string release = 4;
    }
    // REMOVE this line:
    // string schema_version = 2;
  }
```

**Step 3: Generate proto code**

Run: `cd proto && buf generate`

**Step 4: Format proto files**

Run: `buf format -w proto`

**Step 5: Commit proto changes**

```bash
git add proto/store/store/task.proto proto/v1/v1/rollout_service.proto backend/generated-go
git commit -m "refactor: remove redundant task proto fields

Remove duplicate fields from Task proto that are already stored in PlanConfig specs:
- environment_id, database_name, table_name, character_set, collation
- password, format
- schema_version from DatabaseUpdate

These fields will be retrieved from the plan spec using spec_id.

Breaking change: stored Task payloads no longer contain these fields."
```

---

## Task 2: Update task creation logic

**Files:**
- Modify: `backend/api/v1/rollout_service_task.go:153-173`
- Modify: `backend/api/v1/rollout_service_task.go:301-326`

**Step 1: Remove field assignments from getTaskCreatesFromCreateDatabaseConfig**

Edit `backend/api/v1/rollout_service_task.go` lines 153-173:

```go
v := &store.TaskMessage{
	InstanceID:   instance.ResourceID,
	DatabaseName: &databaseName,
	Environment:  effectiveEnvironmentID,
	Type:         storepb.Task_DATABASE_CREATE,
	Payload: &storepb.Task{
		SpecId:        spec.Id,
		// REMOVE these lines:
		// CharacterSet:  c.CharacterSet,
		// TableName:     c.Table,
		// Collation:     c.Collation,
		// EnvironmentId: dbEnvironmentID,
		// DatabaseName:  databaseName,
		Source: &storepb.Task_SheetSha256{
			SheetSha256: sheet.Sha256,
		},
	},
}
```

**Step 2: Remove field assignments from getTaskCreatesFromExportDataConfig**

Edit `backend/api/v1/rollout_service_task.go` lines 301-326:

```go
payload := &storepb.Task{
	SpecId: spec.Id,
	Source: &storepb.Task_SheetSha256{
		SheetSha256: c.SheetSha256,
	},
	// REMOVE these lines:
	// Format: c.Format,
}
// REMOVE these lines:
// if c.Password != nil {
// 	payload.Password = *c.Password
// }
```

**Step 3: Run Go formatter**

Run: `gofmt -w backend/api/v1/rollout_service_task.go`

**Step 4: Run linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS (no new issues)

**Step 5: Commit**

```bash
git add backend/api/v1/rollout_service_task.go
git commit -m "refactor: stop setting redundant fields in task creation

Task payloads no longer duplicate data from plan specs."
```

---

## Task 3: Add helper function to retrieve spec from plan

**Files:**
- Modify: `backend/api/v1/rollout_service_converter.go`

**Step 1: Add getSpecFromPlan helper function**

Add this function after line 17 in `backend/api/v1/rollout_service_converter.go`:

```go
// getSpecFromPlan retrieves a spec by ID from a plan's configuration.
func getSpecFromPlan(plan *store.PlanMessage, specID string) (*storepb.PlanConfig_Spec, error) {
	if plan.Config == nil {
		return nil, errors.Errorf("plan config is nil")
	}
	for _, spec := range plan.Config.Specs {
		if spec.Id == specID {
			return spec, nil
		}
	}
	return nil, errors.Errorf("spec %q not found in plan", specID)
}
```

**Step 2: Run Go formatter**

Run: `gofmt -w backend/api/v1/rollout_service_converter.go`

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/rollout_service_converter.go`

Expected: PASS

**Step 4: Commit**

```bash
git add backend/api/v1/rollout_service_converter.go
git commit -m "feat: add helper to retrieve spec from plan by ID"
```

---

## Task 4: Update rollout converter to get fields from plan spec

**Files:**
- Modify: `backend/api/v1/rollout_service_converter.go:151-220`
- Modify: `backend/api/v1/rollout_service_converter.go:239-267`
- Modify: `backend/api/v1/rollout_service_converter.go:310-341`

**Step 1: Update convertToRollout to pass plan to converters**

Modify `convertToRollout` function signature and calls (lines 151-220):

```go
func convertToRollout(project *store.ProjectMessage, plan *store.PlanMessage, tasks []*store.TaskMessage, environmentOrderMap map[string]int) (*v1pb.Rollout, error) {
	// ... existing code ...

	// Update this section (lines 200-208):
	var v1Tasks []*v1pb.Task
	for _, task := range envTasks {
		v1Task, err := convertToTask(project, plan, task) // ADD plan parameter
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert task"))
		}
		v1Tasks = append(v1Tasks, v1Task)
	}
	// ... rest of function ...
}
```

**Step 2: Update convertToTask signature and dispatch**

Modify lines 222-237:

```go
func convertToTask(project *store.ProjectMessage, plan *store.PlanMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_CREATE:
		return convertToTaskFromDatabaseCreate(project, plan, task) // ADD plan
	case storepb.Task_DATABASE_MIGRATE:
		return convertToTaskFromSchemaUpdate(project, task) // No change
	case storepb.Task_DATABASE_EXPORT:
		return convertToTaskFromDatabaseDataExport(project, plan, task) // ADD plan
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	default:
		return nil, errors.Errorf("task type %v is not supported", task.Type)
	}
}
```

**Step 3: Update convertToTaskFromDatabaseCreate to get fields from spec**

Modify lines 239-267:

```go
func convertToTaskFromDatabaseCreate(project *store.ProjectMessage, plan *store.PlanMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	// Retrieve the spec to get configuration details
	spec, err := getSpecFromPlan(plan, task.Payload.GetSpecId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spec for task")
	}

	createConfig := spec.GetCreateDatabaseConfig()
	if createConfig == nil {
		return nil, errors.Errorf("spec does not contain create database config")
	}

	stageID := common.FormatStageID(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:          common.FormatTask(project.ResourceID, task.PlanID, stageID, task.ID),
		SpecId:        task.Payload.GetSpecId(),
		Type:          convertToTaskType(task),
		Status:        convertToTaskStatus(task.LatestTaskRunStatus, task.Payload.GetSkipped()),
		SkippedReason: task.Payload.GetSkippedReason(),
		Target:        common.FormatInstance(task.InstanceID),
		Payload: &v1pb.Task_DatabaseCreate_{
			DatabaseCreate: &v1pb.Task_DatabaseCreate{
				Project:      "",
				Database:     createConfig.Database,          // FROM SPEC
				Table:        createConfig.Table,             // FROM SPEC
				Sheet:        common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256()),
				CharacterSet: createConfig.CharacterSet,      // FROM SPEC
				Collation:    createConfig.Collation,         // FROM SPEC
				Environment:  createConfig.Environment,       // FROM SPEC
			},
		},
	}
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}
```

**Step 4: Update convertToTaskFromDatabaseDataExport to get fields from spec**

Modify lines 310-341:

```go
func convertToTaskFromDatabaseDataExport(project *store.ProjectMessage, plan *store.PlanMessage, task *store.TaskMessage) (*v1pb.Task, error) {
	if task.DatabaseName == nil {
		return nil, errors.Errorf("data export task database is nil")
	}

	// Retrieve the spec to get configuration details
	spec, err := getSpecFromPlan(plan, task.Payload.GetSpecId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spec for task")
	}

	exportConfig := spec.GetExportDataConfig()
	if exportConfig == nil {
		return nil, errors.Errorf("spec does not contain export data config")
	}

	targetDatabaseName := fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, task.InstanceID, common.DatabaseIDPrefix, *(task.DatabaseName))
	sheet := common.FormatSheet(project.ResourceID, task.Payload.GetSheetSha256())

	var password *string
	if exportConfig.Password != nil {
		password = exportConfig.Password  // FROM SPEC
	}

	v1pbTaskPayload := v1pb.Task_DatabaseDataExport_{
		DatabaseDataExport: &v1pb.Task_DatabaseDataExport{
			Target:   targetDatabaseName,
			Sheet:    sheet,
			Format:   convertExportFormat(exportConfig.Format),  // FROM SPEC
			Password: password,                                  // FROM SPEC
		},
	}
	stageID := common.FormatStageID(task.Environment)
	v1pbTask := &v1pb.Task{
		Name:    common.FormatTask(project.ResourceID, task.PlanID, stageID, task.ID),
		SpecId:  task.Payload.GetSpecId(),
		Type:    convertToTaskType(task),
		Status:  convertToTaskStatus(task.LatestTaskRunStatus, false),
		Target:  targetDatabaseName,
		Payload: &v1pbTaskPayload,
	}
	if task.UpdatedAt != nil {
		v1pbTask.UpdateTime = timestamppb.New(*task.UpdatedAt)
	}
	if task.RunAt != nil {
		v1pbTask.RunTime = timestamppb.New(*task.RunAt)
	}
	return v1pbTask, nil
}
```

**Step 5: Find and update all calls to convertToRollout**

Run: `grep -n "convertToRollout" backend/api/v1/rollout_service.go`

Update each call site to ensure all parameters are correct.

**Step 6: Run Go formatter**

Run: `gofmt -w backend/api/v1/rollout_service_converter.go backend/api/v1/rollout_service.go`

**Step 7: Run linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS (fix any issues)

**Step 8: Commit**

```bash
git add backend/api/v1/rollout_service_converter.go backend/api/v1/rollout_service.go
git commit -m "refactor: get task display fields from plan spec

Converters now retrieve database_name, table_name, character_set,
collation, environment, format, and password from plan specs instead
of task payloads."
```

---

## Task 5: Update database create executor to get fields from spec

**Files:**
- Modify: `backend/runner/taskrun/database_create_executor.go:36-121`

**Step 1: Update RunOnce to retrieve spec and get environment_id and database_name**

Modify the `RunOnce` method (lines 36-121):

```go
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, _ int) (*storepb.TaskRunResult, error) {
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet: %s", task.Payload.GetSheetSha256())
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	statement := sheet.Statement

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, errors.Errorf("empty create database statement")
	}

	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, err
	}

	if !common.EngineSupportCreateDatabase(instance.Metadata.GetEngine()) {
		return nil, errors.Errorf("creating database is not supported for engine %v", instance.Metadata.GetEngine().String())
	}

	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan %v", task.PlanID)
	}
	if plan == nil {
		return nil, errors.Errorf("plan %v not found", task.PlanID)
	}

	// NEW: Retrieve the spec to get configuration details
	spec, err := getSpecFromPlan(plan, task.Payload.GetSpecId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spec for task")
	}

	createConfig := spec.GetCreateDatabaseConfig()
	if createConfig == nil {
		return nil, errors.Errorf("spec does not contain create database config")
	}

	// Create database.
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Metadata.GetTitle()),
		slog.String("database", createConfig.Database),  // FROM SPEC
		slog.String("statement", statement),
	)

	// NEW: Get environment_id from spec
	envID := ""
	if createConfig.Environment != "" {
		envID = strings.TrimPrefix(createConfig.Environment, common.EnvironmentNamePrefix)
	}

	var environmentID *string
	if envID != "" {
		environmentID = &envID
	}
	database, err := exec.store.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:     plan.ProjectID,
		InstanceID:    instance.ResourceID,
		DatabaseName:  createConfig.Database,  // FROM SPEC
		EnvironmentID: environmentID,
		Metadata:      &storepb.DatabaseMetadata{},
	})
	if err != nil {
		return nil, err
	}

	var defaultDBDriver db.Driver
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_MONGODB:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
	}
	defer defaultDBDriver.Close(ctx)
	if _, err := defaultDBDriver.Execute(driverCtx, statement, db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return nil, err
	}

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return &storepb.TaskRunResult{}, nil
}
```

**Step 2: Add getSpecFromPlan helper function**

Add this function at the top of the file after imports:

```go
// getSpecFromPlan retrieves a spec by ID from a plan's configuration.
func getSpecFromPlan(plan *store.PlanMessage, specID string) (*storepb.PlanConfig_Spec, error) {
	if plan.Config == nil {
		return nil, errors.Errorf("plan config is nil")
	}
	for _, spec := range plan.Config.Specs {
		if spec.Id == specID {
			return spec, nil
		}
	}
	return nil, errors.Errorf("spec %q not found in plan", specID)
}
```

**Step 3: Add import for strings package if not already present**

Check if `strings` is imported, add if needed.

**Step 4: Run Go formatter**

Run: `gofmt -w backend/runner/taskrun/database_create_executor.go`

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS (fix any issues)

**Step 6: Commit**

```bash
git add backend/runner/taskrun/database_create_executor.go
git commit -m "refactor: get database create fields from plan spec

DatabaseCreateExecutor now retrieves environment_id and database_name
from the plan spec instead of task payload."
```

---

## Task 6: Update data export executor to get fields from spec

**Files:**
- Modify: `backend/runner/taskrun/data_export_executor.go:47-106`

**Step 1: Update RunOnce to retrieve spec and get format**

Modify the `RunOnce` method (lines 47-106):

```go
func (exec *DataExportExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, _ int) (*storepb.TaskRunResult, error) {
	issue, err := exec.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue")
	}

	database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID, ShowDeleted: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance")
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found")
	}

	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	statement := sheet.Statement

	// NEW: Get plan and spec to retrieve export configuration
	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan %v", task.PlanID)
	}
	if plan == nil {
		return nil, errors.Errorf("plan %v not found", task.PlanID)
	}

	spec, err := getSpecFromPlan(plan, task.Payload.GetSpecId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spec for task")
	}

	exportConfig := spec.GetExportDataConfig()
	if exportConfig == nil {
		return nil, errors.Errorf("spec does not contain export data config")
	}

	dataSource := apiv1.GetQueriableDataSource(instance)
	creatorUser, err := exec.store.GetUserByEmail(ctx, issue.CreatorEmail)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get creator user for issue %d", issue.UID)
	}
	if creatorUser == nil {
		return nil, errors.Errorf("creator user not found for issue %d", issue.UID)
	}

	// Execute the export without masking.
	// For approved DATABASE_EXPORT tasks, the approval itself authorizes access to the data.
	bytes, exportErr := exec.executeExport(ctx, instance, database, dataSource, statement, exportConfig.Format, creatorUser)  // FROM SPEC
	if exportErr != nil {
		return nil, errors.Wrap(exportErr, "failed to export data")
	}

	exportArchive, err := exec.store.CreateExportArchive(ctx, &store.ExportArchiveMessage{
		Bytes: bytes,
		Payload: &storepb.ExportArchivePayload{
			FileFormat: exportConfig.Format,  // FROM SPEC
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create export archive")
	}

	return &storepb.TaskRunResult{
		ExportArchiveUid: int32(exportArchive.UID),
	}, nil
}
```

**Step 2: Add getSpecFromPlan helper function (if not already present)**

Add this function at the top of the file after imports (only if it doesn't already exist from a previous task):

```go
// getSpecFromPlan retrieves a spec by ID from a plan's configuration.
func getSpecFromPlan(plan *store.PlanMessage, specID string) (*storepb.PlanConfig_Spec, error) {
	if plan.Config == nil {
		return nil, errors.Errorf("plan config is nil")
	}
	for _, spec := range plan.Config.Specs {
		if spec.Id == specID {
			return spec, nil
		}
	}
	return nil, errors.Errorf("spec %q not found in plan", specID)
}
```

**Step 3: Run Go formatter**

Run: `gofmt -w backend/runner/taskrun/data_export_executor.go`

**Step 4: Run linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS (fix any issues)

**Step 5: Commit**

```bash
git add backend/runner/taskrun/data_export_executor.go
git commit -m "refactor: get data export format from plan spec

DataExportExecutor now retrieves export format from the plan spec
instead of task payload."
```

---

## Task 7: Extract getSpecFromPlan to common utility

**Files:**
- Create: `backend/common/plan_utils.go`
- Modify: `backend/api/v1/rollout_service_converter.go`
- Modify: `backend/runner/taskrun/database_create_executor.go`
- Modify: `backend/runner/taskrun/data_export_executor.go`

**Step 1: Create plan_utils.go with shared helper**

Create `backend/common/plan_utils.go`:

```go
package common

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// GetSpecFromPlan retrieves a spec by ID from a plan's configuration.
func GetSpecFromPlan(plan *store.PlanMessage, specID string) (*storepb.PlanConfig_Spec, error) {
	if plan.Config == nil {
		return nil, errors.Errorf("plan config is nil")
	}
	for _, spec := range plan.Config.Specs {
		if spec.Id == specID {
			return spec, nil
		}
	}
	return nil, errors.Errorf("spec %q not found in plan", specID)
}
```

**Step 2: Replace local getSpecFromPlan in rollout_service_converter.go**

Edit `backend/api/v1/rollout_service_converter.go`:
- Remove the local `getSpecFromPlan` function
- Replace calls to `getSpecFromPlan` with `common.GetSpecFromPlan`

**Step 3: Replace local getSpecFromPlan in database_create_executor.go**

Edit `backend/runner/taskrun/database_create_executor.go`:
- Remove the local `getSpecFromPlan` function
- Replace calls to `getSpecFromPlan` with `common.GetSpecFromPlan`

**Step 4: Replace local getSpecFromPlan in data_export_executor.go**

Edit `backend/runner/taskrun/data_export_executor.go`:
- Remove the local `getSpecFromPlan` function
- Replace calls to `getSpecFromPlan` with `common.GetSpecFromPlan`

**Step 5: Run Go formatter**

Run: `gofmt -w backend/common/plan_utils.go backend/api/v1/rollout_service_converter.go backend/runner/taskrun/database_create_executor.go backend/runner/taskrun/data_export_executor.go`

**Step 6: Run linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS

**Step 7: Commit**

```bash
git add backend/common/plan_utils.go backend/api/v1/rollout_service_converter.go backend/runner/taskrun/database_create_executor.go backend/runner/taskrun/data_export_executor.go
git commit -m "refactor: extract GetSpecFromPlan to common utility

Consolidate duplicate getSpecFromPlan implementations into a single
shared utility function in the common package."
```

---

## Task 8: Run tests and fix issues

**Files:**
- Various test files may need updates

**Step 1: Run backend tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run TestRollout`

Expected: Tests may fail due to proto changes

**Step 2: Identify failing tests**

Review test output and identify tests that fail due to:
- Missing fields in task payloads
- Converter expectations
- Mock data setup

**Step 3: Update test fixtures and mocks**

For each failing test:
1. Update mock plan configs to include proper specs
2. Update task creation to use new field locations
3. Update assertions to check v1pb fields (not storepb fields)

**Step 4: Run executor tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/runner/taskrun`

Expected: Tests should pass after executor changes

**Step 5: Fix any remaining test failures**

Iterate on test fixes until all tests pass.

**Step 6: Run full linter**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS

**Step 7: Commit test fixes**

```bash
git add backend/api/v1/*_test.go backend/runner/taskrun/*_test.go
git commit -m "test: update tests for removed task proto fields

Update test fixtures and assertions to work with new architecture
where fields are retrieved from plan specs."
```

---

## Task 9: Update migrator test for schema changes

**Files:**
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Check TestLatestVersion test**

The CLAUDE.md instructions state that `TestLatestVersion` needs update after migration file changes. Since we're changing stored proto structure, check if this test needs updating.

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run TestLatestVersion`

**Step 2: Update test if needed**

If the test fails, update the test expectations to match the new proto structure.

**Step 3: Run test again**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run TestLatestVersion`

Expected: PASS

**Step 4: Commit if changes made**

```bash
git add backend/migrator/migrator_test.go
git commit -m "test: update migrator test for proto changes"
```

---

## Task 10: Build and verify

**Files:**
- None (verification only)

**Step 1: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

Expected: Build succeeds

**Step 2: Run all linters**

Run: `golangci-lint run --allow-parallel-runners`

Expected: PASS (run multiple times until no issues)

**Step 3: Run full backend test suite**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`

Expected: All tests pass

**Step 4: Verify proto formatting**

Run: `buf format -w proto && buf lint proto`

Expected: No changes, no lint errors

**Step 5: Final commit if any formatting changes**

```bash
git add -A
git commit -m "chore: final cleanup and formatting"
```

---

## Task 11: Create pull request

**Files:**
- None (git operations only)

**Step 1: Push branch**

Run: `git push -u origin main`

**Step 2: Verify all commits**

Run: `git log --oneline -15`

Review commit history to ensure all changes are properly committed.

**Step 3: Run final verification**

```bash
# Ensure build works
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

# Ensure tests pass
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1
go test -v -count=1 github.com/bytebase/bytebase/backend/runner/taskrun

# Ensure linter passes
golangci-lint run --allow-parallel-runners
```

Expected: All commands succeed

---

## Summary

This plan removes redundant fields from task proto messages and updates all code to retrieve these fields from plan specs instead. The changes include:

1. **Proto changes**: Removed 7 fields from `storepb.Task` and 1 field from `v1pb.Task.DatabaseUpdate`
2. **Task creation**: Stopped setting redundant fields when creating tasks
3. **Converters**: Updated to retrieve fields from plan specs via `spec_id`
4. **Executors**: Updated to retrieve fields from plan specs via `spec_id`
5. **Common utility**: Created shared `GetSpecFromPlan` helper
6. **Tests**: Updated all affected tests

**Breaking Changes:**
- Existing stored task payloads in the database will not have these fields
- Old tasks will need migration if they need to display these fields (handled by looking up the spec)

**Benefits:**
- Single source of truth for task configuration (plan spec)
- Reduced data duplication
- Smaller task payloads in database
- Easier to update task configuration centrally

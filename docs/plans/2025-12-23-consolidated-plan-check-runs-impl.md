# Consolidated Plan Check Runs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate plan check runs from N×types records per plan to a single record, reducing DB row explosion.

**Architecture:** One `plan_check_run` record per plan containing all targets and check types. Single combined executor processes all checks sequentially, aggregating results with instance/database/type tagging.

**Tech Stack:** Go, PostgreSQL, Protocol Buffers, golangci-lint

---

## Task 1: Update Proto - PlanCheckRunConfig

**Files:**
- Modify: `proto/store/store/plan_check_run.proto`

**Step 1: Add CheckType enum and replace PlanCheckRunConfig**

Replace the entire `PlanCheckRunConfig` message with the new consolidated version:

```protobuf
enum PlanCheckType {
  PLAN_CHECK_TYPE_UNSPECIFIED = 0;
  PLAN_CHECK_TYPE_STATEMENT_ADVISE = 1;
  PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT = 2;
  PLAN_CHECK_TYPE_GHOST_SYNC = 3;
}

message PlanCheckRunConfig {
  repeated CheckTarget targets = 1;

  message CheckTarget {
    // Format: instances/{instance}/databases/{database}
    string target = 1;
    string sheet_sha256 = 2;
    bool enable_prior_backup = 3;
    bool enable_ghost = 4;
    bool enable_sdl = 5;
    map<string, string> ghost_flags = 6;
    repeated PlanCheckType types = 7;
  }
}
```

Note: We reuse the name `PlanCheckRunConfig` but with new structure. Migration handles the conversion.

**Step 2: Add target fields to Result message**

Modify the `Result` message inside `PlanCheckRunResult` (around line 32), add after `code` field:

```protobuf
message Result {
  Advice.Status status = 1;
  string title = 2;
  string content = 3;
  int32 code = 4;

  // Target identification for consolidated results
  // Format: instances/{instance}/databases/{database}
  string target = 7;
  PlanCheckType type = 8;

  oneof report {
    SqlSummaryReport sql_summary_report = 5;
    SqlReviewReport sql_review_report = 6;
  }
  // ... rest unchanged
}
```

**Step 3: Generate proto**

Run: `cd proto && buf generate`

**Step 4: Verify generation**

Run: `ls -la ../backend/generated-go/store/plan_check_run.pb.go`
Expected: File updated with new timestamp

**Step 5: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "proto: add PlanCheckRunConfigV2 and result target fields"
```

---

## Task 2: Update Store - PlanCheckRunMessage

**Files:**
- Modify: `backend/store/plan_check_run.go`

**Step 1: Simplify PlanCheckRunMessage struct**

Around line 42, update the struct (remove Type field, Config uses new structure):

```go
// PlanCheckRunMessage is the message for a plan check run.
type PlanCheckRunMessage struct {
	UID       int
	CreatedAt time.Time
	UpdatedAt time.Time

	PlanUID int64

	Status PlanCheckRunStatus
	Config *storepb.PlanCheckRunConfig  // New consolidated config with targets
	Result *storepb.PlanCheckRunResult
}
```

**Step 2: Remove PlanCheckRunType constants**

Delete the type constants (lines 18-25):

```go
// DELETE THESE:
// PlanCheckDatabaseStatementAdvise
// PlanCheckDatabaseStatementSummaryReport
// PlanCheckDatabaseGhostSync
```

**Step 3: Update CreatePlanCheckRuns**

Simplify the INSERT (remove type column):

```go
// Insert new plan check runs
q = qb.Q().Space("INSERT INTO plan_check_run (plan_id, status, config, result) VALUES")
for i, create := range creates {
	config, err := protojson.Marshal(create.Config)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal create config")
	}
	result, err := protojson.Marshal(create.Result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal create result %v", create.Result)
	}
	if i > 0 {
		q.Space(",")
	}
	q.Space("(?, ?, ?, ?)", create.PlanUID, create.Status, config, result)
}
```

**Step 4: Update ListPlanCheckRuns**

Simplify the scan (remove type):

```go
for rows.Next() {
	planCheckRun := PlanCheckRunMessage{
		Config: &storepb.PlanCheckRunConfig{},
		Result: &storepb.PlanCheckRunResult{},
	}
	var config, result string
	if err := rows.Scan(
		&planCheckRun.UID,
		&planCheckRun.CreatedAt,
		&planCheckRun.UpdatedAt,
		&planCheckRun.PlanUID,
		&planCheckRun.Status,
		&config,
		&result,
	); err != nil {
		return nil, err
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(config), planCheckRun.Config); err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(result), planCheckRun.Result); err != nil {
		return nil, err
	}
	planCheckRuns = append(planCheckRuns, &planCheckRun)
}
```

**Step 5: Update FindPlanCheckRunMessage**

Remove Type filter field:

```go
type FindPlanCheckRunMessage struct {
	PlanUID      *int64
	UIDs         *[]int
	Status       *[]PlanCheckRunStatus
	ResultStatus *[]storepb.Advice_Status
	// Type field removed
}
```

**Step 6: Add GetPlanCheckRun helper**

Add after `ListPlanCheckRuns`:

```go
// GetPlanCheckRun returns the plan check run for a plan.
func (s *Store) GetPlanCheckRun(ctx context.Context, planUID int64) (*PlanCheckRunMessage, error) {
	runs, err := s.ListPlanCheckRuns(ctx, &FindPlanCheckRunMessage{PlanUID: &planUID})
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	// With consolidated model, there should be only one record per plan
	// For backward compatibility during migration, return the first one
	return runs[0], nil
}
```

**Step 7: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/store/plan_check_run.go`
Expected: No errors (fix any that appear)

**Step 8: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "store: simplify PlanCheckRunMessage for consolidated model"
```

---

## Task 3: Create Combined Executor

**Files:**
- Create: `backend/runner/plancheck/executor_combined.go`

**Step 1: Create the combined executor file**

```go
package plancheck

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
)

// CombinedExecutor processes all plan check types for a consolidated plan check run.
type CombinedExecutor struct {
	store          *store.Store
	sheetManager   *sheet.Manager
	dbFactory      *dbfactory.DBFactory
	licenseService *enterprise.LicenseService
}

// NewCombinedExecutor creates a combined executor.
func NewCombinedExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	licenseService *enterprise.LicenseService,
) *CombinedExecutor {
	return &CombinedExecutor{
		store:          store,
		sheetManager:   sheetManager,
		dbFactory:      dbFactory,
		licenseService: licenseService,
	}
}

// Run runs all checks for a consolidated config.
func (e *CombinedExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, target := range config.Targets {
		for _, checkType := range target.CheckTypes {
			results, err := e.runCheck(ctx, target, checkType)
			if err != nil {
				// Add error result for this target/type, continue to next
				allResults = append(allResults, &storepb.PlanCheckRunResult_Result{
					Status:       storepb.Advice_ERROR,
					InstanceId:   target.InstanceId,
					DatabaseName: target.DatabaseName,
					CheckType:    checkType,
					Title:        "Check failed",
					Content:      err.Error(),
					Code:         common.Internal.Int32(),
				})
				continue
			}
			// Tag results with target info
			for _, r := range results {
				r.InstanceId = target.InstanceId
				r.DatabaseName = target.DatabaseName
				r.CheckType = checkType
			}
			allResults = append(allResults, results...)
		}
	}

	return allResults, nil
}

func (e *CombinedExecutor) runCheck(ctx context.Context, target *storepb.PlanCheckRunConfig_CheckTarget, checkType storepb.PlanCheckType) ([]*storepb.PlanCheckRunResult_Result, error) {
	switch checkType {
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE:
		return e.runStatementAdvise(ctx, target)
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT:
		return e.runStatementReport(ctx, target)
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC:
		return e.runGhostSync(ctx, target)
	default:
		return nil, nil
	}
}

// Helper methods that call into existing executor logic
// These will need to be refactored from the existing executors
func (e *CombinedExecutor) runStatementAdvise(ctx context.Context, target *storepb.PlanCheckRunConfig_CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	// Inline the logic from StatementAdviseExecutor.Run
	// or refactor to share code
	// For now, create executor instance
	executor := &StatementAdviseExecutor{
		store:          e.store,
		sheetManager:   e.sheetManager,
		dbFactory:      e.dbFactory,
		licenseService: e.licenseService,
	}
	return executor.runForTarget(ctx, target)
}

func (e *CombinedExecutor) runStatementReport(ctx context.Context, target *storepb.PlanCheckRunConfig_CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &StatementReportExecutor{
		store:        e.store,
		sheetManager: e.sheetManager,
		dbFactory:    e.dbFactory,
	}
	return executor.runForTarget(ctx, target)
}

func (e *CombinedExecutor) runGhostSync(ctx context.Context, target *storepb.PlanCheckRunConfig_CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &GhostSyncExecutor{
		store:     e.store,
		dbFactory: e.dbFactory,
	}
	return executor.runForTarget(ctx, target)
}
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/plancheck/executor_combined.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "plancheck: add CombinedExecutor for consolidated runs"
```

---

## Task 4: Simplify Scheduler

**Files:**
- Modify: `backend/runner/plancheck/scheduler.go`

**Step 1: Simplify Scheduler struct**

Remove the type-based executor map, use only CombinedExecutor:

```go
// Scheduler is the plan check run scheduler.
type Scheduler struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	stateCfg       *state.State
	executor       *CombinedExecutor
}

// NewScheduler creates a new plan check scheduler.
func NewScheduler(s *store.Store, licenseService *enterprise.LicenseService, stateCfg *state.State, executor *CombinedExecutor) *Scheduler {
	return &Scheduler{
		store:          s,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		executor:       executor,
	}
}
```

**Step 2: Remove Register method**

Delete the `Register` method entirely - no longer needed.

**Step 3: Simplify runPlanCheckRun**

Replace the `runPlanCheckRun` method:

```go
func (s *Scheduler) runPlanCheckRun(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) {
	// Skip if already running
	if _, ok := s.stateCfg.RunningPlanChecks.Load(planCheckRun.UID); ok {
		return
	}

	s.stateCfg.RunningPlanChecks.Store(planCheckRun.UID, true)
	go func() {
		defer func() {
			s.stateCfg.RunningPlanChecks.Delete(planCheckRun.UID)
			s.stateCfg.RunningPlanCheckRunsCancelFunc.Delete(planCheckRun.UID)
		}()

		ctxWithCancel, cancel := context.WithCancel(ctx)
		defer cancel()
		s.stateCfg.RunningPlanCheckRunsCancelFunc.Store(planCheckRun.UID, cancel)

		results, err := s.executor.Run(ctxWithCancel, planCheckRun.Config)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.markPlanCheckRunCanceled(ctx, planCheckRun, err.Error())
			} else {
				s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			}
		} else {
			s.markPlanCheckRunDone(ctx, planCheckRun, results)
		}
	}()
}
```

**Step 4: Add missing import**

Add `"github.com/pkg/errors"` to imports if not present.

**Step 5: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/plancheck/scheduler.go`
Expected: No errors

**Step 6: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "plancheck: simplify scheduler for consolidated model"
```

---

## Task 5: Update Server Registration

**Files:**
- Modify: `backend/server/server.go`

**Step 1: Replace executor registrations with CombinedExecutor**

Around line 196-202, replace the old executor setup:

```go
	// OLD CODE TO REMOVE:
	// s.planCheckScheduler = plancheck.NewScheduler(stores, s.licenseService, s.stateCfg)
	// statementAdviseExecutor := plancheck.NewStatementAdviseExecutor(...)
	// s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementAdvise, statementAdviseExecutor)
	// ghostSyncExecutor := plancheck.NewGhostSyncExecutor(...)
	// s.planCheckScheduler.Register(store.PlanCheckDatabaseGhostSync, ghostSyncExecutor)
	// statementReportExecutor := plancheck.NewStatementReportExecutor(...)
	// s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementSummaryReport, statementReportExecutor)

	// NEW CODE:
	combinedExecutor := plancheck.NewCombinedExecutor(stores, sheetManager, s.dbFactory, s.licenseService)
	s.planCheckScheduler = plancheck.NewScheduler(stores, s.licenseService, s.stateCfg, combinedExecutor)
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/server/server.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "server: use CombinedExecutor for plan checks"
```

---

## Task 6: Update Config Generation

**Files:**
- Modify: `backend/api/v1/plan_service_plan_check.go`

**Step 1: Rewrite getPlanCheckRunsFromPlan to return single consolidated config**

Replace the entire function:

```go
func getPlanCheckRunFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) (*store.PlanCheckRunMessage, error) {
	var targets []*storepb.PlanCheckRunConfig_CheckTarget

	for _, spec := range plan.Config.Specs {
		switch config := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			// No checks for create database.
		case *storepb.PlanConfig_Spec_ExportDataConfig:
			// No checks for export data.
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			// Skip plan checks for releases.
			if config.ChangeDatabaseConfig.Release != "" {
				continue
			}

			var databases []string
			if len(config.ChangeDatabaseConfig.Targets) == 1 && databaseGroup != nil && config.ChangeDatabaseConfig.Targets[0] == databaseGroup.Name {
				for _, m := range databaseGroup.MatchedDatabases {
					databases = append(databases, m.Name)
				}
			} else {
				databases = config.ChangeDatabaseConfig.Targets
			}

			// Apply sampling upfront
			if samplingSize := project.Setting.GetCiSamplingSize(); samplingSize > 0 {
				if len(databases) > int(samplingSize) {
					databases = databases[:samplingSize]
				}
			}

			enableSDL := config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_SDL
			enableGhost := config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE && config.ChangeDatabaseConfig.EnableGhost

			for _, target := range databases {
				instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse %q", target)
				}

				checkTypes := []storepb.PlanCheckType{
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				}
				if enableGhost {
					checkTypes = append(checkTypes, storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC)
				}

				targets = append(targets, &storepb.PlanCheckRunConfig_CheckTarget{
					InstanceId:        instanceID,
					DatabaseName:      databaseName,
					SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
					EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
					EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
					EnableSdl:         enableSDL,
					GhostFlags:        config.ChangeDatabaseConfig.GhostFlags,
					CheckTypes:        checkTypes,
				})
			}
		default:
			return nil, errors.Errorf("unknown spec config type %T", config)
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Config:  &storepb.PlanCheckRunConfig{Targets: targets},
	}, nil
}
```

**Step 2: Find and update callers of getPlanCheckRunsFromPlan**

Search for callers:
Run: `grep -rn "getPlanCheckRunsFromPlan" backend/`

Update each caller to use the new function signature (returns single `*PlanCheckRunMessage` instead of slice).

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/plan_service_plan_check.go`
Expected: No errors

**Step 4: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "api: rewrite config generation for consolidated format"
```

---

## Task 7: Update Consumer - Approval Runner

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Update plan check status check (around line 448)**

Find the `ListPlanCheckRuns` call and update:

```go
// Check plan check runs status
planCheckRun, err := r.store.GetPlanCheckRun(ctx, plan.UID)
if err != nil {
	return nil, false, errors.Wrapf(err, "failed to get plan check run for plan %v", plan.UID)
}

// No plan check configured
if planCheckRun == nil {
	// Continue with existing logic for no checks
}

// Wait for plan check to complete
if planCheckRun.Status == store.PlanCheckRunStatusRunning {
	return nil, false, nil // Not ready yet, retry later
}

// Build latestPlanCheckRun map from results
type Key struct {
	InstanceID   string
	DatabaseName string
}
latestPlanCheckRun := map[Key]*storepb.PlanCheckRunResult_Result{}

for _, result := range planCheckRun.Result.Results {
	// Only consider summary report results
	if result.CheckType != storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT {
		continue
	}
	key := Key{
		InstanceID:   result.InstanceId,
		DatabaseName: result.DatabaseName,
	}
	latestPlanCheckRun[key] = result
}
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/approval/runner.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "approval: update to use consolidated plan check run"
```

---

## Task 8: Update Consumer - Auto Rollout Scheduler

**Files:**
- Modify: `backend/runner/taskrun/auto_rollout_scheduler.go`

**Step 1: Update plan check status check (around line 130)**

Replace the existing check:

```go
// Check the latest plan checks based on project settings (error only)
if project.Setting.RequirePlanCheckNoError {
	pass, err := func() (bool, error) {
		plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{PipelineID: &task.PipelineID})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get plan")
		}
		if plan == nil {
			return true, nil
		}

		planCheckRun, err := s.store.GetPlanCheckRun(ctx, plan.UID)
		if err != nil {
			return false, errors.Wrapf(err, "failed to get plan check run")
		}
		if planCheckRun == nil {
			return true, nil // No checks configured
		}
		if planCheckRun.Status != store.PlanCheckRunStatusDone {
			return false, nil
		}
		for _, result := range planCheckRun.Result.Results {
			if result.Status == storepb.Advice_ERROR {
				return false, nil
			}
		}
		return true, nil
	}()
	// ... rest unchanged
}
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/runner/taskrun/auto_rollout_scheduler.go`
Expected: No errors

**Step 3: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "taskrun: update to use consolidated plan check run"
```

---

## Task 9: Create Migration Script

**Files:**
- Create: `backend/migrator/migration/3.14/0004##consolidate_plan_check_runs.sql`

**Step 1: Write the migration SQL**

```sql
-- Consolidate plan_check_run records: one record per plan instead of N×types

-- Step 1: Create temp table with deduplicated latest records (last 30 days only)
CREATE TEMP TABLE plan_check_run_deduped AS
SELECT DISTINCT ON (plan_id, type, config->>'instanceId', config->>'databaseName')
    *
FROM plan_check_run
WHERE created_at >= NOW() - INTERVAL '30 days'
  AND status != 'CANCELED'
ORDER BY plan_id, type, config->>'instanceId', config->>'databaseName', created_at DESC;

-- Step 2: Delete all old records
DELETE FROM plan_check_run;

-- Step 3: Insert consolidated records (one per plan)
INSERT INTO plan_check_run (plan_id, status, type, config, result, created_at, updated_at)
SELECT
    plan_id,
    -- Status aggregation
    CASE
        WHEN bool_or(status = 'RUNNING') THEN 'RUNNING'
        WHEN bool_or(status = 'FAILED') THEN 'FAILED'
        ELSE 'DONE'
    END,
    -- Type is deprecated but keep a value for compatibility
    'bb.plan-check.database.statement.advise',
    -- Config: if any RUNNING, empty (will be re-run); otherwise aggregate
    CASE
        WHEN bool_or(status = 'RUNNING') THEN '{"targets": []}'::jsonb
        ELSE jsonb_build_object('targets', (
            SELECT jsonb_agg(target)
            FROM (
                SELECT jsonb_build_object(
                    'instanceId', d2.config->>'instanceId',
                    'databaseName', d2.config->>'databaseName',
                    'sheetSha256', d2.config->>'sheetSha256',
                    'enablePriorBackup', COALESCE((d2.config->>'enablePriorBackup')::boolean, false),
                    'enableGhost', COALESCE((d2.config->>'enableGhost')::boolean, false),
                    'enableSdl', COALESCE((d2.config->>'enableSdl')::boolean, false),
                    'ghostFlags', COALESCE(d2.config->'ghostFlags', '{}'::jsonb),
                    'checkTypes', array_agg(
                        CASE d2.type
                            WHEN 'bb.plan-check.database.statement.advise' THEN 'PLAN_CHECK_TYPE_STATEMENT_ADVISE'
                            WHEN 'bb.plan-check.database.statement.summary.report' THEN 'PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT'
                            WHEN 'bb.plan-check.database.ghost.sync' THEN 'PLAN_CHECK_TYPE_GHOST_SYNC'
                        END
                    )
                ) as target
                FROM plan_check_run_deduped d2
                WHERE d2.plan_id = d.plan_id
                GROUP BY
                    d2.config->>'instanceId',
                    d2.config->>'databaseName',
                    d2.config->>'sheetSha256',
                    d2.config->>'enablePriorBackup',
                    d2.config->>'enableGhost',
                    d2.config->>'enableSdl',
                    d2.config->'ghostFlags'
            ) targets
        ))
    END,
    -- Results: empty if RUNNING, otherwise aggregate with type tagging
    CASE
        WHEN bool_or(status = 'RUNNING') THEN '{"results": []}'::jsonb
        ELSE jsonb_build_object('results', (
            SELECT COALESCE(jsonb_agg(
                r || jsonb_build_object(
                    'instanceId', d3.config->>'instanceId',
                    'databaseName', d3.config->>'databaseName',
                    'checkType', CASE d3.type
                        WHEN 'bb.plan-check.database.statement.advise' THEN 'PLAN_CHECK_TYPE_STATEMENT_ADVISE'
                        WHEN 'bb.plan-check.database.statement.summary.report' THEN 'PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT'
                        WHEN 'bb.plan-check.database.ghost.sync' THEN 'PLAN_CHECK_TYPE_GHOST_SYNC'
                    END
                )
            ), '[]'::jsonb)
            FROM plan_check_run_deduped d3
            LEFT JOIN LATERAL jsonb_array_elements(d3.result->'results') r ON true
            WHERE d3.plan_id = d.plan_id
        ))
    END,
    MAX(created_at),
    MAX(updated_at)
FROM plan_check_run_deduped d
GROUP BY plan_id;

-- Step 4: Cleanup temp table
DROP TABLE plan_check_run_deduped;

-- Step 5: Drop type column (no longer used)
ALTER TABLE plan_check_run DROP COLUMN IF EXISTS type;
```

**Step 2: Update LATEST.sql**

Modify `backend/migrator/migration/LATEST.sql` to remove the `type` column from `plan_check_run` table:

```sql
CREATE TABLE plan_check_run (
    id serial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    plan_id bigint NOT NULL REFERENCES plan(id),
    status text NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    -- type column removed - check types now in config.targets[].checkTypes
    -- Stored as PlanCheckRunConfig (proto/store/store/plan_check_run.proto)
    config jsonb NOT NULL DEFAULT '{}',
    -- Stored as PlanCheckRunResult (proto/store/store/plan_check_run.proto)
    result jsonb NOT NULL DEFAULT '{}',
    payload jsonb NOT NULL DEFAULT '{}'
);
```

**Step 3: Update migrator test version**

Update `TestLatestVersion` in `backend/migrator/migrator_test.go` if needed.

**Step 4: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "migration: consolidate plan_check_runs and drop type column"
```

---

## Task 10: Update API Compatibility Layer

**Files:**
- Modify: `backend/api/v1/plan_service.go`
- Modify: `backend/api/v1/plan_service_converter.go`

**Step 1: Update ListPlanCheckRuns API**

In `plan_service.go`, the API now returns results from the single consolidated record. Update to expand results into the API response format that clients expect.

**Step 2: Update converter**

In `plan_service_converter.go`, update plan check run conversion to use new `Config.Targets` structure instead of old single-target config.

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/plan_service.go backend/api/v1/plan_service_converter.go`

**Step 4: Commit**

```bash
but commit consolidated-plan-check-runs-design -m "api: update for consolidated plan check run format"
```

---

## Task 11: Build and Test

**Step 1: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run linter on all changed files**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors (run repeatedly until clean)

**Step 3: Run related tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run PlanCheck`
Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/runner/plancheck`

**Step 4: Final commit**

```bash
but commit consolidated-plan-check-runs-design -m "chore: fix build and lint issues"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Proto changes | `proto/store/store/plan_check_run.proto` |
| 2 | Store changes | `backend/store/plan_check_run.go` |
| 3 | Combined executor | `backend/runner/plancheck/executor_combined.go` |
| 4 | Scheduler update | `backend/runner/plancheck/scheduler.go` |
| 5 | Server registration | `backend/server/server.go` |
| 6 | Config generation | `backend/api/v1/plan_service_plan_check.go` |
| 7 | Approval runner | `backend/runner/approval/runner.go` |
| 8 | Auto rollout | `backend/runner/taskrun/auto_rollout_scheduler.go` |
| 9 | Migration | `backend/migrator/migration/3.14/0004##...sql` |
| 10 | API compat | `backend/api/v1/plan_service*.go` |
| 11 | Build & test | All |

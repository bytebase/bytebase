# Plan Check Run Runtime Derivation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor plan check run to derive configuration from the plan at runtime instead of storing a self-contained copy.

**Architecture:** Remove `config` and `payload` columns from `plan_check_run` table. The scheduler fetches the plan and calls a derivation function to get check targets at runtime. Executors receive derived config instead of stored config.

**Tech Stack:** Go, PostgreSQL, Protocol Buffers

---

## Task 1: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/3.14/0021##remove_plan_check_run_config_payload.sql`
- Modify: `backend/migrator/migration/LATEST.sql:213-230`
- Modify: `backend/migrator/migrator_test.go:15-16`

**Step 1: Create migration file**

```sql
-- Drop config and payload columns from plan_check_run
ALTER TABLE plan_check_run DROP COLUMN config;
ALTER TABLE plan_check_run DROP COLUMN payload;
```

**Step 2: Update LATEST.sql**

Change:
```sql
CREATE TABLE plan_check_run (
    id serial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    plan_id bigint NOT NULL REFERENCES plan(id),
    status text NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    -- Stored as PlanCheckRunConfig (proto/store/store/plan_check_run.proto)
    config jsonb NOT NULL DEFAULT '{}',
    -- Stored as PlanCheckRunResult (proto/store/store/plan_check_run.proto)
    result jsonb NOT NULL DEFAULT '{}',
    payload jsonb NOT NULL DEFAULT '{}'
);
```

To:
```sql
CREATE TABLE plan_check_run (
    id serial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    plan_id bigint NOT NULL REFERENCES plan(id),
    status text NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    -- Stored as PlanCheckRunResult (proto/store/store/plan_check_run.proto)
    result jsonb NOT NULL DEFAULT '{}'
);
```

**Step 3: Update migrator_test.go**

Change:
```go
require.Equal(t, semver.MustParse("3.14.20"), *files[len(files)-1].version)
require.Equal(t, "migration/3.14/0020##remove_task_run_code_and_sheet_sha256.sql", files[len(files)-1].path)
```

To:
```go
require.Equal(t, semver.MustParse("3.14.21"), *files[len(files)-1].version)
require.Equal(t, "migration/3.14/0021##remove_plan_check_run_config_payload.sql", files[len(files)-1].path)
```

**Step 4: Commit**

```bash
but commit plan-check-run-refactor -m "chore(migration): drop config and payload from plan_check_run"
```

---

## Task 2: Update Proto - Remove PlanCheckRunConfig

**Files:**
- Modify: `proto/store/store/plan_check_run.proto:17-29`

**Step 1: Remove PlanCheckRunConfig message**

Delete the `PlanCheckRunConfig` message and `CheckTarget` nested message (lines 17-29):

```protobuf
message PlanCheckRunConfig {
  repeated CheckTarget targets = 1;

  message CheckTarget {
    // Format: instances/{instance}/databases/{database}
    string target = 1;
    string sheet_sha256 = 2;
    bool enable_prior_backup = 3;
    bool enable_ghost = 4;
    map<string, string> ghost_flags = 6;
    repeated PlanCheckType types = 7;
  }
}
```

**Step 2: Run buf format and lint**

```bash
buf format -w proto && buf lint proto
```

**Step 3: Regenerate proto**

```bash
cd proto && buf generate
```

**Step 4: Commit**

```bash
but commit plan-check-run-refactor -m "proto: remove PlanCheckRunConfig message"
```

---

## Task 3: Update Store Layer

**Files:**
- Modify: `backend/store/plan_check_run.go`

**Step 1: Remove Config from PlanCheckRunMessage**

Change struct (lines 29-40):
```go
// PlanCheckRunMessage is the message for a plan check run.
type PlanCheckRunMessage struct {
	UID       int
	CreatedAt time.Time
	UpdatedAt time.Time

	PlanUID int64

	Status PlanCheckRunStatus
	Config *storepb.PlanCheckRunConfig
	Result *storepb.PlanCheckRunResult
}
```

To:
```go
// PlanCheckRunMessage is the message for a plan check run.
type PlanCheckRunMessage struct {
	UID       int
	CreatedAt time.Time
	UpdatedAt time.Time

	PlanUID int64

	Status PlanCheckRunStatus
	Result *storepb.PlanCheckRunResult
}
```

**Step 2: Update CreatePlanCheckRun**

Change (lines 51-74):
```go
// CreatePlanCheckRun creates or replaces the plan check run for a plan.
func (s *Store) CreatePlanCheckRun(ctx context.Context, create *PlanCheckRunMessage) error {
	config, err := protojson.Marshal(create.Config)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal config")
	}
	result, err := protojson.Marshal(create.Result)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal result")
	}

	query := `
		INSERT INTO plan_check_run (plan_id, status, config, result)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (plan_id) DO UPDATE SET
			status = EXCLUDED.status,
			config = EXCLUDED.config,
			result = EXCLUDED.result,
			updated_at = now()
	`
	if _, err := s.GetDB().ExecContext(ctx, query, create.PlanUID, create.Status, config, result); err != nil {
		return errors.Wrapf(err, "failed to upsert plan check run")
	}
	return nil
}
```

To:
```go
// CreatePlanCheckRun creates or replaces the plan check run for a plan.
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
	if _, err := s.GetDB().ExecContext(ctx, query, create.PlanUID, create.Status, result); err != nil {
		return errors.Wrapf(err, "failed to upsert plan check run")
	}
	return nil
}
```

**Step 3: Update ListPlanCheckRuns**

Change query and scan (lines 77-148):
```go
// ListPlanCheckRuns returns a list of plan check runs based on find.
func (s *Store) ListPlanCheckRuns(ctx context.Context, find *FindPlanCheckRunMessage) ([]*PlanCheckRunMessage, error) {
	q := qb.Q().Space(`
SELECT
	plan_check_run.id,
	plan_check_run.created_at,
	plan_check_run.updated_at,
	plan_check_run.plan_id,
	plan_check_run.status,
	plan_check_run.result
FROM plan_check_run
WHERE TRUE`)
	if v := find.PlanUID; v != nil {
		q.Space("AND plan_check_run.plan_id = ?", *v)
	}
	if v := find.UIDs; v != nil {
		q.Space("AND plan_check_run.id = ANY(?)", *v)
	}
	if v := find.Status; v != nil {
		q.Space("AND plan_check_run.status = ANY(?)", *v)
	}
	if v := find.ResultStatus; v != nil {
		statusStrings := make([]string, len(*v))
		for i, status := range *v {
			statusStrings[i] = status.String()
		}
		q.Space("AND EXISTS (SELECT 1 FROM jsonb_array_elements(plan_check_run.result->'results') AS elem WHERE elem->>'status' = ANY(?))", statusStrings)
	}
	q.Space("ORDER BY plan_check_run.id ASC")
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var planCheckRuns []*PlanCheckRunMessage
	for rows.Next() {
		planCheckRun := PlanCheckRunMessage{
			Result: &storepb.PlanCheckRunResult{},
		}
		var result string
		if err := rows.Scan(
			&planCheckRun.UID,
			&planCheckRun.CreatedAt,
			&planCheckRun.UpdatedAt,
			&planCheckRun.PlanUID,
			&planCheckRun.Status,
			&result,
		); err != nil {
			return nil, err
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(result), planCheckRun.Result); err != nil {
			return nil, err
		}
		planCheckRuns = append(planCheckRuns, &planCheckRun)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return planCheckRuns, nil
}
```

**Step 4: Run lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/store/...
```

**Step 5: Commit**

```bash
but commit plan-check-run-refactor -m "store: remove config from plan_check_run"
```

---

## Task 4: Add Check Target Derivation Types

**Files:**
- Create: `backend/runner/plancheck/check_target.go`

**Step 1: Create new file with CheckTarget struct**

```go
package plancheck

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// CheckTarget represents a derived check target from a plan.
// This is computed at runtime from the plan's specs, not stored.
type CheckTarget struct {
	// Target is the database resource name: instances/{instance}/databases/{database}
	Target string
	// SheetSHA256 is the content hash of the SQL sheet
	SheetSHA256 string
	// EnablePriorBackup indicates if backup before migration is enabled
	EnablePriorBackup bool
	// EnableGhost indicates if gh-ost online migration is enabled
	EnableGhost bool
	// GhostFlags are configuration flags for gh-ost
	GhostFlags map[string]string
	// Types are the plan check types to run for this target
	Types []storepb.PlanCheckType
}
```

**Step 2: Commit**

```bash
but commit plan-check-run-refactor -m "plancheck: add CheckTarget type for runtime derivation"
```

---

## Task 5: Extract Derivation Logic to Plancheck Package

**Files:**
- Create: `backend/runner/plancheck/derive.go`
- Modify: `backend/api/v1/plan_service.go`

**Step 1: Create derive.go with derivation function**

```go
package plancheck

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// DeriveCheckTargets derives check targets from a plan and optional database group.
// This replaces the stored config by computing targets at runtime.
func DeriveCheckTargets(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) ([]*CheckTarget, error) {
	var targets []*CheckTarget

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

			enableGhost := config.ChangeDatabaseConfig.EnableGhost

			for _, target := range databases {
				types := []storepb.PlanCheckType{
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				}
				if enableGhost {
					types = append(types, storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC)
				}

				targets = append(targets, &CheckTarget{
					Target:            target,
					SheetSHA256:       config.ChangeDatabaseConfig.SheetSha256,
					EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
					EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
					GhostFlags:        config.ChangeDatabaseConfig.GhostFlags,
					Types:             types,
				})
			}
		default:
			return nil, errors.Errorf("unknown spec config type %T", config)
		}
	}

	return targets, nil
}
```

**Step 2: Update plan_service.go - simplify getPlanCheckRunFromPlan**

Change function (lines 729-796):
```go
func getPlanCheckRunFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) (*store.PlanCheckRunMessage, error) {
	targets, err := plancheck.DeriveCheckTargets(project, plan, databaseGroup)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
	}, nil
}
```

**Step 3: Add import for plancheck package in plan_service.go**

Add to imports:
```go
"github.com/bytebase/bytebase/backend/runner/plancheck"
```

**Step 4: Run lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/api/v1/... ./backend/runner/plancheck/...
```

**Step 5: Commit**

```bash
but commit plan-check-run-refactor -m "plancheck: extract DeriveCheckTargets function"
```

---

## Task 6: Update Executors to Use CheckTarget

**Files:**
- Modify: `backend/runner/plancheck/executor.go`
- Modify: `backend/runner/plancheck/executor_combined.go`
- Modify: `backend/runner/plancheck/statement_advise_executor.go`
- Modify: `backend/runner/plancheck/statement_report_executor.go`
- Modify: `backend/runner/plancheck/ghost_sync_executor.go`

**Step 1: Update Executor interface**

Change `backend/runner/plancheck/executor.go`:
```go
package plancheck

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Executor is the plan check executor.
type Executor interface {
	// RunForTarget will be called periodically by the plan check scheduler for each target
	RunForTarget(ctx context.Context, target *CheckTarget) (results []*storepb.PlanCheckRunResult_Result, err error)
}
```

**Step 2: Update CombinedExecutor**

Change `backend/runner/plancheck/executor_combined.go`:
```go
package plancheck

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// CombinedExecutor processes all plan check types.
type CombinedExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// NewCombinedExecutor creates a combined executor.
func NewCombinedExecutor(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
) *CombinedExecutor {
	return &CombinedExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// RunForTarget runs all checks for the given target.
func (e *CombinedExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, checkType := range target.Types {
		results, err := e.runCheck(ctx, target, checkType)
		if err != nil {
			// Add error result for this target/type, continue to next
			allResults = append(allResults, &storepb.PlanCheckRunResult_Result{
				Status:  storepb.Advice_ERROR,
				Target:  target.Target,
				Type:    checkType,
				Title:   "Check failed",
				Content: err.Error(),
				Code:    common.Internal.Int32(),
			})
			continue
		}
		// Tag results with target info
		for _, r := range results {
			r.Target = target.Target
			r.Type = checkType
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (e *CombinedExecutor) runCheck(ctx context.Context, target *CheckTarget, checkType storepb.PlanCheckType) ([]*storepb.PlanCheckRunResult_Result, error) {
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

func (e *CombinedExecutor) runStatementAdvise(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &StatementAdviseExecutor{
		store:        e.store,
		sheetManager: e.sheetManager,
		dbFactory:    e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}

func (e *CombinedExecutor) runStatementReport(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &StatementReportExecutor{
		store:        e.store,
		sheetManager: e.sheetManager,
		dbFactory:    e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}

func (e *CombinedExecutor) runGhostSync(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	executor := &GhostSyncExecutor{
		store:     e.store,
		dbFactory: e.dbFactory,
	}
	return executor.RunForTarget(ctx, target)
}
```

**Step 3: Update StatementAdviseExecutor**

Change function signature and field access in `backend/runner/plancheck/statement_advise_executor.go`:
```go
// RunForTarget runs the statement advise check for a single target.
func (e *StatementAdviseExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	fullSheet, err := e.store.GetSheetFull(ctx, target.SheetSHA256)
	// ... rest uses target.Target, target.EnablePriorBackup, target.EnableGhost
```

**Step 4: Update StatementReportExecutor**

Change function signature in `backend/runner/plancheck/statement_report_executor.go`:
```go
// RunForTarget runs the statement report check for a single target.
func (e *StatementReportExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	fullSheet, err := e.store.GetSheetFull(ctx, target.SheetSHA256)
	// ... rest uses target.Target
```

**Step 5: Update GhostSyncExecutor**

Change function signature in `backend/runner/plancheck/ghost_sync_executor.go`:
```go
// RunForTarget runs the gh-ost sync check for a single target.
func (e *GhostSyncExecutor) RunForTarget(ctx context.Context, target *CheckTarget) (results []*storepb.PlanCheckRunResult_Result, err error) {
	// ... uses target.Target, target.SheetSHA256, target.GhostFlags
```

**Step 6: Run lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/runner/plancheck/...
```

**Step 7: Commit**

```bash
but commit plan-check-run-refactor -m "plancheck: update executors to use CheckTarget"
```

---

## Task 7: Update Scheduler for Runtime Derivation

**Files:**
- Modify: `backend/runner/plancheck/scheduler.go`

**Step 1: Update Scheduler struct to include store for plan fetching**

The scheduler already has `store *store.Store`, so it can fetch plans.

**Step 2: Update runPlanCheckRun to fetch plan and derive targets**

Change the `runPlanCheckRun` function (lines 83-120):
```go
func (s *Scheduler) runPlanCheckRun(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) {
	// Skip the plan check run if it is already running.
	if _, ok := s.bus.RunningPlanChecks.Load(planCheckRun.UID); ok {
		return
	}

	s.bus.RunningPlanChecks.Store(planCheckRun.UID, true)
	go func() {
		defer func() {
			s.bus.RunningPlanChecks.Delete(planCheckRun.UID)
			s.bus.RunningPlanCheckRunsCancelFunc.Delete(planCheckRun.UID)
		}()

		ctxWithCancel, cancel := context.WithCancel(ctx)
		defer cancel()
		s.bus.RunningPlanCheckRunsCancelFunc.Store(planCheckRun.UID, cancel)

		// Fetch plan to derive check targets at runtime
		plan, err := s.store.GetPlan(ctxWithCancel, planCheckRun.PlanUID)
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}
		if plan == nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, "plan not found")
			return
		}

		project, err := s.store.GetProject(ctxWithCancel, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}
		if project == nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, "project not found")
			return
		}

		// Get database group if needed (for spec expansion)
		var databaseGroup *v1pb.DatabaseGroup
		for _, spec := range plan.Config.Specs {
			if cfg, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); ok {
				if len(cfg.ChangeDatabaseConfig.Targets) == 1 {
					target := cfg.ChangeDatabaseConfig.Targets[0]
					if dbg, err := s.store.GetDatabaseGroup(ctxWithCancel, &store.FindDatabaseGroupMessage{ResourceID: &target}); err == nil && dbg != nil {
						databaseGroup = convertToDatabaseGroup(dbg)
						break
					}
				}
			}
		}

		// Derive check targets from plan
		targets, err := DeriveCheckTargets(project, plan, databaseGroup)
		if err != nil {
			s.markPlanCheckRunFailed(ctx, planCheckRun, err.Error())
			return
		}

		var results []*storepb.PlanCheckRunResult_Result
		for _, target := range targets {
			targetResults, targetErr := s.executor.RunForTarget(ctxWithCancel, target)
			if targetErr != nil {
				err = targetErr
				break
			}
			results = append(results, targetResults...)
		}
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

**Step 3: Add helper function for database group conversion**

Add at end of file:
```go
func convertToDatabaseGroup(dbg *store.DatabaseGroupMessage) *v1pb.DatabaseGroup {
	if dbg == nil {
		return nil
	}
	result := &v1pb.DatabaseGroup{
		Name: dbg.ResourceID,
	}
	for _, db := range dbg.MatchedDatabases {
		result.MatchedDatabases = append(result.MatchedDatabases, &v1pb.DatabaseGroup_Database{
			Name: db,
		})
	}
	return result
}
```

**Step 4: Add imports**

Add to imports:
```go
v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
```

**Step 5: Run lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/runner/plancheck/...
```

**Step 6: Commit**

```bash
but commit plan-check-run-refactor -m "plancheck: scheduler derives targets at runtime"
```

---

## Task 8: Build and Test

**Step 1: Build backend**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 2: Run lint on all changed packages**

```bash
golangci-lint run --allow-parallel-runners ./backend/store/... ./backend/api/v1/... ./backend/runner/plancheck/...
```

**Step 3: Commit any fixes**

```bash
but commit plan-check-run-refactor -m "fix: lint issues"
```

---

## Task 9: Final Cleanup

**Step 1: Remove unused imports in plan_service.go**

If `storepb.PlanCheckRunConfig` is no longer referenced, the import may need cleanup.

**Step 2: Run full lint**

```bash
golangci-lint run --allow-parallel-runners
```

**Step 3: Final commit**

```bash
but commit plan-check-run-refactor -m "refactor: plan check run derives config at runtime

- Remove config and payload columns from plan_check_run table
- Remove PlanCheckRunConfig proto message
- Add CheckTarget struct for runtime derivation
- Scheduler fetches plan and derives targets before execution
- Executors receive CheckTarget instead of stored proto"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Database migration to drop columns |
| 2 | Proto changes to remove PlanCheckRunConfig |
| 3 | Store layer updates |
| 4 | New CheckTarget type |
| 5 | Extract DeriveCheckTargets function |
| 6 | Update all executors |
| 7 | Update scheduler for runtime derivation |
| 8 | Build and test |
| 9 | Final cleanup |

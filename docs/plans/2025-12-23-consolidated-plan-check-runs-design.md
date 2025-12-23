# Consolidated Plan Check Runs Design

## Problem

Currently, plan check runs create `#specs × #databases × #types` records per plan:
- 100 databases × 2-3 types = 200-300 rows per plan
- Causes DB row explosion affecting performance/storage
- CI sampling size is a post-hoc band-aid

## Solution

Consolidate to **one record per plan** containing all check configs and results.

## Key Decisions

| Decision | Choice |
|----------|--------|
| Records per plan | One (all types combined) |
| Execution model | Sequential, single executor (optimize to parallel later if needed) |
| Sampling | Applied at config generation time, not post-hoc |
| Result structure | Flat array with metadata tags (instance_id, database_name, check_type) |
| Status model | Record-level for execution; per-result for check outcomes |
| Migration | SQL-only with aggregation |

## Data Model Changes

### PlanCheckRunConfig (proto)

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
    string instance_id = 1;
    string database_name = 2;
    string sheet_sha256 = 3;
    bool enable_prior_backup = 4;
    bool enable_ghost = 5;
    bool enable_sdl = 6;
    map<string, string> ghost_flags = 7;
    repeated PlanCheckType check_types = 8;
  }
}
```

### PlanCheckRunResult.Result (proto)

Add fields to existing Result message:
```protobuf
message Result {
  // existing fields...
  string instance_id = 7;
  string database_name = 8;
  PlanCheckType check_type = 9;
}
```

### Store Changes

- Remove `Type` field from `PlanCheckRunMessage`
- Add `GetPlanCheckRun(ctx, planUID)` for single-record access
- Update `CreatePlanCheckRuns` → `CreatePlanCheckRun` (singular)

## Executor Changes

### New Combined Executor

```go
// backend/runner/plancheck/executor_combined.go
type CombinedExecutor struct {
    statementAdviseExecutor  *StatementAdviseExecutor
    summaryReportExecutor    *SummaryReportExecutor
    ghostSyncExecutor        *GhostSyncExecutor
}

func (e *CombinedExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) ([]*storepb.PlanCheckRunResult_Result, error) {
    var results []*storepb.PlanCheckRunResult_Result

    for _, target := range config.Targets {
        for _, checkType := range target.CheckTypes {
            targetResults, err := e.runCheck(ctx, target, checkType)
            if err != nil {
                // Add error result, continue to next
                results = append(results, &storepb.PlanCheckRunResult_Result{
                    Status:       storepb.Advice_ERROR,
                    InstanceId:   target.InstanceId,
                    DatabaseName: target.DatabaseName,
                    CheckType:    checkType,
                    Title:        "Check failed",
                    Content:      err.Error(),
                })
                continue
            }
            results = append(results, targetResults...)
        }
    }
    return results, nil
}
```

### Scheduler Simplification

- Remove type-based executor map
- Single executor handles all plan check runs
- Record status: RUNNING → DONE (or FAILED on infra error)

## Config Generation

```go
func getPlanCheckRunFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) (*store.PlanCheckRunMessage, error) {
    var targets []*storepb.PlanCheckRunConfig_CheckTarget

    for _, spec := range plan.Config.Specs {
        switch config := spec.Config.(type) {
        case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
            if config.ChangeDatabaseConfig.Release != "" {
                continue
            }

            databases := resolveDatabases(config, databaseGroup)

            // Apply sampling upfront
            if samplingSize := project.Setting.GetCiSamplingSize(); samplingSize > 0 {
                if len(databases) > int(samplingSize) {
                    databases = databases[:samplingSize]
                }
            }

            for _, db := range databases {
                target := &storepb.PlanCheckRunConfig_CheckTarget{
                    InstanceId:        instanceID,
                    DatabaseName:      databaseName,
                    SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
                    EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
                    EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
                    EnableSdl:         enableSDL,
                    CheckTypes:        []storepb.CheckType{STATEMENT_ADVISE, STATEMENT_SUMMARY_REPORT},
                }
                if config.ChangeDatabaseConfig.EnableGhost {
                    target.CheckTypes = append(target.CheckTypes, GHOST_SYNC)
                }
                targets = append(targets, target)
            }
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

## Consumer Updates

### approval/runner.go

```go
// Before: query by type, iterate records
planCheckRuns, _ := r.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
    PlanUID: &plan.UID,
    Type:    &[]store.PlanCheckRunType{store.PlanCheckDatabaseStatementSummaryReport},
})

// After: single record, filter results
planCheckRun, _ := r.store.GetPlanCheckRun(ctx, plan.UID)
if planCheckRun.Status == store.PlanCheckRunStatusRunning {
    return nil, false, nil
}
for _, result := range planCheckRun.Result.Results {
    if result.CheckType == storepb.CheckType_STATEMENT_SUMMARY_REPORT {
        // existing logic
    }
}
```

### auto_rollout_scheduler.go

```go
// Simplified single-record check
planCheckRun, _ := s.store.GetPlanCheckRun(ctx, plan.UID)
if planCheckRun == nil {
    return true, nil
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
```

### API ListPlanCheckRuns

Keep API backward compatible. Internally transform single record to list of virtual `PlanCheckRun` objects grouped by type.

## Status & Results Model

| Status | Results | Meaning |
|--------|---------|---------|
| RUNNING | Empty `[]` | Executor still processing |
| DONE | All results | All checks completed |
| FAILED | Partial results + error | Infrastructure error, partial work preserved |

- CANCELED records are discarded (rare, user-initiated)
- If any record is RUNNING during migration → consolidated is RUNNING, Bytebase re-runs

## Migration Script

```sql
-- Step 1: Deduplicate - keep latest per (plan, type, instance, database), last 30 days, exclude CANCELED
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
INSERT INTO plan_check_run (plan_id, status, config, result, created_at, updated_at)
SELECT
    plan_id,
    -- Status aggregation
    CASE
        WHEN bool_or(status = 'RUNNING') THEN 'RUNNING'
        WHEN bool_or(status = 'FAILED') THEN 'FAILED'
        ELSE 'DONE'
    END,
    -- Config: if any RUNNING, empty (will be re-run); otherwise aggregate
    CASE
        WHEN bool_or(status = 'RUNNING') THEN '{"targets": []}'::jsonb
        ELSE jsonb_build_object('targets', (
            SELECT jsonb_agg(target)
            FROM (
                SELECT jsonb_build_object(
                    'instanceId', config->>'instanceId',
                    'databaseName', config->>'databaseName',
                    'sheetSha256', config->>'sheetSha256',
                    'enablePriorBackup', COALESCE((config->>'enablePriorBackup')::boolean, false),
                    'enableGhost', COALESCE((config->>'enableGhost')::boolean, false),
                    'enableSdl', COALESCE((config->>'enableSdl')::boolean, false),
                    'ghostFlags', COALESCE(config->'ghostFlags', '{}'::jsonb),
                    'checkTypes', array_agg(type)
                ) as target
                FROM plan_check_run_deduped d2
                WHERE d2.plan_id = d.plan_id
                GROUP BY
                    config->>'instanceId',
                    config->>'databaseName',
                    config->>'sheetSha256',
                    config->>'enablePriorBackup',
                    config->>'enableGhost',
                    config->>'enableSdl',
                    config->'ghostFlags'
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
                    'checkType', d3.type
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

-- Step 4: Cleanup
DROP TABLE plan_check_run_deduped;

-- Step 5: Drop type column
ALTER TABLE plan_check_run DROP COLUMN IF EXISTS type;
```

## Files Affected

| Area | Files |
|------|-------|
| Proto | `proto/store/plan_check_run.proto` |
| Store | `backend/store/plan_check_run.go` |
| Config gen | `backend/api/v1/plan_service_plan_check.go` |
| Executor | `backend/runner/plancheck/executor_combined.go` (new) |
| Scheduler | `backend/runner/plancheck/scheduler.go` |
| Consumers | `backend/runner/approval/runner.go` |
| | `backend/runner/taskrun/auto_rollout_scheduler.go` |
| | `backend/api/v1/plan_service.go` |
| | `backend/api/v1/plan_service_converter.go` |
| Migration | `backend/migrator/migration/prod/NEXT_VERSION/...` |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| In-flight RUNNING checks during migration | Re-run after migration (cheap, plan checks are idempotent) |
| API backward compatibility | Transform single record to virtual list in ListPlanCheckRuns |
| Sequential execution bottleneck | Can evolve to parallel goroutines if needed |

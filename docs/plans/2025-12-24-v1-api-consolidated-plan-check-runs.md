# V1 API Migration: Consolidated Plan Check Runs

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Simplify v1 API to match consolidated store model - one `PlanCheckRun` per plan with results tagged by target + type.

**Breaking Change:** Yes - removes `type`, `target`, `sheet` fields from `PlanCheckRun`, adds them to `Result`.

---

## Task 1: Update Proto - PlanCheckRun Message

**Files:**
- Modify: `proto/v1/v1/plan_service.proto`

**Step 1: Remove fields from PlanCheckRun**

Remove these fields from `PlanCheckRun` message (around line 412):
- `Type` enum (lines 416-425)
- `type` field (line 426)
- `target` field (line 438)
- `sheet` field (line 441)

**Step 2: Add Type enum to Result message**

Inside `PlanCheckRun.Result` message, add:

```protobuf
message Result {
  Advice.Level status = 1;
  string title = 2;
  string content = 3;
  int32 code = 4;

  // Target identification for consolidated results
  // Format: instances/{instance}/databases/{database}
  string target = 7;
  Type type = 8;

  enum Type {
    TYPE_UNSPECIFIED = 0;
    STATEMENT_ADVISE = 1;
    STATEMENT_SUMMARY_REPORT = 2;
    GHOST_SYNC = 3;
  }

  oneof report {
    SqlSummaryReport sql_summary_report = 5;
    SqlReviewReport sql_review_report = 6;
  }
  // ... rest unchanged
}
```

**Step 3: Generate proto**

```bash
buf format -w proto && buf lint proto && cd proto && buf generate
```

**Step 4: Commit**

```bash
but commit <branch> -m "proto(v1): consolidate PlanCheckRun - move type/target to Result"
```

---

## Task 2: Update Go API Converters

**Files:**
- Modify: `backend/api/v1/plan_service.go`

**Step 1: Simplify convertToPlanCheckRuns**

Replace the function (around line 1058):

```go
func convertToPlanCheckRuns(projectID string, planUID int64, runs []*store.PlanCheckRunMessage) []*v1pb.PlanCheckRun {
	var planCheckRuns []*v1pb.PlanCheckRun
	for _, run := range runs {
		planCheckRuns = append(planCheckRuns, &v1pb.PlanCheckRun{
			Name:       common.FormatPlanCheckRun(projectID, planUID, int64(run.UID)),
			Status:     convertToPlanCheckRunStatus(run.Status),
			Results:    convertToPlanCheckRunResults(run.Result.GetResults()),
			Error:      run.Result.Error,
			CreateTime: timestamppb.New(run.CreatedAt),
		})
	}
	return planCheckRuns
}
```

**Step 2: Update convertToPlanCheckRunResult**

Add target and type fields (around line 1334):

```go
func convertToPlanCheckRunResult(result *storepb.PlanCheckRunResult_Result) *v1pb.PlanCheckRun_Result {
	resultV1 := &v1pb.PlanCheckRun_Result{
		Status:  convertToPlanCheckRunResultStatus(result.Status),
		Title:   result.Title,
		Content: result.Content,
		Code:    result.Code,
		Target:  result.Target,
		Type:    convertToV1ResultType(result.Type),
	}
	// ... report conversion unchanged
	return resultV1
}

func convertToV1ResultType(t storepb.PlanCheckType) v1pb.PlanCheckRun_Result_Type {
	switch t {
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE:
		return v1pb.PlanCheckRun_Result_STATEMENT_ADVISE
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT:
		return v1pb.PlanCheckRun_Result_STATEMENT_SUMMARY_REPORT
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC:
		return v1pb.PlanCheckRun_Result_GHOST_SYNC
	default:
		return v1pb.PlanCheckRun_Result_TYPE_UNSPECIFIED
	}
}
```

**Step 3: Delete obsolete functions**

Remove:
- `expandPlanCheckRun` function
- `convertPlanCheckType` function

**Step 4: Run linter**

```bash
golangci-lint run --allow-parallel-runners backend/api/v1/plan_service.go
```

**Step 5: Commit**

```bash
but commit <branch> -m "api: update converters for consolidated PlanCheckRun"
```

---

## Task 3: Update Frontend Types

**Files:**
- Verify: `frontend/src/types/proto-es/v1/plan_service_pb.ts` (auto-generated)

After proto generation, verify the TypeScript types are updated:
- `PlanCheckRun` no longer has `type`, `target`, `sheet`
- `PlanCheckRun_Result` has `type` and `target`
- `PlanCheckRun_Result_Type` enum exists

---

## Task 4: Update Frontend - common.ts

**Files:**
- Modify: `frontend/src/components/PlanCheckRun/common.ts`

**Step 1: Update HiddenPlanCheckTypes**

```typescript
import {
  type PlanCheckRun,
  PlanCheckRun_Status,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";

export const HiddenPlanCheckTypes = new Set<PlanCheckRun_Result_Type>([
  PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
]);
```

**Step 2: Rewrite planCheckRunSummaryForCheckRunList**

```typescript
export const planCheckRunSummaryForCheckRunList = (
  planCheckRuns: PlanCheckRun[]
) => {
  const summary: PlanCheckRunSummary = {
    runningCount: 0,
    successCount: 0,
    warnCount: 0,
    errorCount: 0,
  };

  for (const checkRun of planCheckRuns) {
    // If still running, count as running
    if (checkRun.status === PlanCheckRun_Status.RUNNING) {
      summary.runningCount++;
      continue;
    }
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      summary.errorCount++;
      continue;
    }
    if (checkRun.status === PlanCheckRun_Status.CANCELED) {
      continue;
    }

    // For DONE status, analyze results by type
    // Group results by type-target to count unique checks
    const resultsByTypeTarget = new Map<string, Advice_Level>();
    for (const result of checkRun.results) {
      if (HiddenPlanCheckTypes.has(result.type)) {
        continue;
      }
      const key = `${result.type}-${result.target}`;
      const existing = resultsByTypeTarget.get(key);
      // Keep worst status
      if (!existing || result.status > existing) {
        resultsByTypeTarget.set(key, result.status);
      }
    }

    for (const status of resultsByTypeTarget.values()) {
      switch (status) {
        case Advice_Level.SUCCESS:
          summary.successCount++;
          break;
        case Advice_Level.WARNING:
          summary.warnCount++;
          break;
        case Advice_Level.ERROR:
          summary.errorCount++;
          break;
      }
    }
  }

  return summary;
};
```

---

## Task 5: Update Frontend - Plan plan-check.ts

**Files:**
- Modify: `frontend/src/components/Plan/logic/plan-check.ts`

**Step 1: Rewrite planCheckRunListForSpec**

```typescript
export const planCheckRunListForSpec = (
  planCheckRuns: PlanCheckRun[],
  spec: Plan_Spec
): PlanCheckRun[] => {
  const targets = flattenTargetsOfSpec(spec);

  // With consolidated model, filter runs that have results matching our targets
  return planCheckRuns.filter((run) => {
    return run.results.some((result) => targets.includes(result.target));
  });
};
```

---

## Task 6: Update Frontend - IssueV1 plan-check.ts

**Files:**
- Modify: `frontend/src/components/IssueV1/logic/plan-check.ts`

**Step 1: Rewrite planCheckRunListForTask**

```typescript
export const planCheckRunListForTask = (issue: ComposedIssue, task: Task) => {
  const target = databaseForTask(projectOfIssue(issue), task).name;

  // With consolidated model, return runs that have results for this target
  return issue.planCheckRunList.filter((run) => {
    return run.results.some((result) => result.target === target);
  });
};
```

**Step 2: Update planCheckRunSummaryForIssue**

```typescript
export const planCheckRunSummaryForIssue = (issue: ComposedIssue) => {
  // With consolidated model, just use all plan check runs
  return planCheckRunSummaryForCheckRunList(issue.planCheckRunList);
};
```

---

## Task 7: Update Frontend - PlanCheckRunBadgeBar.vue

**Files:**
- Modify: `frontend/src/components/PlanCheckRun/PlanCheckRunBadgeBar.vue`

**Step 1: Group by result type instead of checkRun type**

```typescript
const groupedByType = computed(() => {
  const typeMap = new Map<PlanCheckRun_Result_Type, {
    type: PlanCheckRun_Result_Type;
    results: PlanCheckRun_Result[];
    status: PlanCheckRun_Status;
  }>();

  for (const run of props.planCheckRunList) {
    for (const result of run.results) {
      if (!typeMap.has(result.type)) {
        typeMap.set(result.type, {
          type: result.type,
          results: [],
          status: run.status,
        });
      }
      typeMap.get(result.type)!.results.push(result);
    }
  }

  return orderBy(
    Array.from(typeMap.values()),
    [(g) => PlanCheckTypeOrderDict.get(g.type) ?? 99999],
    ["asc"]
  );
});
```

---

## Task 8: Update Frontend - PlanCheckRunPanel.vue

**Files:**
- Modify: `frontend/src/components/PlanCheckRun/PlanCheckRunPanel.vue`

Update to select by result type and filter results accordingly.

---

## Task 9: Update Frontend - Other Components

**Files to check and update:**
- `frontend/src/components/PlanCheckRun/PlanCheckRunDetail.vue`
- `frontend/src/components/Plan/components/PlanCheckRunStatusIcon.vue`
- `frontend/src/components/IssueV1/components/StatementSection/useSQLAdviceMarkers.ts`
- `frontend/src/components/Plan/components/StatementSection/useSQLAdviceMarkers.ts`
- Any other files using `PlanCheckRun.type`, `.target`, or `.sheet`

---

## Task 10: Build and Test

**Step 1: Build backend**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 2: Run backend linter**

```bash
golangci-lint run --allow-parallel-runners
```

**Step 3: Build frontend**

```bash
pnpm --dir frontend type-check
```

**Step 4: Run frontend linter**

```bash
pnpm --dir frontend biome:check
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Proto changes | `proto/v1/v1/plan_service.proto` |
| 2 | Go API converters | `backend/api/v1/plan_service.go` |
| 3 | Verify TS types | `frontend/src/types/proto-es/v1/plan_service_pb.ts` |
| 4 | Frontend common.ts | `frontend/src/components/PlanCheckRun/common.ts` |
| 5 | Plan plan-check.ts | `frontend/src/components/Plan/logic/plan-check.ts` |
| 6 | IssueV1 plan-check.ts | `frontend/src/components/IssueV1/logic/plan-check.ts` |
| 7 | PlanCheckRunBadgeBar | `frontend/src/components/PlanCheckRun/PlanCheckRunBadgeBar.vue` |
| 8 | PlanCheckRunPanel | `frontend/src/components/PlanCheckRun/PlanCheckRunPanel.vue` |
| 9 | Other components | Various |
| 10 | Build & test | All |

# Plan Check Run Singleton API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert plan check run from a collection resource to a singleton resource under plan.

**Architecture:** Since each plan now has exactly one plan check run record (consolidated model), the API changes from List/Batch operations to Get/single operations. Resource pattern changes from `planCheckRuns/{id}` to `planCheckRun` (singleton).

**Tech Stack:** Protocol Buffers, Go, TypeScript, Vue 3

---

## Task 1: Update Proto - PlanService

**Files:**
- Modify: `proto/v1/v1/plan_service.proto`

**Step 1: Replace ListPlanCheckRuns with GetPlanCheckRun RPC**

Replace lines 77-84:

```protobuf
  // Gets the plan check run for a deployment plan.
  // Permissions required: bb.planCheckRuns.get
  rpc GetPlanCheckRun(GetPlanCheckRunRequest) returns (PlanCheckRun) {
    option (google.api.http) = {get: "/v1/{name=projects/*/plans/*/planCheckRun}"};
    option (google.api.method_signature) = "name";
    option (bytebase.v1.permission) = "bb.planCheckRuns.get";
    option (bytebase.v1.auth_method) = IAM;
  }
```

**Step 2: Replace BatchCancelPlanCheckRuns with CancelPlanCheckRun RPC**

Replace lines 98-108:

```protobuf
  // Cancels the plan check run for a deployment plan.
  // Permissions required: bb.planCheckRuns.run
  rpc CancelPlanCheckRun(CancelPlanCheckRunRequest) returns (CancelPlanCheckRunResponse) {
    option (google.api.http) = {
      post: "/v1/{name=projects/*/plans/*/planCheckRun}:cancel"
      body: "*"
    };
    option (google.api.method_signature) = "name";
    option (bytebase.v1.permission) = "bb.planCheckRuns.run";
    option (bytebase.v1.auth_method) = IAM;
  }
```

**Step 3: Replace ListPlanCheckRunsRequest/Response with GetPlanCheckRunRequest**

Replace lines 350-379:

```protobuf
message GetPlanCheckRunRequest {
  // The name of the plan check run to retrieve.
  // Format: projects/{project}/plans/{plan}/planCheckRun
  string name = 1 [
    (google.api.field_behavior) = REQUIRED
  ];
}
```

**Step 4: Replace BatchCancelPlanCheckRunsRequest/Response with CancelPlanCheckRun messages**

Replace lines 397-410:

```protobuf
message CancelPlanCheckRunRequest {
  // The name of the plan check run to cancel.
  // Format: projects/{project}/plans/{plan}/planCheckRun
  string name = 1 [
    (google.api.field_behavior) = REQUIRED
  ];
}

message CancelPlanCheckRunResponse {}
```

**Step 5: Update PlanCheckRun resource name comment**

Update line 413:

```protobuf
message PlanCheckRun {
  // Format: projects/{project}/plans/{plan}/planCheckRun
  string name = 1;
```

**Step 6: Format and lint proto**

Run: `buf format -w proto && buf lint proto`
Expected: No errors

**Step 7: Generate proto**

Run: `cd proto && buf generate`
Expected: Generated files updated

**Step 8: Commit**

```bash
but commit consolidated-plan-check-runs -m "proto: convert plan check run to singleton resource API"
```

---

## Task 2: Update Permissions

**Files:**
- Modify: `backend/component/iam/permission.yaml`
- Modify: `backend/component/iam/permission.go`
- Modify: `backend/component/iam/acl.yaml`
- Modify: `frontend/src/types/iam/permission.ts`

**Step 1: Update permission.yaml**

Replace line 44 (`bb.planCheckRuns.list`) with:

```yaml
  - bb.planCheckRuns.get
```

**Step 2: Regenerate permission.go**

Run: `go generate ./backend/component/iam/...`
Expected: permission.go updated with new constant

**Step 3: Update acl.yaml**

Search for `bb.planCheckRuns.list` and replace with `bb.planCheckRuns.get`.

Run: `grep -n "planCheckRuns.list" backend/component/iam/acl.yaml`

Replace all occurrences.

**Step 4: Update frontend permission.ts**

Replace line 46:

```typescript
  | "bb.planCheckRuns.get"
```

**Step 5: Commit**

```bash
but commit consolidated-plan-check-runs -m "iam: rename planCheckRuns.list to planCheckRuns.get"
```

---

## Task 2.5: Create Permission Migration Script

**Files:**
- Create: `backend/migrator/migration/3.14/0008##rename_plan_check_runs_permission.sql`

**Step 1: Write the migration SQL**

```sql
-- Rename bb.planCheckRuns.list to bb.planCheckRuns.get in custom roles
UPDATE role
SET permissions = jsonb_set(
    permissions,
    '{permissions}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN elem = 'bb.planCheckRuns.list' THEN 'bb.planCheckRuns.get'
                ELSE elem
            END
        )
        FROM jsonb_array_elements_text(permissions->'permissions') AS elem
    )
)
WHERE permissions->'permissions' @> '"bb.planCheckRuns.list"';
```

**Step 2: Commit**

```bash
but commit consolidated-plan-check-runs -m "migration: rename planCheckRuns.list to planCheckRuns.get in custom roles"
```

---

## Task 3: Update Backend Common Helpers

**Files:**
- Modify: `backend/common/resource_name.go`

**Step 1: Add GetProjectIDPlanID from planCheckRun name**

Add after `GetProjectIDPlanIDPlanCheckRunID` (around line 241):

```go
// GetProjectIDPlanIDFromPlanCheckRun returns the project ID and plan ID from a plan check run singleton resource name.
// Format: projects/{project}/plans/{plan}/planCheckRun
func GetProjectIDPlanIDFromPlanCheckRun(name string) (string, int64, error) {
	// Remove the trailing "/planCheckRun" suffix
	if !strings.HasSuffix(name, "/planCheckRun") {
		return "", 0, errors.Errorf("invalid plan check run name %q, expected suffix /planCheckRun", name)
	}
	planName := strings.TrimSuffix(name, "/planCheckRun")
	projectID, planID, err := GetProjectIDPlanID(planName)
	if err != nil {
		return "", 0, err
	}
	return projectID, int64(planID), nil
}
```

**Step 2: Update FormatPlanCheckRun to singleton pattern**

Replace the function at line 565:

```go
// FormatPlanCheckRun formats a plan check run singleton resource name.
// Format: projects/{project}/plans/{plan}/planCheckRun
func FormatPlanCheckRun(projectID string, planUID int64) string {
	return fmt.Sprintf("%s/planCheckRun", FormatPlan(projectID, planUID))
}
```

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/common/resource_name.go`
Expected: No errors

**Step 4: Commit**

```bash
but commit consolidated-plan-check-runs -m "common: update plan check run helpers for singleton resource"
```

---

## Task 4: Update Backend API - PlanService

**Files:**
- Modify: `backend/api/v1/plan_service.go`

**Step 1: Replace ListPlanCheckRuns with GetPlanCheckRun**

Replace the method (around line 437-463):

```go
// GetPlanCheckRun gets the plan check run for the plan.
func (s *PlanService) GetPlanCheckRun(ctx context.Context, request *connect.Request[v1pb.GetPlanCheckRunRequest]) (*connect.Response[v1pb.PlanCheckRun], error) {
	req := request.Msg
	projectID, planUID, err := common.GetProjectIDPlanIDFromPlanCheckRun(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	planCheckRun, err := s.store.GetPlanCheckRun(ctx, planUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check run, error: %v", err))
	}
	if planCheckRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan check run not found for plan %d", planUID))
	}

	converted := convertToPlanCheckRun(projectID, planUID, planCheckRun)
	return connect.NewResponse(converted), nil
}
```

**Step 2: Replace BatchCancelPlanCheckRuns with CancelPlanCheckRun**

Replace the method (around line 658-721):

```go
// CancelPlanCheckRun cancels the plan check run for a plan.
func (s *PlanService) CancelPlanCheckRun(ctx context.Context, request *connect.Request[v1pb.CancelPlanCheckRunRequest]) (*connect.Response[v1pb.CancelPlanCheckRunResponse], error) {
	req := request.Msg
	projectID, planUID, err := common.GetProjectIDPlanIDFromPlanCheckRun(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	planCheckRun, err := s.store.GetPlanCheckRun(ctx, planUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check run, error: %v", err))
	}
	if planCheckRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan check run not found for plan %d", planUID))
	}

	if planCheckRun.Status != store.PlanCheckRunStatusRunning {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("plan check run is not running"))
	}

	// Cancel the plan check run.
	if cancelFunc, ok := s.stateCfg.RunningPlanCheckRunsCancelFunc.Load(planCheckRun.UID); ok {
		cancelFunc.(context.CancelFunc)()
	}

	// Update the status to canceled.
	if err := s.store.BatchCancelPlanCheckRuns(ctx, []int{planCheckRun.UID}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to cancel plan check run, error: %v", err))
	}

	return connect.NewResponse(&v1pb.CancelPlanCheckRunResponse{}), nil
}
```

**Step 3: Delete parsePlanCheckRunFilter method**

Remove the entire `parsePlanCheckRunFilter` method (lines 465-593) - no longer needed.

**Step 4: Update convertToPlanCheckRuns to convertToPlanCheckRun**

Replace around line 1058:

```go
func convertToPlanCheckRun(projectID string, planUID int64, run *store.PlanCheckRunMessage) *v1pb.PlanCheckRun {
	return &v1pb.PlanCheckRun{
		Name:       common.FormatPlanCheckRun(projectID, planUID),
		Status:     convertToPlanCheckRunStatus(run.Status),
		Results:    convertToPlanCheckRunResults(run.Result.GetResults()),
		Error:      run.Result.Error,
		CreateTime: timestamppb.New(run.CreatedAt),
	}
}
```

**Step 5: Update convertToPlan to use new helper**

In `convertToPlan` function (around line 1031-1043), update the plan check run status count logic:

```go
	planCheckRun, err := s.GetPlanCheckRun(ctx, plan.UID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan check run for plan uid %d", plan.UID)
	}
	if planCheckRun != nil {
		p.PlanCheckRunStatusCount[string(planCheckRun.Status)]++
		for _, result := range planCheckRun.Result.Results {
			p.PlanCheckRunStatusCount[storepb.Advice_Status_name[int32(result.Status)]]++
		}
	}
```

**Step 6: Remove unused imports**

Remove CEL-related imports that were used by `parsePlanCheckRunFilter`:
- `"github.com/google/cel-go/cel"`
- `celast "github.com/google/cel-go/common/ast"`
- `celoperators "github.com/google/cel-go/common/operators"`

**Step 7: Run linter**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/plan_service.go`
Expected: No errors (fix any that appear)

**Step 8: Commit**

```bash
but commit consolidated-plan-check-runs -m "api: convert plan check run to singleton resource API"
```

---

## Task 5: Update Frontend - Store and Components

**Files:**
- Modify: `frontend/src/store/modules/v1/experimental-issue.ts`
- Modify: `frontend/src/store/modules/v1/common.ts`
- Modify: `frontend/src/components/Plan/logic/poller/utils.ts`
- Modify: `frontend/src/components/PlanCheckRun/PlanCheckRunDetail.vue`

**Step 1: Update experimental-issue.ts**

Replace the plan check run fetching logic (lines 66-76):

```typescript
    if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.get")) {
      const request = create(GetPlanCheckRunRequestSchema, {
        name: `${issue.plan}/planCheckRun`,
      });
      try {
        const response =
          await planServiceClientConnect.getPlanCheckRun(request);
        issue.planCheckRunList = [response];
      } catch {
        // Plan check run might not exist yet
        issue.planCheckRunList = [];
      }
    }
```

Update imports at the top:

```typescript
import {
  CreatePlanRequestSchema,
  GetPlanRequestSchema,
  GetPlanCheckRunRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
```

**Step 2: Update common.ts - add getProjectNamePlanId helper**

Add after `getProjectNamePlanIdPlanCheckRunId` (around line 85):

```typescript
export const getProjectNamePlanIdFromPlanCheckRun = (name: string): [string, string] => {
  // Format: projects/{project}/plans/{plan}/planCheckRun
  if (!name.endsWith("/planCheckRun")) {
    throw new Error(`Invalid plan check run name: ${name}`);
  }
  const planName = name.replace(/\/planCheckRun$/, "");
  const tokens = getNameParentTokens(planName, [
    projectNamePrefix,
    planNamePrefix,
  ]);
  return [tokens[0], tokens[1]];
};
```

**Step 3: Update poller/utils.ts**

Replace `refreshPlanCheckRuns` function (lines 31-46):

```typescript
export const refreshPlanCheckRuns = async (
  plan: Plan,
  project: Project,
  planCheckRuns: Ref<PlanCheckRun[]>
): Promise<void> => {
  if (!hasProjectPermissionV2(project, "bb.planCheckRuns.get")) {
    return;
  }

  const request = create(GetPlanCheckRunRequestSchema, {
    name: `${plan.name}/planCheckRun`,
  });
  try {
    const response = await planServiceClientConnect.getPlanCheckRun(request);
    planCheckRuns.value = [response];
  } catch {
    // Plan check run might not exist yet
    planCheckRuns.value = [];
  }
};
```

Update imports:

```typescript
import {
  GetPlanRequestSchema,
  GetPlanCheckRunRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
```

**Step 4: Update PlanCheckRunDetail.vue**

Update imports (around line 246-268):

```typescript
import {
  getProjectNamePlanIdFromPlanCheckRun,
  planNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
```

```typescript
import {
  CancelPlanCheckRunRequestSchema,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
```

Update `cancelPlanCheckRun` function (lines 560-572):

```typescript
const cancelPlanCheckRun = async () => {
  const request = create(CancelPlanCheckRunRequestSchema, {
    name: props.planCheckRun.name,
  });
  await planServiceClientConnect.cancelPlanCheckRun(request);
  if (usePlanCheckRunContext()) {
    usePlanCheckRunContext().events.emit("status-changed");
  }
};
```

**Step 5: Format frontend**

Run: `pnpm --dir frontend biome:check`
Expected: No errors

**Step 6: Type check frontend**

Run: `pnpm --dir frontend type-check`
Expected: No errors

**Step 7: Commit**

```bash
but commit consolidated-plan-check-runs -m "frontend: update to singleton plan check run API"
```

---

## Task 6: Build and Test

**Step 1: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run backend linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors (run repeatedly until clean)

**Step 3: Run related tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run PlanCheck`
Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run PlanCheck`
Expected: Tests pass

**Step 4: Final commit**

```bash
but commit consolidated-plan-check-runs -m "chore: fix build and lint issues"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Proto changes | `proto/v1/v1/plan_service.proto` |
| 2 | Permission changes | `backend/component/iam/*.yaml`, `frontend/src/types/iam/permission.ts` |
| 2.5 | Permission migration | `backend/migrator/migration/3.14/0008##rename_plan_check_runs_permission.sql` |
| 3 | Backend common helpers | `backend/common/resource_name.go` |
| 4 | Backend API | `backend/api/v1/plan_service.go` |
| 5 | Frontend updates | `frontend/src/store/`, `frontend/src/components/` |
| 6 | Build & test | All |

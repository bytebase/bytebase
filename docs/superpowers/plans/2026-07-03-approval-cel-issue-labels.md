# Approval CEL Issue Labels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `issue.labels` to `CHANGE_DATABASE` approval-flow CEL and keep approval state correct when labels change.

**Architecture:** Treat issue labels as approval-affecting input for database-change issues only. The approval runner canonicalizes labels once, injects the same slice into CEL variables, and passes that slice to a guarded store update so stale label computations cannot write approval state. Label updates reset approval finding only while the linked plan has no rollout.

**Tech Stack:** Go, PostgreSQL JSONB, cel-go, ConnectRPC services, React, TypeScript, existing CEL expression editor.

---

## File Structure

- Modify `backend/common/cel_attributes.go`: add the shared `issue.labels` CEL attribute name.
- Modify `backend/common/cel.go`: add `issue.labels` to source-specific approval CEL factors as `list<string>`.
- Modify `backend/common/cel_test.go`: cover compile behavior for `issue.labels` and fallback rejection.
- Modify `backend/store/issue.go`: add a label-aware guarded approval payload update.
- Modify `backend/store/plan_approval_input_version_test.go`: cover label-aware write success and stale-label rejection.
- Modify `backend/runner/approval/runner.go`: canonicalize labels, inject them into database-change CEL variables, and use the label-aware guarded write.
- Modify `backend/runner/approval/runner_test.go`: cover approval rule matching on `issue.labels`.
- Modify `backend/api/v1/issue_service.go`: reset approval finding and enqueue approval finding when labels change before rollout; skip reset after rollout.
- Modify `backend/api/v1/issue_service_test.go`: cover label reset semantics before and after rollout.
- Modify `frontend/src/utils/cel-attributes.ts`: add the frontend `issue.labels` constant.
- Modify `frontend/src/plugins/cel/types/factor.ts`: add a list factor type for `issue.labels`.
- Modify `frontend/src/plugins/cel/types/operator.ts`: support list membership operators for `issue.labels`.
- Modify `frontend/src/plugins/cel/types/simple.ts`: add a list-membership condition shape.
- Modify `frontend/src/plugins/cel/logic/build.ts`: build `"value" in issue.labels` and `!("value" in issue.labels)`.
- Modify `frontend/src/react/components/CustomApproval/utils.ts`: expose `issue.labels` for `CHANGE_DATABASE` only.
- Create `frontend/src/plugins/cel/logic/build.test.ts`: cover list factor build behavior.
- Create `frontend/src/react/components/CustomApproval/utils.test.ts`: cover approval factor exposure.

---

### Task 1: Backend CEL Attribute And Compile Tests

**Files:**
- Modify: `backend/common/cel_attributes.go`
- Modify: `backend/common/cel.go`
- Modify: `backend/common/cel_test.go`

- [ ] **Step 1: Write failing tests for approval and fallback CEL environments**

Add these cases to `backend/common/cel_test.go` near `TestApprovalFactorsIncludesRiskLevel`:

```go
func TestApprovalFactorsIncludesIssueLabels(t *testing.T) {
	a := require.New(t)

	e, err := cel.NewEnv(ApprovalFactors...)
	a.NoError(err)

	_, issues := e.Compile(`"prod" in issue.labels`)
	a.Nil(issues)

	_, issues = e.Compile(`!("security" in issue.labels) && risk.level == "HIGH"`)
	a.Nil(issues)
}

func TestFallbackApprovalFactorsExcludesIssueLabels(t *testing.T) {
	a := require.New(t)

	e, err := cel.NewEnv(FallbackApprovalFactors...)
	a.NoError(err)

	_, issues := e.Compile(`"prod" in issue.labels`)
	a.NotNil(issues)
	a.Error(issues.Err())
}
```

- [ ] **Step 2: Run the focused common tests and verify failure**

Run:

```bash
go test -v -count=1 ./backend/common -run '^(TestApprovalFactorsIncludesIssueLabels|TestFallbackApprovalFactorsExcludesIssueLabels)$'
```

Expected: `TestApprovalFactorsIncludesIssueLabels` fails because `issue.labels` is undeclared.

- [ ] **Step 3: Add the backend CEL attribute and type**

In `backend/common/cel_attributes.go`, add:

```go
// CEL attribute names for issue scope.
const (
	// CELAttributeIssueLabels is the canonical labels attached to the issue.
	CELAttributeIssueLabels = "issue.labels"
)
```

In `backend/common/cel.go`, add this variable to `ApprovalFactors` after the risk scope entry:

```go
	// Issue scope
	cel.Variable(CELAttributeIssueLabels, cel.ListType(cel.StringType)),
```

Do not add the variable to `FallbackApprovalFactors`.

- [ ] **Step 4: Run the focused common tests and verify pass**

Run:

```bash
go test -v -count=1 ./backend/common -run '^(TestApprovalFactorsIncludesIssueLabels|TestFallbackApprovalFactorsExcludesIssueLabels)$'
```

Expected: both tests pass.

- [ ] **Step 5: Commit backend CEL env change**

Run:

```bash
gofmt -w backend/common/cel_attributes.go backend/common/cel.go backend/common/cel_test.go
git add backend/common/cel_attributes.go backend/common/cel.go backend/common/cel_test.go
git commit -m "feat: add issue labels to approval CEL"
```

---

### Task 2: Store Guard For Stale Issue Labels

**Files:**
- Modify: `backend/store/issue.go`
- Modify: `backend/store/plan_approval_input_version_test.go`

- [ ] **Step 1: Write failing store tests for label-aware approval writes**

Add these tests to `backend/store/plan_approval_input_version_test.go` after `TestUpdateIssuePayloadIfPlanApprovalInputVersionSkipsIssueWithoutPlan`:

```go
func TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsUpdatesMatchingLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod", "security"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2, []string{"security", "prod", "prod"})
	require.NoError(t, err)
	require.True(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_HIGH, got.Payload.GetRiskLevel())
	require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
}

func TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsSkipsStaleLabels(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	issue, err := s.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "issue-a",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Description:  "",
		Payload: &storepb.Issue{
			Labels:    []string{"prod"},
			RiskLevel: storepb.RiskLevel_LOW,
		},
		PlanUID: &plan.UID,
	})
	require.NoError(t, err)

	updatedIssue, err := s.UpdateIssue(ctx, "project-a", issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{Labels: []string{"stage"}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"stage"}, updatedIssue.Payload.GetLabels())

	updated, err := s.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, "project-a", issue.UID, &storepb.Issue{
		RiskLevel: storepb.RiskLevel_HIGH,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  true,
			ApprovalInputVersion: 2,
		},
	}, 2, []string{"prod"})
	require.NoError(t, err)
	require.False(t, updated)

	got, err := s.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"project-a"}, UID: &issue.UID})
	require.NoError(t, err)
	require.Equal(t, storepb.RiskLevel_LOW, got.Payload.GetRiskLevel())
	require.Nil(t, got.Payload.GetApproval())
}
```

- [ ] **Step 2: Run the store tests and verify failure**

Run:

```bash
go test -v -count=1 ./backend/store -run '^(TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsUpdatesMatchingLabels|TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsSkipsStaleLabels)$'
```

Expected: compile failure because `UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels` does not exist.

- [ ] **Step 3: Implement the label-aware guarded update**

In `backend/store/issue.go`, add `encoding/json` to imports:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)
```

Add this method after `UpdateIssuePayloadIfPlanApprovalInputVersion`:

```go
func (s *Store) UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx context.Context, projectID string, uid int64, payload *storepb.Issue, approvalInputVersion int64, labels []string) (bool, error) {
	p, err := protojson.Marshal(payload)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal payload")
	}
	labelBytes, err := json.Marshal(CanonicalizeIssueLabels(labels))
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal issue labels")
	}

	q := qb.Q().Space(`
		UPDATE issue
		SET
			updated_at = ?,
			payload = payload || ?
		WHERE project = ?
		  AND id = ?
		  AND COALESCE(payload->'labels', '[]'::jsonb) = ?::jsonb
		  AND EXISTS (
			SELECT 1
			FROM plan
			WHERE plan.project = issue.project
			  AND plan.id = issue.plan_id
			  AND COALESCE((plan.config->>'approvalInputVersion')::bigint, 0) = ?
		  )`, time.Now(), p, projectID, uid, string(labelBytes), approvalInputVersion)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "failed to inspect issue update")
	}
	return rowsAffected > 0, nil
}
```

- [ ] **Step 4: Run the focused store tests and verify pass**

Run:

```bash
go test -v -count=1 ./backend/store -run '^(TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsUpdatesMatchingLabels|TestUpdateIssuePayloadIfPlanApprovalInputVersionAndLabelsSkipsStaleLabels)$'
```

Expected: both tests pass.

- [ ] **Step 5: Commit store guard**

Run:

```bash
gofmt -w backend/store/issue.go backend/store/plan_approval_input_version_test.go
git add backend/store/issue.go backend/store/plan_approval_input_version_test.go
git commit -m "fix: guard approval writes by issue labels"
```

---

### Task 3: Approval Runner Uses Observed Canonical Labels

**Files:**
- Modify: `backend/runner/approval/runner.go`
- Modify: `backend/runner/approval/runner_test.go`

- [ ] **Step 1: Write failing approval runner tests**

Add these tests to `backend/runner/approval/runner_test.go` near `TestApprovalTemplateMatchesUnspecifiedStatementSQLType`:

```go
func TestApprovalTemplateMatchesIssueLabels(t *testing.T) {
	a := require.New(t)

	celVars := map[string]any{
		common.CELAttributeResourceProjectID: "project",
		common.CELAttributeIssueLabels:       []string{"prod", "security"},
	}
	injectRiskLevelIntoCELVars([]map[string]any{celVars}, storepb.RiskLevel_HIGH)

	approvalTemplate, err := getApprovalTemplate(&storepb.WorkspaceApprovalSetting{
		Rules: []*storepb.WorkspaceApprovalSetting_Rule{
			{
				Source:    storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
				Condition: &expr.Expr{Expression: `"prod" in issue.labels && risk.level == "HIGH"`},
				Template:  &storepb.ApprovalTemplate{Title: "Production label rule"},
			},
		},
	}, storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE, []map[string]any{celVars})
	a.NoError(err)
	a.NotNil(approvalTemplate)
	a.Equal("Production label rule", approvalTemplate.Title)
}

func TestInjectIssueLabelsIntoCELVars(t *testing.T) {
	celVarsList := []map[string]any{
		{common.CELAttributeResourceProjectID: "project-a"},
		{common.CELAttributeResourceProjectID: "project-a"},
	}

	injectIssueLabelsIntoCELVars(celVarsList, []string{" security ", "prod", "prod"})

	for _, celVars := range celVarsList {
		require.Equal(t, []string{"prod", "security"}, celVars[common.CELAttributeIssueLabels])
	}
}
```

- [ ] **Step 2: Run the focused approval runner tests and verify failure**

Run:

```bash
go test -v -count=1 ./backend/runner/approval -run '^(TestApprovalTemplateMatchesIssueLabels|TestInjectIssueLabelsIntoCELVars)$'
```

Expected: compile failure for `injectIssueLabelsIntoCELVars`, or CEL compile failure if Task 1 has not been applied.

- [ ] **Step 3: Implement label injection and guarded runner write**

In `backend/runner/approval/runner.go`, add:

```go
func injectIssueLabelsIntoCELVars(celVarsList []map[string]any, labels []string) {
	canonicalLabels := store.CanonicalizeIssueLabels(labels)
	if canonicalLabels == nil {
		canonicalLabels = []string{}
	}
	for _, celVars := range celVarsList {
		celVars[common.CELAttributeIssueLabels] = canonicalLabels
	}
}
```

In `findApprovalTemplateForIssue`, compute labels before the inner closure:

```go
	approvalLabels := store.CanonicalizeIssueLabels(payload.GetLabels())
	if approvalLabels == nil {
		approvalLabels = []string{}
	}
```

After risk-level injection for `CHANGE_DATABASE`, inject labels:

```go
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE {
			riskLevel := calculateRiskLevelFromCELVars(celVarsList)
			injectRiskLevelIntoCELVars(celVarsList, riskLevel)
			injectIssueLabelsIntoCELVars(celVarsList, approvalLabels)
		}
```

Replace the database-change payload write call:

```go
		updated, err := stores.UpdateIssuePayloadIfPlanApprovalInputVersionAndLabels(ctx, issue.ProjectID, issue.UID, payloadPatch, approvalInputVersion, approvalLabels)
```

- [ ] **Step 4: Run the focused approval runner tests and verify pass**

Run:

```bash
go test -v -count=1 ./backend/runner/approval -run '^(TestApprovalTemplateMatchesIssueLabels|TestInjectIssueLabelsIntoCELVars)$'
```

Expected: both tests pass.

- [ ] **Step 5: Commit approval runner label matching**

Run:

```bash
gofmt -w backend/runner/approval/runner.go backend/runner/approval/runner_test.go
git add backend/runner/approval/runner.go backend/runner/approval/runner_test.go
git commit -m "feat: match approval rules on issue labels"
```

---

### Task 4: Label Updates Reset Approval Before Rollout Only

**Files:**
- Modify: `backend/api/v1/issue_service.go`
- Modify: `backend/api/v1/issue_service_test.go`

- [ ] **Step 1: Write failing service tests for label reset behavior**

Add tests in `backend/api/v1/issue_service_test.go` using existing service/store test setup in that file. The tests should create a `DATABASE_CHANGE` issue linked to a plan with `ApprovalInputVersion: 2` and a completed approval payload, then update labels through `IssueService.UpdateIssue`.

The pre-rollout assertion must check:

```go
require.False(t, got.Payload.GetApproval().GetApprovalFindingDone())
require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
```

The post-rollout assertion must mark the plan as rolled out by calling:

```go
approvalInputVersion := int64(2)
marked, _, err := stores.CreateRolloutTasks(ctx, "project-a", plan.UID, &approvalInputVersion, nil)
require.NoError(t, err)
require.True(t, marked)
```

Then update labels and assert:

```go
require.True(t, got.Payload.GetApproval().GetApprovalFindingDone())
require.EqualValues(t, 2, got.Payload.GetApproval().GetApprovalInputVersion())
```

- [ ] **Step 2: Run the focused service tests and verify failure**

Run the exact test names added in Step 1:

```bash
go test -v -count=1 ./backend/api/v1 -run '^(TestUpdateIssueLabelsResetsApprovalBeforeRollout|TestUpdateIssueLabelsDoesNotResetApprovalAfterRollout)$'
```

Expected: the pre-rollout test fails because labels update without resetting approval.

- [ ] **Step 3: Implement label-driven reset in `UpdateIssue`**

In `backend/api/v1/issue_service.go`, track label changes:

```go
	var labelsChanged bool
```

Inside the `"labels"` update-mask case, set it after confirming this is not a no-op:

```go
			labelsChanged = true
```

After `issue, err = s.store.UpdateIssue(...)` and before comment creation, add:

```go
	if labelsChanged && issue.Type == storepb.Issue_DATABASE_CHANGE && issue.PlanUID != nil {
		plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: issue.ProjectID, UID: issue.PlanUID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get plan"))
		}
		if plan == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("plan not found"))
		}
		if !plan.Config.GetHasRollout() {
			updatedIssue, updated, err := resetIssueApprovalFindingIfPlanApprovalInputVersion(ctx, s.store, issue, plan.Config.GetApprovalInputVersion())
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to reset approval finding after issue label update"))
			}
			if updated {
				issue = updatedIssue
				s.bus.ApprovalCheckChan <- bus.IssueRef{ProjectID: issue.ProjectID, UID: issue.UID}
			}
		}
	}
```

- [ ] **Step 4: Run the focused service tests and verify pass**

Run:

```bash
go test -v -count=1 ./backend/api/v1 -run '^(TestUpdateIssueLabelsResetsApprovalBeforeRollout|TestUpdateIssueLabelsDoesNotResetApprovalAfterRollout)$'
```

Expected: both tests pass.

- [ ] **Step 5: Commit label reset behavior**

Run:

```bash
gofmt -w backend/api/v1/issue_service.go backend/api/v1/issue_service_test.go
git add backend/api/v1/issue_service.go backend/api/v1/issue_service_test.go
git commit -m "fix: reset approval on issue label changes"
```

---

### Task 5: Frontend CEL Editor List Factor

**Files:**
- Modify: `frontend/src/utils/cel-attributes.ts`
- Modify: `frontend/src/plugins/cel/types/factor.ts`
- Modify: `frontend/src/plugins/cel/types/operator.ts`
- Modify: `frontend/src/plugins/cel/types/simple.ts`
- Modify: `frontend/src/plugins/cel/logic/build.ts`
- Modify: `frontend/src/react/components/CustomApproval/utils.ts`
- Create: `frontend/src/plugins/cel/logic/build.test.ts`
- Create: `frontend/src/react/components/CustomApproval/utils.test.ts`

- [ ] **Step 1: Write failing frontend tests**

Create `frontend/src/plugins/cel/logic/build.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/plugins/cel";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { buildCELExpr } from "./build";

describe("buildCELExpr", () => {
  it("builds label membership as value in issue.labels", async () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, "prod"],
    };

    const built = await buildCELExpr(expr);

    expect(built?.exprKind.case).toBe("callExpr");
    const call = built?.exprKind.value;
    expect(call?.function).toBe("@in");
    expect(call?.args[0].exprKind.case).toBe("constExpr");
    expect(
      call?.args[0].exprKind.value.constantKind.case
    ).toBe("stringValue");
    expect(
      call?.args[0].exprKind.value.constantKind.value
    ).toBe("prod");
    expect(call?.args[1].exprKind.case).toBe("identExpr");
    expect(call?.args[1].exprKind.value.name).toBe(CEL_ATTRIBUTE_ISSUE_LABELS);
  });
});
```

Create `frontend/src/react/components/CustomApproval/utils.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { getApprovalFactorList } from "./utils";

describe("getApprovalFactorList", () => {
  it("exposes issue labels for database-change approval rules only", () => {
    expect(
      getApprovalFactorList(WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE)
    ).toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
    expect(
      getApprovalFactorList(
        WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
      )
    ).not.toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
    expect(
      getApprovalFactorList(WorkspaceApprovalSetting_Rule_Source.REQUEST_ACCESS)
    ).not.toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
  });
});
```

- [ ] **Step 2: Run frontend tests and verify failure**

Run:

```bash
pnpm --dir frontend test --run build.test.ts utils.test.ts
```

Expected: compile failure because `CEL_ATTRIBUTE_ISSUE_LABELS`, list factors, or `@contains` are not defined.

- [ ] **Step 3: Add frontend constants and list factor type**

In `frontend/src/utils/cel-attributes.ts`, add:

```ts
// CEL attribute names for issue scope.
export const CEL_ATTRIBUTE_ISSUE_LABELS = "issue.labels";
```

In `frontend/src/plugins/cel/types/factor.ts`, import the new constant and add:

```ts
export const ListFactorList = [CEL_ATTRIBUTE_ISSUE_LABELS] as const;
export type ListFactor = (typeof ListFactorList)[number];
```

Extend `Factor`:

```ts
export type Factor =
  | NumberFactor
  | StringFactor
  | BooleanFactor
  | TimestampFactor
  | ListFactor;
```

Add:

```ts
export const isListFactor = (factor: string): factor is ListFactor => {
  return ListFactorList.includes(factor as ListFactor);
};
```

- [ ] **Step 4: Add membership operators and builder support**

In `frontend/src/plugins/cel/types/operator.ts`, add:

```ts
export const ListMembershipOperatorList = ["@contains", "@not_contains"] as const;
export type ListMembershipOperator = (typeof ListMembershipOperatorList)[number];
```

Include `ListMembershipOperator` in `ConditionOperator`, add `isListMembershipOperator`, and add this to `OperatorList`:

```ts
  [CEL_ATTRIBUTE_ISSUE_LABELS]: uniq([...ListMembershipOperatorList]),
```

In `frontend/src/plugins/cel/types/simple.ts`, add:

```ts
export interface ListMembershipExpr extends BaseConditionExpr {
  operator: ListMembershipOperator;
  args: [ListFactor, string];
}
```

Include it in `ConditionExpr` and add:

```ts
export const isListMembershipExpr = (
  expr: SimpleExpr
): expr is ListMembershipExpr => {
  return isConditionExpr(expr) && isListMembershipOperator(expr.operator);
};
```

In `frontend/src/plugins/cel/logic/build.ts`, import `isListMembershipExpr` and handle it before string expressions:

```ts
    if (isListMembershipExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      const membership = wrapCallExpr("@in", [
        wrapConstExpr(value),
        wrapIdentExpr(factor),
      ]);
      if (operator === "@not_contains") {
        return wrapCallExpr("!_", [membership]);
      }
      return membership;
    }
```

- [ ] **Step 5: Expose `issue.labels` for `CHANGE_DATABASE` only**

In `frontend/src/react/components/CustomApproval/utils.ts`, import `CEL_ATTRIBUTE_ISSUE_LABELS` and add it to the `CHANGE_DATABASE` factor list:

```ts
      CEL_ATTRIBUTE_RISK_LEVEL,
      CEL_ATTRIBUTE_ISSUE_LABELS,
```

Do not add it to `SOURCE_UNSPECIFIED` or other source factor lists.

- [ ] **Step 6: Run frontend tests and verify pass**

Run:

```bash
pnpm --dir frontend test --run build.test.ts utils.test.ts
```

Expected: the new frontend tests pass.

- [ ] **Step 7: Commit frontend CEL editor support**

Run:

```bash
pnpm --dir frontend fix
git add frontend/src/utils/cel-attributes.ts frontend/src/plugins/cel/types/factor.ts frontend/src/plugins/cel/types/operator.ts frontend/src/plugins/cel/types/simple.ts frontend/src/plugins/cel/logic/build.ts frontend/src/plugins/cel/logic/build.test.ts frontend/src/react/components/CustomApproval/utils.ts frontend/src/react/components/CustomApproval/utils.test.ts
git commit -m "feat: expose issue labels in approval CEL editor"
```

---

### Task 6: Full Verification

**Files:**
- No code changes expected.

- [ ] **Step 1: Run backend focused tests**

Run:

```bash
go test -v -count=1 ./backend/common ./backend/store ./backend/runner/approval ./backend/api/v1 -run '(IssueLabels|ApprovalInputVersion|UpdateIssueLabels|ApprovalFactors)'
```

Expected: all selected tests pass.

- [ ] **Step 2: Run Go formatting**

Run:

```bash
gofmt -w backend/common/cel_attributes.go backend/common/cel.go backend/common/cel_test.go backend/store/issue.go backend/store/plan_approval_input_version_test.go backend/runner/approval/runner.go backend/runner/approval/runner_test.go backend/api/v1/issue_service.go backend/api/v1/issue_service_test.go
```

Expected: command exits with code 0.

- [ ] **Step 3: Run backend lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: no lint issues. If it reports issues, fix them and rerun the same command until it reports no issues.

- [ ] **Step 4: Run frontend checks**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: all commands pass.

- [ ] **Step 5: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: build succeeds and writes `./bytebase-build/bytebase`.

- [ ] **Step 6: Final git status**

Run:

```bash
git status --short
```

Expected: clean worktree.

# BYT-9422 Plan Check Sheet Identity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist SQL sheet identity on plan-check results so approval evaluation handles multiple specs targeting the same database without spec-order-dependent governance bypass.

**Architecture:** Add `sheet_sha256` to the stored plan-check result proto, propagate it from `plancheck.CheckTarget`, and use it in approval statement-summary lookup keys. Approval generation will detect legacy completed summary results without sheet identity, recreate the plan-check run, tickle the scheduler, and wait for the normal async completion path.

**Tech Stack:** Go, protobuf, buf, protojson, Bytebase store layer, plancheck runner, approval runner, testify.

---

## File Structure

- Modify `proto/store/store/plan_check_run.proto`: add internal stored result field `sheet_sha256`.
- Modify generated files under `backend/generated-go/store/`: regenerate after the proto change.
- Modify `backend/runner/plancheck/executor_combined.go`: extract result-tagging helper and copy `SheetSha256`.
- Create `backend/runner/plancheck/executor_combined_test.go`: test result tagging without database dependencies.
- Modify `backend/runner/approval/runner.go`: pass the bus into approval finding, detect legacy plan-check results, recreate stale runs, and key summary reports by sheet.
- Extend `backend/runner/approval/runner_test.go`: test summary map matching and legacy stale detection.
- Modify approval call sites in `backend/api/v1/issue_hook.go`, `backend/api/v1/issue_service.go`, and `backend/api/v1/plan_service.go`: pass the service bus to `approval.FindAndApplyApprovalTemplate`.

## Task 1: Add Stored Sheet Identity

**Files:**
- Modify: `proto/store/store/plan_check_run.proto`
- Regenerate: `backend/generated-go/store/plan_check_run.pb.go`
- Regenerate: `backend/generated-go/store/plan_check_run_equal.pb.go`

- [ ] **Step 1: Add the proto field**

Edit `proto/store/store/plan_check_run.proto` inside `message PlanCheckRunResult.Result`:

```proto
    // Target identification for consolidated results
    // Format: instances/{instance}/databases/{database}
    string target = 7;
    PlanCheckType type = 8;

    // sheet_sha256 is the content hash of the SQL sheet used to produce this result.
    // Empty for checks that are not tied to a SQL sheet.
    string sheet_sha256 = 9;
```

- [ ] **Step 2: Format and regenerate protobufs**

Run:

```bash
buf format -w proto
cd proto && buf generate
```

Expected: command exits `0`, and generated store Go files include `SheetSha256 string`.

- [ ] **Step 3: Verify generated field exists**

Run:

```bash
rg -n "SheetSha256|sheet_sha256" backend/generated-go/store/plan_check_run.pb.go backend/generated-go/store/plan_check_run_equal.pb.go
```

Expected: matches in both generated files.

- [ ] **Step 4: Commit proto identity field**

Run:

```bash
git add proto/store/store/plan_check_run.proto backend/generated-go/store/plan_check_run.pb.go backend/generated-go/store/plan_check_run_equal.pb.go
git commit -m "feat(plancheck): persist sheet identity in results"
```

## Task 2: Propagate Sheet Identity From Plan Check Targets

**Files:**
- Modify: `backend/runner/plancheck/executor_combined.go`
- Create: `backend/runner/plancheck/executor_combined_test.go`

- [ ] **Step 1: Write the failing result-tagging test**

Create `backend/runner/plancheck/executor_combined_test.go`:

```go
package plancheck

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestTagPlanCheckResultsCopiesTargetMetadata(t *testing.T) {
	results := []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.Advice_SUCCESS,
			Title:  "ok",
		},
	}
	target := &CheckTarget{
		Target:      "instances/prod/databases/app",
		SheetSha256: "abc123",
	}

	tagPlanCheckResults(results, target, storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT)

	require.Len(t, results, 1)
	require.Equal(t, "instances/prod/databases/app", results[0].Target)
	require.Equal(t, "abc123", results[0].SheetSha256)
	require.Equal(t, storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT, results[0].Type)
}
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
go test -v -count=1 ./backend/runner/plancheck -run ^TestTagPlanCheckResultsCopiesTargetMetadata$
```

Expected: FAIL with `undefined: tagPlanCheckResults`.

- [ ] **Step 3: Extract tagging helper and copy sheet identity**

Edit `backend/runner/plancheck/executor_combined.go`:

```go
func (e *CombinedExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	var allResults []*storepb.PlanCheckRunResult_Result

	for _, checkType := range target.Types {
		results, err := e.runCheck(ctx, target, checkType)
		if err != nil {
			// Add error result for this target/type, continue to next
			allResults = append(allResults, &storepb.PlanCheckRunResult_Result{
				Status:      storepb.Advice_ERROR,
				Target:      target.Target,
				Type:        checkType,
				SheetSha256: target.SheetSha256,
				Title:       "Check failed",
				Content:     err.Error(),
				Code:        common.Internal.Int32(),
			})
			continue
		}
		tagPlanCheckResults(results, target, checkType)
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func tagPlanCheckResults(results []*storepb.PlanCheckRunResult_Result, target *CheckTarget, checkType storepb.PlanCheckType) {
	for _, r := range results {
		r.Target = target.Target
		r.Type = checkType
		r.SheetSha256 = target.SheetSha256
	}
}
```

- [ ] **Step 4: Run the plancheck test**

Run:

```bash
go test -v -count=1 ./backend/runner/plancheck -run ^TestTagPlanCheckResultsCopiesTargetMetadata$
```

Expected: PASS.

- [ ] **Step 5: Commit target propagation**

Run:

```bash
gofmt -w backend/runner/plancheck/executor_combined.go backend/runner/plancheck/executor_combined_test.go
git add backend/runner/plancheck/executor_combined.go backend/runner/plancheck/executor_combined_test.go
git commit -m "fix(plancheck): tag results with sheet identity"
```

## Task 3: Match Approval Summaries By Sheet

**Files:**
- Modify: `backend/runner/approval/runner.go`
- Extend: `backend/runner/approval/runner_test.go`

- [ ] **Step 1: Write the failing summary-map test**

Append to `backend/runner/approval/runner_test.go`:

```go
func TestBuildStatementSummaryResultMapUsesSheetSHA256(t *testing.T) {
	results := []*storepb.PlanCheckRunResult_Result{
		{
			Target:      "instances/prod/databases/app",
			Type:        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
			SheetSha256: "sheet-a",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					AffectedRows: 10,
				},
			},
		},
		{
			Target:      "instances/prod/databases/app",
			Type:        storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
			SheetSha256: "sheet-b",
			Report: &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
				SqlSummaryReport: &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
					AffectedRows: 20,
				},
			},
		},
	}

	got := buildStatementSummaryResultMap(results)

	require.Equal(t, int64(10), got[statementSummaryKey{
		InstanceID:    "prod",
		DatabaseName:  "app",
		SheetSHA256:   "sheet-a",
	}].GetSqlSummaryReport().GetAffectedRows())
	require.Equal(t, int64(20), got[statementSummaryKey{
		InstanceID:    "prod",
		DatabaseName:  "app",
		SheetSHA256:   "sheet-b",
	}].GetSqlSummaryReport().GetAffectedRows())
}
```

- [ ] **Step 2: Run the failing summary-map test**

Run:

```bash
go test -v -count=1 ./backend/runner/approval -run ^TestBuildStatementSummaryResultMapUsesSheetSHA256$
```

Expected: FAIL with `undefined: buildStatementSummaryResultMap`.

- [ ] **Step 3: Add the sheet-aware summary map helper**

Edit `backend/runner/approval/runner.go` near `buildCELVariablesForDatabaseChange`:

```go
type statementSummaryKey struct {
	InstanceID    string
	DatabaseName  string
	SheetSHA256   string
}

func buildStatementSummaryResultMap(results []*storepb.PlanCheckRunResult_Result) map[statementSummaryKey]*storepb.PlanCheckRunResult_Result {
	m := map[statementSummaryKey]*storepb.PlanCheckRunResult_Result{}
	for _, result := range results {
		if result.Type != storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT {
			continue
		}
		instanceID, databaseName, err := common.GetInstanceDatabaseID(result.Target)
		if err != nil {
			continue
		}
		m[statementSummaryKey{
			InstanceID:    instanceID,
			DatabaseName:  databaseName,
			SheetSHA256:   result.SheetSha256,
		}] = result
	}
	return m
}
```

- [ ] **Step 4: Use the helper in database-change CEL generation**

Inside `buildCELVariablesForDatabaseChange`, replace the local `Key` type and `latestPlanCheckRun` construction with:

```go
	statementSummaryResults := map[statementSummaryKey]*storepb.PlanCheckRunResult_Result{}
	if planCheckRun != nil {
		statementSummaryResults = buildStatementSummaryResultMap(planCheckRun.Result.GetResults())
	}
```

Then replace the lookup before `report := result.GetSqlSummaryReport()` with:

```go
		result, ok := statementSummaryResults[statementSummaryKey{
			InstanceID:    target.database.InstanceID,
			DatabaseName:  target.database.DatabaseName,
			SheetSHA256:   target.sheetSha256,
		}]
```

- [ ] **Step 5: Run the summary-map test**

Run:

```bash
gofmt -w backend/runner/approval/runner.go backend/runner/approval/runner_test.go
go test -v -count=1 ./backend/runner/approval -run ^TestBuildStatementSummaryResultMapUsesSheetSHA256$
```

Expected: PASS.

- [ ] **Step 6: Commit sheet-aware approval lookup**

Run:

```bash
git add backend/runner/approval/runner.go backend/runner/approval/runner_test.go
git commit -m "fix(approval): match statement summaries by sheet"
```

## Task 4: Auto-Rerun Legacy Plan Checks During Approval

**Files:**
- Modify: `backend/runner/approval/runner.go`
- Extend: `backend/runner/approval/runner_test.go`
- Modify: `backend/api/v1/issue_hook.go`
- Modify: `backend/api/v1/issue_service.go`
- Modify: `backend/api/v1/plan_service.go`

- [ ] **Step 1: Write the failing legacy detection test**

Append to `backend/runner/approval/runner_test.go`:

```go
func TestHasLegacyStatementSummaryResultForSheetTarget(t *testing.T) {
	targets := []specTarget{
		{
			database: &store.DatabaseMessage{
				InstanceID:   "prod",
				DatabaseName: "app",
			},
			sheetSha256: "sheet-a",
		},
	}
	planCheckRun := &store.PlanCheckRunMessage{
		Status: store.PlanCheckRunStatusDone,
		Result: &storepb.PlanCheckRunResult{
			Results: []*storepb.PlanCheckRunResult_Result{
				{
					Target: "instances/prod/databases/app",
					Type:   storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				},
			},
		},
	}

	require.True(t, hasLegacyStatementSummaryResult(planCheckRun, targets))

	planCheckRun.Result.Results[0].SheetSha256 = "sheet-a"
	require.False(t, hasLegacyStatementSummaryResult(planCheckRun, targets))
}
```

Also add the missing store import at the top of `backend/runner/approval/runner_test.go`:

```go
	"github.com/bytebase/bytebase/backend/store"
```

- [ ] **Step 2: Run the failing legacy detection test**

Run:

```bash
go test -v -count=1 ./backend/runner/approval -run ^TestHasLegacyStatementSummaryResultForSheetTarget$
```

Expected: FAIL with `undefined: hasLegacyStatementSummaryResult`.

- [ ] **Step 3: Add legacy detection and rerun helpers**

Add helpers near the statement-summary map helper in `backend/runner/approval/runner.go`. The file already imports `backend/component/bus`, so the new helper can use `*bus.Bus` directly:

```go
func hasLegacyStatementSummaryResult(planCheckRun *store.PlanCheckRunMessage, targets []specTarget) bool {
	if planCheckRun == nil || planCheckRun.Status != store.PlanCheckRunStatusDone {
		return false
	}

	type databaseKey struct {
		InstanceID   string
		DatabaseName string
	}
	sheetBackedTargets := map[databaseKey]struct{}{}
	for _, target := range targets {
		if target.sheetSha256 == "" || target.database == nil {
			continue
		}
		sheetBackedTargets[databaseKey{
			InstanceID:   target.database.InstanceID,
			DatabaseName: target.database.DatabaseName,
		}] = struct{}{}
	}
	if len(sheetBackedTargets) == 0 {
		return false
	}

	for _, result := range planCheckRun.Result.GetResults() {
		if result.Type != storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT || result.SheetSha256 != "" {
			continue
		}
		instanceID, databaseName, err := common.GetInstanceDatabaseID(result.Target)
		if err != nil {
			continue
		}
		if _, ok := sheetBackedTargets[databaseKey{InstanceID: instanceID, DatabaseName: databaseName}]; ok {
			return true
		}
	}
	return false
}

func rerunPlanChecksForApproval(ctx context.Context, stores *store.Store, b *bus.Bus, plan *store.PlanMessage) error {
	if b == nil {
		return errors.New("approval plan-check rerun requires bus")
	}
	if err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: plan.ProjectID,
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{},
	}); err != nil {
		return errors.Wrap(err, "failed to create plan check run")
	}
	b.PlanCheckTickleChan <- 0
	return nil
}
```

- [ ] **Step 4: Pass bus through approval generation**

Change signatures in `backend/runner/approval/runner.go`:

```go
func FindAndApplyApprovalTemplate(ctx context.Context, stores *store.Store, b *bus.Bus, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService, issue *store.IssueMessage) error
func findApprovalTemplateForIssue(ctx context.Context, stores *store.Store, b *bus.Bus, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService, issue *store.IssueMessage, approvalSetting *storepb.WorkspaceApprovalSetting) error
func buildCELVariablesForIssue(ctx context.Context, stores *store.Store, b *bus.Bus, issue *store.IssueMessage) ([]map[string]any, bool, error)
func buildCELVariablesForDatabaseChange(ctx context.Context, stores *store.Store, b *bus.Bus, issue *store.IssueMessage) ([]map[string]any, bool, error)
```

Update calls inside the same file:

```go
	err = findApprovalTemplateForIssue(ctx, stores, b, webhookManager, licenseService, issue, approvalSetting)
	if err := findApprovalTemplateForIssue(ctx, r.store, r.bus, r.webhookManager, r.licenseService, issue, approvalSetting); err != nil {
	celVarsList, done, err := buildCELVariablesForIssue(ctx, stores, b, issue)
	return buildCELVariablesForDatabaseChange(ctx, stores, b, issue)
```

- [ ] **Step 5: Trigger rerun before building summary map**

In `buildCELVariablesForDatabaseChange`, unfold targets before constructing the statement-summary map. After `targets` is available and after the existing `RUNNING` check, add:

```go
	if hasLegacyStatementSummaryResult(planCheckRun, targets) {
		if err := rerunPlanChecksForApproval(ctx, stores, b, plan); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
```

Keep the existing `RUNNING` behavior:

```go
	if planCheckRun != nil && planCheckRun.Status == store.PlanCheckRunStatusRunning {
		return nil, false, nil
	}
```

- [ ] **Step 6: Update API call sites**

In `backend/api/v1/issue_hook.go`, change:

```go
if err := approval.FindAndApplyApprovalTemplate(ctx, stores, b, webhookManager, licenseService, issue); err != nil {
```

In `backend/api/v1/issue_service.go`, change:

```go
if err := approval.FindAndApplyApprovalTemplate(ctx, s.store, s.bus, s.webhookManager, s.licenseService, issue); err != nil {
```

In `backend/api/v1/plan_service.go`, change:

```go
if err := approval.FindAndApplyApprovalTemplate(ctx, s.store, s.bus, s.webhookManager, s.licenseService, updatedIssue); err != nil {
```

- [ ] **Step 7: Run approval tests**

Run:

```bash
gofmt -w backend/runner/approval/runner.go backend/runner/approval/runner_test.go backend/api/v1/issue_hook.go backend/api/v1/issue_service.go backend/api/v1/plan_service.go
go test -v -count=1 ./backend/runner/approval -run '^(TestBuildStatementSummaryResultMapUsesSheetSHA256|TestHasLegacyStatementSummaryResultForSheetTarget)$'
```

Expected: PASS.

- [ ] **Step 8: Commit legacy rerun behavior**

Run:

```bash
git add backend/runner/approval/runner.go backend/runner/approval/runner_test.go backend/api/v1/issue_hook.go backend/api/v1/issue_service.go backend/api/v1/plan_service.go
git commit -m "fix(approval): rerun legacy plan checks before evaluation"
```

## Task 5: Verify Integration Gates

**Files:**
- Verify only.

- [ ] **Step 1: Run focused package tests**

Run:

```bash
go test -v -count=1 ./backend/runner/plancheck ./backend/runner/approval
```

Expected: PASS.

- [ ] **Step 2: Run protobuf formatting and linting**

Run:

```bash
buf format -w proto
buf lint proto
```

Expected: both commands exit `0`.

- [ ] **Step 3: Run Go formatting**

Run:

```bash
gofmt -w backend/runner/plancheck/executor_combined.go backend/runner/plancheck/executor_combined_test.go backend/runner/approval/runner.go backend/runner/approval/runner_test.go backend/api/v1/issue_hook.go backend/api/v1/issue_service.go backend/api/v1/plan_service.go
```

Expected: command exits `0`.

- [ ] **Step 4: Run repository lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If the linter reports issues, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Expected after fixes: PASS.

- [ ] **Step 5: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: PASS and binary written to `./bytebase-build/bytebase`.

- [ ] **Step 6: Commit verification fixes**

Run:

```bash
git status --short
git add proto/store/store/plan_check_run.proto backend/generated-go/store/plan_check_run.pb.go backend/generated-go/store/plan_check_run_equal.pb.go backend/runner/plancheck/executor_combined.go backend/runner/plancheck/executor_combined_test.go backend/runner/approval/runner.go backend/runner/approval/runner_test.go backend/api/v1/issue_hook.go backend/api/v1/issue_service.go backend/api/v1/plan_service.go
git commit -m "chore: fix plan check sheet identity verification"
```

Expected: create this commit only if verification changed files. If `git status --short` is empty, do not create an empty commit.

## Task List

Task Index: T1: Emit plan spec update issue comments [M]

### T1: Emit plan spec update issue comments [M]

**Objective**: Restore issue activity entries for updated plan specs whose statement sheet changes while the plan is attached to an issue, preserving the existing `PlanSpecUpdate` payload and frontend diff path.

**Size**: M (2 files, moderate comparison and regression test logic)

**Files**:
- Modify: `backend/api/v1/plan_service.go`
- Create: `backend/tests/plan_update_test.go`

**Implementation**:
1. In `backend/api/v1/plan_service.go`, in `PlanService.UpdatePlan()` around the existing local variables near line 296, add a slice for pending `*store.IssueCommentMessage` values.
2. In the `specs` update branch around lines 321-335, after converting the request specs into the cloned plan config and after finding the linked issue, collect `PlanSpecUpdate` issue comment payloads when both old and new specs contain the same spec ID and the sheet SHA256 changes.
3. In `backend/api/v1/plan_service.go`, add small unexported helpers near the plan conversion/helper functions:
   - one helper to extract a sheet SHA256 from sheet-backed store plan spec configs (`ChangeDatabaseConfig` and `ExportDataConfig`);
   - one helper to compare old/new specs by ID and return `IssueCommentPayload_PlanSpecUpdate` comments using `common.FormatSpec(projectID, planUID, specID)`.
4. In `PlanService.UpdatePlan()` around line 366, after `s.store.UpdatePlan()` succeeds and before returning the converted plan, call `s.store.CreateIssueComments(ctx, user.Email, pendingComments...)` when comments were collected. Log insertion errors as non-fatal with `slog.Warn`.
5. In `backend/tests/plan_update_test.go`, add `TestUpdatePlanCreatesPlanSpecUpdateIssueComment`: start the existing test controller, create a SQLite instance/database, create two sheets, create a sheet-backed database change plan and issue, update the plan spec to point at the second sheet, list issue comments, and assert exactly one `PlanSpecUpdate` event contains the expected spec resource plus old and new sheet resource names.

**Boundaries**: Do not change proto files, generated files, frontend files, plan permissions, approval eligibility rules, rollout/task behavior, or issue comment storage schema.

**Dependencies**: None

**Expected Outcome**: Updating a sheet-backed plan spec attached to an issue creates a review-visible `PlanSpecUpdate` issue comment with old/new sheet references; existing plan update behavior remains unchanged.

**Validation**:
- `gofmt -w backend/api/v1/plan_service.go backend/tests/plan_update_test.go` - modified Go files are formatted.
- `golangci-lint run --allow-parallel-runners` - lint completes with no issues.
- `golangci-lint run --fix --allow-parallel-runners` - auto-fix completes without introducing required follow-up changes.
- `golangci-lint run --allow-parallel-runners` - repeated lint completes with no issues.
- `go test -v -count=1 github.com/bytebase/bytebase/backend/tests -run ^TestUpdatePlanCreatesPlanSpecUpdateIssueComment$` - focused regression test passes.
- `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go` - backend build succeeds.

## Out-of-Scope Tasks

- Tracking who last edited plan SQL for approval eligibility.
- Blocking self-approval based on plan SQL edits.
- Adding new issue comment event types or proto fields.
- Changing the frontend activity timeline or statement diff renderer.
- Recording plan spec changes for plans without linked issues.
- Reworking rollout, task, or plan check scheduling semantics.

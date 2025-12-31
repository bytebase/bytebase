# Event-Driven Approval Finding Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace polling-based approval finding with event-driven architecture for HA compatibility.

**Architecture:** Use buffered channel (`ApprovalCheckChan`) to signal approval finding needs. Primary trigger is plan check completion. Remove in-memory sync.Map and error persistence. Errors are logged but not saved to database.

**Tech Stack:** Go, Protobuf, Vue.js, TypeScript

---

## Task 1: Remove ApprovalFindingError from Proto

**Files:**
- Modify: `proto/store/approval.proto`

**Step 1: Update proto definition**

Edit `proto/store/approval.proto`, find the `IssuePayloadApproval` message and remove field 4:

```proto
message IssuePayloadApproval {
  ApprovalTemplate approval_template = 1;
  repeated Approver approvers = 2;
  // Whether the system has finished finding a matching approval template.
  // False means the backend is still searching for matching templates.
  bool approval_finding_done = 3;

  // Reserve field number to prevent reuse
  reserved 4;
  reserved "approval_finding_error";

  message Approver {
    // The new status.
    ApprovalStep_Status status = 1;
    // The principal.
    // Format: users/{email}
    string principal = 2;
  }
}
```

**Step 2: Regenerate proto files**

Run: `cd proto && buf generate`

Expected: Proto files regenerated in `backend/generated-go/store/`

**Step 3: Format proto file**

Run: `buf format -w proto`

Expected: Proto file formatted

**Step 4: Commit**

```bash
git add proto/store/approval.proto backend/generated-go/store/
git commit -m "refactor: remove ApprovalFindingError from proto

Remove approval_finding_error field from IssuePayloadApproval.
Errors are now transient (logged, not persisted).

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Add ApprovalCheckChan to State

**Files:**
- Modify: `backend/component/state/state.go`

**Step 1: Add channel field**

Edit `backend/component/state/state.go`, add new field to State struct:

```go
// State is the state for all in-memory states within the server.
type State struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan int64 // issue UID

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// RunningPlanChecks is the set of running plan checks.
	RunningPlanChecks sync.Map
	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan int64

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan int64
}
```

**Step 2: Initialize channel in New()**

Edit the `New()` function:

```go
func New() (*State, error) {
	return &State{
		ApprovalCheckChan:       make(chan int64, 1000),
		PlanCheckTickleChan:     make(chan int, 1000),
		TaskRunTickleChan:       make(chan int, 1000),
		RolloutCreationChan:     make(chan int64, 100),
		PlanCompletionCheckChan: make(chan int64, 1000),
	}, nil
}
```

**Step 3: Commit**

```bash
git add backend/component/state/state.go
git commit -m "feat: add ApprovalCheckChan to State

Add buffered channel for event-driven approval finding.
Buffered with 1000 capacity to handle bursts.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Convert Approval Runner to Event Listener

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Replace Run() method**

Find the `Run()` method in `backend/runner/approval/runner.go` and replace it:

```go
// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Debug("Approval runner started (event-driven)")

	for {
		select {
		case issueUID := <-r.stateCfg.ApprovalCheckChan:
			r.processIssue(ctx, issueUID)
		case <-ctx.Done():
			return
		}
	}
}
```

**Step 2: Add processIssue() method**

Add new method after `Run()`:

```go
func (r *Runner) processIssue(ctx context.Context, issueUID int64) {
	// Get fresh issue from database
	issue, err := r.store.GetIssue(ctx, &store.FindIssueMessage{UID: &issueUID})
	if err != nil {
		slog.Error("failed to get issue for approval check",
			slog.Int64("issue_uid", issueUID), log.BBError(err))
		return
	}
	if issue == nil {
		return // Issue deleted, nothing to do
	}

	approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
	if err != nil {
		slog.Error("failed to get workspace approval setting", log.BBError(err))
		return
	}

	// Find approval template - errors are logged, not persisted
	_, err = r.findApprovalTemplateForIssue(ctx, issue, approvalSetting)
	if err != nil {
		slog.Error("failed to find approval template",
			slog.Int64("issue_uid", issueUID),
			slog.String("issue_title", issue.Title),
			log.BBError(err))
		// Don't persist error - user can rerun plan check to retry
	}
}
```

**Step 3: Commit**

```bash
git add backend/runner/approval/runner.go
git commit -m "refactor: convert approval runner to event listener

Replace polling with event-driven architecture.
Process issues when signaled via ApprovalCheckChan.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Remove Error Persistence from Approval Finding

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Simplify error handling**

Find the error handling block in `findApprovalTemplateForIssue()` (around line 176):

```go
// BEFORE (delete this):
if err != nil {
	if updateErr := updateIssueApprovalPayload(ctx, r.store, issue, &storepb.IssuePayloadApproval{
		ApprovalFindingDone:  true,
		ApprovalFindingError: err.Error(),
	}, storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED); updateErr != nil {
		return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
	}
	return false, err
}

// AFTER (replace with this):
if err != nil {
	// Don't persist error - it will be logged by caller
	// User can rerun plan check to retry
	return false, err
}
```

**Step 2: Commit**

```bash
git add backend/runner/approval/runner.go
git commit -m "refactor: remove error persistence from approval finding

Errors are now transient - logged but not saved to database.
Users rerun plan check to retry.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Remove Deprecated Approval Runner Methods

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Delete runOnce() method**

Find and delete the entire `runOnce()` method (approximately lines 80-101):

```go
// DELETE this entire function:
func (r *Runner) runOnce(ctx context.Context) {
	approvalSetting, err := r.store.GetWorkspaceApprovalSetting(ctx)
	if err != nil {
		slog.Error("failed to get workspace approval setting", log.BBError(err))
		return
	}

	r.stateCfg.ApprovalFinding.Range(func(key, value any) bool {
		issue, ok := value.(*store.IssueMessage)
		if !ok {
			return true
		}
		done, err := r.findApprovalTemplateForIssue(ctx, issue, approvalSetting)
		if err != nil {
			slog.Error("failed to find approval template for issue", slog.Int("issue", issue.UID), log.BBError(err))
		}
		if err != nil || done {
			r.stateCfg.ApprovalFinding.Delete(key)
		}
		return true
	})
}
```

**Step 2: Delete retryFindApprovalTemplate() method**

Find and delete the entire `retryFindApprovalTemplate()` method (approximately lines 103-117):

```go
// DELETE this entire function:
func (r *Runner) retryFindApprovalTemplate(ctx context.Context) {
	issues, err := r.store.ListIssues(ctx, &store.FindIssueMessage{
		StatusList: []storepb.Issue_Status{storepb.Issue_OPEN},
	})
	if err != nil {
		err := errors.Wrap(err, "failed to list issues")
		slog.Error("failed to retry finding approval template", log.BBError(err))
	}
	for _, issue := range issues {
		payload := issue.Payload
		if payload.Approval == nil || !payload.Approval.ApprovalFindingDone {
			r.stateCfg.ApprovalFinding.Store(issue.UID, issue)
		}
	}
}
```

**Step 3: Commit**

```bash
git add backend/runner/approval/runner.go
git commit -m "refactor: remove deprecated approval runner methods

Remove runOnce() and retryFindApprovalTemplate().
No longer needed with event-driven architecture.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Trigger Approval Check on Plan Check Completion

**Files:**
- Modify: `backend/runner/plancheck/scheduler.go`

**Step 1: Add approval trigger to markPlanCheckRunDone()**

Find the `markPlanCheckRunDone()` method and update it:

```go
func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results: results,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusDone,
		result,
		planCheckRun.UID,
	); err != nil {
		slog.Error("failed to mark plan check run done", log.BBError(err))
		return
	}

	// Get issue for this plan
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planCheckRun.PlanUID})
	if err != nil {
		slog.Error("failed to get issue for approval check after plan check",
			slog.Int("plan_id", int(planCheckRun.PlanUID)),
			log.BBError(err))
		return
	}
	if issue != nil && issue.PlanUID != nil {
		// Trigger approval finding
		s.stateCfg.ApprovalCheckChan <- issue.UID
		// Trigger rollout creation (existing behavior)
		s.stateCfg.RolloutCreationChan <- planCheckRun.PlanUID
	}
}
```

**Step 2: Commit**

```bash
git add backend/runner/plancheck/scheduler.go
git commit -m "feat: trigger approval check on plan check completion

Signal ApprovalCheckChan when plan checks complete successfully.
Primary trigger for event-driven approval finding.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Add tryTriggerApprovalCheck Helper to IssueService

**Files:**
- Modify: `backend/api/v1/issue_service.go`

**Step 1: Add helper method**

Add this method to the IssueService (after the struct definition, before CreateIssue):

```go
// tryTriggerApprovalCheck checks if plan check is already done and triggers approval finding.
// Called after issue creation to handle the case where plan checks completed before issue was created.
func (s *IssueService) tryTriggerApprovalCheck(ctx context.Context, issue *store.IssueMessage) {
	if issue.PlanUID == nil {
		return
	}

	planCheckRun, err := s.store.GetPlanCheckRun(ctx, *issue.PlanUID)
	if err != nil {
		slog.Debug("failed to get plan check run for approval trigger check",
			slog.Int("plan_uid", int(*issue.PlanUID)), log.BBError(err))
		return
	}

	// If plan check is already DONE, trigger approval finding
	if planCheckRun != nil && planCheckRun.Status == store.PlanCheckRunStatusDone {
		s.stateCfg.ApprovalCheckChan <- issue.UID
	}
}
```

**Step 2: Commit**

```bash
git add backend/api/v1/issue_service.go
git commit -m "feat: add helper to trigger approval check on issue creation

Helper checks if plan check already completed and triggers approval.
Handles case where plan check finished before issue created.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Update Issue Creation to Use New Trigger

**Files:**
- Modify: `backend/api/v1/issue_service.go`

**Step 1: Update first CreateIssue call (DATABASE_CHANGE)**

Find the first issue creation block (around line 429) and replace:

```go
// OLD (delete):
s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

// NEW (replace with):
s.tryTriggerApprovalCheck(ctx, issue)
```

**Step 2: Update second CreateIssue call (GRANT_REQUEST)**

Find the second issue creation block (around line 509) and replace:

```go
// OLD (delete):
s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

// NEW (replace with):
s.tryTriggerApprovalCheck(ctx, issue)
```

**Step 3: Update third CreateIssue call (DATABASE_EXPORT)**

Find the third issue creation block (around line 578) and replace:

```go
// OLD (delete):
s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

// NEW (replace with):
s.tryTriggerApprovalCheck(ctx, issue)
```

**Step 4: Commit**

```bash
git add backend/api/v1/issue_service.go
git commit -m "refactor: use event trigger instead of sync.Map in issue creation

Replace sync.Map.Store() with tryTriggerApprovalCheck().
Triggers approval finding if plan check already done.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update Plan Service Approval Reset

**Files:**
- Modify: `backend/api/v1/plan_service.go`

**Step 1: Update approval reset after plan update**

Find the approval reset code (around line 318-325) and update:

```go
// Reset approval finding status
if _, err := s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
	PayloadUpsert: &storepb.Issue{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
		},
	},
}); err != nil {
	slog.Error("failed to reset approval finding status after plan update", log.BBError(err))
}
// Note: Don't trigger ApprovalCheckChan here - plan update creates new plan check run,
// which will trigger approval finding on completion
```

Remove the old line:

```go
// DELETE this:
s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
```

**Step 2: Commit**

```bash
git add backend/api/v1/plan_service.go
git commit -m "refactor: simplify approval reset on plan update

Remove sync.Map.Store() - plan update triggers new plan check,
which will trigger approval finding on completion.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Remove ApprovalFinding sync.Map from State

**Files:**
- Modify: `backend/component/state/state.go`

**Step 1: Remove field from State struct**

Edit `backend/component/state/state.go` and delete the ApprovalFinding field:

```go
// State is the state for all in-memory states within the server.
type State struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan int64 // issue UID

	// DELETE this line:
	// ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// ... rest of fields
}
```

**Step 2: Commit**

```bash
git add backend/component/state/state.go
git commit -m "refactor: remove ApprovalFinding sync.Map from State

No longer needed with event-driven architecture.
Replaced by ApprovalCheckChan.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Remove approval_status Update Path

**Files:**
- Modify: `backend/api/v1/issue_service.go`

**Step 1: Delete approval_status case**

Find and delete the entire `case "approval_status":` block in UpdateIssue() (around lines 1052-1069):

```go
// DELETE this entire case block:
case "approval_status":
	if req.Msg.Issue.ApprovalStatus != v1pb.Issue_CHECKING {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("can only set approval_status to CHECKING to trigger re-finding approval templates"))
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("issue payload approval is nil"))
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding is not done"))
	}

	if patch.PayloadUpsert == nil {
		patch.PayloadUpsert = &storepb.Issue{}
	}
	patch.PayloadUpsert.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone: false,
	}
```

**Step 2: Delete dead code checking approval_finding_done**

Find and delete the dead code (around lines 1143-1145):

```go
// DELETE these lines:
if updateMasks["approval_finding_done"] {
	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
}
```

**Step 3: Commit**

```bash
git add backend/api/v1/issue_service.go
git commit -m "refactor: remove approval_status update path

Delete manual retry mechanism - users now rerun plan check.
Also remove dead code checking approval_finding_done mask.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Remove ApprovalFindingError Checks from Backend

**Files:**
- Modify: `backend/api/v1/issue_service.go`
- Modify: `backend/api/v1/issue_service_converter.go`
- Modify: `backend/utils/utils.go`

**Step 1: Remove error checks from issue_service.go**

Find and delete the error checks in three methods (ApprovalApprove, RejectIssue, UpdateIssue):

In `ApprovalApprove()` (around line 621-622):
```go
// DELETE:
if payload.Approval.ApprovalFindingError != "" {
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding failed: %v", payload.Approval.ApprovalFindingError))
}
```

In `RejectIssue()` (around line 797-798):
```go
// DELETE:
if payload.Approval.ApprovalFindingError != "" {
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding failed: %v", payload.Approval.ApprovalFindingError))
}
```

In UpdateIssue()'s approvers update section (around line 903-904):
```go
// DELETE:
if payload.Approval.ApprovalFindingError != "" {
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding failed: %v", payload.Approval.ApprovalFindingError))
}
```

**Step 2: Remove error mapping from issue_service_converter.go**

Delete line 117:
```go
// DELETE:
issueV1.ApprovalStatusError = approval.GetApprovalFindingError()
```

Delete lines 130-132 in `computeApprovalStatus()`:
```go
// DELETE:
// If there's an error finding approval, status is error
if approval.GetApprovalFindingError() != "" {
	return v1pb.Issue_ERROR
}
```

**Step 3: Remove error check from utils.go**

In `CheckApprovalApproved()`, delete lines 63-65:
```go
// DELETE:
if approval.ApprovalFindingError != "" {
	return false, nil
}
```

**Step 4: Commit**

```bash
git add backend/api/v1/issue_service.go backend/api/v1/issue_service_converter.go backend/utils/utils.go
git commit -m "refactor: remove ApprovalFindingError references

Remove all checks and mappings for approval finding errors.
Errors are now transient (logged, not persisted).

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 13: Remove Retry Button from Frontend

**Files:**
- Modify: `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue`

**Step 1: Remove ERROR state UI**

Delete the ERROR state block (lines 18-30):

```vue
<!-- DELETE this block: -->
<div
  v-else-if="issue.approvalStatus === Issue_ApprovalStatus.ERROR"
  class="flex items-center gap-x-2"
>
  <span class="text-error text-sm">{{ issue.approvalStatusError }}</span>
  <NButton
    size="tiny"
    :loading="retrying"
    @click="retryFindingApprovalFlow"
  >
    {{ $t("common.retry") }}
  </NButton>
</div>
```

**Step 2: Remove retry function and ref**

Delete the retry-related code (lines 84-94):

```typescript
// DELETE:
const retrying = ref(false);

const retryFindingApprovalFlow = async () => {
  retrying.value = true;
  try {
    await useIssueV1Store().regenerateReviewV1(props.issue.name);
    emit("issue-updated");
  } finally {
    retrying.value = false;
  }
};
```

**Step 3: Commit**

```bash
git add frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue
git commit -m "refactor: remove approval retry button from UI

Remove ERROR state and retry button.
Users now rerun plan check to retry approval finding.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 14: Remove regenerateReviewV1 from Issue Store

**Files:**
- Modify: `frontend/src/store/modules/v1/issue.ts`

**Step 1: Delete regenerateReviewV1 function**

Delete the function (lines 80-89):

```typescript
// DELETE:
const regenerateReviewV1 = async (name: string) => {
  const request = create(UpdateIssueRequestSchema, {
    issue: create(IssueSchema, {
      name,
      approvalStatus: Issue_ApprovalStatus.CHECKING,
    }),
    updateMask: { paths: ["approval_status"] },
  });
  await issueServiceClientConnect.updateIssue(request);
};
```

**Step 2: Remove from return statement**

Delete from the return statement (line 139):

```typescript
return {
  listIssues,
  fetchIssueByName,
  // DELETE this line:
  // regenerateReviewV1,
};
```

**Step 3: Commit**

```bash
git add frontend/src/store/modules/v1/issue.ts
git commit -m "refactor: remove regenerateReviewV1 from issue store

Function no longer needed - users rerun plan check to retry.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 15: Run Backend Linting

**Files:**
- All modified Go files

**Step 1: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners`

Expected: No linting errors

**Step 2: Fix any linting issues**

If there are linting errors, fix them according to the output.

**Step 3: Run golangci-lint again**

Run: `golangci-lint run --allow-parallel-runners`

Expected: No linting errors (0 issues)

**Step 4: Commit if fixes were needed**

```bash
git add .
git commit -m "chore: fix linting issues

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 16: Run Frontend Linting and Formatting

**Files:**
- All modified frontend files

**Step 1: Format with Biome**

Run: `pnpm --dir frontend biome:format`

Expected: Files formatted

**Step 2: Lint with Biome**

Run: `pnpm --dir frontend biome:lint`

Expected: No linting errors

**Step 3: Lint with ESLint**

Run: `pnpm --dir frontend lint --fix`

Expected: No linting errors

**Step 4: Type check**

Run: `pnpm --dir frontend type-check`

Expected: No type errors

**Step 5: Commit if fixes were needed**

```bash
git add frontend/
git commit -m "chore: fix frontend linting and formatting

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 17: Manual Testing

**Test Scenario 1: Plan check triggers approval**

1. Start backend: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
2. Create a new DATABASE_CHANGE issue with SQL
3. Observe logs: Plan check completes → approval finding triggered
4. Check issue in UI: approval status should show appropriate template or "No approval needed"

Expected: Approval found immediately after plan check completes (no 1-second delay)

**Test Scenario 2: Issue creation with completed plan check**

1. Create plan check first (via API)
2. Wait for plan check to complete
3. Create issue for that plan
4. Observe logs: Approval finding triggered immediately

Expected: Approval found right after issue creation

**Test Scenario 3: Error handling**

1. Temporarily break approval rule CEL expression in workspace settings
2. Create issue and wait for plan check
3. Check logs: Error logged but not persisted
4. Fix CEL expression
5. Rerun plan check
6. Check issue: Approval found successfully

Expected: Error logged, not shown in UI. Retry works via plan check rerun.

**Test Scenario 4: Plan update**

1. Create issue with SQL
2. Wait for approval to be found
3. Update SQL in plan
4. Observe: Approval status resets to CHECKING
5. Wait for new plan check
6. Check issue: New approval found

Expected: Approval resets and re-finds after plan update

---

## Task 18: Update Documentation (Optional)

**Files:**
- Create: `docs/architecture/approval-finding.md` (optional)

**Step 1: Document architecture (optional)**

If desired, create architecture documentation explaining:
- Event-driven flow
- Trigger points
- Error handling philosophy
- HA compatibility

**Step 2: Commit documentation**

```bash
git add docs/architecture/approval-finding.md
git commit -m "docs: add approval finding architecture

Document event-driven approval finding architecture.

🤖 Generated with Claude Code

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Verification Checklist

- [ ] Proto regenerated without ApprovalFindingError
- [ ] ApprovalCheckChan added to State and initialized
- [ ] Approval runner converted to event listener
- [ ] Plan check completion triggers approval
- [ ] Issue creation triggers approval (if plan check done)
- [ ] Plan update resets approval status
- [ ] sync.Map removed from State
- [ ] approval_status update path removed
- [ ] Error references removed from backend
- [ ] Retry button removed from frontend
- [ ] regenerateReviewV1 removed from store
- [ ] Backend linting passes
- [ ] Frontend linting passes
- [ ] Manual testing scenarios pass
- [ ] No console errors in browser
- [ ] No HA-related issues (test with multiple instances if possible)

---

## Success Criteria

1. **HA Compatible**: Multiple instances can run without duplicate processing
2. **Immediate Processing**: Approval found right after plan check (no polling delay)
3. **Clean Error Handling**: Errors logged, not persisted; retry via plan check rerun
4. **Code Reduction**: ~230 lines removed (sync.Map, error persistence, dead code, retry UI)
5. **No Regressions**: All existing approval workflows still work

---

## Rollback Plan

If issues are discovered:

1. Revert all commits on this branch
2. Return to main branch
3. In-memory sync.Map approach still works (not HA-safe but functional)

Critical files to check for regressions:
- `backend/runner/approval/runner.go`
- `backend/api/v1/issue_service.go`
- `backend/runner/plancheck/scheduler.go`

# Event-Driven Approval Finding Design

**Date:** 2025-12-31
**Status:** Design
**Author:** Danny

## Overview

This design replaces the polling-based approval finding system with an event-driven architecture that is HA-compatible. Approval templates will be determined automatically and instantly when plan checks complete, eliminating the in-memory sync.Map and 1-second polling loop.

## Problem

The current approval finding system uses an in-memory `sync.Map` which breaks in High Availability (HA) deployments:

```go
// backend/component/state/state.go
type State struct {
    ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage
}

// backend/runner/approval/runner.go
func (r *Runner) runOnce(ctx context.Context) {
    r.stateCfg.ApprovalFinding.Range(func(key, value any) bool {
        // Process issues in the map
    })
}
```

**HA Incompatibilities:**

1. **Not shared** - Each replica has its own sync.Map
2. **Duplicate processing** - Multiple replicas might process the same issue simultaneously
3. **Lost work** - If a replica crashes, queued issues are lost from its map
4. **Inconsistent state** - Database has `ApprovalFindingDone` flag but no cross-instance coordination
5. **Polling overhead** - Runs every 1 second checking all issues in map

**Additional Problems:**

6. **Complex error handling** - `ApprovalFindingError` persisted in database requires manual retry UI
7. **Dead code** - "approval_status" update path has incorrect mask check, never triggers
8. **Dependency on plan check** - Approval finding waits for plan check summary report data

## Requirements

1. **HA-compatible** - Multiple replicas can run without conflicts
2. **Event-driven** - No polling, instant processing when plan checks complete
3. **Simple error handling** - Transient errors logged, not persisted; users rerun plan check to retry
4. **No startup recovery** - Avoid duplicate processing across replicas at startup
5. **Clean architecture** - Remove sync.Map, dead code, and error persistence

## Solution: Event-Driven Approval Finding

Use a channel-based architecture similar to the existing `RolloutCreationChan` pattern. Approval finding is triggered by events, primarily plan check completion.

### Key Insight

Approval finding depends on plan check summary report data, so the natural trigger point is plan check completion. If users need to trigger approval finding manually, they can click "Rerun plan checks" in the UI.

### Architecture

```
┌─────────────────────┐
│ Plan Check Complete │
│   (status = DONE)   │
└──────────┬──────────┘
           │
           ▼
    ┌──────────────────┐
    │ Signal channel:  │
    │ ApprovalCheckChan│
    └──────────┬───────┘
           │
           ▼
┌──────────────────────────┐
│ Approval Runner (event   │
│ listener) receives signal│
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│ Get fresh issue from DB  │
│ Find approval template   │
│ Update issue payload     │
└──────────────────────────┘
```

**Benefits:**
- ✅ HA-compatible: channel is per-instance but that's fine (any instance can process)
- ✅ No polling: only runs when triggered
- ✅ Immediate: processes as soon as plan check completes
- ✅ Simple: no sync.Map, no error persistence, just signal → process
- ✅ Efficient: CEL evaluation only runs once per event

**HA Consideration:**

The channel is in-memory per-instance, which is acceptable because:
- Plan check completion happens on one instance → signals its channel → processes
- If that instance crashes before processing, issue is left with `approval_finding_done=false`
- User can rerun plan check to trigger again (no automatic recovery to avoid duplicate processing)

## Component Changes

### 1. State Structure

**File:** `backend/component/state/state.go`

```go
type State struct {
    // ADD new channel:
    // ApprovalCheckChan signals when an issue needs approval template finding.
    // Triggered by plan check completion.
    ApprovalCheckChan chan int64 // issue UID

    // REMOVE old sync.Map:
    // ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

    TaskRunSchedulerInfo sync.Map
    RunningTaskRunsCancelFunc sync.Map
    RunningPlanChecks sync.Map
    RunningPlanCheckRunsCancelFunc sync.Map
    PlanCheckTickleChan chan int
    TaskRunTickleChan chan int
    RolloutCreationChan chan int64
    PlanCompletionCheckChan chan int64
}

func New() (*State, error) {
    return &State{
        ApprovalCheckChan:       make(chan int64, 1000), // buffered for bursts
        PlanCheckTickleChan:     make(chan int, 1000),
        TaskRunTickleChan:       make(chan int, 1000),
        RolloutCreationChan:     make(chan int64, 100),
        PlanCompletionCheckChan: make(chan int64, 1000),
    }, nil
}
```

### 2. Approval Runner

**File:** `backend/runner/approval/runner.go`

**New Run() method (event listener):**

```go
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

**Remove deprecated methods:**

- `runOnce()` - polling method no longer needed
- `retryFindApprovalTemplate()` - startup recovery removed for HA safety

**Update error handling in findApprovalTemplateForIssue():**

```go
// OLD: Persisted error to database
if err != nil {
    if updateErr := updateIssueApprovalPayload(ctx, r.store, issue, &storepb.IssuePayloadApproval{
        ApprovalFindingDone:  true,
        ApprovalFindingError: err.Error(),
    }, storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED); updateErr != nil {
        return false, multierr.Append(errors.Wrap(updateErr, "failed to update issue payload"), err)
    }
    return false, err
}

// NEW: Just return error (logged by caller)
if err != nil {
    // Don't persist error - it will be logged by caller
    // User can rerun plan check to retry
    return false, err
}
```

## Trigger Points

### Trigger 1: Plan Check Completion (Primary)

**Location:** `backend/runner/plancheck/scheduler.go`

**In markPlanCheckRunDone():**

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

    if issue != nil {
        // Trigger approval finding
        s.stateCfg.ApprovalCheckChan <- issue.UID

        // Also trigger rollout creation (existing behavior)
        s.stateCfg.RolloutCreationChan <- planCheckRun.PlanUID
    }
}
```

### Trigger 2: Plan Update

**Location:** `backend/api/v1/plan_service.go`

When user updates plan (e.g., changes SQL), reset approval status but don't trigger immediately. The plan update will trigger a new plan check run, which will trigger approval finding when it completes.

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

### Trigger 4: Manual Retry (Via Plan Check Rerun)

User clicks "Rerun plan check" button → plan check runs → triggers approval finding on completion.

**No code changes needed** - existing plan check rerun functionality handles this.

### No Startup Recovery

**Rationale:** Startup recovery is NOT HA-compatible:
- Multiple instances start simultaneously
- Each would try to process all pending issues
- No coordination mechanism → duplicate processing

**What happens to crashed work?**
- Issue stuck with `approval_finding_done=false`
- User notices approval stuck at CHECKING status
- User clicks "Rerun plan check"
- Plan check completes → triggers approval finding

Since crashes are rare and users have a clear fix (rerun plan check), we accept this trade-off for HA safety.

## Error Handling

### Transient Error Philosophy

Approval finding errors are treated as **transient** and not persisted to the database:

**Error Sources:**

1. **Database errors** - network issues, timeouts (transient)
2. **Missing data** - plan/instance/database not found (should not happen in normal flow)
3. **CEL compilation errors** - invalid approval rule expression (configuration error)
4. **CEL evaluation errors** - type mismatches, missing variables (configuration error)

**Error Handling:**

1. **Log error** - Include issue UID, title, and full error details
2. **Don't persist** - No `ApprovalFindingError` field in database
3. **User retry** - User clicks "Rerun plan check" to retry
4. **Issue visible** - Issue stays at `CHECKING` approval status

**User Experience:**

- Approval status shows "Checking..." (spinner in UI)
- If stuck, user checks server logs or clicks "Rerun plan check"
- Much simpler than error message + retry button

### Removed Error Persistence

**Delete from proto:** `proto/store/approval.proto`

```proto
message IssuePayloadApproval {
  ApprovalTemplate approval_template = 1;
  repeated Approver approvers = 2;
  bool approval_finding_done = 3;

  // REMOVE field 4:
  // string approval_finding_error = 4;

  // Reserve to prevent field number reuse:
  reserved 4;
  reserved "approval_finding_error";
}
```

**Regenerate:** `cd proto && buf generate`

## Cleanup: Remove Deprecated Code

### 1. Backend Cleanup

**File:** `backend/api/v1/issue_service.go`

**Remove "approval_status" update path:**

```go
// DELETE entire case block (lines 1052-1069):
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

**Remove dead code:**

```go
// DELETE lines 1143-1145:
if updateMasks["approval_finding_done"] {
    s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
}
```

**Remove error checks (3 places):**

In `ApprovalApprove()`, `RejectIssue()`, and `UpdateIssue()`:

```go
// DELETE:
if payload.Approval.ApprovalFindingError != "" {
    return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding failed: %v", payload.Approval.ApprovalFindingError))
}
```

**File:** `backend/api/v1/issue_service_converter.go`

```go
// DELETE line 117:
issueV1.ApprovalStatusError = approval.GetApprovalFindingError()

// DELETE lines 130-132 in computeApprovalStatus():
if approval.GetApprovalFindingError() != "" {
    return v1pb.Issue_ERROR
}
```

**File:** `backend/utils/utils.go`

In `CheckApprovalApproved()`:

```go
// DELETE:
if approval.ApprovalFindingError != "" {
    return false, nil
}
```

### 2. Frontend Cleanup

**File:** `frontend/src/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue`

**Remove ERROR state and retry button:**

```vue
<!-- DELETE lines 18-30: -->
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

**Remove retry function:**

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

**File:** `frontend/src/store/modules/v1/issue.ts`

**Remove regenerateReviewV1:**

```typescript
// DELETE function (lines 80-89):
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

// DELETE from return statement:
return {
  listIssues,
  fetchIssueByName,
  // regenerateReviewV1,  // DELETE this
};
```

## Migration Considerations

### Database Migration

**No database migration needed** because:
- `ApprovalFindingError` is stored in JSONB `issue.payload` column
- Removing the field from proto doesn't affect existing data
- Old data with `approval_finding_error` will be ignored (field not in proto)
- New data won't have the field

### Backward Compatibility

**Proto field reservation:**
```proto
reserved 4;
reserved "approval_finding_error";
```

This prevents:
- Future proto changes from reusing field number 4
- Confusion from old clients that might still reference the field

### Rollout Strategy

1. Deploy backend with event-driven approval finding
2. Deploy frontend with removed retry button
3. No downtime required
4. Old in-flight issues with `approval_finding_done=false` will be processed on next plan check rerun

## Testing Considerations

### Unit Tests

1. **Approval runner event processing**
   - Test `processIssue()` with valid issue
   - Test with missing issue (deleted)
   - Test with database errors
   - Test with CEL evaluation errors

2. **Trigger points**
   - Test plan check completion triggers channel
   - Test plan update resets approval status

### Integration Tests

1. **End-to-end flow**
   - Create issue → plan check runs → approval found
   - Update plan → new plan check → new approval

2. **Error scenarios**
   - Invalid CEL expression in approval rule
   - Missing plan check data
   - Database temporarily unavailable

### HA Testing

1. **Multiple replicas**
   - Two instances running
   - Plan check completes on instance A
   - Verify approval processed (by either instance)
   - Verify no duplicate processing

2. **Crash recovery**
   - Instance crashes after plan check, before approval finding
   - Verify issue stuck at CHECKING status
   - User reruns plan check
   - Verify approval found

## Success Metrics

1. **HA Compatibility**
   - No duplicate approval finding across replicas
   - No lost work from crashes (user can retry)

2. **Performance**
   - Approval finding happens immediately after plan check (no 1s delay)
   - No polling overhead

3. **Simplicity**
   - Removed sync.Map (~100 lines)
   - Removed error persistence (~50 lines)
   - Removed dead code (~30 lines)
   - Removed retry UI (~50 lines frontend)
   - Total: ~230 lines removed

4. **User Experience**
   - Immediate approval status (no waiting)
   - Simple retry (rerun plan check button already exists)
   - No confusing error messages

## Future Enhancements

If orphaned issues (stuck at CHECKING due to crashes) become a problem:

**Option: Add lightweight polling safety net**
- Keep event-driven as primary mechanism
- Add slow background poll (e.g., every 60 seconds)
- Use `FOR UPDATE SKIP LOCKED` for HA-safe claim
- Poll only processes issues older than 5 minutes
- Catches rare crash edge cases without overhead

**Implementation:**
```go
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
    defer wg.Done()
    slog.Debug("Approval runner started (event-driven)")

    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case issueUID := <-r.stateCfg.ApprovalCheckChan:
            r.processIssue(ctx, issueUID)
        case <-ticker.C:
            // Safety net: process old stuck issues
            r.processStuckIssues(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

This can be added later if needed, keeping the design simple initially.

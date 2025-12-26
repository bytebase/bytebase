# Event-Driven Rollout Creation Design

**Date:** 2025-12-25
**Status:** Approved

## Overview

This design replaces the polling-based auto-rollout scheduler with an event-driven architecture. Rollouts will be created automatically and instantly when approval and plan check conditions are met, eliminating both the 5-second polling delay and the need for users to manually click "Create Rollout".

## Motivation

**Current Problems:**
- 5-second polling delay before rollouts are created
- Users must manually click "Create Rollout" button even when conditions are met
- Periodic scheduler wastes resources checking issues that haven't changed

**Goals:**
- Instant rollout creation when conditions are met
- Better user experience (no manual button click needed)
- Cleaner event-driven architecture
- Lower resource usage

**Scope:**
- Backend-only changes
- Frontend continues to work as-is (button stays as fallback)

## Current State

**Auto-Rollout Scheduler:**
- Location: `backend/runner/auto_rollout/scheduler.go`
- Runs every 5 seconds
- Checks all open issues with plans for:
  - Approval status (if `project.RequireIssueApproval == true`)
  - Plan check status (if `project.RequirePlanCheckNoError == true`)
- Calls `CreateRollout()` when conditions are met

**Manual Button:**
- Location: `frontend/src/components/Plan/components/HeaderSection/Actions/registry/actions/rollout.ts`
- Shows "Create Rollout" button when `!plan.hasRollout` and preconditions met
- Calls `CreateRollout()` RPC

**CreateRollout():**
- Location: `backend/api/v1/rollout_service.go`
- Converts plan specs to tasks
- Sets `plan.Config.HasRollout = true` to prevent further spec modifications

## Design

### Architecture

**Event-Driven Flow:**

```
┌─────────────────┐         ┌──────────────────┐
│ ApproveIssue()  │────────▶│ TryCreateRollout │
│ (all approved)  │         │   (Issue ID)     │
└─────────────────┘         └──────────────────┘
                                      │
                                      ▼
┌─────────────────┐         ┌──────────────────┐
│ PlanCheckRunner │────────▶│  Check all       │
│ (status = DONE) │         │  conditions      │
└─────────────────┘         └──────────────────┘
                                      │
                                      ▼
                            ┌──────────────────┐
                            │  CreateRollout() │
                            │  (if ready)      │
                            └──────────────────┘
```

**Components to Remove:**
- `backend/runner/auto_rollout/scheduler.go` - Entire scheduler
- Auto-rollout scheduler registration in server startup

**Components to Add:**
- `TryCreateRollout(ctx, issueID)` - Centralized condition checking

**Components to Modify:**
- `CreateRollout()` - Add idempotency check
- `ApproveIssue()` - Call TryCreateRollout when fully approved
- Plan check executor - Call TryCreateRollout when status = DONE

### Core Function: TryCreateRollout()

**Location:** `backend/api/v1/rollout_service.go`

**Signature:**
```go
func (s *RolloutService) TryCreateRollout(ctx context.Context, issueID int) error
```

**Logic Flow:**
1. Fetch issue by ID (include plan, project)
2. Early return if issue is nil or plan is nil
3. **Idempotency check:** Early return if `plan.Config.HasRollout == true`
4. **Check approval condition:**
   - If `project.RequireIssueApproval == true`:
     - Verify `issue.approvalStatus == APPROVED`
   - Else: approval condition passes
5. **Check plan check condition:**
   - If `project.RequirePlanCheckNoError == true`:
     - Fetch all plan check runs for the plan
     - Verify no ERROR-level results exist
     - Verify no RUNNING plan checks exist
   - Else: plan check condition passes
   - **Edge case:** Treat "no plan checks" as passing (same as old scheduler)
6. If BOTH conditions pass:
   - Call `CreateRollout()` with plan
7. **Error handling:** Log errors but don't propagate (fire-and-forget)

**Characteristics:**
- Non-blocking (called in goroutine)
- Idempotent (safe to call multiple times)
- Best-effort (errors logged but not returned)

### CreateRollout() Idempotency

**Add at the beginning of CreateRollout():**

```go
// Idempotency check
if plan.Config.HasRollout {
    return nil, status.Errorf(codes.AlreadyExists, "rollout already exists for plan")
}
```

This ensures:
- Race conditions between events are handled safely
- Manual button click after auto-creation returns graceful error
- Database transaction provides additional safety

### Trigger Point 1: Approval

**Location:** `backend/api/v1/issue_service.go` in `ApproveIssue()`

**When to trigger:**
- After updating `issue.approvers` in database
- Only when `issue.approvalStatus` becomes `APPROVED` (all approvers stamped)
- Not after each individual approval, only when fully approved

**Implementation:**
```go
// After database update
if issue.approvalStatus == Issue_ApprovalStatus_APPROVED {
    go func() {
        if err := s.rolloutService.TryCreateRollout(ctx, issueID); err != nil {
            slog.Error("failed to auto-create rollout after approval",
                log.BBError(err),
                slog.Int("issue_id", issueID))
        }
    }()
}
```

### Trigger Point 2: Plan Check Completion

**Location:** `backend/runner/plancheck/executor.go` (or wherever plan_check_run status is updated to DONE)

**When to trigger:**
- After updating `plan_check_run.status = DONE` in database
- Only for status `DONE` (success)
- Not for `FAILED` or `CANCELED`

**Implementation:**
```go
// After updating plan check run to DONE
if planCheckRun.Status == PlanCheckRun_Status_DONE {
    // Get issue from plan
    issueID := getIssueIDFromPlan(plan)
    if issueID > 0 {
        go func() {
            if err := rolloutService.TryCreateRollout(ctx, issueID); err != nil {
                slog.Error("failed to auto-create rollout after plan check",
                    log.BBError(err),
                    slog.Int("issue_id", issueID))
            }
        }()
    }
}
```

### Edge Cases

1. **Race condition between events:**
   - Both approval and plan check complete simultaneously
   - Both call `TryCreateRollout()`
   - **Solution:** Idempotent check on `HasRollout` + database transaction in CreateRollout()

2. **No plan checks exist:**
   - Some plans may not have plan checks (freemium users, simple changes)
   - **Solution:** Treat "no plan checks" as passing (same behavior as old scheduler)

3. **Approval template discovery incomplete:**
   - Template discovery runs async (every 1 second)
   - If approval required but template not found, status stays PENDING_APPROVAL
   - **Solution:** No special handling - TryCreateRollout only triggers on APPROVED status

4. **Manual button after auto-creation:**
   - User clicks button after auto-creation happens
   - **Solution:** CreateRollout() returns AlreadyExists error, frontend handles gracefully

5. **Plan specs change after approval:**
   - If `plan.Config.HasRollout == true`, specs can't be modified (existing validation)
   - **Solution:** No special handling needed

## Testing Strategy

### Unit Tests

1. **TryCreateRollout() tests:**
   - Approval required + plan checks required → both must pass
   - Approval skipped + plan checks required → only plan checks matter
   - Approval required + no plan check enforcement → only approval matters
   - Idempotency: calling twice returns early on second call
   - Plan checks with ERROR results → rollout not created
   - Plan checks still RUNNING → rollout not created
   - No plan checks exist → treated as passing

2. **CreateRollout() idempotency:**
   - Calling CreateRollout on plan with HasRollout=true returns AlreadyExists error

### Integration Tests

1. **Approval flow:**
   - Create issue requiring 2 approvals + plan checks passing
   - Approve with first user → no rollout created
   - Approve with second user → rollout created automatically

2. **Plan check flow:**
   - Create issue with approval passed + plan checks running
   - Complete plan checks with DONE status → rollout created automatically

3. **Race condition:**
   - Approval and plan check complete simultaneously
   - Verify only one rollout created
   - Verify no errors in logs

## Rollout Plan

1. **Code changes:**
   - Add TryCreateRollout() function
   - Add idempotency check to CreateRollout()
   - Hook approval trigger in ApproveIssue()
   - Hook plan check trigger in plan check executor
   - Remove auto_rollout scheduler initialization

2. **Testing:**
   - Run unit tests
   - Run integration tests
   - Manual testing in development environment

3. **Deployment:**
   - Deploy to staging
   - Verify auto-creation works
   - Verify manual button still works as fallback
   - Deploy to production

4. **Monitoring:**
   - Log every TryCreateRollout call (issue ID, rollout created or not)
   - Log any errors in TryCreateRollout
   - Alert if CreateRollout returns AlreadyExists frequently (indicates race conditions)

## Success Metrics

- Rollouts created instantly when conditions met (vs 0-5 second delay)
- Zero manual "Create Rollout" button clicks needed in normal flow
- No errors or race conditions in production logs
- Manual button still works as fallback for edge cases

## Rollback Plan

If issues are discovered:
1. Re-enable auto_rollout_scheduler
2. Remove event triggers from ApproveIssue and plan check executor
3. Keep idempotency check in CreateRollout (harmless safety measure)

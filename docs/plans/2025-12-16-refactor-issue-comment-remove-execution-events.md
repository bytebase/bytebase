# Refactor IssueComment: Remove Execution Events from Issue Timeline

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove execution-related events (StageEnd, TaskUpdate for status changes, TaskPriorBackup) from the issue timeline, keeping only issue-level events. Replace TaskUpdate with PlanSpecUpdate to track plan spec modifications (sheet changes).

**Architecture:** Following GitHub/GitLab patterns, issue timeline should only contain issue-level events. Execution details are already tracked in `task_run` and `task_run_log` tables. This refactor removes redundant comment types and renames TaskUpdate to PlanSpecUpdate for clarity.

**Tech Stack:** Go backend, Protocol Buffers, Vue.js frontend with TypeScript

---

## Summary of Changes

### Events to Keep
| Event | Purpose |
|-------|---------|
| `comment` | User comments |
| `approval` | Approval status changes |
| `issue_update` | Title/description/status/labels changes |
| `plan_spec_update` | **NEW** - Plan spec modifications (replaces TaskUpdate for sheet changes) |

### Events to Remove
| Event | Why Remove | Already Tracked In |
|-------|------------|-------------------|
| `stage_end` | Execution event | Derivable from task statuses |
| `task_update` (status part) | Execution event | `task_run` table |
| `task_prior_backup` | Execution event | `task_run_log` with `PRIOR_BACKUP_START/END` |

### Key Insight
The current `TaskUpdate` has two purposes:
1. **Status updates** (PENDING, RUNNING, DONE, etc.) - Remove this
2. **Sheet changes** (from_sheet, to_sheet) - Keep this, rename to `PlanSpecUpdate`

---

## Task 1: Update Store Proto Definition

**Files:**
- Modify: `proto/store/store/issue_comment.proto`

Replace the entire content of the `IssueCommentPayload` message:

```protobuf
message IssueCommentPayload {
  string comment = 1;

  oneof event {
    Approval approval = 2;
    IssueUpdate issue_update = 3;
    // REMOVED: StageEnd stage_end = 4;
    // REMOVED: TaskUpdate task_update = 5;
    // REMOVED: TaskPriorBackup task_prior_backup = 6;
    PlanSpecUpdate plan_spec_update = 7;  // NEW: replaces sheet change part of TaskUpdate
  }

  message Approval {
    IssuePayloadApproval.Approver.Status status = 1;
  }

  message IssueUpdate {
    optional string from_title = 1;
    optional string to_title = 2;
    optional string from_description = 3;
    optional string to_description = 4;
    optional Issue.Status from_status = 5;
    optional Issue.Status to_status = 6;
    repeated string from_labels = 7;
    repeated string to_labels = 8;
  }

  // REMOVED: message StageEnd
  // REMOVED: message TaskUpdate
  // REMOVED: message TaskPriorBackup

  // NEW: Plan spec update event (tracks sheet changes to plan specs)
  message PlanSpecUpdate {
    // The spec that was updated
    // Format: projects/{project}/plans/{plan}/specs/{spec}
    string spec = 1;
    // Format: projects/{project}/sheets/{sheet}
    optional string from_sheet = 2;
    // Format: projects/{project}/sheets/{sheet}
    optional string to_sheet = 3;
  }
}
```

---

## Task 2: Update V1 Proto Definition

**Files:**
- Modify: `proto/v1/v1/issue_service.proto`

Locate the `IssueComment` message (around line 575) and update it:

```protobuf
message IssueComment {
  // ... existing fields (name, comment, payload, create_time, update_time, creator) ...

  // The event associated with this comment.
  oneof event {
    // Approval event.
    Approval approval = 7;
    // Issue update event.
    IssueUpdate issue_update = 8;
    // REMOVED: StageEnd stage_end = 9;
    // REMOVED: TaskUpdate task_update = 10;
    // REMOVED: TaskPriorBackup task_prior_backup = 11;
    // Plan spec update event.
    PlanSpecUpdate plan_spec_update = 12;  // NEW
  }

  // ... keep existing Approval message ...
  // ... keep existing IssueUpdate message ...

  // REMOVED: message StageEnd
  // REMOVED: message TaskUpdate
  // REMOVED: message TaskPriorBackup

  // NEW: Plan spec update event (tracks sheet changes to plan specs)
  message PlanSpecUpdate {
    // The spec that was updated.
    // Format: projects/{project}/plans/{plan}/specs/{spec}
    string spec = 1;
    // The previous sheet.
    // Format: projects/{project}/sheets/{sheet}
    optional string from_sheet = 2;
    // The new sheet.
    // Format: projects/{project}/sheets/{sheet}
    optional string to_sheet = 3;
  }
}
```

**After Tasks 1-2, run proto generation:**

```bash
cd proto && buf generate
buf lint proto
```

---

## Task 3: Remove StageEnd Comment Creation

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go`

The StageEnd comment is created around lines 854-872 in the `ListenTaskSkippedOrDone` function.

Remove this entire block:

```go
// REMOVE THIS BLOCK:
if err := func() error {
    p := &storepb.IssueCommentPayload{
        Event: &storepb.IssueCommentPayload_StageEnd_{
            StageEnd: &storepb.IssueCommentPayload_StageEnd{
                Stage: common.FormatStage(project.ResourceID, task.PipelineID, common.FormatStageID(task.Environment)),
            },
        },
    }
    if issueN != nil {
        _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
            IssueUID: issueN.UID,
            Payload:  p,
        }, common.SystemBotEmail)
        return err
    }
    return nil
}(); err != nil {
    slog.Warn("failed to create issue comment", log.BBError(err))
}
```

---

## Task 4: Remove TaskUpdate Status Comment Creation

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go`
- Modify: `backend/store/issue_comment.go`
- Modify: `backend/api/v1/rollout_service.go`

**4.1: Remove CreateIssueCommentTaskUpdateStatus function**

In `backend/store/issue_comment.go`, remove the entire `CreateIssueCommentTaskUpdateStatus` function (lines 121-137).

**4.2: Remove TaskUpdate status calls from scheduler.go**

Remove these calls:

1. Line ~250 in `scheduleAutoRolloutTask`:
```go
// REMOVE:
if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.TaskRun_PENDING, common.SystemBotEmail, ""); err != nil {
    slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
}
```

2. Line ~662 in `runTaskRunOnce` for FAILED status:
```go
// REMOVE:
return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.TaskRun_FAILED, common.SystemBotEmail, "")
```

3. Line ~706 in `runTaskRunOnce` for DONE status:
```go
// REMOVE:
return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.TaskRun_DONE, common.SystemBotEmail, "")
```

**4.3: Remove TaskUpdate status call from rollout_service.go**

In `backend/api/v1/rollout_service.go`, remove the call in `BatchSkipTasks` (around line 739):

```go
// REMOVE:
if issueN != nil {
    if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issueN.UID, request.Tasks, storepb.TaskRun_SKIPPED, user.Email, request.Reason); err != nil {
        slog.Warn("failed to create issue comment", "issueUID", issueN.UID, log.BBError(err))
    }
}
```

---

## Task 5: Rename TaskUpdate to PlanSpecUpdate for Sheet Changes

**Files:**
- Modify: `backend/api/v1/plan_service.go`
- Modify: `backend/common/resource_name.go`

**5.1: Add FormatSpec helper function**

In `backend/common/resource_name.go`, add after the `FormatPlan` function:

```go
func FormatSpec(projectID string, planUID int64, specID string) string {
	return fmt.Sprintf("%s/%s%s", FormatPlan(projectID, planUID), "specs/", specID)
}
```

**5.2: Update sheet change comment creation**

In `backend/api/v1/plan_service.go`, around lines 626-640, replace:

```go
issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
    IssueUID: issue.UID,
    Payload: &storepb.IssueCommentPayload{
        Event: &storepb.IssueCommentPayload_TaskUpdate_{
            TaskUpdate: &storepb.IssueCommentPayload_TaskUpdate{
                Tasks:     []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.Environment, task.ID)},
                FromSheet: &oldSheet,
                ToSheet:   &newSheet,
            },
        },
    },
})
```

With:

```go
specName := common.FormatSpec(issue.Project.ResourceID, oldPlan.UID, specID)
issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
    IssueUID: issue.UID,
    Payload: &storepb.IssueCommentPayload{
        Event: &storepb.IssueCommentPayload_PlanSpecUpdate_{
            PlanSpecUpdate: &storepb.IssueCommentPayload_PlanSpecUpdate{
                Spec:      specName,  // Format: projects/{project}/plans/{plan}/specs/{spec_id}
                FromSheet: &oldSheet,
                ToSheet:   &newSheet,
            },
        },
    },
})
```

---

## Task 6: Update Issue Service Converter

**Files:**
- Modify: `backend/api/v1/issue_service_converter.go`

**6.1: Remove converter functions for deleted events**

Remove these functions entirely:
- `convertToIssueCommentEventStageEnd`
- `convertToIssueCommentEventTaskUpdate`
- `convertToIssueCommentEventTaskUpdateStatus`
- `convertToIssueCommentEventTaskPriorBackup`
- `convertToIssueCommentEventTaskPriorBackupTables`

**6.2: Add converter for PlanSpecUpdate**

```go
func convertToIssueCommentEventPlanSpecUpdate(u *storepb.IssueCommentPayload_PlanSpecUpdate_) *v1pb.IssueComment_PlanSpecUpdate_ {
    return &v1pb.IssueComment_PlanSpecUpdate_{
        PlanSpecUpdate: &v1pb.IssueComment_PlanSpecUpdate{
            Spec:      u.PlanSpecUpdate.Spec,
            FromSheet: u.PlanSpecUpdate.FromSheet,
            ToSheet:   u.PlanSpecUpdate.ToSheet,
        },
    }
}
```

**6.3: Update convertToIssueComment switch statement**

Replace:
```go
switch e := ic.Payload.Event.(type) {
case *storepb.IssueCommentPayload_Approval_:
    r.Event = convertToIssueCommentEventApproval(e)
case *storepb.IssueCommentPayload_IssueUpdate_:
    r.Event = convertToIssueCommentEventIssueUpdate(e)
case *storepb.IssueCommentPayload_StageEnd_:
    r.Event = convertToIssueCommentEventStageEnd(e)
case *storepb.IssueCommentPayload_TaskUpdate_:
    r.Event = convertToIssueCommentEventTaskUpdate(e)
case *storepb.IssueCommentPayload_TaskPriorBackup_:
    r.Event = convertToIssueCommentEventTaskPriorBackup(e)
}
```

With:
```go
switch e := ic.Payload.Event.(type) {
case *storepb.IssueCommentPayload_Approval_:
    r.Event = convertToIssueCommentEventApproval(e)
case *storepb.IssueCommentPayload_IssueUpdate_:
    r.Event = convertToIssueCommentEventIssueUpdate(e)
case *storepb.IssueCommentPayload_PlanSpecUpdate_:
    r.Event = convertToIssueCommentEventPlanSpecUpdate(e)
}
```

---

## Task 7: Update Frontend Store

**Files:**
- Modify: `frontend/src/store/modules/v1/issueComment.ts`

**7.1: Update IssueCommentType enum**

Replace:
```typescript
export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  STAGE_END = "STAGE_END",
  TASK_UPDATE = "TASK_UPDATE",
  TASK_PRIOR_BACKUP = "TASK_PRIOR_BACKUP",
}
```

With:
```typescript
export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_SPEC_UPDATE = "PLAN_SPEC_UPDATE",
}
```

**7.2: Update getIssueCommentType function**

Replace:
```typescript
export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "stageEnd") {
    return IssueCommentType.STAGE_END;
  } else if (issueComment.event?.case === "taskUpdate") {
    return IssueCommentType.TASK_UPDATE;
  } else if (issueComment.event?.case === "taskPriorBackup") {
    return IssueCommentType.TASK_PRIOR_BACKUP;
  }
  return IssueCommentType.USER_COMMENT;
};
```

With:
```typescript
export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "planSpecUpdate") {
    return IssueCommentType.PLAN_SPEC_UPDATE;
  }
  return IssueCommentType.USER_COMMENT;
};
```

---

## Task 8: Update Frontend IssueCommentView Components

**Files:**
- Modify: `frontend/src/components/IssueV1/components/IssueCommentSection/IssueCommentView/ActionSentence.vue`
- Modify: `frontend/src/components/IssueV1/components/IssueCommentSection/IssueCommentView/ActionIcon.vue`
- Modify: `frontend/src/components/IssueV1/components/IssueCommentSection/IssueCommentView/common.ts`

**8.1: Update ActionSentence.vue**

Remove rendering logic for `STAGE_END`, `TASK_UPDATE` (status), `TASK_PRIOR_BACKUP`.

Add rendering for `PLAN_SPEC_UPDATE`:
```typescript
} else if (
  commentType === IssueCommentType.PLAN_SPEC_UPDATE &&
  issueComment.event?.case === "planSpecUpdate"
) {
  const { fromSheet, toSheet } = issueComment.event.value;
  if (fromSheet !== undefined && toSheet !== undefined) {
    return (
      <Translation keypath="dynamic.activity.sentence.changed-x-link">
        {{
          name: () => "SQL",
          link: () => (
            <StatementUpdate oldSheet={fromSheet} newSheet={toSheet} />
          ),
        }}
      </Translation>
    );
  }
}
```

**8.2: Update ActionIcon.vue**

Remove icon cases for removed event types, add:
```typescript
case IssueCommentType.PLAN_SPEC_UPDATE:
  return heroicons.DocumentTextIcon;
```

**8.3: Update common.ts**

Update `isSimilarIssueComment` - remove `TASK_UPDATE` handling, add `PLAN_SPEC_UPDATE`:
```typescript
if (aType === IssueCommentType.PLAN_SPEC_UPDATE) {
  const fromPlanSpecUpdate =
    a.event?.case === "planSpecUpdate" ? a.event.value : null;
  const toPlanSpecUpdate =
    b.event?.case === "planSpecUpdate" ? b.event.value : null;
  if (!fromPlanSpecUpdate || !toPlanSpecUpdate) {
    return false;
  }
  if (
    fromPlanSpecUpdate.toSheet &&
    fromPlanSpecUpdate.toSheet === toPlanSpecUpdate.toSheet
  ) {
    return true;
  }
}
```

---

## Task 9: Update Plan IssueReviewView Components

**Files:**
- Modify: `frontend/src/components/Plan/components/IssueReviewView/ActivitySection/IssueCommentView/ActionSentence.vue`
- Modify: `frontend/src/components/Plan/components/IssueReviewView/ActivitySection/IssueCommentView/ActionIcon.vue`
- Modify: `frontend/src/components/Plan/components/IssueReviewView/ActivitySection/IssueCommentView/common.ts`

Apply the same changes as Task 8 to these files.

---

## Task 10: Build and Test

```bash
# Backend linting
golangci-lint run --allow-parallel-runners

# Backend build
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

# Frontend linting and type check
pnpm --dir frontend biome:check
pnpm --dir frontend type-check
```

---

## Task 11: Database Migration (Last)

**Files:**
- Create: `backend/migrator/migration/3.13/0030##remove_execution_events_from_issue_comment.sql`

```sql
-- Migration: Remove execution events from issue_comment
--
-- This migration:
-- 1. Deletes stageEnd comments (derivable from task statuses)
-- 2. Deletes taskPriorBackup comments (tracked in task_run_log)
-- 3. Deletes taskUpdate comments with toStatus (execution status tracked in task_run)
-- 4. Migrates taskUpdate comments with sheet changes to planSpecUpdate

-- Step 1: Delete stageEnd comments
DELETE FROM issue_comment WHERE payload ? 'stageEnd';

-- Step 2: Delete taskPriorBackup comments
DELETE FROM issue_comment WHERE payload ? 'taskPriorBackup';

-- Step 3: Delete taskUpdate comments that only have status changes (no sheet changes)
DELETE FROM issue_comment
WHERE payload ? 'taskUpdate'
  AND (payload->'taskUpdate') ? 'toStatus'
  AND NOT ((payload->'taskUpdate') ? 'fromSheet' OR (payload->'taskUpdate') ? 'toSheet');

-- Step 4: Migrate taskUpdate comments with sheet changes to planSpecUpdate
-- The task name format is: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task_id}
-- The spec format is: projects/{project}/plans/{plan_id}/specs/{spec_id}
UPDATE issue_comment ic
SET payload = jsonb_build_object(
    'comment', COALESCE(ic.payload->>'comment', ''),
    'planSpecUpdate', jsonb_build_object(
        'spec', 'projects/' || i.project || '/plans/' || i.plan_id || '/specs/' || COALESCE(t.payload->>'specId', ''),
        'fromSheet', ic.payload->'taskUpdate'->>'fromSheet',
        'toSheet', ic.payload->'taskUpdate'->>'toSheet'
    )
)
FROM issue i, task t
WHERE ic.payload ? 'taskUpdate'
  AND ((ic.payload->'taskUpdate') ? 'fromSheet' OR (ic.payload->'taskUpdate') ? 'toSheet')
  AND ic.issue_id = i.id
  AND i.plan_id IS NOT NULL
  AND t.id = (
      SELECT CAST(
          split_part(
              ic.payload->'taskUpdate'->'tasks'->>0,
              '/tasks/',
              2
          ) AS INTEGER
      )
  );

-- Step 5: Delete any remaining taskUpdate comments that couldn't be migrated
DELETE FROM issue_comment
WHERE payload ? 'taskUpdate';
```

---

## Task 12: Final Commit

```bash
git add -A
git commit -m "$(cat <<'EOF'
refactor(api): remove execution events from issue timeline

Remove execution-related events from IssueCommentPayload:
- StageEnd (derivable from task statuses)
- TaskUpdate status changes (tracked in task_run table)
- TaskPriorBackup (tracked in task_run_log)

Replace TaskUpdate with PlanSpecUpdate for sheet changes.
Sheet changes are plan-level events, not task-level execution events.

This follows GitHub/GitLab patterns where issue timeline only contains
issue-level events, not execution details.

Migration included to:
- Delete old execution event comments
- Migrate sheet change comments to new PlanSpecUpdate format

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Files Changed Summary

| File | Change Type |
|------|-------------|
| `proto/store/store/issue_comment.proto` | Modify - Remove events, add PlanSpecUpdate |
| `proto/v1/v1/issue_service.proto` | Modify - Remove events, add PlanSpecUpdate |
| `backend/runner/taskrun/scheduler.go` | Modify - Remove StageEnd and TaskUpdate creation |
| `backend/store/issue_comment.go` | Modify - Remove CreateIssueCommentTaskUpdateStatus |
| `backend/api/v1/rollout_service.go` | Modify - Remove TaskUpdate call |
| `backend/api/v1/plan_service.go` | Modify - Use PlanSpecUpdate |
| `backend/common/resource_name.go` | Modify - Add FormatSpec helper |
| `backend/api/v1/issue_service_converter.go` | Modify - Update converters |
| `frontend/src/store/modules/v1/issueComment.ts` | Modify - Update enum and type function |
| `frontend/src/components/IssueV1/...` | Modify - Update UI components |
| `frontend/src/components/Plan/...` | Modify - Update UI components |
| `backend/migrator/migration/3.13/0030##...sql` | Create - Database migration |

---

## Backward Compatibility Notes

### Data Migration Strategy
The migration (Task 11) handles existing data:
- **Delete**: `stageEnd`, `taskPriorBackup`, and `taskUpdate` status-only comments
- **Migrate**: `taskUpdate` with sheet changes → `planSpecUpdate`

### Why Delete Instead of Keep?
1. **Execution events are redundant** - Already tracked in `task_run` and `task_run_log`
2. **Clean timeline** - Issue timeline should only show issue-level events
3. **No data loss** - Execution history remains in `task_run` table

### API Breaking Change
- The v1 API removes event types from IssueComment
- Clients using `stage_end`, `task_update`, `task_prior_backup` events will need to adapt
- **Add `breaking` label to PR**

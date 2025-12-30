# Webhook Events Redesign

**Date:** 2025-12-30
**Status:** Design
**Author:** Danny

## Overview

This document outlines a redesign of the webhook notification system to reduce noise, improve signal quality, and provide better targeting for notifications. The current system sends too many notifications for granular events (field updates, task status changes) while sometimes missing important outcomes (pipeline failures, completions).

## Goals

1. **Reduce noise** - From 9 event types to 5 focused events
2. **Better signal** - Notify on outcomes (failed/completed) not intermediate steps
3. **Smart routing** - Direct messages for action-required events, channels for awareness
4. **Failure aggregation** - 5-minute window prevents notification spam from cascading failures

## Event Model

### New Event Types (5 total)

**Approval Events (Direct Message capable):**
- `ISSUE_APPROVAL_REQUESTED` - Replaces `ISSUE_APPROVAL_NOTIFY`
  - Triggered: When issue approval flow starts OR user re-requests approval
  - Audience: Users with required approval role (from IAM policies)

- `ISSUE_SENT_BACK` - New event type
  - Triggered: When approver rejects/sends back the issue
  - Audience: Issue creator

**Issue Lifecycle Events (Channel only):**
- `ISSUE_CREATED` - Replaces `ISSUE_CREATE`
  - Triggered: New issue created via API
  - Audience: All subscribed webhooks

**Pipeline Events (Channel only):**
- `PIPELINE_FAILED` - New event type with aggregation
  - Triggered: 5 minutes after first task failure (debounced aggregation)
  - Audience: All subscribed webhooks
  - Note: Non-HA implementation initially (in-memory state)

- `PIPELINE_COMPLETED` - New event type
  - Triggered: All tasks in plan reach terminal state (DONE/SKIPPED) with no failures
  - Audience: All subscribed webhooks
  - Note: Non-HA implementation initially

### Removed Events

These existing events will be deprecated:

- ~~`ISSUE_COMMENT_CREATE`~~ - Too noisy
- ~~`ISSUE_FIELD_UPDATE`~~ - Too noisy
- ~~`ISSUE_STATUS_UPDATE`~~ - Redundant with lifecycle events
- ~~`NOTIFY_ISSUE_APPROVED`~~ - Implicit in issue completion
- ~~`NOTIFY_PIPELINE_ROLLOUT`~~ - Redundant with pipeline events
- ~~`ISSUE_PIPELINE_STAGE_STATUS_UPDATE`~~ - Too granular, replaced by `PIPELINE_COMPLETED`
- ~~`ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE`~~ - Too granular, replaced by `PIPELINE_FAILED`

### Event Type Mapping

| Old Event Type | New Event Type | Notes |
|----------------|----------------|-------|
| `ISSUE_CREATE` (1) | `ISSUE_CREATED` (10) | Renamed for consistency |
| `ISSUE_APPROVAL_NOTIFY` (6) | `ISSUE_APPROVAL_REQUESTED` (11) | Covers initial + re-request |
| `ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE` (7) | `PIPELINE_FAILED` (13) | Aggregated failures |
| `ISSUE_PIPELINE_STAGE_STATUS_UPDATE` (5) | `PIPELINE_COMPLETED` (14) | Success only |

## Data Architecture

### Component Layer (backend/component/webhook)

The component layer provides raw event data using proto-generated enums:

```go
type WebhookContext struct {
    EventType storepb.Activity_Type  // Proto enum: ISSUE_CREATED, etc.

    // Direct messaging configuration
    DirectMessage bool

    // Event-specific data (only one populated per event)
    IssueCreated         *IssueCreatedEvent
    ApprovalRequested    *IssueApprovalRequestedEvent
    SentBack             *IssueSentBackEvent
    PipelineFailed       *PipelineFailedEvent
    PipelineCompleted    *PipelineCompletedEvent
}

type IssueCreatedEvent struct {
    IssueID      int64
    IssueTitle   string
    IssueType    string  // "DATABASE_CHANGE", "GRANT_REQUEST", etc.
    ProjectID    string
    CreatorName  string
    CreatorEmail string
}

type IssueApprovalRequestedEvent struct {
    IssueID       int64
    IssueTitle    string
    ProjectID     string
    ApprovalRole  string     // "Project Owner"
    RequiredCount int        // How many approvals needed
    Approvers     []User     // List of people to notify
}

type IssueSentBackEvent struct {
    IssueID        int64
    IssueTitle     string
    ProjectID      string
    ApproverName   string
    ApproverEmail  string
    Reason         string    // Rejection comment
    CreatorName    string
    CreatorEmail   string    // Person to notify
}

type PipelineFailedEvent struct {
    IssueID       int64
    IssueTitle    string
    PlanID        int64
    FailedTasks   []FailedTask
}

type FailedTask struct {
    TaskID       int64
    TaskName     string
    DatabaseName string
    InstanceName string
    ErrorMessage string
    FailedAt     time.Time
}

type PipelineCompletedEvent struct {
    IssueID       int64
    IssueTitle    string
    PlanID        int64
    TotalTasks    int
    StartedAt     time.Time
    CompletedAt   time.Time
}

type User struct {
    Name  string
    Email string
}
```

### Plugin Layer (backend/plugin/webhook)

Each plugin (Slack, Discord, Teams, Feishu, etc.) receives the `WebhookContext` and handles:
- Title/description rendering (in their preferred language)
- URL construction from IssueID to Bytebase UI
- Message formatting according to platform
- Direct message delivery (if DirectMessage = true)

Plugins decide all presentation details. The component layer only provides raw data.

## Configuration & Storage

### Database Schema

Uses existing `project_webhook` table - **no schema changes needed**:

```sql
CREATE TABLE project_webhook (
    id serial PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    -- Stored as ProjectWebhook proto
    payload jsonb NOT NULL DEFAULT '{}'
);
```

### Proto Changes

Update `proto/store/store/project_webhook.proto` - modify Activity.Type enum:

**Add new activity types:**
```proto
enum Activity {
  enum Type {
    // ... existing types ...

    // New focused event types
    ISSUE_CREATED = 10;
    ISSUE_APPROVAL_REQUESTED = 11;
    ISSUE_SENT_BACK = 12;
    PIPELINE_FAILED = 13;
    PIPELINE_COMPLETED = 14;
  }
}
```

**Existing ProjectWebhook proto (no changes):**
```proto
message ProjectWebhook {
  WebhookType type = 1;                // SLACK, DISCORD, TEAMS, etc.
  string title = 2;
  string url = 3;
  repeated Activity.Type activities = 4;  // Event subscriptions
  bool direct_message = 5;                // Enable IM direct messaging
}
```

### Direct Message Behavior

The `direct_message` field in ProjectWebhook controls delivery:

**When `direct_message = true`:**
- `ISSUE_APPROVAL_REQUESTED` → sends individual DMs to each user in `Approvers[]`
- `ISSUE_SENT_BACK` → sends DM to `CreatorEmail`
- Other events (ISSUE_CREATED, PIPELINE_*) → POST to webhook `url` (channel events ignore DM flag)

**When `direct_message = false`:**
- All subscribed events POST to webhook `url`

**IM Integration:**
Direct messaging requires IM integration settings in the webhook payload JSONB:
```json
{
  "im_type": "feishu",
  "app_id": "cli_xxx",
  "app_secret": "xxx"
}
```

## Pipeline Failure Aggregation

### Problem

Multiple tasks failing in quick succession creates notification spam. For example, a migration failing across 10 databases could trigger 10 separate notifications.

### Solution

5-minute debounced aggregation window:

1. **First task fails** → Start 5-minute timer, collect failed task info
2. **More tasks fail within 5 minutes** → Add to collection, don't notify yet
3. **After 5 minutes** → Send single `PIPELINE_FAILED` event with all failed tasks
4. **Subsequent failures after notification** → Start new 5-minute window

### Implementation

**Approach:** In-memory state tracking (non-HA)

Pipeline events will use in-memory aggregation similar to the existing `RunningTaskRuns` pattern:

```go
type PipelineFailureWindow struct {
    FirstFailureTime time.Time
    FailedTasks      []*FailedTaskInfo
    NotificationSent bool
    Timer            *time.Timer
}
```

**Location:** Extend existing `ListenTaskSkippedOrDone` in `backend/runner/taskrun/scheduler.go`

**Future Migration:**
- Can migrate to DB-based `webhook_pending_event` table for HA when needed
- Would use `FOR UPDATE SKIP LOCKED` pattern for distributed coordination
- Won't require payload changes, only implementation changes

## Event Triggering & Delivery

### Trigger Points

| Event | Trigger Location | Condition |
|-------|-----------------|-----------|
| `ISSUE_CREATED` | `backend/api/v1/issue_service.go` - `CreateIssue()` | After issue persisted to DB |
| `ISSUE_APPROVAL_REQUESTED` | Approval runner or issue update handler | Approval flow starts or re-requested |
| `ISSUE_SENT_BACK` | Approval update handler | Approver rejects/sends back |
| `PIPELINE_FAILED` | `backend/runner/taskrun/scheduler.go` | 5-min after first failure |
| `PIPELINE_COMPLETED` | Extend `ListenTaskSkippedOrDone` in scheduler.go | All tasks terminal, no failures |

### Delivery Flow

1. Event occurs → Component layer creates `WebhookContext` with appropriate event data
2. Query `project_webhook` table for webhooks subscribed to this event type
3. For each matching webhook:
   - If `direct_message = true` AND event is approval-related → send individual DMs
   - Otherwise → POST to webhook URL
4. Delivery happens asynchronously (background goroutine to avoid blocking)
5. 3-second timeout per webhook, with retry on transient errors

### Audience Resolution

**ISSUE_APPROVAL_REQUESTED:**
- Query IAM policies (project + workspace level)
- Find users with the required approval role
- Filter: `END_USER` principal type only (no service accounts)
- Exclude deleted members

**ISSUE_SENT_BACK:**
- Issue creator (stored in issue record)

**Other events:**
- No user-specific targeting, all go to configured webhook URLs

## Implementation Checklist

### Proto Changes
- [ ] Update `proto/store/store/project_webhook.proto` - add new Activity.Type values (10-14)
- [ ] Run `cd proto && buf format -w` and `buf generate`

### Component Layer
- [ ] Create new event structs in `backend/component/webhook/`
- [ ] Update `WebhookContext` struct to use proto enums
- [ ] Implement audience resolution for approval events

### Event Triggers
- [ ] Add `ISSUE_CREATED` trigger in `backend/api/v1/issue_service.go`
- [ ] Add `ISSUE_APPROVAL_REQUESTED` trigger in approval handlers
- [ ] Add `ISSUE_SENT_BACK` trigger in approval update handler
- [ ] Implement pipeline failure aggregation in `backend/runner/taskrun/scheduler.go`
- [ ] Implement pipeline completion detection in scheduler

### Plugin Layer
- [ ] Update plugins to handle new event types:
  - `backend/plugin/webhook/slack/`
  - `backend/plugin/webhook/discord.go`
  - `backend/plugin/webhook/teams.go`
  - `backend/plugin/webhook/feishu.go`
  - Other webhook plugins

### Testing
- [ ] Unit tests for event creation
- [ ] Unit tests for failure aggregation logic
- [ ] Integration tests for direct messaging
- [ ] Integration tests for channel webhooks

## Migration Strategy

### Backward Compatibility

Old Activity.Type values (1-9) will remain in the proto but marked as deprecated. Existing webhooks subscribed to old events will continue working until users update their subscriptions.

**Migration path:**
1. Deploy new code with both old and new event types supported
2. UI can show "deprecated" tags on old event types
3. Provide migration tool or guide for users to update webhook subscriptions
4. Eventually remove old event handling code in future major version

### Rollout Plan

1. **Phase 1:** Deploy proto changes and new event structs (no functional change yet)
2. **Phase 2:** Deploy new event triggers alongside old ones (both systems running)
3. **Phase 3:** Update plugins to handle new event types
4. **Phase 4:** UI updates to show new event types in webhook configuration
5. **Phase 5:** Deprecation notices for old event types
6. **Future:** Remove old event handling code

## Future Enhancements

### HA Support for Pipeline Events

Current implementation uses in-memory state for failure aggregation. For multi-replica HA deployments:

**Proposed approach:**
- Add `webhook_pending_event` table
- Use `FOR UPDATE SKIP LOCKED` for distributed coordination
- Background job aggregates and sends notifications
- Details TBD when HA requirement becomes active

### Additional Event Types

Potential future additions based on user feedback:
- `ISSUE_CANCELED` - When issue is explicitly canceled
- `ISSUE_COMPLETED` - When issue reaches done state
- `PIPELINE_CANCELED` - When pipeline is stopped by user

These are intentionally excluded from initial design to keep it focused.

## Summary

This redesign reduces webhook noise while improving signal quality:

| Metric | Before | After |
|--------|--------|-------|
| Event types | 9 | 5 |
| Noisy events | Yes (comments, field updates) | No |
| Pipeline spam | Yes (per-task notifications) | No (5-min aggregation) |
| Action targeting | Partial (DM for approvals) | Better (DM for all approval events) |

**Key benefits:**
- Teams receive fewer, more meaningful notifications
- Approval events properly target specific users via DM
- Pipeline failures aggregate to prevent notification storms
- Clean data architecture with proto enums throughout

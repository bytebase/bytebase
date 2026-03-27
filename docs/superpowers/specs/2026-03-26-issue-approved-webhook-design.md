# Issue Approved Webhook Event

**Linear Issue:** BYT-9100 — Support Issue Approved event for webhook
**Date:** 2026-03-26

## Problem

Customers rely on webhook notifications to know when their submitted issues (change data requests, JIT access) have been approved, so they can act on the approval (e.g., go run their query). This was supported in release/3.12.2 via `NOTIFY_ISSUE_APPROVED` but was removed in the webhook redesign (commit `a5de11313b`) that consolidated 9 event types down to 5.

Currently, when all approval steps complete in `ApproveIssue()`, `NotifyApprovalRequested()` is called but returns early because `FindNextPendingRole()` returns `""` — no webhook fires.

## Solution

Add `ISSUE_APPROVED` as the 6th event type in the current webhook framework. DM the issue creator when all approval steps complete.

## Changes

### 1. Proto: `proto/store/store/project_webhook.proto`

Add `ISSUE_APPROVED = 15` to the `Activity.Type` enum:

```protobuf
enum Type {
    TYPE_UNSPECIFIED = 0;
    ISSUE_CREATED = 10;
    ISSUE_APPROVAL_REQUESTED = 11;
    ISSUE_SENT_BACK = 12;
    PIPELINE_FAILED = 13;
    PIPELINE_COMPLETED = 14;
    // ISSUE_APPROVED represents an issue being fully approved.
    ISSUE_APPROVED = 15;
}
```

Run `cd proto && buf generate` to regenerate Go and TypeScript types.

### 2. Event struct: `backend/component/webhook/event.go`

Add `EventIssueApproved` struct and field on `Event`:

```go
type EventIssueApproved struct {
    Approver *User
    Creator  *User
    Issue    *Issue
}
```

Add `IssueApproved *EventIssueApproved` to the `Event` struct between `ApprovalRequested` and `SentBack`.

### 3. Manager: `backend/component/webhook/manager.go`

Add switch case in `getWebhookContextFromEvent` between `ISSUE_APPROVAL_REQUESTED` and `ISSUE_SENT_BACK`:

```go
case storepb.Activity_ISSUE_APPROVED:
    level = webhook.WebhookSuccess
    title = "Issue approved"
    titleZh = "工单审批通过"
    if e.IssueApproved != nil {
        actor = e.IssueApproved.Approver
        issue = e.IssueApproved.Issue
        link = fmt.Sprintf("%s/projects/%s/issues/%d", externalURL, e.Project.ResourceID, issue.UID)
        webhookCtx.Description = fmt.Sprintf("%s approved the issue", e.IssueApproved.Approver.Name)
        mentionUsers = []*store.UserMessage{{
            Name:  e.IssueApproved.Creator.Name,
            Email: e.IssueApproved.Creator.Email,
            Type:  storepb.PrincipalType_END_USER,
        }}
    }
```

Key choices:
- **Level:** `WebhookSuccess` (positive event, same as `PIPELINE_COMPLETED`)
- **Actor:** The approver who gave the final approval (not SystemBot like old system)
- **DM target:** Issue creator via `mentionUsers`
- **Link:** Points to the issue page (same pattern as other issue events)

### 4. Helper function: `backend/runner/approval/runner.go`

Add `NotifyIssueApproved` alongside existing `NotifyApprovalRequested`:

```go
func NotifyIssueApproved(ctx context.Context, stores *store.Store, webhookManager *webhook.Manager, issue *store.IssueMessage, project *store.ProjectMessage, approver *store.UserMessage) {
    creatorAccount, err := stores.GetAccountByEmail(ctx, issue.CreatorEmail)
    if err != nil {
        slog.Warn("failed to get issue creator", log.BBError(err))
        return
    }
    if creatorAccount == nil {
        slog.Warn("issue creator account not found", slog.String("email", issue.CreatorEmail))
        return
    }

    webhookManager.CreateEvent(ctx, &webhook.Event{
        Type:    storepb.Activity_ISSUE_APPROVED,
        Project: webhook.NewProject(project),
        IssueApproved: &webhook.EventIssueApproved{
            Approver: &webhook.User{Name: approver.Name, Email: approver.Email},
            Creator:  &webhook.User{Name: creatorAccount.Name, Email: creatorAccount.Email},
            Issue:    webhook.NewIssue(issue),
        },
    })
}
```

Same error-handling style as `NotifyApprovalRequested` — warn and return on failure, don't block the approval flow. Includes nil check for `creatorAccount` since `GetAccountByEmail` can return `(nil, nil)`.

### 5. Trigger: `backend/api/v1/issue_service.go`

In `ApproveIssue()`, call `NotifyIssueApproved` inside the existing `if approved` block (line 635) and update the comment:

```go
approval.NotifyApprovalRequested(ctx, s.store, s.webhookManager, issue, project)

// If the issue is approved, notify the creator and complete access request if applicable.
if approved {
    approval.NotifyIssueApproved(ctx, s.store, s.webhookManager, issue, project, user)
    issue, err = completeAccessRequestIssue(ctx, s.store, user.Email, issue)
    if err != nil {
        slog.Debug("failed to complete role grant issue", log.BBError(err))
    }
}
```

### 6. Frontend: `frontend/src/types/v1/projectWebhook.ts`

Add new activity item after `issue-sent-back`:

```typescript
{
    title: t("project.webhook.activity-item.issue-approved.title"),
    label: t("project.webhook.activity-item.issue-approved.label"),
    activity: Activity_Type.ISSUE_APPROVED,
    supportDirectMessage: true,
},
```

### 7. i18n: locale files under `frontend/src/locales/`

Add `issue-approved` keys in all 5 locale files (`en-US`, `zh-CN`, `ja-JP`, `es-ES`, `vi-VN`) between `issue-sent-back` and `pipeline-failed`:

**en-US:**
```json
"issue-approved": {
    "title": "Issue approved",
    "label": "Notify when all approval steps are completed"
}
```

**zh-CN:**
```json
"issue-approved": {
    "title": "工单审批通过",
    "label": "当所有审批步骤完成时通知"
}
```

Other locales: use English as placeholder (existing pattern for non-translated strings).

## What's NOT needed

- **No database migration.** The `project_webhook` table stores activities as a JSONB proto payload. Existing webhooks won't have `ISSUE_APPROVED` selected until users configure it.
- **No `ROLLOUT_READY` event.** The old system had `NOTIFY_PIPELINE_ROLLOUT` alongside `NOTIFY_ISSUE_APPROVED`, but `PIPELINE_COMPLETED` already covers the rollout-done case. Adding `ROLLOUT_READY` would re-introduce the noise the redesign intentionally removed.
- **No webhook plugin changes.** All platforms (Slack, Discord, Teams, DingTalk, Feishu, WeCom, Lark) use the same `webhook.Context` struct. The new event works automatically across all platforms.

## Bug fixes in `manager.go` (pre-existing, fixed as part of this change)

### Fix 1: `Issue.Creator` incorrectly uses `actor.Email` instead of `issue.CreatorEmail`

**Bug:** In `getWebhookContextFromEvent`, the shared post-switch code (line 185) populates `webhookCtx.Issue.Creator` using `actor.Email`. For events where the actor is not the issue creator (`ISSUE_SENT_BACK`, and our new `ISSUE_APPROVED`), the "Issue Creator" field in webhook messages (rendered by `GetMetaList()` in all plugins — Slack, Discord, Teams, DingTalk, Feishu, WeCom, Lark) shows the approver instead of the actual issue creator.

**Fix:** Use `issue.CreatorEmail` instead of `actor.Email` in the shared post-switch code:

```go
// Before (buggy):
creatorAccount, err := m.store.GetAccountByEmail(ctx, actor.Email)
...
    Creator: webhook.Creator{
        Name:  creatorName,
        Email: actor.Email,
    },

// After (fixed):
creatorAccount, err := m.store.GetAccountByEmail(ctx, issue.CreatorEmail)
...
    Creator: webhook.Creator{
        Name:  creatorName,
        Email: issue.CreatorEmail,
    },
```

This fixes `ISSUE_SENT_BACK` (existing) and `ISSUE_APPROVED` (new) at once. For `ISSUE_CREATED` and `ISSUE_APPROVAL_REQUESTED`, actor is already the creator, so behavior is unchanged.

### Fix 2: `Description` and `Environment` silently wiped by struct reassignment

**Bug:** In `getWebhookContextFromEvent`, switch cases set `webhookCtx.Description` and `webhookCtx.Environment` on the initial `webhookCtx` variable (line 64). But line 162 fully reassigns `webhookCtx = webhook.Context{...}` without including `Description` or `Environment`, silently discarding them. This affects all events that set descriptions (`ISSUE_CREATED`, `ISSUE_SENT_BACK`, `PIPELINE_FAILED`, `PIPELINE_COMPLETED`) and environment (`PIPELINE_FAILED`, `PIPELINE_COMPLETED`).

`Description` is rendered by all webhook plugins (e.g., Slack renders it as a code block at slack.go line 98). `Environment` is rendered in the metadata list for rollout events (webhook.go line 146-150).

**Fix:** Capture `description` and `environment` as local variables (like `title`, `link`, etc.) and include them in the struct literal:

```go
// Add to local variable declarations (after line 76):
description := ""
environment := ""

// In switch cases, use local vars instead of webhookCtx fields:
description = fmt.Sprintf("%s approved the issue", e.IssueApproved.Approver.Name)
environment = e.RolloutFailed.Environment

// Include in struct literal (line 162):
webhookCtx = webhook.Context{
    Level:           level,
    EventType:       string(eventType),
    Title:           title,
    TitleZh:         titleZh,
    Description:     description,
    Link:            link,
    Environment:     environment,
    MentionEndUsers: mentionEndUsers,
    Project: &webhook.Project{
        Name:  common.FormatProject(e.Project.ResourceID),
        Title: e.Project.Title,
    },
}
```

All existing switch cases that set `webhookCtx.Description` or `webhookCtx.Environment` must be updated to use the local variables instead.

## Differences from old implementation

| Aspect | Old (`NOTIFY_ISSUE_APPROVED = 23`) | New (`ISSUE_APPROVED = 15`) |
|---|---|---|
| Actor | SystemBot | Actual approver |
| Event data | Flat on `Event.Issue` | Dedicated `EventIssueApproved` struct |
| DM target | `e.Issue.Creator` (`*store.UserMessage`) | `e.IssueApproved.Creator` (`*User`) → `*store.UserMessage` |
| Description | none | `"<name> approved the issue"` |
| Level | default (Info) | `WebhookSuccess` |
| Trigger location | Closure in `ApproveIssue` | Helper `NotifyIssueApproved` called from `if approved` block |

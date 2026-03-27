# Issue Approved Webhook Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ISSUE_APPROVED` webhook event so users get notified (via DM or channel) when all approval steps complete on their issue.

**Architecture:** New event type `ISSUE_APPROVED = 15` in the existing webhook framework. Triggered in `ApproveIssue()` when the final approval is given. Also fixes two pre-existing bugs in the webhook manager: wrong `Issue.Creator` email and silently wiped `Description`/`Environment` fields.

**Tech Stack:** Go (backend), Protocol Buffers (proto), TypeScript/Vue (frontend), i18n (locales)

**Spec:** `docs/superpowers/specs/2026-03-26-issue-approved-webhook-design.md`

---

## Chunk 1: Proto + Codegen

### Task 1: Add `ISSUE_APPROVED` to proto definitions

Both the store proto and v1 proto have independent `Activity.Type` enums that must stay in sync.

**Files:**
- Modify: `proto/store/store/project_webhook.proto:24` (add enum value before closing brace)
- Modify: `proto/v1/v1/project_service.proto:534-540,558` (add enum value + update comment)

- [ ] **Step 1: Add `ISSUE_APPROVED = 15` to store proto**

In `proto/store/store/project_webhook.proto`, add after `PIPELINE_COMPLETED = 14;`:

```protobuf
    // ISSUE_APPROVED represents an issue being fully approved.
    ISSUE_APPROVED = 15;
```

- [ ] **Step 2: Add `ISSUE_APPROVED = 15` to v1 proto**

In `proto/v1/v1/project_service.proto`, add after `PIPELINE_COMPLETED = 14;` (line 558):

```protobuf
    // ISSUE_APPROVED represents an issue being fully approved.
    ISSUE_APPROVED = 15;
```

Also update the `notification_types` comment (line 534-539) to include `ISSUE_APPROVED`:

```protobuf
  // It should not be empty, and should be a subset of the following:
  // - ISSUE_CREATED
  // - ISSUE_APPROVAL_REQUESTED
  // - ISSUE_SENT_BACK
  // - ISSUE_APPROVED
  // - PIPELINE_FAILED
  // - PIPELINE_COMPLETED
```

- [ ] **Step 3: Format protos**

Run: `buf format -w proto`

- [ ] **Step 4: Lint protos**

Run: `buf lint proto`
Expected: No errors.

- [ ] **Step 5: Generate code from protos**

Run: `cd proto && buf generate`
Expected: Regenerates Go types in `backend/generated-go/` and TypeScript types in `frontend/src/types/proto-es/`.

- [ ] **Step 6: Commit**

```bash
git add proto/ backend/generated-go/ frontend/src/types/proto-es/
git commit -m "feat: add ISSUE_APPROVED enum to proto definitions"
```

---

## Chunk 2: Bug fixes in `manager.go`

Fix two pre-existing bugs before adding the new event type. This keeps the bug fixes in their own commit for clean `git blame`.

### Task 2: Fix `Issue.Creator` using wrong email

**Files:**
- Modify: `backend/component/webhook/manager.go:185,201`

The shared post-switch code uses `actor.Email` to populate `Issue.Creator`. For events where actor != creator (e.g., `ISSUE_SENT_BACK`), this shows the approver as the issue creator in webhook messages. Fix by using `issue.CreatorEmail` instead.

- [ ] **Step 1: Fix `GetAccountByEmail` call**

In `backend/component/webhook/manager.go`, change line 185 from:

```go
		creatorAccount, err := m.store.GetAccountByEmail(ctx, actor.Email)
```

to:

```go
		creatorAccount, err := m.store.GetAccountByEmail(ctx, issue.CreatorEmail)
```

- [ ] **Step 2: Fix `Creator.Email` assignment**

In the same file, change line 201 from:

```go
				Email: actor.Email,
```

to:

```go
				Email: issue.CreatorEmail,
```

### Task 3: Fix `Description` and `Environment` silently wiped

**Files:**
- Modify: `backend/component/webhook/manager.go:76,89,118,135-136,146-147,162`

Switch cases set `webhookCtx.Description` and `webhookCtx.Environment`, but line 162 reassigns the entire `webhookCtx` struct without including them. Fix by using local variables.

- [ ] **Step 1: Add local variable declarations**

In `backend/component/webhook/manager.go`, after line 76 (`link := ""`), add:

```go
	description := ""
	environment := ""
```

- [ ] **Step 2: Update `ISSUE_CREATED` case**

Change line 89 from:

```go
			webhookCtx.Description = fmt.Sprintf("%s created issue %s", actor.Name, issue.Title)
```

to:

```go
			description = fmt.Sprintf("%s created issue %s", actor.Name, issue.Title)
```

- [ ] **Step 3: Update `ISSUE_SENT_BACK` case**

Change line 118 from:

```go
			webhookCtx.Description = fmt.Sprintf("%s sent back the issue: %s", e.SentBack.Approver.Name, e.SentBack.Reason)
```

to:

```go
			description = fmt.Sprintf("%s sent back the issue: %s", e.SentBack.Approver.Name, e.SentBack.Reason)
```

- [ ] **Step 4: Update `PIPELINE_FAILED` case**

Change lines 135-136 from:

```go
			webhookCtx.Description = "Rollout failed"
			webhookCtx.Environment = e.RolloutFailed.Environment
```

to:

```go
			description = "Rollout failed"
			environment = e.RolloutFailed.Environment
```

- [ ] **Step 5: Update `PIPELINE_COMPLETED` case**

Change lines 146-147 from:

```go
			webhookCtx.Description = "Rollout completed successfully"
			webhookCtx.Environment = e.RolloutCompleted.Environment
```

to:

```go
			description = "Rollout completed successfully"
			environment = e.RolloutCompleted.Environment
```

- [ ] **Step 6: Add `Description` and `Environment` to struct literal**

In the struct literal at line 162, add `Description` and `Environment` fields:

```go
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

- [ ] **Step 7: Format Go code**

Run: `gofmt -w backend/component/webhook/manager.go`

- [ ] **Step 8: Run Go lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No new issues.

- [ ] **Step 9: Commit**

```bash
git add backend/component/webhook/manager.go
git commit -m "fix: correct Issue.Creator email and preserve Description/Environment in webhook context"
```

---

## Chunk 3: Backend — new event type

### Task 4: Add `EventIssueApproved` struct

**Files:**
- Modify: `backend/component/webhook/event.go:14-15,87`

- [ ] **Step 1: Add field to `Event` struct**

In `backend/component/webhook/event.go`, add `IssueApproved` between `ApprovalRequested` (line 14) and `SentBack` (line 15):

```go
	// Focused event types (only one is set)
	IssueCreated      *EventIssueCreated
	ApprovalRequested *EventIssueApprovalRequested
	IssueApproved     *EventIssueApproved
	SentBack          *EventIssueSentBack
	RolloutFailed     *EventRolloutFailed
	RolloutCompleted  *EventRolloutCompleted
```

- [ ] **Step 2: Add `EventIssueApproved` struct**

After `EventIssueSentBack` (line 87), add:

```go
type EventIssueApproved struct {
	Approver *User
	Creator  *User
	Issue    *Issue
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/component/webhook/event.go
git commit -m "feat: add EventIssueApproved struct"
```

### Task 5: Add `ISSUE_APPROVED` handler in manager

**Files:**
- Modify: `backend/component/webhook/manager.go` (insert new case between `ISSUE_APPROVAL_REQUESTED` and `ISSUE_SENT_BACK`)

- [ ] **Step 1: Add switch case**

In `backend/component/webhook/manager.go`, insert after the closing brace of the `ISSUE_APPROVAL_REQUESTED` case and before the `ISSUE_SENT_BACK` case (note: line numbers will have shifted from Chunk 2 edits — match by the case labels, not line numbers):

```go
	case storepb.Activity_ISSUE_APPROVED:
		level = webhook.WebhookSuccess
		title = "Issue approved"
		titleZh = "工单审批通过"
		if e.IssueApproved != nil {
			actor = e.IssueApproved.Approver
			issue = e.IssueApproved.Issue
			link = fmt.Sprintf("%s/projects/%s/issues/%d", externalURL, e.Project.ResourceID, issue.UID)
			description = fmt.Sprintf("%s approved the issue", e.IssueApproved.Approver.Name)
			mentionUsers = []*store.UserMessage{{
				Name:  e.IssueApproved.Creator.Name,
				Email: e.IssueApproved.Creator.Email,
				Type:  storepb.PrincipalType_END_USER,
			}}
		}
```

Note: Uses `description` local variable (from Chunk 2 fix), not `webhookCtx.Description`.

- [ ] **Step 2: Format Go code**

Run: `gofmt -w backend/component/webhook/manager.go`

- [ ] **Step 3: Run Go lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No new issues.

- [ ] **Step 4: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Compiles without errors.

- [ ] **Step 5: Commit**

```bash
git add backend/component/webhook/manager.go
git commit -m "feat: add ISSUE_APPROVED handler in webhook manager"
```

### Task 6: Add `NotifyIssueApproved` helper and trigger

**Files:**
- Modify: `backend/runner/approval/runner.go` (add function after `NotifyApprovalRequested`)
- Modify: `backend/api/v1/issue_service.go:634-640`

- [ ] **Step 1: Add `NotifyIssueApproved` function**

In `backend/runner/approval/runner.go`, add after line 1032 (end of `NotifyApprovalRequested`):

```go
// NotifyIssueApproved sends the ISSUE_APPROVED webhook event when all approval steps complete.
// It notifies the issue creator that their issue has been fully approved.
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

- [ ] **Step 2: Wire up trigger in `ApproveIssue`**

In `backend/api/v1/issue_service.go`, replace lines 634-640:

```go
	// If the issue is a role grant request and approved, complete it.
	if approved {
		issue, err = completeAccessRequestIssue(ctx, s.store, user.Email, issue)
		if err != nil {
			slog.Debug("failed to complete role grant issue", log.BBError(err))
		}
	}
```

with:

```go
	// If the issue is approved, notify the creator and complete access request if applicable.
	if approved {
		approval.NotifyIssueApproved(ctx, s.store, s.webhookManager, issue, project, user)
		issue, err = completeAccessRequestIssue(ctx, s.store, user.Email, issue)
		if err != nil {
			slog.Debug("failed to complete role grant issue", log.BBError(err))
		}
	}
```

- [ ] **Step 3: Format Go code**

Run: `gofmt -w backend/runner/approval/runner.go backend/api/v1/issue_service.go`

- [ ] **Step 4: Run Go lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No new issues.

- [ ] **Step 5: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Compiles without errors.

- [ ] **Step 6: Commit**

```bash
git add backend/runner/approval/runner.go backend/api/v1/issue_service.go
git commit -m "feat: trigger ISSUE_APPROVED webhook when all approval steps complete"
```

---

## Chunk 4: Frontend + i18n

### Task 7: Add i18n strings

**Files:**
- Modify: `frontend/src/locales/en-US.json:1850`
- Modify: `frontend/src/locales/zh-CN.json:1850`
- Modify: `frontend/src/locales/ja-JP.json:1850`
- Modify: `frontend/src/locales/es-ES.json:1850`
- Modify: `frontend/src/locales/vi-VN.json:1850`

In all files, insert after the `issue-sent-back` closing `}` (line 1850) and before `"pipeline-failed"` (line 1851).

- [ ] **Step 1: Add en-US locale strings**

In `frontend/src/locales/en-US.json`, after line 1850 (`}`), add:

```json
        "issue-approved": {
          "title": "Issue approved",
          "label": "Notify when all approval steps are completed"
        },
```

- [ ] **Step 2: Add zh-CN locale strings**

In `frontend/src/locales/zh-CN.json`, after line 1850 (`}`), add:

```json
        "issue-approved": {
          "title": "工单审批通过",
          "label": "当所有审批步骤完成时通知"
        },
```

- [ ] **Step 3: Add ja-JP locale strings**

In `frontend/src/locales/ja-JP.json`, after line 1850 (`}`), add:

```json
        "issue-approved": {
          "title": "Issue approved",
          "label": "Notify when all approval steps are completed"
        },
```

- [ ] **Step 4: Add es-ES locale strings**

In `frontend/src/locales/es-ES.json`, after line 1850 (`}`), add:

```json
        "issue-approved": {
          "title": "Issue approved",
          "label": "Notify when all approval steps are completed"
        },
```

- [ ] **Step 5: Add vi-VN locale strings**

In `frontend/src/locales/vi-VN.json`, after line 1850 (`}`), add:

```json
        "issue-approved": {
          "title": "Issue approved",
          "label": "Notify when all approval steps are completed"
        },
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/locales/
git commit -m "feat: add i18n strings for issue-approved webhook event"
```

### Task 8: Add activity item to frontend webhook config

**Files:**
- Modify: `frontend/src/types/v1/projectWebhook.ts:119`

- [ ] **Step 1: Add activity item**

In `frontend/src/types/v1/projectWebhook.ts`, after the `issue-sent-back` entry (line 113-119) and before the `pipeline-failed` entry, add:

```typescript
      {
        title: t("project.webhook.activity-item.issue-approved.title"),
        label: t("project.webhook.activity-item.issue-approved.label"),
        activity: Activity_Type.ISSUE_APPROVED,
        supportDirectMessage: true,
      },
```

- [ ] **Step 2: Run frontend fix**

Run: `pnpm --dir frontend fix`
Expected: No errors.

- [ ] **Step 3: Run frontend type check**

Run: `pnpm --dir frontend type-check`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/v1/projectWebhook.ts
git commit -m "feat: add issue-approved to frontend webhook activity list"
```

---

## Chunk 5: Verification

### Task 9: Full build and lint verification

- [ ] **Step 1: Backend build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Success.

- [ ] **Step 2: Backend lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No new issues. Run repeatedly until clean.

- [ ] **Step 3: Frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors.

- [ ] **Step 4: Frontend type check**

Run: `pnpm --dir frontend type-check`
Expected: No errors.

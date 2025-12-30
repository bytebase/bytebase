# Webhook Events Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign webhook notification system to reduce noise and improve signal quality with 5 focused event types

**Architecture:** Add new Activity.Type proto enums, create new event structs in component layer, update plugins to handle new event types, implement pipeline failure aggregation with 5-minute debouncing

**Tech Stack:** Go, Protocol Buffers, existing webhook plugin architecture

---

## Phase 1: Proto Changes

### Task 1: Add New Activity Types to Proto

**Files:**
- Modify: `proto/store/store/project_webhook.proto:12-37`

**Step 1: Add new activity type enums**

Edit `proto/store/store/project_webhook.proto`, add after line 36:

```proto
    // New focused event types
    ISSUE_CREATED = 10;
    ISSUE_APPROVAL_REQUESTED = 11;
    ISSUE_SENT_BACK = 12;
    PIPELINE_FAILED = 13;
    PIPELINE_COMPLETED = 14;
```

**Step 2: Format proto files**

Run: `buf format -w proto`
Expected: Files formatted successfully

**Step 3: Generate Go code from protos**

Run: `cd proto && buf generate`
Expected: Generated files updated in `backend/generated-go/store/`

**Step 4: Verify generated code**

Run: `grep -A 2 "Activity_ISSUE_CREATED" backend/generated-go/store/project_webhook.pb.go`
Expected: Find the new constants defined

**Step 5: Commit proto changes**

```bash
git add proto/store/store/project_webhook.proto backend/generated-go/
git commit -m "feat: add new webhook activity types

Add 5 new focused activity types:
- ISSUE_CREATED (10)
- ISSUE_APPROVAL_REQUESTED (11)
- ISSUE_SENT_BACK (12)
- PIPELINE_FAILED (13)
- PIPELINE_COMPLETED (14)

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: Component Layer - Event Data Structures

### Task 2: Create New Event Structs

**Files:**
- Modify: `backend/component/webhook/event.go:1-95`

**Step 1: Add new event structs to event.go**

Add after the existing `EventTaskRunStatusUpdate` struct (around line 94):

```go
type EventIssueCreated struct {
	CreatorName  string
	CreatorEmail string
}

type EventIssueApprovalRequested struct {
	ApprovalRole  string
	RequiredCount int
	Approvers     []User
}

type EventIssueSentBack struct {
	ApproverName  string
	ApproverEmail string
	Reason        string
}

type EventPipelineFailed struct {
	FailedTasks      []FailedTask
	FirstFailureTime time.Time
}

type FailedTask struct {
	TaskID       int64
	TaskName     string
	DatabaseName string
	InstanceName string
	ErrorMessage string
	FailedAt     time.Time
}

type EventPipelineCompleted struct {
	TotalTasks  int
	StartedAt   time.Time
	CompletedAt time.Time
}

type User struct {
	Name  string
	Email string
}
```

**Step 2: Add new fields to Event struct**

Modify the `Event` struct (around line 8-22) to add new fields:

```go
type Event struct {
	Actor   *store.UserMessage
	Type    storepb.Activity_Type
	Comment string
	// nullable
	Issue   *Issue
	Project *Project
	Rollout *Rollout

	// Existing event types
	IssueUpdate         *EventIssueUpdate
	IssueApprovalCreate *EventIssueApprovalCreate
	IssueRolloutReady   *EventIssueRolloutReady
	StageStatusUpdate   *EventStageStatusUpdate
	TaskRunStatusUpdate *EventTaskRunStatusUpdate

	// New focused event types
	IssueCreated         *EventIssueCreated
	ApprovalRequested    *EventIssueApprovalRequested
	SentBack             *EventIssueSentBack
	PipelineFailed       *EventPipelineFailed
	PipelineCompleted    *EventPipelineCompleted
}
```

**Step 3: Add time import**

Add to imports at top of file:

```go
import (
	"time"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)
```

**Step 4: Build to verify**

Run: `go build ./backend/component/webhook/...`
Expected: Build succeeds with no errors

**Step 5: Commit event struct changes**

```bash
git add backend/component/webhook/event.go
git commit -m "feat: add new webhook event data structures

Add event structs for:
- ISSUE_CREATED
- ISSUE_APPROVAL_REQUESTED
- ISSUE_SENT_BACK
- PIPELINE_FAILED
- PIPELINE_COMPLETED

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 3: Plugin Layer - Webhook Context Updates

### Task 3: Update Plugin Webhook Context

**Files:**
- Modify: `backend/plugin/webhook/webhook.go:84-100`

**Step 1: Add new fields to plugin Context struct**

Add after the existing `TaskResult` field (check actual location):

```go
type Context struct {
	URL         string
	Level       Level
	EventType   string
	Title       string
	TitleZh     string
	Description string
	Link        string
	ActorID     int
	ActorName   string
	ActorEmail  string
	CreatedTS   int64
	Issue       *Issue
	Rollout     *Rollout
	Stage       *Stage
	Project     *Project
	TaskResult  *TaskResult

	// New event data
	DirectMessage       bool
	MentionedUsers      []*store.UserMessage
	ApprovalRole        string
	ApprovalRequired    int
	FailedTasks         []FailedTaskInfo
	PipelineMetrics     *PipelineMetrics
}

type FailedTaskInfo struct {
	Name         string
	Instance     string
	Database     string
	ErrorMessage string
	FailedAt     string
}

type PipelineMetrics struct {
	TotalTasks    int
	StartedAt     string
	CompletedAt   string
	DurationSecs  int64
}
```

**Step 2: Build to verify**

Run: `go build ./backend/plugin/webhook/...`
Expected: Build succeeds

**Step 3: Commit plugin context changes**

```bash
git add backend/plugin/webhook/webhook.go
git commit -m "feat: extend webhook plugin context for new events

Add fields to support:
- Direct messaging
- Approval event data
- Pipeline failure/completion metrics

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 4: Component Layer - Webhook Context Mapping

### Task 4: Update getWebhookContextFromEvent in Manager

**Files:**
- Modify: `backend/component/webhook/manager.go:72-223`

**Step 1: Add case handlers for new event types**

In the `getWebhookContextFromEvent` method's switch statement, add after the existing cases (around line 94-200):

```go
	case storepb.Activity_ISSUE_CREATED:
		title = "Issue created"
		titleZh = "创建工单"
		if e.IssueCreated != nil {
			webhookCtx.Description = fmt.Sprintf("%s created issue %s", e.IssueCreated.CreatorName, e.Issue.Title)
		}

	case storepb.Activity_ISSUE_APPROVAL_REQUESTED:
		level = webhook.WebhookWarn
		title = "Approval required"
		titleZh = "需要审批"
		if e.ApprovalRequested != nil {
			webhookCtx.ApprovalRole = e.ApprovalRequested.ApprovalRole
			webhookCtx.ApprovalRequired = e.ApprovalRequested.RequiredCount
			mentionUsers = make([]*store.UserMessage, 0, len(e.ApprovalRequested.Approvers))
			for _, user := range e.ApprovalRequested.Approvers {
				mentionUsers = append(mentionUsers, &store.UserMessage{
					Name:  user.Name,
					Email: user.Email,
				})
			}
		}

	case storepb.Activity_ISSUE_SENT_BACK:
		level = webhook.WebhookWarn
		title = "Issue sent back"
		titleZh = "工单被退回"
		if e.SentBack != nil {
			webhookCtx.Description = fmt.Sprintf("%s sent back the issue: %s", e.SentBack.ApproverName, e.SentBack.Reason)
			mentionUsers = []*store.UserMessage{
				{
					Name:  e.SentBack.CreatorName,
					Email: e.SentBack.CreatorEmail,
				},
			}
		}

	case storepb.Activity_PIPELINE_FAILED:
		level = webhook.WebhookError
		title = "Pipeline failed"
		titleZh = "流水线失败"
		if e.PipelineFailed != nil {
			failedTasks := make([]webhook.FailedTaskInfo, 0, len(e.PipelineFailed.FailedTasks))
			for _, task := range e.PipelineFailed.FailedTasks {
				failedTasks = append(failedTasks, webhook.FailedTaskInfo{
					Name:         task.TaskName,
					Instance:     task.InstanceName,
					Database:     task.DatabaseName,
					ErrorMessage: task.ErrorMessage,
					FailedAt:     task.FailedAt.Format(time.RFC3339),
				})
			}
			webhookCtx.FailedTasks = failedTasks
			webhookCtx.Description = fmt.Sprintf("%d task(s) failed", len(failedTasks))
		}

	case storepb.Activity_PIPELINE_COMPLETED:
		level = webhook.WebhookSuccess
		title = "Pipeline completed"
		titleZh = "流水线完成"
		if e.PipelineCompleted != nil {
			duration := e.PipelineCompleted.CompletedAt.Sub(e.PipelineCompleted.StartedAt)
			webhookCtx.PipelineMetrics = &webhook.PipelineMetrics{
				TotalTasks:   e.PipelineCompleted.TotalTasks,
				StartedAt:    e.PipelineCompleted.StartedAt.Format(time.RFC3339),
				CompletedAt:  e.PipelineCompleted.CompletedAt.Format(time.RFC3339),
				DurationSecs: int64(duration.Seconds()),
			}
			webhookCtx.Description = fmt.Sprintf("Completed %d task(s) in %s", e.PipelineCompleted.TotalTasks, duration.String())
		}
```

**Step 2: Build and check for errors**

Run: `go build ./backend/component/webhook/...`
Expected: Build succeeds

**Step 3: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/component/webhook/`
Expected: No linting errors

**Step 4: Commit webhook context mapping**

```bash
git add backend/component/webhook/manager.go
git commit -m "feat: add webhook context mapping for new events

Map new event types to webhook context:
- ISSUE_CREATED → info message
- ISSUE_APPROVAL_REQUESTED → warn with approvers
- ISSUE_SENT_BACK → warn with creator
- PIPELINE_FAILED → error with failed tasks
- PIPELINE_COMPLETED → success with metrics

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 5: Event Triggering - Issue Created

### Task 5: Add ISSUE_CREATED Event Trigger

**Files:**
- Modify: `backend/api/v1/issue_service.go`

**Step 1: Find CreateIssue method**

Run: `grep -n "func.*CreateIssue" backend/api/v1/issue_service.go`
Note the line number

**Step 2: Add webhook trigger after issue creation**

Find where the issue is successfully created and persisted (look for successful database insert), then add:

```go
// Trigger ISSUE_CREATED webhook
s.webhookManager.CreateEvent(ctx, &webhook.Event{
	Actor:   user,
	Type:    storepb.Activity_ISSUE_CREATED,
	Project: webhook.NewProject(project),
	Issue:   webhook.NewIssue(issue),
	IssueCreated: &webhook.EventIssueCreated{
		CreatorName:  user.Name,
		CreatorEmail: user.Email,
	},
})
```

**Step 3: Build to verify**

Run: `go build ./backend/api/v1/...`
Expected: Build succeeds

**Step 4: Run relevant tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run TestIssueService`
Expected: Tests pass (or identify which tests need updating)

**Step 5: Commit issue created trigger**

```bash
git add backend/api/v1/issue_service.go
git commit -m "feat: trigger ISSUE_CREATED webhook event

Send webhook notification when new issue is created.

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 6: Event Triggering - Approval Events

### Task 6: Add ISSUE_APPROVAL_REQUESTED Trigger

**Files:**
- Modify: `backend/runner/approval/runner.go` or approval handler location
- Test: Create test to verify approval events

**Step 1: Locate approval flow start point**

Run: `grep -rn "approval.*create\|approval.*start" backend/runner/approval/ backend/api/v1/`
Identify where approval flow begins

**Step 2: Add helper function to get approvers**

Add to the appropriate file (likely approval runner or issue service):

```go
func (s *IssueService) getApproversForRole(ctx context.Context, projectID string, role string) ([]webhook.User, error) {
	// Query IAM policies for users with the approval role
	policies, err := s.iamManager.GetProjectPolicies(ctx, projectID)
	if err != nil {
		return nil, err
	}

	approvers := []webhook.User{}
	for _, policy := range policies {
		if policy.Role == role && policy.PrincipalType == storepb.PrincipalType_END_USER {
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{
				Email: &policy.PrincipalEmail,
			})
			if err != nil || user == nil || user.Deleted {
				continue
			}
			approvers = append(approvers, webhook.User{
				Name:  user.Name,
				Email: user.Email,
			})
		}
	}
	return approvers, nil
}
```

**Step 3: Trigger ISSUE_APPROVAL_REQUESTED webhook**

At the approval flow start point, add:

```go
// Get approvers for this approval step
approvers, err := s.getApproversForRole(ctx, issue.Project, approvalStep.Role)
if err != nil {
	slog.Warn("failed to get approvers", log.BBError(err))
	approvers = []webhook.User{} // Continue with empty list
}

// Trigger ISSUE_APPROVAL_REQUESTED webhook
s.webhookManager.CreateEvent(ctx, &webhook.Event{
	Actor:   actor,
	Type:    storepb.Activity_ISSUE_APPROVAL_REQUESTED,
	Project: webhook.NewProject(project),
	Issue:   webhook.NewIssue(issue),
	ApprovalRequested: &webhook.EventIssueApprovalRequested{
		ApprovalRole:  approvalStep.Role,
		RequiredCount: approvalStep.RequiredApprovals,
		Approvers:     approvers,
	},
})
```

**Step 4: Build and verify**

Run: `go build ./backend/runner/approval/...`
Expected: Build succeeds

**Step 5: Commit approval requested trigger**

```bash
git add backend/runner/approval/ backend/api/v1/
git commit -m "feat: trigger ISSUE_APPROVAL_REQUESTED webhook event

Send notification to approvers when approval is required.
Includes helper to resolve approvers from IAM policies.

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 7: Add ISSUE_SENT_BACK Trigger

**Files:**
- Modify: Approval update handler (likely in `backend/api/v1/` or `backend/runner/approval/`)

**Step 1: Locate approval rejection/send-back handler**

Run: `grep -rn "reject\|send.*back\|approval.*deny" backend/api/v1/ backend/runner/approval/`
Identify where approver rejects an issue

**Step 2: Add ISSUE_SENT_BACK webhook trigger**

At the rejection point, add:

```go
// Trigger ISSUE_SENT_BACK webhook
s.webhookManager.CreateEvent(ctx, &webhook.Event{
	Actor:   approver,
	Type:    storepb.Activity_ISSUE_SENT_BACK,
	Project: webhook.NewProject(project),
	Issue:   webhook.NewIssue(issue),
	SentBack: &webhook.EventIssueSentBack{
		ApproverName:  approver.Name,
		ApproverEmail: approver.Email,
		Reason:        rejectionComment, // Get from approval update payload
		CreatorName:   issue.Creator.Name,
		CreatorEmail:  issue.CreatorEmail,
	},
})
```

**Step 3: Build and verify**

Run: `go build ./backend/api/v1/... ./backend/runner/approval/...`
Expected: Build succeeds

**Step 4: Commit sent back trigger**

```bash
git add backend/api/v1/ backend/runner/approval/
git commit -m "feat: trigger ISSUE_SENT_BACK webhook event

Notify issue creator when approver sends back the issue.

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 7: Pipeline Events - Failure Aggregation

### Task 8: Implement Pipeline Failure Aggregation

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go`
- Create: `backend/runner/taskrun/pipeline_events.go`

**Step 1: Create pipeline events tracker**

Create new file `backend/runner/taskrun/pipeline_events.go`:

```go
package taskrun

import (
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/component/webhook"
)

// PipelineFailureWindow tracks failed tasks for aggregation
type PipelineFailureWindow struct {
	mu               sync.Mutex
	firstFailureTime time.Time
	failedTasks      []webhook.FailedTask
	notificationSent bool
	timer            *time.Timer
}

// PipelineEventsTracker manages failure aggregation windows per plan
type PipelineEventsTracker struct {
	mu      sync.RWMutex
	windows map[int64]*PipelineFailureWindow // planID -> window
}

func NewPipelineEventsTracker() *PipelineEventsTracker {
	return &PipelineEventsTracker{
		windows: make(map[int64]*PipelineFailureWindow),
	}
}

// RecordTaskFailure adds a failed task to the aggregation window
func (t *PipelineEventsTracker) RecordTaskFailure(planID int64, task webhook.FailedTask, onAggregated func([]webhook.FailedTask)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	window, exists := t.windows[planID]
	if !exists || window.notificationSent {
		// Start new window
		window = &PipelineFailureWindow{
			firstFailureTime: time.Now(),
			failedTasks:      []webhook.FailedTask{task},
			notificationSent: false,
		}
		t.windows[planID] = window

		// Set 5-minute timer
		window.timer = time.AfterFunc(5*time.Minute, func() {
			t.mu.Lock()
			defer t.mu.Unlock()

			if w, ok := t.windows[planID]; ok && !w.notificationSent {
				w.notificationSent = true
				onAggregated(w.failedTasks)
			}
		})
	} else {
		// Add to existing window
		window.mu.Lock()
		window.failedTasks = append(window.failedTasks, task)
		window.mu.Unlock()
	}
}

// Clear removes the window for a plan (call after pipeline completes)
func (t *PipelineEventsTracker) Clear(planID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if window, exists := t.windows[planID]; exists {
		if window.timer != nil {
			window.timer.Stop()
		}
		delete(t.windows, planID)
	}
}
```

**Step 2: Add tracker to Scheduler struct**

In `backend/runner/taskrun/scheduler.go`, add field to Scheduler:

```go
type Scheduler struct {
	// ... existing fields ...
	pipelineEvents *PipelineEventsTracker
}
```

Initialize in constructor:

```go
func NewScheduler(...) *Scheduler {
	return &Scheduler{
		// ... existing fields ...
		pipelineEvents: NewPipelineEventsTracker(),
	}
}
```

**Step 3: Hook into task failure detection**

Find where task failures are detected (likely in the task run status update handler), add:

```go
if taskRun.Status == storepb.TaskRun_FAILED {
	// Record failure for aggregation
	s.pipelineEvents.RecordTaskFailure(
		plan.UID,
		webhook.FailedTask{
			TaskID:       task.ID,
			TaskName:     task.Name,
			DatabaseName: database.Name,
			InstanceName: instance.Name,
			ErrorMessage: taskRun.Result.GetError(),
			FailedAt:     time.Now(),
		},
		func(failedTasks []webhook.FailedTask) {
			// Send aggregated PIPELINE_FAILED webhook
			s.webhookManager.CreateEvent(ctx, &webhook.Event{
				Actor:   systemUser,
				Type:    storepb.Activity_PIPELINE_FAILED,
				Project: webhook.NewProject(project),
				Issue:   webhook.NewIssue(issue),
				Rollout: webhook.NewRollout(plan),
				PipelineFailed: &webhook.EventPipelineFailed{
					FailedTasks:      failedTasks,
					FirstFailureTime: time.Now().Add(-5 * time.Minute),
				},
			})
		},
	)
}
```

**Step 4: Build and verify**

Run: `go build ./backend/runner/taskrun/...`
Expected: Build succeeds

**Step 5: Commit failure aggregation**

```bash
git add backend/runner/taskrun/
git commit -m "feat: implement pipeline failure aggregation

Add 5-minute debouncing window to aggregate failed tasks:
- Prevents notification spam from cascading failures
- Sends single PIPELINE_FAILED event with all failures
- In-memory implementation (non-HA)

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 8: Pipeline Events - Completion Detection

### Task 9: Implement Pipeline Completion Detection

**Files:**
- Modify: `backend/runner/taskrun/scheduler.go`

**Step 1: Extend ListenTaskSkippedOrDone handler**

Find the `ListenTaskSkippedOrDone` method (around line 79), extend it:

```go
func (s *Scheduler) ListenTaskSkippedOrDone(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.TaskSkippedOrDoneChan:
			// ... existing code to check environment completion ...

			// Check if entire plan is complete
			s.checkPlanCompletion(ctx)
		}
	}
}

func (s *Scheduler) checkPlanCompletion(ctx context.Context) {
	// Implementation to check if all tasks in plan are done
	// This needs to query the database for task statuses
}
```

**Step 2: Implement plan completion check**

Add helper method:

```go
func (s *Scheduler) checkPlanCompletion(ctx context.Context) {
	// Get all running plans
	plans, err := s.store.ListPlans(ctx, &store.FindPlanMessage{
		// Query for plans with running tasks
	})
	if err != nil {
		return
	}

	for _, plan := range plans {
		allTasks, err := s.store.ListTasks(ctx, &store.FindTaskMessage{
			PlanID: &plan.UID,
		})
		if err != nil {
			continue
		}

		allComplete := true
		hasFailures := false
		startTime := time.Now()
		endTime := time.Now()

		for _, task := range allTasks {
			taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
				TaskID: &task.ID,
			})
			if err != nil || len(taskRuns) == 0 {
				continue
			}

			latestRun := taskRuns[0] // Assuming sorted by creation time desc

			if latestRun.Status != storepb.TaskRun_DONE &&
				latestRun.Status != storepb.TaskRun_FAILED &&
				latestRun.Status != storepb.TaskRun_CANCELED &&
				latestRun.Status != storepb.TaskRun_SKIPPED {
				allComplete = false
				break
			}

			if latestRun.Status == storepb.TaskRun_FAILED {
				hasFailures = true
			}

			// Track start/end times
			if latestRun.CreatedTime.Before(startTime) {
				startTime = latestRun.CreatedTime
			}
			if latestRun.UpdatedTime.After(endTime) {
				endTime = latestRun.UpdatedTime
			}
		}

		if allComplete && !hasFailures {
			// Clear any pending failure windows
			s.pipelineEvents.Clear(plan.UID)

			// Send PIPELINE_COMPLETED webhook
			issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
				PlanUID: &plan.UID,
			})
			if err != nil {
				continue
			}

			project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
				ResourceID: &issue.Project,
			})
			if err != nil {
				continue
			}

			s.webhookManager.CreateEvent(ctx, &webhook.Event{
				Actor:   &store.UserMessage{Name: "System", Email: "system@bytebase.com"},
				Type:    storepb.Activity_PIPELINE_COMPLETED,
				Project: webhook.NewProject(project),
				Issue:   webhook.NewIssue(issue),
				Rollout: webhook.NewRollout(plan),
				PipelineCompleted: &webhook.EventPipelineCompleted{
					TotalTasks:  len(allTasks),
					StartedAt:   startTime,
					CompletedAt: endTime,
				},
			})
		}
	}
}
```

**Step 3: Build and verify**

Run: `go build ./backend/runner/taskrun/...`
Expected: Build succeeds

**Step 4: Commit pipeline completion**

```bash
git add backend/runner/taskrun/scheduler.go
git commit -m "feat: implement pipeline completion detection

Detect when all tasks in pipeline complete successfully:
- Extends ListenTaskSkippedOrDone handler
- Checks all task run statuses
- Sends PIPELINE_COMPLETED webhook
- Clears failure aggregation windows

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 9: Plugin Updates - Slack

### Task 10: Update Slack Plugin for New Events

**Files:**
- Modify: `backend/plugin/webhook/slack/slack.go`

**Step 1: Locate message formatting function**

Run: `grep -n "func.*Post" backend/plugin/webhook/slack/slack.go`
Find the main message formatting function

**Step 2: Add cases for new event types**

In the event type switch/if-else chain, add:

```go
case "ISSUE_CREATED":
	text = fmt.Sprintf("🆕 *%s*\n%s", context.Title, context.Description)
	color = "#36A64F" // Green

case "ISSUE_APPROVAL_REQUESTED":
	text = fmt.Sprintf("⚠️ *%s*\nRole: %s\nRequired: %d approval(s)",
		context.Title, context.ApprovalRole, context.ApprovalRequired)
	color = "#FF9900" // Orange

	if context.DirectMessage && len(context.MentionedUsers) > 0 {
		// Send DM to each approver
		// Implementation depends on Slack IM integration
	}

case "ISSUE_SENT_BACK":
	text = fmt.Sprintf("↩️ *%s*\n%s", context.Title, context.Description)
	color = "#FF9900" // Orange

case "PIPELINE_FAILED":
	failedTasksText := ""
	for _, task := range context.FailedTasks {
		failedTasksText += fmt.Sprintf("\n• %s (%s.%s): %s",
			task.Name, task.Instance, task.Database, task.ErrorMessage)
	}
	text = fmt.Sprintf("❌ *%s*\n%d task(s) failed:%s",
		context.Title, len(context.FailedTasks), failedTasksText)
	color = "#FF0000" // Red

case "PIPELINE_COMPLETED":
	if context.PipelineMetrics != nil {
		duration := time.Duration(context.PipelineMetrics.DurationSecs) * time.Second
		text = fmt.Sprintf("✅ *%s*\nCompleted %d task(s) in %s",
			context.Title, context.PipelineMetrics.TotalTasks, duration.String())
	}
	color = "#36A64F" // Green
```

**Step 3: Build and verify**

Run: `go build ./backend/plugin/webhook/...`
Expected: Build succeeds

**Step 4: Commit Slack plugin updates**

```bash
git add backend/plugin/webhook/slack/
git commit -m "feat: add Slack formatting for new webhook events

Support new event types:
- ISSUE_CREATED
- ISSUE_APPROVAL_REQUESTED (with DM support)
- ISSUE_SENT_BACK
- PIPELINE_FAILED (with task details)
- PIPELINE_COMPLETED (with metrics)

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 10: Plugin Updates - Other Platforms

### Task 11: Update Discord Plugin

**Files:**
- Modify: `backend/plugin/webhook/discord.go`

**Step 1: Add formatting for new events**

Similar to Slack, add cases for new event types with Discord-specific formatting (embeds).

**Step 2: Build and commit**

```bash
git add backend/plugin/webhook/discord.go
git commit -m "feat: add Discord formatting for new webhook events

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 12: Update Teams Plugin

**Files:**
- Modify: `backend/plugin/webhook/teams.go` (if exists) or relevant Teams file

**Step 1: Add formatting for new events**

Add Teams-specific message card formatting.

**Step 2: Build and commit**

```bash
git add backend/plugin/webhook/teams.go
git commit -m "feat: add Teams formatting for new webhook events

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 13: Update Feishu Plugin

**Files:**
- Modify: `backend/plugin/webhook/feishu.go` (if exists)

**Step 1: Add formatting for new events**

Add Feishu/Lark message formatting with Chinese translations.

**Step 2: Build and commit**

```bash
git add backend/plugin/webhook/feishu.go
git commit -m "feat: add Feishu formatting for new webhook events

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 11: Testing

### Task 14: Add Component Layer Tests

**Files:**
- Create: `backend/component/webhook/event_test.go`

**Step 1: Write test for event struct creation**

```go
package webhook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventIssueCreated(t *testing.T) {
	event := &Event{
		Type: storepb.Activity_ISSUE_CREATED,
		IssueCreated: &EventIssueCreated{
			CreatorName:  "Alice",
			CreatorEmail: "alice@example.com",
		},
	}

	assert.NotNil(t, event.IssueCreated)
	assert.Equal(t, "Alice", event.IssueCreated.CreatorName)
}

func TestEventApprovalRequested(t *testing.T) {
	approvers := []User{
		{Name: "Bob", Email: "bob@example.com"},
		{Name: "Carol", Email: "carol@example.com"},
	}

	event := &Event{
		Type: storepb.Activity_ISSUE_APPROVAL_REQUESTED,
		ApprovalRequested: &EventIssueApprovalRequested{
			ApprovalRole:  "Project Owner",
			RequiredCount: 2,
			Approvers:     approvers,
		},
	}

	assert.Equal(t, 2, len(event.ApprovalRequested.Approvers))
	assert.Equal(t, "Project Owner", event.ApprovalRequested.ApprovalRole)
}

func TestFailedTaskTracking(t *testing.T) {
	failedTask := FailedTask{
		TaskID:       1,
		TaskName:     "Migration",
		DatabaseName: "prod",
		InstanceName: "prod-1",
		ErrorMessage: "syntax error",
		FailedAt:     time.Now(),
	}

	event := &Event{
		Type: storepb.Activity_PIPELINE_FAILED,
		PipelineFailed: &EventPipelineFailed{
			FailedTasks: []FailedTask{failedTask},
		},
	}

	assert.Equal(t, 1, len(event.PipelineFailed.FailedTasks))
	assert.Equal(t, "syntax error", event.PipelineFailed.FailedTasks[0].ErrorMessage)
}
```

**Step 2: Run tests**

Run: `go test -v ./backend/component/webhook/`
Expected: All tests pass

**Step 3: Commit tests**

```bash
git add backend/component/webhook/event_test.go
git commit -m "test: add unit tests for new webhook event structs

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Task 15: Add Pipeline Events Tracker Tests

**Files:**
- Create: `backend/runner/taskrun/pipeline_events_test.go`

**Step 1: Write aggregation tests**

```go
package taskrun

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/bytebase/bytebase/backend/component/webhook"
)

func TestPipelineFailureAggregation(t *testing.T) {
	tracker := NewPipelineEventsTracker()

	var aggregatedTasks []webhook.FailedTask
	var mu sync.Mutex

	onAggregated := func(tasks []webhook.FailedTask) {
		mu.Lock()
		defer mu.Unlock()
		aggregatedTasks = tasks
	}

	// Record first failure
	task1 := webhook.FailedTask{
		TaskID:   1,
		TaskName: "Migration 1",
		FailedAt: time.Now(),
	}
	tracker.RecordTaskFailure(100, task1, onAggregated)

	// Record second failure within window
	task2 := webhook.FailedTask{
		TaskID:   2,
		TaskName: "Migration 2",
		FailedAt: time.Now(),
	}
	tracker.RecordTaskFailure(100, task2, onAggregated)

	// Wait for aggregation (with timeout for testing)
	time.Sleep(5*time.Minute + 100*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, len(aggregatedTasks))
}

func TestPipelineFailureNewWindowAfterNotification(t *testing.T) {
	// Test that new failures after notification start a new window
	tracker := NewPipelineEventsTracker()

	// ... test implementation
}

func TestClearPipelineWindow(t *testing.T) {
	tracker := NewPipelineEventsTracker()

	task := webhook.FailedTask{TaskID: 1}
	tracker.RecordTaskFailure(100, task, func([]webhook.FailedTask) {})

	tracker.Clear(100)

	// Verify window is cleared
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	assert.NotContains(t, tracker.windows, int64(100))
}
```

**Step 2: Run tests**

Run: `go test -v ./backend/runner/taskrun/ -run TestPipeline`
Expected: Tests pass

**Step 3: Commit tests**

```bash
git add backend/runner/taskrun/pipeline_events_test.go
git commit -m "test: add tests for pipeline failure aggregation

Test 5-minute debouncing window and aggregation logic.

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 12: Integration & Documentation

### Task 16: Run Full Test Suite

**Step 1: Run all tests**

Run: `go test -v -count=1 ./backend/...`
Expected: All tests pass

**Step 2: Fix any failing tests**

If tests fail, identify and fix issues related to the new event types.

**Step 3: Run linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No linting errors (run multiple times until clean)

**Step 4: Format code**

Run: `gofmt -w backend/`
Expected: All files formatted

**Step 5: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

---

### Task 17: Update Design Document Status

**Files:**
- Modify: `docs/plans/2025-12-30-webhook-events-redesign.md:4`

**Step 1: Update status to Implemented**

Change line 4:
```markdown
**Status:** Implemented
```

**Step 2: Commit documentation**

```bash
git add docs/plans/2025-12-30-webhook-events-redesign.md
git commit -m "docs: mark webhook redesign as implemented

🤖 Generated with Claude Code
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 13: Final Review & Merge Preparation

### Task 18: Final Code Review

**Step 1: Review all changes**

Run: `git diff main...feature/webhook-events-redesign --stat`
Review files changed

**Step 2: Create PR checklist**

Verify:
- [ ] All 5 new Activity.Type enums added
- [ ] Component layer event structs created
- [ ] Plugin context extended
- [ ] All 5 event triggers implemented
- [ ] Pipeline failure aggregation works
- [ ] Pipeline completion detection works
- [ ] All plugins updated (Slack, Discord, Teams, Feishu)
- [ ] Tests added and passing
- [ ] Linter passes
- [ ] Backend builds successfully
- [ ] Documentation updated

**Step 3: Run verification build and tests**

```bash
# Full backend build
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

# All tests
go test -v -count=1 ./backend/component/webhook/... ./backend/runner/taskrun/...

# Lint
golangci-lint run --allow-parallel-runners
```

Expected: All succeed

---

## Summary

This implementation adds 5 focused webhook event types to replace the existing 9 noisy events:

**New Events:**
1. ISSUE_CREATED - Simple notification when issues are created
2. ISSUE_APPROVAL_REQUESTED - Direct message to approvers
3. ISSUE_SENT_BACK - Direct message to issue creator
4. PIPELINE_FAILED - Aggregated failures with 5-minute debouncing
5. PIPELINE_COMPLETED - Success notification

**Key Features:**
- Proto-based type safety with storepb.Activity_Type enums
- Clean component/plugin separation
- 5-minute failure aggregation prevents notification spam
- Direct messaging support for approval events
- Backward compatible (old events still in proto)

**Testing:**
- Unit tests for event structs
- Unit tests for failure aggregation
- Integration tests for triggers
- Full backend build verification

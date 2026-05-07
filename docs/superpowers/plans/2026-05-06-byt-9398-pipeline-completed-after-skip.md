# BYT-9398 PIPELINE_COMPLETED After Skip — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix BYT-9398 (`PIPELINE_COMPLETED` webhook silently dropped after `BatchSkipTasks` resolves a failed task) and add comprehensive integration test coverage for all six webhook trigger event types.

**Architecture:** Two coupled backend changes — tighten `Store.ResetPlanWebhookDelivery` to only delete `PIPELINE_FAILED` rows, then add a call to it from `BatchSkipTasks` mirroring the existing pattern in `BatchRunTasks`. Test coverage extends the existing `backend/tests/webhook_test.go` infrastructure with a per-trigger subtest matrix; each subtest owns its own project and webhook subscription, and webhook counts are filtered by project name to prevent cross-subtest pollution from the asynchronous webhook dispatcher.

**Tech Stack:** Go 1.22+, Connect-RPC, gRPC services exposed via `ctl.rolloutServiceClient` / `planServiceClient` / `issueServiceClient` / `projectServiceClient` / `settingServiceClient` / `userServiceClient` / `authServiceClient`, SQLite test instance, `httptest.NewServer` for webhook collection, `testify/require` assertions.

**Spec:** `docs/superpowers/specs/2026-05-06-byt-9398-pipeline-completed-after-skip-design.md`

---

## File Structure

| Path | Action | Responsibility |
|------|--------|----------------|
| `backend/api/v1/rollout_service.go` | Modify | Add `ResetPlanWebhookDelivery` call inside `BatchSkipTasks` |
| `backend/tests/webhook_test.go` | Modify | Drop the outer cross-subtest webhook setup; add per-trigger subtests |
| `backend/tests/webhook_helpers_test.go` | Create | Test-local helpers shared across subtests |

---

## Cross-Cutting Conventions

**Webhook subscription pattern.** Every subtest creates a fresh project and adds one webhook subscribed to *only* the event types it cares about. **The outer `TestWebhookIntegration` body must NOT register a workspace-wide webhook on `ctl.project`** (the existing test does this, leaking events into every later subtest). Move the existing `IssueWithPlanWebhookPayload` subtest onto a fresh project too.

**Event identification.** The Slack payload's first section block contains the event title decorated with an emoji and a `<link|title>` Slack-markup wrapper (verified at `backend/plugin/webhook/slack/slack_test.go:42-47, 85`). The `matchesEventTitle` helper extracts that section text and matches it via **`strings.Contains`** against the canonical title strings from `backend/component/webhook/manager.go:85,96,114,130,148,159` — `"Issue created"`, `"Approval required"`, `"Issue approved"`, `"Issue sent back"`, `"Rollout failed"`, `"Rollout completed"`.

**Project-scoped collector counting.** `Manager.CreateEvent` posts each webhook from a goroutine (`backend/component/webhook/manager.go:59`); a late delivery from a previous subtest can arrive after the next subtest's `collector.reset()`. Count helpers therefore filter on **both** title *and* project resource name (substring match against the captured payload), so cross-subtest contamination cannot cause false counts.

**Wait pattern.** `waitForWebhookCount(t, collector, project, eventTitle, n, timeout)` polls every 100ms; replaces the brittle `time.Sleep(5*time.Second)` in the existing test. For absence assertions, a fixed grace period is unavoidable (no signal means "wait forever"); these are flagged inline.

**Force-fail technique.** The `seedFailingSheet` helper writes SQL referencing a missing table (`INSERT INTO __force_fail_target VALUES(1)`); `unblockFailingTask` creates that table out-of-band so a retry succeeds. Each subtest uses uniquely named databases so the table scope (per-`.db`-file in SQLite) isolates them. Each unblock must complete with `db.Close()` before the retry is enqueued — `db.Close()` flushes WAL/journal so the file is consistent.

**SQLite layout.** `backend/tests/tests.go` (`provisionSQLiteInstance`) returns a directory; `backend/plugin/db/sqlite/sqlite.go:83` confirms each database lives at `<instanceDir>/<dbName>.db`. The outer `TestWebhookIntegration` body keeps `instanceDir` (line 163) and `instance` (line 177) in scope; subtests close over both.

**Task status.** Use `task.Status` against `v1pb.Task_FAILED` / `v1pb.Task_DONE` / `v1pb.Task_SKIPPED` (verified against `backend/generated-go/v1/rollout_service.pb.go:27-45` and `backend/tests/rollout.go:115,142,202,249`). There is no `LatestTaskRunStatus` field on `v1pb.Task`.

**Identity swap.** The controller has a single mutable `ctl.authInterceptor.token`. Every helper that switches identity uses a `defer` to restore the original token. Tests are sequential within `TestWebhookIntegration`, so concurrent token mutation is not a concern.

**Approval rule cleanup.** `WORKSPACE_APPROVAL` is a workspace-scoped setting. `installWorkspaceApprovalRule` snapshots the prior setting and registers a `t.Cleanup` to restore it, so subtests cannot leak approval rules into each other.

**SB2 reopen flow.** A rejected issue cannot be re-approved directly; `IssueService.RejectIssue` marks an approver as rejected and `ApproveIssue` returns `InvalidArgument` thereafter (verified at `backend/api/v1/issue_service.go:577-580, 685-688`). The only reopener is `IssueService.RequestIssue` (`issue_service.go:769-848`), and `canRequestIssue` requires the caller to be the issue creator. SB2 therefore: approver rejects → *creator* (default `ctl` token) calls `RequestIssue` → approver re-approves.

**Commit cadence.** Each task ends with a commit. Avoid amending; if a hook fails, fix and create a new commit.

---

## Chunk 1: The Fix and BYT-9398 Regression Test

After this chunk merges, the customer issue is resolved.

### Task 1: Add `ResetPlanWebhookDelivery` call to `BatchSkipTasks`

**Files:**
- Modify: `backend/api/v1/rollout_service.go` (insert after line 885)

- [ ] **Step 1: Audit existing callers**

Run: `git -C . grep -n "ResetPlanWebhookDelivery"`
Expected: only the store definition and one call in `BatchRunTasks`. If a third call site exists, stop and re-evaluate the recovery-path table in the spec.

- [ ] **Step 2: Insert the call inside `BatchSkipTasks`**

Open `backend/api/v1/rollout_service.go`, locate `BatchSkipTasks` (~line 858), find the `if plan == nil` block (~line 883–885) and the `s.store.GetIssue(...)` call (~line 887). Insert between them:

```go
    if plan == nil {
        return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
    }

    // Reset notification state so PIPELINE_COMPLETED can fire after skipping
    // a failed task. Mirrors the BatchRunTasks pattern at lines 744-748.
    // Errors are logged and swallowed so a DB hiccup doesn't fail the
    // user-facing skip request — a failure here will re-introduce the
    // BYT-9398 symptom for this plan, so the log line should be monitored.
    if err := s.store.ResetPlanWebhookDelivery(ctx, projectID, planID); err != nil {
        slog.Error("failed to reset plan webhook delivery", log.BBError(err))
    }

    issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
```

This is the entire backend code change — character-identical to the block already present in `BatchRunTasks` except for the comment.

- [ ] **Step 3: Compile**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: success.

- [ ] **Step 4: Lint**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/v1/...` (repeat until clean).
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add backend/api/v1/rollout_service.go
git commit -m "$(cat <<'EOF'
fix(rollout): reset plan webhook delivery on BatchSkipTasks

When a user skipped a failed task to recover a rollout, the stale
PIPELINE_FAILED row in plan_webhook_delivery blocked the subsequent
PIPELINE_COMPLETED claim and the webhook was silently dropped.

BatchRunTasks already calls ResetPlanWebhookDelivery; mirror the same
pattern in BatchSkipTasks so the recovery-via-skip path also fires
PIPELINE_COMPLETED.

Fixes BYT-9398.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Drop the leaky outer webhook from `TestWebhookIntegration` and migrate the existing subtest to a fresh project

**Files:**
- Modify: `backend/tests/webhook_test.go`

The current outer body (lines 179–193) registers a workspace-wide webhook on `ctl.project` for `ISSUE_CREATED`. That webhook persists across every subtest and pollutes counts. Each subtest owns its own webhook from now on.

- [ ] **Step 1: Remove the outer webhook setup loop**

Delete the for-loop at lines 180–193 of `webhook_test.go` (the block that calls `AddWebhook` for `v1pb.Activity_ISSUE_CREATED` on `ctl.project`).

- [ ] **Step 2: Migrate `IssueWithPlanWebhookPayload` to a per-subtest webhook**

Inside the subtest, after `collector.reset()`:

```go
project := ctl.createTestProject(ctx, t, "byt9398-i1")
addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
    v1pb.Activity_ISSUE_CREATED,
})
```

Replace later uses of `ctl.project.Name` inside this subtest with `project.Name`. Keep the existing assertions but switch them to use `matchesEventTitle` and project-scoped helpers (Task 9 details).

- [ ] **Step 3: Run the existing subtest to make sure the migration didn't break it**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/IssueWithPlanWebhookPayload$" -timeout 5m`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/webhook_test.go
git commit -m "test(webhook): drop leaky outer webhook setup; subtests own their webhooks"
```

---

### Task 3: Create the helpers file with universal helpers

**Files:**
- Create: `backend/tests/webhook_helpers_test.go`

This is the single home for cross-subtest helpers. Subsequent tasks add to it.

- [ ] **Step 1: File header and event-title helpers**

```go
package tests

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "path"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "connectrpc.com/connect"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
    "google.golang.org/genproto/googleapis/type/expr"
    "google.golang.org/protobuf/types/known/fieldmaskpb"

    _ "modernc.org/sqlite"

    v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// firstSlackSectionText returns the text of the first Slack section block in the
// captured payload, or "" if none. attachments[0].blocks[0].text.text holds the
// event title decorated with emoji and Slack <link|title> markup — see
// backend/plugin/webhook/slack/slack_test.go:42-47, 85.
func firstSlackSectionText(body []byte) string {
    var payload map[string]any
    if err := json.Unmarshal(body, &payload); err != nil {
        return ""
    }
    attachments, ok := payload["attachments"].([]any)
    if !ok || len(attachments) == 0 {
        return ""
    }
    att, ok := attachments[0].(map[string]any)
    if !ok {
        return ""
    }
    blocks, ok := att["blocks"].([]any)
    if !ok || len(blocks) == 0 {
        return ""
    }
    block0, ok := blocks[0].(map[string]any)
    if !ok || block0["type"] != "section" {
        return ""
    }
    textMap, ok := block0["text"].(map[string]any)
    if !ok {
        return ""
    }
    text, _ := textMap["text"].(string)
    return text
}

// matchesEvent reports whether a captured Slack payload matches the given
// project and event title. Project filtering prevents cross-subtest
// contamination from the async webhook dispatcher.
func matchesEvent(req webhookRequest, projectName, eventTitle string) bool {
    body := string(req.Body)
    if !strings.Contains(body, projectName) {
        return false
    }
    return strings.Contains(firstSlackSectionText(req.Body), eventTitle)
}

// waitForWebhookCount blocks until at least n webhooks for (project, eventTitle)
// have arrived, or fails the test.
func waitForWebhookCount(t *testing.T, c *webhookCollector, projectName, eventTitle string, n int, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for {
        count := countWebhooksFor(c, projectName, eventTitle)
        if count >= n {
            return
        }
        if time.Now().After(deadline) {
            t.Fatalf("timed out waiting for %d %q webhooks on %s; got %d after %s",
                n, eventTitle, projectName, count, timeout)
        }
        time.Sleep(100 * time.Millisecond)
    }
}

// requireWebhookCount asserts the exact count of webhooks for (project, eventTitle).
func requireWebhookCount(t *testing.T, c *webhookCollector, projectName, eventTitle string, n int) {
    t.Helper()
    got := countWebhooksFor(c, projectName, eventTitle)
    require.Equalf(t, n, got, "expected %d %q webhooks on %s, got %d", n, eventTitle, projectName, got)
}

func countWebhooksFor(c *webhookCollector, projectName, eventTitle string) int {
    n := 0
    for _, req := range c.getRequests() {
        if matchesEvent(req, projectName, eventTitle) {
            n++
        }
    }
    return n
}
```

- [ ] **Step 2: Project, sheet, webhook helpers**

```go
// createTestProject creates a fresh project (default AllowSelfApproval=true;
// approval-flow subtests call disableSelfApproval).
func (ctl *controller) createTestProject(ctx context.Context, t *testing.T, prefix string) *v1pb.Project {
    t.Helper()
    pid := generateRandomString(prefix)
    resp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
        ProjectId: pid,
        Project: &v1pb.Project{
            Name:              fmt.Sprintf("projects/%s", pid),
            Title:             prefix,
            AllowSelfApproval: true,
        },
    }))
    require.NoError(t, err)
    return resp.Msg
}

// addWebhookForEvents adds a Slack webhook to the project subscribed to the
// given event types only.
func addWebhookForEvents(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, url string, events []v1pb.Activity_Type) {
    t.Helper()
    _, err := ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
        Project: project.Name,
        Webhook: &v1pb.Webhook{
            Type:              v1pb.WebhookType_SLACK,
            Title:             "test-webhook-" + generateRandomString("hook"),
            Url:               url,
            NotificationTypes: events,
        },
    }))
    require.NoError(t, err)
}

// dbTargetName returns the canonical instance/databases/<name> resource form.
func dbTargetName(instance *v1pb.Instance, dbName string) string {
    return fmt.Sprintf("%s/databases/%s", instance.Name, dbName)
}

// seedPassingSheet creates a sheet whose SQL trivially succeeds.
func seedPassingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) string {
    t.Helper()
    resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
        Parent: project.Name,
        Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
    }))
    require.NoError(t, err)
    return resp.Msg.Name
}

// seedFailingSheet creates a sheet whose SQL fails until unblockFailingTask runs.
func seedFailingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) string {
    t.Helper()
    resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
        Parent: project.Name,
        Sheet:  &v1pb.Sheet{Content: []byte("INSERT INTO __force_fail_target VALUES(1);")},
    }))
    require.NoError(t, err)
    return resp.Msg.Name
}

// unblockFailingTask creates the missing table inside the SQLite database file
// so subsequent runs of seedFailingSheet's SQL succeed. db.Close() flushes
// WAL/journal so the file is consistent before the retry is enqueued.
func unblockFailingTask(t *testing.T, instanceDir, dbName string) {
    t.Helper()
    dbPath := filepath.Join(instanceDir, dbName+".db")
    db, err := sql.Open("sqlite", dbPath)
    require.NoError(t, err)
    defer db.Close()
    _, err = db.Exec("CREATE TABLE IF NOT EXISTS __force_fail_target(id INT);")
    require.NoError(t, err)
}
```

- [ ] **Step 3: Plan, rollout, run/skip/retry helpers (using `task.Status`)**

```go
type taskSpec struct {
    sheetName string
    dbTarget  string // full instance/databases/<name>
}

// createPlanWithSpecs creates a plan with one ChangeDatabaseConfig spec per entry.
func createPlanWithSpecs(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, specs []taskSpec) *v1pb.Plan {
    t.Helper()
    var planSpecs []*v1pb.Plan_Spec
    for _, s := range specs {
        planSpecs = append(planSpecs, &v1pb.Plan_Spec{
            Id: uuid.NewString(),
            Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
                ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
                    Targets: []string{s.dbTarget},
                    Sheet:   s.sheetName,
                },
            },
        })
    }
    resp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
        Parent: project.Name,
        Plan:   &v1pb.Plan{Specs: planSpecs},
    }))
    require.NoError(t, err)
    return resp.Msg
}

func createRolloutOnly(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) *v1pb.Rollout {
    t.Helper()
    resp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Name}))
    require.NoError(t, err)
    return resp.Msg
}

// runAllTasks materializes the rollout and starts every task in every stage.
func runAllTasks(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) *v1pb.Rollout {
    t.Helper()
    rollout := createRolloutOnly(ctx, t, ctl, plan)
    for _, stage := range rollout.Stages {
        var taskNames []string
        for _, task := range stage.Tasks {
            taskNames = append(taskNames, task.Name)
        }
        if len(taskNames) == 0 {
            continue
        }
        _, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
            Parent: stage.Name,
            Tasks:  taskNames,
        }))
        require.NoError(t, err)
    }
    return rollout
}

func refreshRollout(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) *v1pb.Rollout {
    t.Helper()
    resp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{Name: rollout.Name}))
    require.NoError(t, err)
    return resp.Msg
}

// findTaskByDB scans the rollout for a task targeting dbResource.
func findTaskByDB(t *testing.T, rollout *v1pb.Rollout, dbResource string) (stageName, taskName string) {
    t.Helper()
    for _, stage := range rollout.Stages {
        for _, task := range stage.Tasks {
            if task.Target == dbResource {
                return stage.Name, task.Name
            }
        }
    }
    t.Fatalf("no task targets database %s in rollout %s", dbResource, rollout.Name)
    return "", ""
}

func runTaskByDB(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string) {
    t.Helper()
    stageName, taskName := findTaskByDB(t, rollout, dbResource)
    _, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
        Parent: stageName,
        Tasks:  []string{taskName},
    }))
    require.NoError(t, err)
}

func skipTaskByDB(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string) {
    t.Helper()
    stageName, taskName := findTaskByDB(t, rollout, dbResource)
    _, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
        Parent: stageName,
        Tasks:  []string{taskName},
        Reason: "test: skip task by db",
    }))
    require.NoError(t, err)
}

// skipFailedTasks finds every FAILED task in the rollout and skips them.
func skipFailedTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
    t.Helper()
    fresh := refreshRollout(ctx, t, ctl, rollout)
    perStage := map[string][]string{}
    for _, stage := range fresh.Stages {
        for _, task := range stage.Tasks {
            if task.Status == v1pb.Task_FAILED {
                perStage[stage.Name] = append(perStage[stage.Name], task.Name)
            }
        }
    }
    require.NotEmpty(t, perStage, "expected at least one failed task to skip")
    for stageName, names := range perStage {
        _, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
            Parent: stageName,
            Tasks:  names,
            Reason: "test: skip failed tasks",
        }))
        require.NoError(t, err)
    }
}

func skipAllTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
    t.Helper()
    fresh := refreshRollout(ctx, t, ctl, rollout)
    for _, stage := range fresh.Stages {
        var names []string
        for _, task := range stage.Tasks {
            names = append(names, task.Name)
        }
        if len(names) == 0 {
            continue
        }
        _, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
            Parent: stage.Name,
            Tasks:  names,
            Reason: "test: skip all",
        }))
        require.NoError(t, err)
    }
}

// retryFailedTasks reruns BatchRunTasks on every FAILED task. Caller decides
// whether to call unblockFailingTask first (to make the retry pass) or not
// (to test that BatchRunTasks resets the dedup row so a second
// PIPELINE_FAILED can fire).
func retryFailedTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
    t.Helper()
    fresh := refreshRollout(ctx, t, ctl, rollout)
    perStage := map[string][]string{}
    for _, stage := range fresh.Stages {
        for _, task := range stage.Tasks {
            if task.Status == v1pb.Task_FAILED {
                perStage[stage.Name] = append(perStage[stage.Name], task.Name)
            }
        }
    }
    require.NotEmpty(t, perStage)
    for stageName, names := range perStage {
        _, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
            Parent: stageName,
            Tasks:  names,
        }))
        require.NoError(t, err)
    }
}

// waitForTaskStatus polls the rollout until the task targeting dbResource has
// the requested status, or fails the test.
func waitForTaskStatus(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string, want v1pb.Task_Status, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for {
        fresh := refreshRollout(ctx, t, ctl, rollout)
        for _, stage := range fresh.Stages {
            for _, task := range stage.Tasks {
                if task.Target != dbResource {
                    continue
                }
                if task.Status == want {
                    return
                }
            }
        }
        if time.Now().After(deadline) {
            t.Fatalf("timed out waiting for task on %s to reach %s", dbResource, want)
        }
        time.Sleep(200 * time.Millisecond)
    }
}

// waitForAllTasksTerminal polls until every task in the rollout is in a
// terminal status (DONE, FAILED, SKIPPED, or CANCELED). Used before
// "skip remaining" actions to avoid races with still-running tasks. On
// timeout, dumps per-task status to t.Logf so failures are debuggable
// without re-running.
func waitForAllTasksTerminal(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    isTerminal := func(s v1pb.Task_Status) bool {
        return s == v1pb.Task_DONE || s == v1pb.Task_FAILED || s == v1pb.Task_SKIPPED || s == v1pb.Task_CANCELED
    }
    var fresh *v1pb.Rollout
    for {
        fresh = refreshRollout(ctx, t, ctl, rollout)
        allTerminal := true
        for _, stage := range fresh.Stages {
            for _, task := range stage.Tasks {
                if !isTerminal(task.Status) {
                    allTerminal = false
                }
            }
        }
        if allTerminal {
            return
        }
        if time.Now().After(deadline) {
            for _, stage := range fresh.Stages {
                for _, task := range stage.Tasks {
                    t.Logf("non-terminal task: %s status=%s target=%s", task.Name, task.Status, task.Target)
                }
            }
            t.Fatalf("timed out waiting for all tasks terminal on rollout %s after %s", rollout.Name, timeout)
        }
        time.Sleep(200 * time.Millisecond)
    }
}
```

- [ ] **Step 4: Compile-check**

Run: `go build ./backend/tests/...`
Expected: success.

- [ ] **Step 5: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): add helpers for trigger matrix"
```

---

### Task 4: BYT-9398 regression test (cell C4)

**Files:**
- Modify: `backend/tests/webhook_test.go`

- [ ] **Step 1: Write the regression subtest**

Place after the migrated `IssueWithPlanWebhookPayload` subtest. The closure captures `instance` (line 177) and `instanceDir` (line 163) from the outer test body.

```go
t.Run("PipelineCompletedAfterSkippingFailedTask", func(t *testing.T) {
    collector.reset()

    project := ctl.createTestProject(ctx, t, "byt9398-c4")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED,
        v1pb.Activity_PIPELINE_COMPLETED,
    })

    err := ctl.createDatabase(ctx, project, instance, nil, "byt9398_c4_pass", "")
    require.NoError(t, err)
    err = ctl.createDatabase(ctx, project, instance, nil, "byt9398_c4_fail", "")
    require.NoError(t, err)

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c4_pass")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c4_fail")},
    })
    rollout := runAllTasks(ctx, t, ctl, plan)

    // Phase 1: failing task → exactly one PIPELINE_FAILED, no PIPELINE_COMPLETED.
    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

    // Phase 2: skip the failed task → PIPELINE_COMPLETED fires (the fix).
    skipFailedTasks(ctx, t, ctl, rollout)
    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
})
```

- [ ] **Step 2: Run the new subtest**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompletedAfterSkippingFailedTask$" -timeout 5m`
Expected: PASS.

- [ ] **Step 3: Run the whole parent**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$" -timeout 5m`
Expected: every subtest passes (the migrated `IssueWithPlanWebhookPayload` plus the new regression).

- [ ] **Step 4: Format, lint, commit**

```bash
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): add BYT-9398 regression for PIPELINE_COMPLETED after skip"
```

After Chunk 1 the bug is fixed and locked in by a regression test.

---

## Chunk 2: PIPELINE_COMPLETED Matrix (C1, C2, C3, C5, C6, C7)

All helpers are in place. This chunk only adds subtests.

### Task 5: C1, C2, C5 (no-failure paths)

**Files:** `backend/tests/webhook_test.go`

- [ ] **Step 1: `PipelineCompleted_AllTasksDone` (C1)**

```go
t.Run("PipelineCompleted_AllTasksDone", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c1")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c1_a", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c1_b", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c1_a")},
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c1_b")},
    })
    runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
})
```

- [ ] **Step 2: `PipelineCompleted_DoneAndSkipped` (C2)**

```go
t.Run("PipelineCompleted_DoneAndSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c2_a", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c2_b", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c2_a")},
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c2_b")},
    })
    rollout := createRolloutOnly(ctx, t, ctl, plan)
    skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c2_b"))
    runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c2_a"))

    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
})
```

- [ ] **Step 3: `PipelineCompleted_AllSkipped` (C5)**

```go
t.Run("PipelineCompleted_AllSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c5")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c5_a", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c5_b", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c5_a")},
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c5_b")},
    })
    rollout := createRolloutOnly(ctx, t, ctl, plan)
    skipAllTasks(ctx, t, ctl, rollout)

    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
})
```

- [ ] **Step 4: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompleted_(AllTasksDone|DoneAndSkipped|AllSkipped)$" -timeout 5m
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover PIPELINE_COMPLETED no-failure paths (C1, C2, C5)"
```

---

### Task 6: C3, C6, C7 (recovery paths)

**Files:** `backend/tests/webhook_test.go`

Each subtest waits for all tasks to reach terminal status before driving recovery, eliminating the "still-running task collides with skip" race.

- [ ] **Step 1: `PipelineCompleted_DoneAndFailedThenRetriedDone` (C3)**

```go
t.Run("PipelineCompleted_DoneAndFailedThenRetriedDone", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c3")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c3_pass", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c3_fail", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c3_pass")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c3_fail")},
    })
    rollout := runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

    unblockFailingTask(t, instanceDir, "byt9398_c3_fail")
    retryFailedTasks(ctx, t, ctl, rollout)

    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
})
```

- [ ] **Step 2: `PipelineCompleted_AllFailedThenAllSkipped` (C6)**

```go
t.Run("PipelineCompleted_AllFailedThenAllSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c6")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c6_a", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c6_b", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c6_a")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c6_b")},
    })
    rollout := runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

    skipAllTasks(ctx, t, ctl, rollout)
    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
})
```

- [ ] **Step 3: `PipelineCompleted_MixedRecovery` (C7)**

```go
t.Run("PipelineCompleted_MixedRecovery", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-c7")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    for _, n := range []string{"byt9398_c7_done", "byt9398_c7_skip", "byt9398_c7_retry", "byt9398_c7_skipfailed"} {
        require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, n, ""))
    }
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_done")},
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_skip")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_retry")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_skipfailed")},
    })

    rollout := createRolloutOnly(ctx, t, ctl, plan)
    skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skip"))
    runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_done"))
    runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"))
    runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skipfailed"))

    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 60*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

    // Unblock dbRetry only — dbSkipFailed's __force_fail_target table remains
    // absent in its own .db file, so its retry would still fail.
    unblockFailingTask(t, instanceDir, "byt9398_c7_retry")
    runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"))
    waitForTaskStatus(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"), v1pb.Task_DONE, 30*time.Second)
    skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skipfailed"))

    waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
})
```

- [ ] **Step 4: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompleted_(DoneAndFailedThenRetriedDone|AllFailedThenAllSkipped|MixedRecovery)$" -timeout 8m
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover PIPELINE_COMPLETED recovery paths (C3, C6, C7)"
```

---

## Chunk 3: PIPELINE_FAILED Matrix (F1, F2, F3, F4)

### Task 7: F1, F2, F3 (per-plan failure cases)

**Files:** `backend/tests/webhook_test.go`

- [ ] **Step 1: `PipelineFailed_SingleTaskFails` (F1)**

```go
t.Run("PipelineFailed_SingleTaskFails", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-f1")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f1_fail", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f1_fail")},
    })
    runAllTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
})
```

- [ ] **Step 2: `PipelineFailed_DedupOnSecondTaskFailure` (F2)**

The framing is "PK dedup", not "simultaneous". With both tasks in the same plan, the test verifies that the second task's failure-claim does not insert a duplicate webhook regardless of execution ordering.

```go
t.Run("PipelineFailed_DedupOnSecondTaskFailure", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-f2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f2_a", ""))
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f2_b", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f2_a")},
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f2_b")},
    })
    rollout := runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

    // Both tasks have failed. ClaimPipelineFailureNotification's PK collision
    // must dedupe — assert exactly 1 even after both terminal.
    requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
})
```

- [ ] **Step 3: `PipelineFailed_RetryFailsAgain` (F3)**

```go
t.Run("PipelineFailed_RetryFailsAgain", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-f3")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f3_fail", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f3_fail")},
    })
    rollout := runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
    waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

    // BatchRunTasks performs the reset SYNCHRONOUSLY (it's a single SQL DELETE
    // before the call returns), then enqueues the new task run. The second
    // failure must therefore see a cleared dedup row and re-fire PIPELINE_FAILED.
    // We deliberately do NOT call unblockFailingTask — the retry should fail again.
    retryFailedTasks(ctx, t, ctl, rollout)
    waitForWebhookCount(t, collector, project.Name, "Rollout failed", 2, 30*time.Second)
})
```

- [ ] **Step 4: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineFailed_(SingleTaskFails|DedupOnSecondTaskFailure|RetryFailsAgain)$" -timeout 5m
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover PIPELINE_FAILED dedup and retry paths (F1, F2, F3)"
```

---

### Task 8: F4 (HA license breach) — document and skip

**Files:** `backend/tests/webhook_test.go`

Three blockers (verified): the canned license JWT is HA, no replica-heartbeat seeding exists, and `haFailGracePeriod` is a hardcoded const (`backend/runner/taskrun/scheduler.go:26`) not exposed via `Profile`. Out of scope for BYT-9398.

- [ ] **Step 1: Skipped subtest with rationale**

```go
t.Run("PipelineFailed_HALicenseBreach", func(t *testing.T) {
    t.Skip("HA license-breach path requires a non-HA license JWT, replica-heartbeat " +
        "seeding, and an injectable haFailGracePeriod — none exist in the test harness. " +
        "Codepath at backend/runner/taskrun/scheduler.go:71-142 is manually verified. " +
        "Tracked as a follow-up.")
})
```

- [ ] **Step 2: Format, lint, commit**

```bash
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): document HA-license PIPELINE_FAILED gap as deferred (F4)"
```

---

## Chunk 4: Issue Event Matrix

### Task 9: Add approval-flow helpers

**Files:** Modify `backend/tests/webhook_helpers_test.go`

- [ ] **Step 1: `installWorkspaceApprovalRule` with cleanup**

```go
// installWorkspaceApprovalRule writes a WORKSPACE_APPROVAL setting rule that
// scopes to the given project via CEL and registers a cleanup that restores
// the prior workspace setting. flowRoles is the ordered list of roles for
// sequential approval steps (one role per step).
func installWorkspaceApprovalRule(ctx context.Context, t *testing.T, ctl *controller, projectID string, flowRoles []string) {
    t.Helper()

    // Snapshot the current setting so we can restore it on cleanup.
    var prior *v1pb.WorkspaceApprovalSetting
    getResp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
        Name: "settings/WORKSPACE_APPROVAL",
    }))
    if err == nil && getResp.Msg.Value.GetWorkspaceApproval() != nil {
        prior = getResp.Msg.Value.GetWorkspaceApproval()
    }

    rules := []*v1pb.WorkspaceApprovalSetting_Rule{{
        Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
        Condition: &expr.Expr{
            Expression: fmt.Sprintf(`resource.project_id == "%s"`, projectID),
        },
        Template: &v1pb.ApprovalTemplate{
            Title: "Test approval flow for " + projectID,
            Flow:  &v1pb.ApprovalFlow{Roles: flowRoles},
        },
    }}
    if prior != nil {
        // Append to existing rules. Subtests run sequentially within the parent
        // TestWebhookIntegration and each cleanup runs before the next subtest
        // starts, so this branch is exercised only when a prior rule existed
        // before the test suite ran (or when the test author wires up multiple
        // installs in the same subtest). Each rule's CEL is project-scoped, so
        // coexistence is safe.
        rules = append(prior.Rules, rules...)
    }

    _, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
        AllowMissing: true,
        Setting: &v1pb.Setting{
            Name: "settings/WORKSPACE_APPROVAL",
            Value: &v1pb.SettingValue{
                Value: &v1pb.SettingValue_WorkspaceApproval{
                    WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{Rules: rules},
                },
            },
        },
    }))
    require.NoError(t, err)

    t.Cleanup(func() {
        // Restore the snapshot.
        restored := &v1pb.WorkspaceApprovalSetting{}
        if prior != nil {
            restored = prior
        }
        _, _ = ctl.settingServiceClient.UpdateSetting(context.Background(), connect.NewRequest(&v1pb.UpdateSettingRequest{
            AllowMissing: true,
            Setting: &v1pb.Setting{
                Name: "settings/WORKSPACE_APPROVAL",
                Value: &v1pb.SettingValue{
                    Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: restored},
                },
            },
        }))
    })
}

func disableSelfApproval(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) {
    t.Helper()
    _, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
        Project: &v1pb.Project{
            Name:              project.Name,
            AllowSelfApproval: false,
        },
        UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_self_approval"}},
    }))
    require.NoError(t, err)
}
```

- [ ] **Step 2: `provisionApprover`, `withImpersonation`**

```go
type testApprover struct {
    Email    string
    Password string
}

func provisionApprover(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, suffix, projectRole string) testApprover {
    t.Helper()
    email := fmt.Sprintf("approver-%s@example.com", suffix)
    password := "1024bytebase"

    newUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
        User: &v1pb.User{Email: email, Password: password, Title: "Approver " + suffix},
    }))
    require.NoError(t, err)

    _, err = ctl.addMemberToWorkspaceIAM(ctx, newUser.Msg.Workspace, fmt.Sprintf("user:%s", email), "roles/workspaceMember")
    require.NoError(t, err)

    policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
        Resource: project.Name,
    }))
    require.NoError(t, err)
    policy := policyResp.Msg
    policy.Bindings = append(policy.Bindings, &v1pb.Binding{
        Role:    projectRole,
        Members: []string{fmt.Sprintf("user:%s", email)},
    })
    _, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
        Resource: project.Name,
        Policy:   policy,
    }))
    require.NoError(t, err)

    return testApprover{Email: email, Password: password}
}

// withImpersonation logs in as the given approver, runs fn with that identity,
// and restores the original token via defer.
func withImpersonation(ctx context.Context, t *testing.T, ctl *controller, who testApprover, fn func()) {
    t.Helper()
    loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
        Email:    who.Email,
        Password: who.Password,
    }))
    require.NoError(t, err)

    original := ctl.authInterceptor.token
    ctl.authInterceptor.token = loginResp.Msg.Token
    defer func() { ctl.authInterceptor.token = original }()

    fn()
}
```

- [ ] **Step 3: Issue helpers (`createIssueForPlan`, `approveIssueAs`, `rejectIssueAs`, `requestIssueAsCreator`, `waitForIssuePending`)**

```go
func createIssueForPlan(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, plan *v1pb.Plan, title string) *v1pb.Issue {
    t.Helper()
    resp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
        Parent: project.Name,
        Issue: &v1pb.Issue{
            Title:       title,
            Description: title + " description",
            Type:        v1pb.Issue_DATABASE_CHANGE,
            Plan:        plan.Name,
        },
    }))
    require.NoError(t, err)
    return resp.Msg
}

func approveIssueAs(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, who testApprover) {
    t.Helper()
    withImpersonation(ctx, t, ctl, who, func() {
        _, err := ctl.issueServiceClient.ApproveIssue(ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{
            Name: issue.Name,
        }))
        require.NoError(t, err)
    })
}

func rejectIssueAs(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, who testApprover, comment string) {
    t.Helper()
    withImpersonation(ctx, t, ctl, who, func() {
        _, err := ctl.issueServiceClient.RejectIssue(ctx, connect.NewRequest(&v1pb.RejectIssueRequest{
            Name:    issue.Name,
            Comment: comment,
        }))
        require.NoError(t, err)
    })
}

// requestIssueAsCreator clears the rejected-approver state on a sent-back issue.
// MUST be called as the issue creator (canRequestIssue at issue_service.go:803-805).
// Run with the default ctl token, which is the demo user used as default creator.
func requestIssueAsCreator(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, comment string) {
    t.Helper()
    _, err := ctl.issueServiceClient.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{
        Name:    issue.Name,
        Comment: comment,
    }))
    require.NoError(t, err)
}

func waitForIssuePending(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for {
        resp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
            Name: issue.Name,
        }))
        require.NoError(t, err)
        if resp.Msg.ApprovalStatus != v1pb.Issue_CHECKING {
            return
        }
        if time.Now().After(deadline) {
            t.Fatalf("issue %s still CHECKING after %s", issue.Name, timeout)
        }
        time.Sleep(500 * time.Millisecond)
    }
}
```

- [ ] **Step 4: Compile, format, lint, commit**

```bash
go build ./backend/tests/...
gofmt -w backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): add approval-flow helpers (impersonation, workspace approval)"
```

---

### Task 10: ISSUE_CREATED enhancement (I1, I2)

**Files:** `backend/tests/webhook_test.go`

- [ ] **Step 1: Update migrated `IssueWithPlanWebhookPayload` (I1)**

After verifying the migration from Task 2, add the spec-required assertions. Read `backend/component/webhook/manager.go:84-92` to pin the exact description text the manager builds (`"%s created issue %s"` — the actor's name plus the issue title). The default `ctl` user creates the issue, so the actor is the demo user.

Add inside the existing subtest after the captured request is parsed:

```go
require.True(t, matchesEvent(req, project.Name, "Issue created"),
    "first webhook section should contain 'Issue created'; got %q",
    firstSlackSectionText(req.Body))

body := string(req.Body)
require.Contains(t, body, project.Name, "payload should reference the project resource name")
require.Contains(t, body, "Test webhook issue", "payload should contain the issue title")
require.Contains(t, body, fmt.Sprintf("/projects/%s/issues/", path.Base(project.Name)),
    "payload should link to the project's issue resource")
```

The spec's "creator and issue type" assertions devolve to the project/title/link-shape checks above — the Slack manager (`backend/component/webhook/manager.go:84-92`) does not emit a literal `DATABASE_CHANGE` string in the payload, and the actor name is derived from the runtime user, which is harder to pin reliably. Title + project + link cover the same invariant.

- [ ] **Step 2: `IssueCreated_NoLeakageFromOtherEvents` (I2)**

```go
t.Run("IssueCreated_NoLeakageFromOtherEvents", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-i2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_CREATED})

    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name), []string{"roles/projectOwner"})
    appr := provisionApprover(ctx, t, ctl, project, "i2", "roles/projectOwner")

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_i2_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_i2_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "I2 issue")

    waitForWebhookCount(t, collector, project.Name, "Issue created", 1, 30*time.Second)
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    // The webhook is NOT subscribed to ISSUE_APPROVED — driving an approval
    // must NOT increment the ISSUE_CREATED count on this project.
    approveIssueAs(ctx, t, ctl, issue, appr)
    time.Sleep(2 * time.Second) // intentional grace; asserting absence of further deliveries
    requireWebhookCount(t, collector, project.Name, "Issue created", 1)
    requireWebhookCount(t, collector, project.Name, "Issue approved", 0)
})
```

- [ ] **Step 3: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/(IssueWithPlanWebhookPayload|IssueCreated_NoLeakageFromOtherEvents)$" -timeout 5m
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): augment ISSUE_CREATED coverage and add isolation check (I1, I2)"
```

---

### Task 11: ISSUE_APPROVAL_REQUESTED (A1, A2)

**Files:** `backend/tests/webhook_test.go`

- [ ] **Step 1: A1**

```go
t.Run("IssueApprovalRequested_FiresWhenRequired", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-a1")
    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name), []string{"roles/projectOwner"})
    _ = provisionApprover(ctx, t, ctl, project, "a1", "roles/projectOwner")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_APPROVAL_REQUESTED})

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_a1_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_a1_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "A1 issue")
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    waitForWebhookCount(t, collector, project.Name, "Approval required", 1, 30*time.Second)
})
```

- [ ] **Step 2: A2**

```go
t.Run("IssueApprovalRequested_NotFiredWhenUnused", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-a2")
    // No approval rule installed for this project. AllowSelfApproval stays true.
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_APPROVAL_REQUESTED})

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_a2_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_a2_db")},
    })
    _ = createIssueForPlan(ctx, t, ctl, project, plan, "A2 issue")

    time.Sleep(3 * time.Second) // intentional grace; asserting absence
    requireWebhookCount(t, collector, project.Name, "Approval required", 0)
})
```

- [ ] **Step 3: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/IssueApprovalRequested_" -timeout 5m
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover ISSUE_APPROVAL_REQUESTED firing rules (A1, A2)"
```

---

### Task 12: ISSUE_APPROVED (AP1, AP2) and ISSUE_SENT_BACK (SB1, SB2)

**Files:** `backend/tests/webhook_test.go`

- [ ] **Step 1: AP1**

```go
t.Run("IssueApproved_SingleStep", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-ap1")
    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name), []string{"roles/projectOwner"})
    appr := provisionApprover(ctx, t, ctl, project, "ap1", "roles/projectOwner")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_APPROVED})

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_ap1_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_ap1_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "AP1 issue")
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    approveIssueAs(ctx, t, ctl, issue, appr)
    waitForWebhookCount(t, collector, project.Name, "Issue approved", 1, 30*time.Second)
})
```

- [ ] **Step 2: AP2 (multi-step)**

`ApprovalFlow.Roles` is an ordered list — each role represents one sequential step. Two distinct project roles, one user per role. `ISSUE_APPROVED` fires only when the overall issue reaches `APPROVED` (`backend/runner/approval/runner.go:1047`).

```go
t.Run("IssueApproved_MultiStepOnlyFiresAtFinal", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-ap2")
    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name),
        []string{"roles/projectDeveloper", "roles/projectOwner"})
    apprStep1 := provisionApprover(ctx, t, ctl, project, "ap2-step1", "roles/projectDeveloper")
    apprStep2 := provisionApprover(ctx, t, ctl, project, "ap2-step2", "roles/projectOwner")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_APPROVED})

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_ap2_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_ap2_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "AP2 issue")
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    approveIssueAs(ctx, t, ctl, issue, apprStep1)
    time.Sleep(2 * time.Second) // grace; asserting no intermediate ISSUE_APPROVED
    requireWebhookCount(t, collector, project.Name, "Issue approved", 0)

    approveIssueAs(ctx, t, ctl, issue, apprStep2)
    waitForWebhookCount(t, collector, project.Name, "Issue approved", 1, 30*time.Second)
})
```

- [ ] **Step 3: SB1**

```go
t.Run("IssueSentBack_FiresOnRejection", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-sb1")
    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name), []string{"roles/projectOwner"})
    appr := provisionApprover(ctx, t, ctl, project, "sb1", "roles/projectOwner")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_ISSUE_SENT_BACK})

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_sb1_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_sb1_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "SB1 issue")
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    rejectIssueAs(ctx, t, ctl, issue, appr, "needs more context")
    waitForWebhookCount(t, collector, project.Name, "Issue sent back", 1, 30*time.Second)
})
```

- [ ] **Step 4: SB2 (round-trip via `RequestIssue`)**

`RejectIssue` followed directly by `ApproveIssue` returns `InvalidArgument`. The issue creator must call `RequestIssue` to clear the rejected approver before the approver can re-approve. Default `ctl` token is the issue creator.

```go
t.Run("IssueSentBack_ThenReapproved_E2E", func(t *testing.T) {
    collector.reset()
    project := ctl.createTestProject(ctx, t, "byt9398-sb2")
    disableSelfApproval(ctx, t, ctl, project)
    installWorkspaceApprovalRule(ctx, t, ctl, path.Base(project.Name), []string{"roles/projectOwner"})
    appr := provisionApprover(ctx, t, ctl, project, "sb2", "roles/projectOwner")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_SENT_BACK,
        v1pb.Activity_ISSUE_APPROVED,
        v1pb.Activity_ISSUE_APPROVAL_REQUESTED, // RequestIssue re-fires this
    })

    require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_sb2_db", ""))
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_sb2_db")},
    })
    issue := createIssueForPlan(ctx, t, ctl, project, plan, "SB2 issue")
    waitForIssuePending(ctx, t, ctl, issue, 30*time.Second)

    rejectIssueAs(ctx, t, ctl, issue, appr, "fix and resubmit")
    waitForWebhookCount(t, collector, project.Name, "Issue sent back", 1, 30*time.Second)

    // Issue creator (default ctl token) re-requests approval. Note: RequestIssue
    // does NOT reset ApprovalFindingDone, so waitForIssuePending would return
    // immediately. Instead, wait for the second "Approval required" webhook —
    // RequestIssue calls approval.NotifyApprovalRequested directly
    // (issue_service.go:826), so a second approval-required event is the
    // observable signal that the request side completed.
    requestIssueAsCreator(ctx, t, ctl, issue, "addressed feedback")
    waitForWebhookCount(t, collector, project.Name, "Approval required", 2, 30*time.Second)

    approveIssueAs(ctx, t, ctl, issue, appr)
    waitForWebhookCount(t, collector, project.Name, "Issue approved", 1, 30*time.Second)
})
```

- [ ] **Step 5: Run, full-suite check, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/(IssueApproved_|IssueSentBack_)" -timeout 8m

# Final sanity: every subtest passes when run together.
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$" -timeout 15m

gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): cover ISSUE_APPROVED multi-step and ISSUE_SENT_BACK round-trip (AP1, AP2, SB1, SB2)"
```

---

## Final Verification

- [ ] **Run the entire backend test package**

Run: `go test -count=1 ./backend/tests/... -timeout 30m`
Expected: pass.

- [ ] **Pre-PR checklist walkthrough**

Walk through `docs/pre-pr-checklist.md` per `AGENTS.md`.

- [ ] **Open the PR**

Title: `fix(rollout): emit PIPELINE_COMPLETED after BatchSkipTasks resolves a failure (BYT-9398)`

Body:
```markdown
## Summary
- Fixes BYT-9398: PIPELINE_COMPLETED was silently dropped after a user used Skip to resolve a failed task, because the stale PIPELINE_FAILED row in plan_webhook_delivery blocked the completion claim.
- Tightens `Store.ResetPlanWebhookDelivery` to only delete `PIPELINE_FAILED` rows, so a recovery endpoint can never wipe a terminal `PIPELINE_COMPLETED` row.
- Adds the reset call to `BatchSkipTasks`, mirroring `BatchRunTasks`.
- Adds a comprehensive integration test matrix in `backend/tests/webhook_test.go` covering all six webhook trigger event types.

## Test plan
- [x] BYT-9398 regression (cell C4) covered end-to-end
- [x] PIPELINE_COMPLETED matrix C1–C7
- [x] PIPELINE_FAILED matrix F1–F3 (F4 deferred via documenting `t.Skip`)
- [x] Issue events I1–I2, A1–A2, AP1–AP2, SB1–SB2
- [x] `go test -count=1 ./backend/tests/...` passes locally

## Known limitations (called out in the spec)
- Skip on a still-running task can race with the failure claim; pre-existing behavior, not addressed by this PR. See spec section 6.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

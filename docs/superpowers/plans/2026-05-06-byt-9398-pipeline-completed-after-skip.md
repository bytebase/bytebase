# BYT-9398 PIPELINE_COMPLETED After Skip — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix BYT-9398 (`PIPELINE_COMPLETED` webhook silently dropped after `BatchSkipTasks` resolves a failed task) and add comprehensive integration test coverage for all six webhook trigger event types.

**Architecture:** Single one-line code fix in `backend/api/v1/rollout_service.go` mirroring the existing `BatchRunTasks` pattern. Test coverage extends the existing `backend/tests/webhook_test.go` infrastructure (HTTP-test-server collector + Slack payload parser) with a per-trigger subtest matrix. Tests assert observable webhook deliveries via the HTTP collector, not DB rows, to keep them aligned with what customers actually see.

**Tech Stack:** Go 1.22+, Connect-RPC, gRPC services exposed via `ctl.rolloutServiceClient` / `planServiceClient` / `issueServiceClient` / `projectServiceClient`, SQLite test instance, `httptest.NewServer` for webhook collection, `testify/require` assertions.

**Spec:** `docs/superpowers/specs/2026-05-06-byt-9398-pipeline-completed-after-skip-design.md`

---

## File Structure

| Path | Action | Responsibility |
|------|--------|----------------|
| `backend/api/v1/rollout_service.go` | Modify (~line 886) | Add `ResetPlanWebhookDelivery` call inside `BatchSkipTasks` |
| `backend/tests/webhook_test.go` | Modify | Extend with helpers + per-trigger subtest matrix |
| `backend/tests/webhook_helpers_test.go` | Create | Test-local helpers: `waitForWebhookCount`, `webhookEventTitle`, `seedFailingSheet`, `seedPassingSheet`, `unblockFailingTask` |

The helpers live in their own file so the `webhook_test.go` body stays readable. `webhook_helpers_test.go` is a `_test.go` file in package `tests`, so it has full access to `controller`, `webhookCollector`, etc.

---

## Cross-Cutting Conventions Used Throughout

**Webhook subscription pattern:** every subtest creates a fresh project and adds one webhook subscribed to *only* the event types it cares about. This avoids cross-event leakage in the shared collector.

**Event identification:** the existing `parseSlackWebhook` returns body sections; the new `webhookEventTitle` helper returns the first section's text, which is the event title set by `backend/component/webhook/manager.go` (e.g., `"Issue created"`, `"Rollout failed"`, `"Rollout completed"`, `"Issue approved"`, `"Issue sent back"`, `"Approval required"`).

**Wait pattern:** `waitForWebhookCount(t, collector, eventTitle, n, timeout)` polls every 100ms; replaces the brittle `time.Sleep(5*time.Second)` in the existing test.

**Force-fail technique:** the `seedFailingSheet` helper writes SQL that references a missing table/column (`INSERT INTO __force_fail_target VALUES(1)`). The `unblockFailingTask` helper executes `CREATE TABLE __force_fail_target(id INT)` against the SQLite instance file out-of-band, after which a `BatchRunTasks` retry passes. This avoids needing to mutate sheet content (which wouldn't help — tasks capture `sheetSha256`).

**Commit cadence:** each task ends with a commit. Avoid amending; if a hook fails, fix and create a new commit.

---

## Chunk 1: The Fix and BYT-9398 Regression Test

This chunk closes the bug and proves it stays closed. After this chunk merges, the customer issue is resolved.

### Task 1: Add `ResetPlanWebhookDelivery` call to `BatchSkipTasks`

**Files:**
- Modify: `backend/api/v1/rollout_service.go` (insert after line 885)

- [ ] **Step 1: Read the context window of the existing `BatchSkipTasks` function**

Open `backend/api/v1/rollout_service.go` and locate `BatchSkipTasks` (~line 858). Find the `if plan == nil` block (~line 883–885) and the `s.store.GetIssue(...)` call (~line 887). The new code goes between them.

- [ ] **Step 2: Insert the reset call**

Apply this edit:

```go
    if plan == nil {
        return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
    }

    // Reset notification state so PIPELINE_COMPLETED can fire after skipping a failed task.
    if err := s.store.ResetPlanWebhookDelivery(ctx, projectID, planID); err != nil {
        slog.Error("failed to reset plan webhook delivery", log.BBError(err))
        // Don't fail the request - notification is non-critical
    }

    issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
```

The block is character-identical to the one in `BatchRunTasks` at lines 744–748 except for the comment, which names the specific scenario.

- [ ] **Step 3: Verify the file compiles**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: build succeeds with no errors.

- [ ] **Step 4: Run the linter**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/v1/...`
Repeat the command until it reports no issues (the linter has a max-issues cap and may not surface every issue in one run).
Expected: no lint issues.

- [ ] **Step 5: Commit**

```bash
git add backend/api/v1/rollout_service.go
git commit -m "$(cat <<'EOF'
fix(rollout): reset plan webhook delivery on BatchSkipTasks

When a user skipped a failed task to recover a rollout, the stale
PIPELINE_FAILED row in plan_webhook_delivery blocked the subsequent
PIPELINE_COMPLETED claim and the webhook was silently dropped.

BatchRunTasks already resets the row; mirror the same pattern in
BatchSkipTasks so the recovery-via-skip path also fires
PIPELINE_COMPLETED.

Fixes BYT-9398.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Add the BYT-9398 regression test (cell C4)

**Files:**
- Create: `backend/tests/webhook_helpers_test.go`
- Modify: `backend/tests/webhook_test.go`

This task introduces only the helpers needed for cell C4 plus the C4 subtest itself. Helpers needed by later cells are added in Chunk 2.

- [ ] **Step 1: Write the failing regression test (skeleton in `webhook_test.go`)**

Add a new subtest to `TestWebhookIntegration` named `PipelineCompletedAfterSkippingFailedTask`. Place it after the existing `IssueWithPlanWebhookPayload` subtest. The skeleton:

```go
t.Run("PipelineCompletedAfterSkippingFailedTask", func(t *testing.T) {
    collector.reset()

    // Subscribe a fresh webhook to only the two pipeline events.
    project := ctl.createProject(ctx, t, "byt9398-c4")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED,
        v1pb.Activity_PIPELINE_COMPLETED,
    })

    // Two databases on the SQLite instance — one task always passes,
    // one is forced to fail until unblockFailingTask runs.
    dbPass := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c4_pass")
    dbFail := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c4_fail")

    plan := createPlanWithTwoTasks(ctx, t, ctl, project,
        seedPassingSheet(ctx, t, ctl, project, dbPass.Name),
        seedFailingSheet(ctx, t, ctl, project, dbFail.Name),
    )
    runAllTasks(ctx, t, ctl, plan)

    // Phase 1: failing task should drive PIPELINE_FAILED exactly once.
    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout completed", 0)

    // Phase 2: skip the failed task → completion check should fire COMPLETED.
    skipFailedTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1) // still exactly 1 — no spurious re-send
})
```

(Implementations of `createProject`, `createDatabaseInProject`, `createPlanWithTwoTasks`, `runAllTasks`, `skipFailedTasks`, `addWebhookForEvents` follow.)

- [ ] **Step 2: Add minimum-viable helpers in `webhook_helpers_test.go`**

Create `backend/tests/webhook_helpers_test.go`:

```go
package tests

import (
    "context"
    "database/sql"
    "fmt"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "connectrpc.com/connect"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"

    _ "modernc.org/sqlite"

    v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// webhookEventTitle returns the human-readable event title from the first
// section text of a captured Slack payload (e.g. "Rollout completed").
// Empty string means no recognizable event title was present.
func webhookEventTitle(req webhookRequest) string {
    title, _, _ := parseSlackWebhook(req.Body)
    if title != "" {
        return title
    }
    // Fallback: walk all sections and use the first one whose text matches a
    // known event title.
    return extractFirstSectionText(req.Body)
}

// extractFirstSectionText pulls the first Slack section block's text.
// Used as a fallback when the issue-specific parser doesn't apply (e.g.,
// PIPELINE_COMPLETED has no issue tile).
func extractFirstSectionText(body []byte) string {
    // ... walk attachments[0].blocks, return blocks[0].text.text if section ...
    // (Implementation copies the section-walking logic from parseSlackWebhook.)
}

// waitForWebhookCount blocks until the collector has received at least n
// requests whose event title equals the given title, or fails the test.
func waitForWebhookCount(t *testing.T, c *webhookCollector, eventTitle string, n int, timeout time.Duration) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for {
        count := countWebhooksByTitle(c, eventTitle)
        if count >= n {
            return
        }
        if time.Now().After(deadline) {
            t.Fatalf("timed out waiting for %d %q webhooks; got %d after %s",
                n, eventTitle, count, timeout)
        }
        time.Sleep(100 * time.Millisecond)
    }
}

// requireWebhookCount asserts the exact count of webhooks for the given title.
func requireWebhookCount(t *testing.T, c *webhookCollector, eventTitle string, n int) {
    t.Helper()
    got := countWebhooksByTitle(c, eventTitle)
    require.Equalf(t, n, got, "expected %d %q webhooks, got %d", n, eventTitle, got)
}

func countWebhooksByTitle(c *webhookCollector, eventTitle string) int {
    n := 0
    for _, req := range c.getRequests() {
        if webhookEventTitle(req) == eventTitle {
            n++
        }
    }
    return n
}
```

- [ ] **Step 3: Add the project / database / sheet / webhook helpers**

Continue `webhook_helpers_test.go`:

```go
// createProject creates a fresh test project and returns the project resource.
func (ctl *controller) createProject(ctx context.Context, t *testing.T, prefix string) *v1pb.Project {
    t.Helper()
    resp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
        ProjectId: generateRandomString(prefix),
        Project: &v1pb.Project{
            Title: prefix + " test project",
        },
    }))
    require.NoError(t, err)
    return resp.Msg
}

// createDatabaseInProject creates a SQLite database under the given instance,
// transfers it to the given project, and returns the database resource.
func (ctl *controller) createDatabaseInProject(ctx context.Context, t *testing.T, project *v1pb.Project, instance *v1pb.Instance, dbName string) *v1pb.Database {
    t.Helper()
    err := ctl.createDatabase(ctx, project, instance, nil, dbName, "")
    require.NoError(t, err)
    resp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
        Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbName),
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
            Title:             "test-webhook-" + generateRandomString(""),
            Url:               url,
            NotificationTypes: events,
        },
    }))
    require.NoError(t, err)
}

// seedPassingSheet creates a sheet whose SQL trivially succeeds.
func seedPassingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, _ string) string {
    t.Helper()
    resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
        Parent: project.Name,
        Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
    }))
    require.NoError(t, err)
    return resp.Msg.Name
}

// seedFailingSheet creates a sheet whose SQL fails until the target table is
// created via unblockFailingTask.
func seedFailingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, _ string) string {
    t.Helper()
    resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
        Parent: project.Name,
        Sheet:  &v1pb.Sheet{Content: []byte("INSERT INTO __force_fail_target VALUES(1);")},
    }))
    require.NoError(t, err)
    return resp.Msg.Name
}

// unblockFailingTask creates the missing table inside the SQLite instance
// file so subsequent runs of seedFailingSheet's SQL succeed. The instance
// path is the SQLite file directory; the database file is `<dbName>.db`.
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

- [ ] **Step 4: Add plan/run/skip helpers**

Continue `webhook_helpers_test.go`:

```go
// createPlanWithTwoTasks creates a plan with two ChangeDatabaseConfig specs,
// one per (sheet, target) pair, and returns the resulting plan.
func createPlanWithTwoTasks(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, sheetA, sheetB string, dbA, dbB string) *v1pb.Plan {
    t.Helper()
    specs := []*v1pb.Plan_Spec{
        {
            Id: uuid.NewString(),
            Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
                ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
                    Targets: []string{dbA},
                    Sheet:   sheetA,
                },
            },
        },
        {
            Id: uuid.NewString(),
            Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
                ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
                    Targets: []string{dbB},
                    Sheet:   sheetB,
                },
            },
        },
    }
    resp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
        Parent: project.Name,
        Plan:   &v1pb.Plan{Specs: specs},
    }))
    require.NoError(t, err)
    return resp.Msg
}

// runAllTasks creates the rollout for the plan and starts every task.
// Returns the rollout for downstream task-id lookups.
func runAllTasks(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) *v1pb.Rollout {
    t.Helper()
    rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
        Parent: plan.Name,
    }))
    require.NoError(t, err)
    rollout := rolloutResp.Msg
    require.GreaterOrEqual(t, len(rollout.Stages), 1)

    for _, stage := range rollout.Stages {
        var taskNames []string
        for _, task := range stage.Tasks {
            taskNames = append(taskNames, task.Name)
        }
        _, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
            Parent: stage.Name,
            Tasks:  taskNames,
        }))
        require.NoError(t, err)
    }
    return rollout
}

// skipFailedTasks scans the rollout's tasks, identifies which have FAILED
// task runs, and skips them via BatchSkipTasks.
func skipFailedTasks(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) {
    t.Helper()
    // Re-fetch rollout to get latest task statuses.
    rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
        Name: plan.Name + "/rollouts/-",
    }))
    require.NoError(t, err)

    var failedTasks []string
    var stageName string
    for _, stage := range rolloutResp.Msg.Stages {
        for _, task := range stage.Tasks {
            if task.LatestTaskRunStatus == v1pb.TaskRun_FAILED {
                failedTasks = append(failedTasks, task.Name)
                stageName = stage.Name
            }
        }
    }
    require.NotEmpty(t, failedTasks, "expected at least one failed task to skip")

    _, err = ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
        Parent: stageName,
        Tasks:  failedTasks,
        Reason: "test: skipping failed task",
    }))
    require.NoError(t, err)
}
```

(The exact `GetRollout` form may need adjusting — verify the existing rollout-name pattern in `schema_update_test.go`. If the Connect API requires the resolved rollout UID instead of `-`, swap accordingly.)

- [ ] **Step 5: Run only the new subtest**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompletedAfterSkippingFailedTask$" -timeout 5m`
Expected: PASS. The first phase observes one `"Rollout failed"` webhook; the second phase, after `skipFailedTasks`, observes one `"Rollout completed"` webhook with no extra failures.

- [ ] **Step 6: Sanity-check the existing subtest still passes**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$" -timeout 5m`
Expected: both subtests pass.

- [ ] **Step 7: Format and lint**

Run: `gofmt -w backend/tests/webhook_helpers_test.go backend/tests/webhook_test.go`
Run: `golangci-lint run --allow-parallel-runners ./backend/tests/...` (repeat until clean)
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add backend/tests/webhook_helpers_test.go backend/tests/webhook_test.go
git commit -m "$(cat <<'EOF'
test(webhook): add BYT-9398 regression for PIPELINE_COMPLETED after skip

Adds the C4 cell of the webhook trigger matrix: a rollout where one task
fails and is then resolved via BatchSkipTasks must observe both
PIPELINE_FAILED and PIPELINE_COMPLETED webhooks.

Introduces minimum-viable shared helpers (waitForWebhookCount,
seedFailingSheet, unblockFailingTask, etc.) that subsequent webhook
trigger subtests will reuse.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

After Chunk 1, BYT-9398 is fixed and locked in by a regression test. Chunks 2–4 broaden coverage.

---

## Chunk 2: PIPELINE_COMPLETED Matrix (C1, C2, C3, C5, C6, C7)

This chunk fills the rest of the completion-event matrix. Each subtest follows the same pattern as C4 but exercises a different state combination.

### Task 3: Add helpers needed for the remaining cells

**Files:**
- Modify: `backend/tests/webhook_helpers_test.go`

- [ ] **Step 1: Add `skipAllTasks` and `skipSpecificTasks` helpers**

```go
// skipAllTasks skips every task in the plan via BatchSkipTasks.
func skipAllTasks(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) {
    t.Helper()
    rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
        Name: plan.Name + "/rollouts/-",
    }))
    require.NoError(t, err)
    for _, stage := range rolloutResp.Msg.Stages {
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
```

- [ ] **Step 2: Add `retryFailedTasksAfterUnblock` helper**

```go
// retryFailedTasksAfterUnblock re-runs the FAILED tasks via BatchRunTasks.
// Caller must have already invoked unblockFailingTask to make the SQL pass.
func retryFailedTasksAfterUnblock(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) {
    t.Helper()
    rolloutResp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{
        Name: plan.Name + "/rollouts/-",
    }))
    require.NoError(t, err)
    var failed []string
    var stageName string
    for _, stage := range rolloutResp.Msg.Stages {
        for _, task := range stage.Tasks {
            if task.LatestTaskRunStatus == v1pb.TaskRun_FAILED {
                failed = append(failed, task.Name)
                stageName = stage.Name
            }
        }
    }
    require.NotEmpty(t, failed)
    _, err = ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
        Parent: stageName,
        Tasks:  failed,
    }))
    require.NoError(t, err)
}
```

- [ ] **Step 3: Add `createPlanWithSpecs` for variable-task-count cells**

```go
type taskSpec struct {
    sheetName string
    dbName    string
}

func createPlanWithSpecs(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, specs []taskSpec) *v1pb.Plan {
    t.Helper()
    var planSpecs []*v1pb.Plan_Spec
    for _, s := range specs {
        planSpecs = append(planSpecs, &v1pb.Plan_Spec{
            Id: uuid.NewString(),
            Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
                ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
                    Targets: []string{s.dbName},
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
```

- [ ] **Step 4: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): add skip-all / retry helpers for trigger matrix"
```

---

### Task 4: Add subtests C1, C2, C5

**Files:**
- Modify: `backend/tests/webhook_test.go`

These three cells share a structure: no failure ever occurs, one `"Rollout completed"` webhook fires.

- [ ] **Step 1: Write subtest `PipelineCompleted_AllTasksDone` (C1)**

```go
t.Run("PipelineCompleted_AllTasksDone", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c1")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    db1 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c1_a")
    db2 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c1_b")
    sheetA := seedPassingSheet(ctx, t, ctl, project, db1.Name)
    sheetB := seedPassingSheet(ctx, t, ctl, project, db2.Name)
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {sheetA, db1.Name}, {sheetB, db2.Name},
    })
    runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 0)
})
```

- [ ] **Step 2: Write subtest `PipelineCompleted_DoneAndSkipped` (C2)**

```go
t.Run("PipelineCompleted_DoneAndSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    db1 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c2_a")
    db2 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c2_b")
    sheetA := seedPassingSheet(ctx, t, ctl, project, db1.Name)
    sheetB := seedPassingSheet(ctx, t, ctl, project, db2.Name)
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {sheetA, db1.Name}, {sheetB, db2.Name},
    })

    // Skip db2's task; run db1's task. Plan completes via DONE + SKIPPED.
    rollout := createRolloutOnly(ctx, t, ctl, plan)
    skipTaskByDB(ctx, t, ctl, rollout, db2.Name)
    runTaskByDB(ctx, t, ctl, rollout, db1.Name)

    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 0)
})
```

`createRolloutOnly`, `skipTaskByDB`, `runTaskByDB` are new helpers — add them in the same step. They follow the same pattern as the existing helpers but operate on a single targeted task identified by its database name.

- [ ] **Step 3: Write subtest `PipelineCompleted_AllSkipped` (C5)**

```go
t.Run("PipelineCompleted_AllSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c5")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    db1 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c5_a")
    db2 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c5_b")
    sheetA := seedPassingSheet(ctx, t, ctl, project, db1.Name)
    sheetB := seedPassingSheet(ctx, t, ctl, project, db2.Name)
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {sheetA, db1.Name}, {sheetB, db2.Name},
    })
    _ = createRolloutOnly(ctx, t, ctl, plan)
    skipAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 0)
})
```

- [ ] **Step 4: Run the three new subtests**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompleted_(AllTasksDone|DoneAndSkipped|AllSkipped)$" -timeout 5m`
Expected: all three pass.

- [ ] **Step 5: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): cover PIPELINE_COMPLETED no-failure paths (C1, C2, C5)"
```

---

### Task 5: Add subtests C3, C6, C7

**Files:**
- Modify: `backend/tests/webhook_test.go`

These cells use `unblockFailingTask` to fix-then-retry a failing task.

- [ ] **Step 1: Write subtest `PipelineCompleted_DoneAndFailedThenRetriedDone` (C3)**

```go
t.Run("PipelineCompleted_DoneAndFailedThenRetriedDone", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c3")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    dbPass := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c3_pass")
    dbFail := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c3_fail")
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project, dbPass.Name), dbPass.Name},
        {seedFailingSheet(ctx, t, ctl, project, dbFail.Name), dbFail.Name},
    })
    runAllTasks(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout completed", 0)

    // Unblock the failing SQL and retry.
    unblockFailingTask(t, instanceDir, "byt9398_c3_fail")
    retryFailedTasksAfterUnblock(ctx, t, ctl, plan)

    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1)
})
```

- [ ] **Step 2: Write subtest `PipelineCompleted_AllFailedThenAllSkipped` (C6)**

```go
t.Run("PipelineCompleted_AllFailedThenAllSkipped", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c6")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    db1 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c6_a")
    db2 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c6_b")
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project, db1.Name), db1.Name},
        {seedFailingSheet(ctx, t, ctl, project, db2.Name), db2.Name},
    })
    runAllTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)

    skipAllTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1)
})
```

- [ ] **Step 3: Write subtest `PipelineCompleted_MixedRecovery` (C7)**

```go
t.Run("PipelineCompleted_MixedRecovery", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-c7")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
    })
    dbDone := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c7_done")
    dbSkip := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c7_skip")
    dbRetry := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c7_retry")
    dbSkipFailed := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_c7_skipfailed")

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedPassingSheet(ctx, t, ctl, project, dbDone.Name), dbDone.Name},
        {seedPassingSheet(ctx, t, ctl, project, dbSkip.Name), dbSkip.Name},
        {seedFailingSheet(ctx, t, ctl, project, dbRetry.Name), dbRetry.Name},
        {seedFailingSheet(ctx, t, ctl, project, dbSkipFailed.Name), dbSkipFailed.Name},
    })

    // First: skip the always-skipped task, run the rest.
    rollout := createRolloutOnly(ctx, t, ctl, plan)
    skipTaskByDB(ctx, t, ctl, rollout, dbSkip.Name)
    runTaskByDB(ctx, t, ctl, rollout, dbDone.Name)
    runTaskByDB(ctx, t, ctl, rollout, dbRetry.Name)
    runTaskByDB(ctx, t, ctl, rollout, dbSkipFailed.Name)

    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout completed", 0)

    // Second: unblock dbRetry's SQL and retry; skip dbSkipFailed.
    unblockFailingTask(t, instanceDir, "byt9398_c7_retry")
    retryTaskByDB(ctx, t, ctl, plan, dbRetry.Name)
    waitForTaskDone(ctx, t, ctl, plan, dbRetry.Name, 30*time.Second)
    skipTaskByDB(ctx, t, ctl, rollout, dbSkipFailed.Name)

    waitForWebhookCount(t, collector, "Rollout completed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1)
})
```

(`retryTaskByDB`, `waitForTaskDone` are new — add them as needed.)

- [ ] **Step 4: Run new subtests**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineCompleted_(DoneAndFailedThenRetriedDone|AllFailedThenAllSkipped|MixedRecovery)$" -timeout 8m`
Expected: all three pass.

- [ ] **Step 5: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): cover PIPELINE_COMPLETED recovery paths (C3, C6, C7)"
```

---

## Chunk 3: PIPELINE_FAILED Matrix (F1, F2, F3, F4)

### Task 6: F1, F2, F3 (per-plan failure cases)

**Files:**
- Modify: `backend/tests/webhook_test.go`

- [ ] **Step 1: Write subtest `PipelineFailed_SingleTaskFails` (F1)**

```go
t.Run("PipelineFailed_SingleTaskFails", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-f1")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED,
    })
    dbFail := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_f1_fail")
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project, dbFail.Name), dbFail.Name},
    })
    runAllTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1)
})
```

- [ ] **Step 2: Write subtest `PipelineFailed_TwoTasksFailDeduped` (F2)**

```go
t.Run("PipelineFailed_TwoTasksFailDeduped", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-f2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED,
    })
    db1 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_f2_a")
    db2 := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_f2_b")
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project, db1.Name), db1.Name},
        {seedFailingSheet(ctx, t, ctl, project, db2.Name), db2.Name},
    })
    runAllTasks(ctx, t, ctl, plan)

    // Both tasks will fail; the table dedupes — assert exactly one webhook.
    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)
    // Give the second failure a chance to (incorrectly) re-trigger.
    time.Sleep(2 * time.Second)
    requireWebhookCount(t, collector, "Rollout failed", 1)
})
```

- [ ] **Step 3: Write subtest `PipelineFailed_RetryFailsAgain` (F3)**

```go
t.Run("PipelineFailed_RetryFailsAgain", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-f3")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_PIPELINE_FAILED,
    })
    dbFail := ctl.createDatabaseInProject(ctx, t, project, instance, "byt9398_f3_fail")
    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
        {seedFailingSheet(ctx, t, ctl, project, dbFail.Name), dbFail.Name},
    })
    runAllTasks(ctx, t, ctl, plan)
    waitForWebhookCount(t, collector, "Rollout failed", 1, 30*time.Second)

    // BatchRunTasks resets the delivery row; the next failure must fire FAILED again.
    retryFailedTasksAfterUnblock(ctx, t, ctl, plan) // intentionally without unblock — it'll fail again
    waitForWebhookCount(t, collector, "Rollout failed", 2, 30*time.Second)
})
```

- [ ] **Step 4: Run the three new subtests**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/PipelineFailed_(SingleTaskFails|TwoTasksFailDeduped|RetryFailsAgain)$" -timeout 5m`
Expected: all three pass.

- [ ] **Step 5: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover PIPELINE_FAILED dedup and retry paths (F1, F2, F3)"
```

---

### Task 7: F4 (HA license failure path)

**Files:**
- Modify: `backend/tests/webhook_test.go`

The HA-failure path is exercised via `failTaskRunsForHA` in
`backend/runner/taskrun/scheduler.go`. If the test environment cannot
deterministically trigger the HA-license-breach codepath without external
licensing infrastructure, document this limitation explicitly and gate the
subtest behind an environment variable or skip it with `t.Skip` and a
TODO referencing follow-up work.

- [ ] **Step 1: Investigate test feasibility**

Read `backend/runner/taskrun/scheduler.go:71-142` (`failTaskRunsForHA`) and
identify what external state it needs to fire. Look at whether existing
tests under `backend/enterprise/` or `backend/tests/subscription_test.go`
demonstrate a way to drive the HA-limit-exceeded check from a test.

- [ ] **Step 2 (path A — feasible): Write subtest `PipelineFailed_HALicenseBreach`**

If a deterministic trigger exists, add the subtest. Pattern:
- Set up an HA-licensed environment with replica count exceeding the licensed limit.
- Submit a plan that creates pending task runs.
- Wait for `failTaskRunsForHA` to fire.
- Assert exactly one `"Rollout failed"` webhook fires.

- [ ] **Step 2 (path B — infeasible): Skip with documentation**

```go
t.Run("PipelineFailed_HALicenseBreach", func(t *testing.T) {
    t.Skip("Driving HA license breach from an integration test requires harness " +
        "support not yet available; tracked as follow-up. The codepath is " +
        "covered manually and by unit-level checks in backend/enterprise/.")
})
```

Pick whichever path matches reality after Step 1.

- [ ] **Step 3: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go
git commit -m "test(webhook): cover or skip HA-license PIPELINE_FAILED path (F4)"
```

---

## Chunk 4: Issue Event Matrix (CREATED, APPROVAL_REQUESTED, APPROVED, SENT_BACK)

### Task 8: ISSUE_CREATED enhancement (I1, I2)

**Files:**
- Modify: `backend/tests/webhook_test.go`

- [ ] **Step 1: Augment existing `IssueWithPlanWebhookPayload` for I1**

Locate the existing subtest. After the `parseSlackWebhook` block, add:

```go
require.Equal(t, "Issue created", webhookEventTitle(req))
// payload should contain project name and issue type indicator
require.Contains(t, string(req.Body), project.Name)
require.Contains(t, strings.ToLower(string(req.Body)), "database change")
```

(Adjust the issue-type assertion to match what `manager.go` actually emits — may be `"Issue created"` description format.)

- [ ] **Step 2: Add subtest `IssueCreated_NoLeakageFromOtherEvents` (I2)**

```go
t.Run("IssueCreated_NoLeakageFromOtherEvents", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-i2")
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_CREATED,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial passing task */ })
    issue := createIssueForPlan(ctx, t, ctl, project, plan)

    waitForWebhookCount(t, collector, "Issue created", 1, 30*time.Second)

    // Drive an approval (or any non-CREATED event) and confirm the count stays at 1.
    approveIssue(ctx, t, ctl, issue)
    time.Sleep(2 * time.Second)
    requireWebhookCount(t, collector, "Issue created", 1)
    requireWebhookCount(t, collector, "Issue approved", 0) // not subscribed
})
```

(`createIssueForPlan` and `approveIssue` are new helpers — add them, mirroring patterns in `approval_test.go`.)

- [ ] **Step 3: Run the subtests**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/(IssueWithPlanWebhookPayload|IssueCreated_NoLeakageFromOtherEvents)$" -timeout 5m`
Expected: pass.

- [ ] **Step 4: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): augment ISSUE_CREATED coverage and add isolation check (I1, I2)"
```

---

### Task 9: ISSUE_APPROVAL_REQUESTED (A1, A2)

**Files:**
- Modify: `backend/tests/webhook_test.go`

- [ ] **Step 1: Add helpers for approval setup**

```go
// setProjectRequiresApproval enables the require-approval setting on a project.
func setProjectRequiresApproval(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) { ... }

// configureApprovalFlow sets up an approval flow with the given approver users.
func configureApprovalFlow(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, approvers []string) { ... }
```

(Pattern these on existing approval test helpers; if the helpers already exist for `approval_test.go`, hoist or share them rather than duplicating.)

- [ ] **Step 2: Write subtest `IssueApprovalRequested_FiresWhenRequired` (A1)**

```go
t.Run("IssueApprovalRequested_FiresWhenRequired", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-a1")
    setProjectRequiresApproval(ctx, t, ctl, project)
    configureApprovalFlow(ctx, t, ctl, project, []string{"approver@example.com"})
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_APPROVAL_REQUESTED,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial */ })
    _ = createIssueForPlan(ctx, t, ctl, project, plan)

    waitForWebhookCount(t, collector, "Approval required", 1, 30*time.Second)
})
```

- [ ] **Step 3: Write subtest `IssueApprovalRequested_NotFiredWhenUnused` (A2)**

```go
t.Run("IssueApprovalRequested_NotFiredWhenUnused", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-a2")
    // Note: do NOT enable require-approval.
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_APPROVAL_REQUESTED,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial */ })
    _ = createIssueForPlan(ctx, t, ctl, project, plan)

    time.Sleep(3 * time.Second) // give async pipeline time
    requireWebhookCount(t, collector, "Approval required", 0)
})
```

- [ ] **Step 4: Run, format, lint, commit**

```bash
go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/IssueApprovalRequested_" -timeout 5m
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): cover ISSUE_APPROVAL_REQUESTED firing rules (A1, A2)"
```

---

### Task 10: ISSUE_APPROVED (AP1, AP2) and ISSUE_SENT_BACK (SB1, SB2)

**Files:**
- Modify: `backend/tests/webhook_test.go`

- [ ] **Step 1: Write subtest `IssueApproved_SingleStep` (AP1)**

Single approver flow → approve → expect one `"Issue approved"`.

- [ ] **Step 2: Write subtest `IssueApproved_MultiStepOnlyFiresAtFinal` (AP2)**

Two-step flow. After step 1 approval, assert count = 0. After step 2 approval, assert count = 1.

```go
t.Run("IssueApproved_MultiStepOnlyFiresAtFinal", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-ap2")
    setProjectRequiresApproval(ctx, t, ctl, project)
    configureApprovalFlow(ctx, t, ctl, project, []string{"step1@example.com", "step2@example.com"})
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_APPROVED,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial */ })
    issue := createIssueForPlan(ctx, t, ctl, project, plan)

    approveIssueAs(ctx, t, ctl, issue, "step1@example.com")
    time.Sleep(2 * time.Second)
    requireWebhookCount(t, collector, "Issue approved", 0)

    approveIssueAs(ctx, t, ctl, issue, "step2@example.com")
    waitForWebhookCount(t, collector, "Issue approved", 1, 30*time.Second)
})
```

- [ ] **Step 3: Write subtest `IssueSentBack_FiresOnRejection` (SB1)**

```go
t.Run("IssueSentBack_FiresOnRejection", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-sb1")
    setProjectRequiresApproval(ctx, t, ctl, project)
    configureApprovalFlow(ctx, t, ctl, project, []string{"approver@example.com"})
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_SENT_BACK,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial */ })
    issue := createIssueForPlan(ctx, t, ctl, project, plan)
    rejectIssueAs(ctx, t, ctl, issue, "approver@example.com", "needs more context")

    waitForWebhookCount(t, collector, "Issue sent back", 1, 30*time.Second)
})
```

- [ ] **Step 4: Write subtest `IssueSentBack_ThenReapproved_E2E` (SB2)**

```go
t.Run("IssueSentBack_ThenReapproved_E2E", func(t *testing.T) {
    collector.reset()
    project := ctl.createProject(ctx, t, "byt9398-sb2")
    setProjectRequiresApproval(ctx, t, ctl, project)
    configureApprovalFlow(ctx, t, ctl, project, []string{"approver@example.com"})
    addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
        v1pb.Activity_ISSUE_SENT_BACK,
        v1pb.Activity_ISSUE_APPROVED,
    })

    plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{ /* trivial */ })
    issue := createIssueForPlan(ctx, t, ctl, project, plan)
    rejectIssueAs(ctx, t, ctl, issue, "approver@example.com", "fix")
    waitForWebhookCount(t, collector, "Issue sent back", 1, 30*time.Second)

    approveIssueAs(ctx, t, ctl, issue, "approver@example.com")
    waitForWebhookCount(t, collector, "Issue approved", 1, 30*time.Second)
})
```

- [ ] **Step 5: Add `approveIssueAs` and `rejectIssueAs` helpers**

These call the appropriate approval-service endpoints. Mirror the patterns
already in `approval_test.go`; if a shared helper already exists, hoist it
into a `_helpers_test.go` file rather than duplicating.

- [ ] **Step 6: Run all new subtests**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$/(IssueApproved_|IssueSentBack_)" -timeout 8m`
Expected: pass.

- [ ] **Step 7: Run the full test file as a final sanity check**

Run: `go test -v -count=1 ./backend/tests/ -run "^TestWebhookIntegration$" -timeout 15m`
Expected: every subtest passes.

- [ ] **Step 8: Format / lint / commit**

```bash
gofmt -w backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
golangci-lint run --allow-parallel-runners ./backend/tests/...

git add backend/tests/webhook_test.go backend/tests/webhook_helpers_test.go
git commit -m "test(webhook): cover ISSUE_APPROVED multi-step and ISSUE_SENT_BACK round-trip (AP1, AP2, SB1, SB2)"
```

---

## Final Verification

- [ ] **Run the entire backend test suite to confirm no regressions**

Run: `go test -count=1 ./backend/tests/... -timeout 30m`
Expected: pass. If any failures pre-exist on `main`, note them but do not fix in this branch.

- [ ] **Pre-PR checklist walkthrough**

Walk through `docs/pre-pr-checklist.md` per `AGENTS.md`. Pay attention to:
- breaking-change review (none expected)
- composite-PK query safety (no new store methods)
- lint/test gates (run `golangci-lint run --allow-parallel-runners` once more on the whole repo)

- [ ] **Open the PR**

Title: `fix(rollout): emit PIPELINE_COMPLETED after BatchSkipTasks resolves a failure (BYT-9398)`

Body:
```markdown
## Summary
- Fixes BYT-9398: PIPELINE_COMPLETED was silently dropped after a user used Skip to resolve a failed task, because the stale PIPELINE_FAILED row in plan_webhook_delivery blocked the completion claim.
- Mirrors the existing ResetPlanWebhookDelivery call from BatchRunTasks into BatchSkipTasks.
- Adds a comprehensive integration test matrix in backend/tests/webhook_test.go covering all six webhook trigger event types (PIPELINE_COMPLETED, PIPELINE_FAILED, ISSUE_CREATED, ISSUE_APPROVAL_REQUESTED, ISSUE_APPROVED, ISSUE_SENT_BACK) so future regressions are caught at CI time.

## Test plan
- [x] New regression subtest (cell C4) exercises the BYT-9398 scenario end-to-end
- [x] PIPELINE_COMPLETED matrix (C1–C7) covers all paths to the completion event
- [x] PIPELINE_FAILED matrix (F1–F3) locks in the dedup contract; F4 (HA) feasibility documented
- [x] Issue event matrix (I1–I2, A1–A2, AP1–AP2, SB1–SB2) prevents trigger-drop regressions
- [x] `go test -count=1 ./backend/tests/...` passes locally

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

The historical PIPELINE_COMPLETED for plan 2505 cannot be replayed — out of scope for this PR (called out in the spec).

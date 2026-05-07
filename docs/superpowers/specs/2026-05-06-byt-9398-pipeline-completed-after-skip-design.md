# BYT-9398: PIPELINE_COMPLETED never fires after BatchSkipTasks resolves a failure

- Issue: [BYT-9398](https://linear.app/bytebase/issue/BYT-9398)
- Branch: `fix/pipeline-completed`
- Scope: narrow code fix + comprehensive webhook-trigger test coverage

## 1. Problem

A customer running 3.16.1 saw a `PIPELINE_FAILED` webhook for plan 2505 after a
task hit `lock timeout`, then resolved the failure (most likely via the Skip
button), and never received `PIPELINE_COMPLETED`. The downstream workflow that
listens to that event never ran.

## 2. Root cause (verified)

`plan_webhook_delivery` uses primary key `(project, plan_id)` with the invariant
"`PIPELINE_FAILED` and `PIPELINE_COMPLETED` are mutually exclusive — one row per
plan at any time." Recovery requires the row to be cleared, which the store
exposes as `Store.ResetPlanWebhookDelivery`.

Today only `BatchRunTasks` calls `ResetPlanWebhookDelivery` (see
`backend/api/v1/rollout_service.go:744-748`). `BatchSkipTasks`
(`rollout_service.go:858`) signals `PlanCompletionCheckChan` but does not reset
the delivery row, so when the user resolves a failed task by skipping it, the
stale `PIPELINE_FAILED` row blocks the subsequent
`ClaimPipelineCompletionNotification` (the `INSERT … ON CONFLICT DO NOTHING`
returns no row) and the webhook is silently dropped.

## 3. Design gap (per AGENTS.md "trace to design before fixing")

The original design encoded "every recovery endpoint must call
`ResetPlanWebhookDelivery`" as an implicit contract. The complete recovery-path
truth table:

| Endpoint | Can resolve a prior failure into completion? | Calls Reset today? | Should call Reset? |
|---|---|---|---|
| `BatchRunTasks` | yes (re-run failed task) | ✅ | ✅ |
| `BatchSkipTasks` | yes (skip failed task) | ❌ | ✅ ← gap |
| `BatchCancelTaskRuns` | no — only cancels in-flight, user must Run/Skip after | n/a | n/a |
| `runTaskRunOnce` (scheduler) | no — only fires `PIPELINE_FAILED`, never recovers | n/a | n/a |
| `failTaskRunsForHA` (scheduler) | no — only fires `PIPELINE_FAILED`, never recovers | n/a | n/a |

The gap is exactly one cell: `BatchSkipTasks`.

## 4. Fix

Add a `ResetPlanWebhookDelivery` call inside `BatchSkipTasks`, placed
analogously to the existing call in `BatchRunTasks` — i.e. immediately after
the `GetPlan` nil-check and before `GetIssue`. Reusing the existing call's
position, error-handling stance, and comment style:

```go
// rollout_service.go, inside BatchSkipTasks (~line 886, after the plan == nil check)
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

No store-layer, schema, or proto changes.

## 5. Test plan

The existing `backend/tests/webhook_test.go` provides the test infrastructure
(`webhookCollector`, `parseSlackWebhook`, controller bootstrap) but currently
asserts only `ISSUE_CREATED`. We extend that file with a thorough trigger
matrix.

### Helper additions (one-time)

- `waitForWebhookCount(t, collector, n, timeout)` — poll the collector until at
  least `n` requests arrive or fail with a clear diagnostic. Replaces the
  brittle `time.Sleep(5*time.Second)` pattern.
- `webhookEventType(req webhookRequest) string` — extract the event identifier
  from a captured Slack payload so subtests can filter by trigger.
- `seedFailingTask(...)` — helper that sets up a task guaranteed to fail on
  first run (use the SQLite-based "force-fail" technique already proven in
  `task_run_idempotent_test.go`).
- `seedPassingTask(...)` — analogous helper for the success path.

### Subtests by trigger

Each subtest creates a fresh project (or at minimum re-registers a fresh
webhook subscribed to a single event) and calls `collector.reset()` at the top
to keep events isolated.

#### `PIPELINE_COMPLETED` matrix

| # | Scenario | Expected `PIPELINE_FAILED` | Expected `PIPELINE_COMPLETED` |
|---|---|---|---|
| C1 | All tasks DONE on first run | 0 | 1 |
| C2 | Some DONE + some SKIPPED (no failure) | 0 | 1 |
| C3 | Some DONE + some FAILED, retried via `BatchRunTasks` and DONE | 1 | 1 |
| C4 | Some DONE + some FAILED, resolved via `BatchSkipTasks` | 1 | 1 ← **BYT-9398 regression** |
| C5 | All tasks SKIPPED before any run | 0 | 1 |
| C6 | All tasks FAILED, all skipped to resolve | 1 | 1 |
| C7 | Mix: DONE + SKIPPED + (FAILED→retried→DONE) + (FAILED→skipped) | 1 | 1 |

#### `PIPELINE_FAILED` matrix

| # | Scenario | Expected `PIPELINE_FAILED` |
|---|---|---|
| F1 | One task fails on first run | 1 |
| F2 | Two tasks fail simultaneously in the same plan | 1 (the table dedupes — that is the contract) |
| F3 | Task fails → `BatchRunTasks` → fails again | 2 (Reset on `BatchRunTasks` clears the row, second FAILED fires) |
| F4 | HA license breach drives `failTaskRunsForHA` | 1 |

#### `ISSUE_CREATED`

- I1 (existing): `IssueWithPlanWebhookPayload` — keep, augment to assert the
  event identifier matches `ISSUE_CREATED` and the payload contains issue type,
  project, and creator.
- I2 (new): subscribe a webhook to `ISSUE_CREATED` only; trigger an approval
  later and assert the `ISSUE_CREATED` webhook count stays at 1 (no leakage
  from other event types).

#### `ISSUE_APPROVAL_REQUESTED`

- A1: project setting requires approval → create issue → assert exactly one
  `ISSUE_APPROVAL_REQUESTED` webhook fires after approval flow registers
  approvers.
- A2: project without approval requirement → create issue → assert no
  `ISSUE_APPROVAL_REQUESTED` fires.

#### `ISSUE_APPROVED`

- AP1: single-approver flow → approve → assert exactly one `ISSUE_APPROVED`
  fires.
- AP2: multi-step approval flow → approve at each step → assert
  `ISSUE_APPROVED` fires only once, at the final step.

#### `ISSUE_SENT_BACK`

- SB1: approver rejects → assert exactly one `ISSUE_SENT_BACK` fires.
- SB2: after send-back, re-approve → assert subsequent `ISSUE_APPROVED` fires
  (end-to-end).

### What the matrix locks in

- BYT-9398 is caught specifically by **C4**.
- Future code paths that mutate task state without going through the
  Run/Skip/Cancel triad will fail at least one of C2/C5/C6/C7 if they bypass
  webhook triggers.
- The `PIPELINE_FAILED` cells lock in the dedup contract — any future change
  that over-fires `PIPELINE_FAILED` (e.g., a new claim path that bypasses the
  table) will fail F2.

### Non-goals

- Rich payload-schema assertions for every event. The existing
  `parseSlackWebhook` helper performs structural checks; deep payload
  validation belongs in a separate change.
- Coverage for non-Slack webhook types (Discord, Lark, Google Chat, MS Teams,
  generic). All routes share `webhook.Manager`'s switch, so trigger correctness
  is type-independent; per-type formatting is exercised elsewhere.
- New `TestCollision_*` cases. We are not adding a new store method touching a
  composite-PK table; AGENTS.md only mandates collision tests for new methods.

## 6. Risks

- **Spurious reset in non-failure flow.** If a user clicks Skip on a plan that
  never had a `PIPELINE_FAILED` row, `ResetPlanWebhookDelivery` is a no-op
  (`DELETE … WHERE` matches zero rows). Safe.
- **Race vs. concurrent `ClaimPipelineFailureNotification`.** A failing task on
  a different stage could claim FAILED at the same instant Skip is issued. The
  reset removes the just-claimed row, so the FAILED webhook is sent (the claim
  already fired the webhook in the same code path) but the row is wiped, so a
  later COMPLETED claim can succeed. This matches the existing
  `BatchRunTasks` behavior and is the intended semantic.
- **Operational recovery for plan 2505.** The historical event cannot be
  replayed automatically. Customer recovery is out of scope for this PR (the
  Linear issue notes it explicitly).

## 7. Out-of-scope follow-ups (not part of this PR)

- A principled redesign that lets `ClaimPipelineCompletionNotification`
  atomically supersede a prior `PIPELINE_FAILED` row via
  `ON CONFLICT … DO UPDATE WHERE event_type = 'PIPELINE_FAILED'`. This would
  remove the per-endpoint reset obligation but is a larger change; tracked
  separately if we want to harden the contract.

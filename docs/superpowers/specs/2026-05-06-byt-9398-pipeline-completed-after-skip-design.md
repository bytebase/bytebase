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
analogously to the existing call in `BatchRunTasks` — immediately after the
`GetPlan` nil-check and before `GetIssue`. The store function itself is
unchanged.

```go
// rollout_service.go, inside BatchSkipTasks (~line 886, after the plan == nil check)
if plan == nil {
    return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("rollout (plan) %v not found", planID))
}

// Reset notification state so PIPELINE_COMPLETED can fire after skipping a
// failed task. Errors are logged and swallowed so a transient DB hiccup
// doesn't fail the user-facing skip request — note that a failure here will
// re-introduce the BYT-9398 symptom for this plan, so the log line should
// be monitored.
if err := s.store.ResetPlanWebhookDelivery(ctx, projectID, planID); err != nil {
    slog.Error("failed to reset plan webhook delivery", log.BBError(err))
}

issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
```

No store-layer, schema, or proto changes — only a new call site in
`BatchSkipTasks` that mirrors `BatchRunTasks` exactly.

This means a `Skip` on a plan that has already reached `PIPELINE_COMPLETED`
will (just like a `Run` on such a plan today) wipe the existing dedup row and
allow a duplicate completion webhook to fire when the next completion check
runs. We treat this as the same shape of behavior the existing `BatchRunTasks`
path already has, and address it (if at all) in the section-7 follow-up that
makes `ClaimPipelineCompletionNotification` atomically supersede a FAILED row
via `ON CONFLICT … DO UPDATE`, removing the per-endpoint reset entirely.

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
| F2 | Two failing tasks in the same plan exercise PK dedup | 1 (the table dedupes — that is the contract) |
| F3 | Task fails → `BatchRunTasks` → fails again | 2 (Reset on `BatchRunTasks` clears the row, second FAILED fires) |
| F4 | HA license breach drives `failTaskRunsForHA` | **Deferred.** Requires a non-HA license JWT, replica-heartbeat seeding, and an injectable `haFailGracePeriod` — none currently exist in the test harness. Codepath at `backend/runner/taskrun/scheduler.go:71-142` is manually verified and the gap is documented via `t.Skip` in the suite. Tracked as a follow-up. |

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

## 6. Risks and known limitations

- **Spurious reset in non-failure flow.** If a user clicks Skip on a plan that
  never had a `PIPELINE_FAILED` row, the new `event_type = 'PIPELINE_FAILED'`
  WHERE clause makes `ResetPlanWebhookDelivery` a no-op. Safe.
- **Reset runs before permission/validation checks.** The call is placed
  immediately after `GetPlan` so it mirrors the existing `BatchRunTasks`
  pattern. A request that fails permission/approval validation will still
  clear the failure-notification row. Tradeoff accepted for symmetry with
  `BatchRunTasks`; a future cleanup can move both call sites after validation.
- **In-flight skip race (pre-existing, unrelated to this fix).** `BatchSkipTasks`
  sets `payload.skipped = true` but does not cancel a `PENDING`/`RUNNING` task
  run. If a user skips a still-running task that subsequently fails in
  `runTaskRunOnce`, the failure-claim and completion-claim race on the
  `(project, plan_id)` PK and one of the two events is dropped. This is a
  pre-existing behavior in BatchSkipTasks and is *not* addressed by this PR.
  The customer's BYT-9398 case is a sequential one (the failed task was
  terminal-`FAILED` for minutes before the user skipped it), so it's not
  affected. Tracked as a follow-up in section 7.
- **Race vs. concurrent `ClaimPipelineFailureNotification` from a different
  stage.** A failing task on a different stage could claim FAILED at the same
  instant Skip is issued. With the tightened SQL, reset only deletes the
  FAILED row, leaving any subsequent COMPLETED claim able to succeed. The
  failure webhook for that other stage's task still fires (the claim already
  delivered it in the same code path). Intended semantic.
- **Operational recovery for plan 2505.** The historical event cannot be
  replayed automatically. Customer recovery is out of scope for this PR (the
  Linear issue notes it explicitly).

## 7. Out-of-scope follow-ups (not part of this PR)

- A principled redesign that lets `ClaimPipelineCompletionNotification`
  atomically supersede a prior `PIPELINE_FAILED` row via
  `ON CONFLICT … DO UPDATE WHERE event_type = 'PIPELINE_FAILED'`. This would
  remove the per-endpoint reset obligation but is a larger change; tracked
  separately if we want to harden the contract.
- **Cancel in-flight task runs in `BatchSkipTasks`.** Today Skip flips a flag
  but lets a running task run to completion. To make Skip semantically
  consistent with the user's intent ("I am done with this task — don't fire
  any more events for it"), the running task run should be cancelled. This
  closes the race described in the third bullet of section 6.
- **Move `ResetPlanWebhookDelivery` after permission/approval validation in
  both `BatchRunTasks` and `BatchSkipTasks`.** Avoids clearing the dedup
  ledger when the request is going to be rejected. Out of scope here for
  symmetry; safe to do as a small follow-up.

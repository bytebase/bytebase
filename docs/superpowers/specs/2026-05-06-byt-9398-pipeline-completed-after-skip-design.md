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

Two coupled changes in `backend/api/v1/rollout_service.go`. The store function
itself is unchanged.

### 4.1 Add the reset call to `BatchSkipTasks`

`BatchSkipTasks` did not previously clear the dedup row, which is the BYT-9398
root cause. Add a `Store.ResetPlanWebhookDelivery` call inside the success
path so that PIPELINE_COMPLETED can fire after a skip resolves a prior
failure.

### 4.2 Move the reset in BOTH endpoints to AFTER validation

`Store.ResetPlanWebhookDelivery` is a state mutation (DELETE row). The
existing call site in `BatchRunTasks` (and the new one in `BatchSkipTasks`)
must run only after the request has passed authorization, approval (where
applicable), and the actual store mutation. A request that is going to be
rejected (invalid input, denied permission, missing approval) must not have
visible side effects on the dedup ledger — otherwise a later legitimate
webhook event can fire as a duplicate, because the ledger was spuriously
cleared.

The correct order in both endpoints:

```
1. parse and look up project/plan/issue (read-only)
2. parse and validate task IDs (return error → no state change)
3. canUserRunEnvironmentTasks (return error → no state change)
4. CheckIssueApproved (BatchRunTasks only; return error → no state change)
5. mutate (atomic-ish):
   a. store mutation (CreatePendingTaskRuns / store.BatchSkipTasks)
   b. ResetPlanWebhookDelivery
   c. signal next stage (TaskRunTickleChan / PlanCompletionCheckChan)
6. return success
```

The reset must come AFTER the store mutation (so a failed mutation doesn't
clear the ledger) and BEFORE the signal (so the async completion check sees
the cleared ledger).

### 4.3 Inside `BatchSkipTasks` the placement looks like

```go
if err := s.store.BatchSkipTasks(ctx, projectID, taskUIDs, request.Reason); err != nil {
    return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to skip tasks"))
}

// Reset notification state so PIPELINE_COMPLETED can fire after skipping a
// failed task. Placed after authorization + the store mutation so a rejected
// request leaves the dedup ledger untouched. Errors are logged and swallowed
// so a transient DB hiccup doesn't fail the user-facing skip request — note
// that a failure here will re-introduce the BYT-9398 symptom for this plan.
if err := s.store.ResetPlanWebhookDelivery(ctx, projectID, planID); err != nil {
    slog.Error("failed to reset plan webhook delivery", log.BBError(err))
}

s.bus.PlanCompletionCheckChan <- bus.PlanRef{ProjectID: projectID, PlanID: planID}
```

`BatchRunTasks` gets the same restructuring: reset moves from immediately
after `GetPlan` to immediately after `CreatePendingTaskRuns` and before
`TaskRunTickleChan`.

### Behavioral consequences

- Success paths are unchanged: by the time the request returns success, the
  reset has run.
- Rejected paths (auth fail, missing tasks, etc.) no longer mutate the
  dedup ledger.
- A `Skip` or `Run` on a plan that has already reached `PIPELINE_COMPLETED`
  still wipes the COMPLETED row when the call succeeds and re-arms the
  dedup ledger — that's the existing BatchRunTasks behavior and is treated
  as the same-shape "atomic-supersede-via-ON-CONFLICT" follow-up tracked
  in section 7.

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

**Out of scope:** the HA-license-breach path (`failTaskRunsForHA` in `backend/runner/taskrun/scheduler.go:71-142`). Driving it deterministically from an integration test needs harness pieces that do not currently exist (a non-HA license JWT, replica-heartbeat seeding, and an injectable `haFailGracePeriod`). The codepath is manually verified. Build the harness and add the test in a separate PR if regression coverage there becomes necessary.

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
- ~~**Reset runs before permission/validation checks.**~~ Addressed in
  section 4.2 — both endpoints now reset only after authorization and the
  store mutation succeed.
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
- ~~Move `ResetPlanWebhookDelivery` after permission/approval validation in
  both `BatchRunTasks` and `BatchSkipTasks`.~~ Done in section 4.2 (in
  response to a Codex P1 finding on the PR).

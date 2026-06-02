# ServiceNow ↔ Bytebase Integration (Bytebase-first)

**Status:** Design note · **Scope:** integrating Bytebase database-change approval & rollout with ServiceNow ITSM (CAB) approval.

## Goal

Let a user author a database change in **Bytebase's UI**, mirror it as a **ServiceNow change request (CHG)**, and have **ServiceNow gate both the approval and the rollout**, with execution status reflected back onto the CHG — keeping ITSM as the system of record for change control.

## Actors

- **Bytebase** — authoring + execution engine. Exposes a public REST/gRPC API at `/v1/*`.
- **ServiceNow** — approval system of record (CAB).
- **Orchestrator** — the only active caller. Either **ServiceNow Flow Designer + MID Server**, or a small **middleware service**. Authenticates as a **project-scoped Bytebase service account**.

## Architecture

```
  Orchestrator = ServiceNow Flow Designer + MID Server  (or a small middleware service)
  It is the ONLY active caller: discovers issues, approves, triggers + follows rollouts.

                 BYTEBASE · CREATE
        +-----------------------------------+
        | 1) User creates plan + issue (UI) |
        |    catch-all approval rule routes |---> if NO rule matches, issue auto-
        |    it to SN role -> parks PENDING |     approves & BYPASSES ServiceNow
        +-----------------------------------+
                         |
                         v
        +===================================+
        | /!\ GAP: Bytebase can't push      |
        | 2) Orchestrator DISCOVERS issue   |
        |    poll ListIssues:               |
        |    create_time >= cursor &&       |
        |    approval_status == PENDING     |
        |    (idempotent on issue name)     |
        +===================================+
                         |
                         v
               SERVICENOW · APPROVE
        +-----------------------------------+
        | 3) Create CHG -> CAB approval     |
        |    reject  -> RejectIssue         |
        |    rework  -> RequestIssue        |
        +-----------------------------------+
                         |
                  CHG approved
                         v
        BYTEBASE · CONTROL ROLLOUT (via API)
        +-----------------------------------+
        | 4) ApproveIssue (service acct)    |
        +-----------------------------------+
                         |
                         v
        +- - - - - - - - - - - - - - - - - -+
        | 5) WAIT: rollout is created ASYNC |   race fix: do NOT call
        |    poll GetRollout until           |   BatchRunTasks immediately
        |    has_rollout == true             |   (or CreateRollout explicitly,
        +- - - - - - - - - - - - - - - - - -+    guarded on has_rollout)
                         |
                         v
        +-----------------------------------+
        | 6) BatchRunTasks per stage        |
        |    test -> staging -> prod         |
        |    fail -> BatchCancel / Skip      |
        +-----------------------------------+
                         |
                         v
        +-----------------------------------+
        | 7) FOLLOW to completion           |   short rollouts: same execution
        |    poll GetRollout/ListTaskRuns    |   long rollouts: in-flight-only
        |    until DONE/FAILED, update CHG   |   poller (NOT a full scan)
        +-----------------------------------+

  Note: steps 2 and 7 (the polls) exist ONLY because Bytebase can't push.
        A generic outbound webhook (ISSUE_CREATED / PIPELINE_*) removes both.
        Interim push (no code change): Bytebase TEAMS webhook -> Power Automate
        -> ServiceNow connector; keep a slow reconciliation poll as a backstop.
```

## Enforcement prerequisites

Get these wrong and changes silently bypass ServiceNow:

1. **Catch-all approval rule.** The SN gate only engages if a `WorkspaceApprovalSetting` rule *matches* the issue and routes it to the SN-only role. **An unmatched issue auto-approves (`SKIPPED`) and bypasses ServiceNow entirely.** Configure a catch-all rule covering every in-scope project/issue-type; treat "no rule matched" as a misconfiguration to alert on.
2. **No UI bypass of rollout.** `BatchRunTasks` passes for anyone holding the `bb.taskRuns.create` permission — which `projectOwner`/`workspaceAdmin` carry by default. Give human users a **custom role without `bb.taskRuns.create`** and reserve that permission for the service account; otherwise owners/admins can roll out straight from the UI and skip ServiceNow.
3. **Per-environment `RolloutPolicy.automatic = false`.** It is the default, but verify it on **every** gated environment — any environment left `automatic` auto-runs that stage and skips the gate.

## Flow

1. **Create (Bytebase UI).** User authors plan + issue; the catch-all rule parks it `PENDING` on the SN role.
2. **Discover (orchestrator).** Poll `ListIssues` with a server-side filter — `create_time >= <cursor> && approval_status == "PENDING"` (optionally `&& labels == "needs-cab"`), `order_by create_time desc`. **Idempotency keyed on the issue resource name** so overlapping polls / restarts don't create duplicate CHGs.
3. **Approve (ServiceNow).** Create CHG; run CAB.
   - **Rejected →** `RejectIssue`. **Rework →** `RequestIssue`. Store the CHG ↔ issue mapping; link back via `CreateIssueComment` (include the **SN approver identity + CHG number** — see Security/Audit).
4. **Approve back (API).** On CHG approval, call `ApproveIssue` as the service account.
5. **Wait for the rollout to exist (race fix).** `ApproveIssue` creates the rollout **asynchronously** (it returns before stages/tasks exist). Do **not** call `BatchRunTasks` immediately. Either:
   - poll `GetRollout` until `has_rollout == true` (simplest), **or**
   - call `CreateRollout` explicitly (synchronous; returns stages + tasks). `CreateRollout` is **not idempotent**, so guard on `has_rollout` / catch the duplicate error if the async runner already created it.
6. **Roll out (API).** `BatchRunTasks` per stage (one stage = one environment; optional `run_time` for a maintenance window) to promote test → staging → prod.
   - **Failure →** `BatchCancelTaskRuns` / `BatchSkipTasks`; reflect partial-promotion state onto the CHG.
7. **Follow to completion.** Poll `GetRollout` / `ListTaskRuns` until DONE/FAILED, update the CHG. Short rollouts: inside the same triggered execution. Long rollouts: an in-flight-only poller.

## Polling: who does it, and is it scalable?

There are two polls, and both exist for one reason: **Bytebase cannot push to ServiceNow** (no generic outbound webhook). Whoever wants to know "did something change in Bytebase?" has to ask.

- **Discovery poll (step 2)** — detect a newly created issue.
- **Status poll (step 7)** — detect that the rollout reached DONE/FAILED so the CHG can be closed out.

**Who polls** = whichever side hosts the orchestrator:

- **(A) ServiceNow-native** — Flow Designer makes the outbound REST calls; a **MID Server** reaches a self-hosted Bytebase behind a firewall. Standard ITSM integration shape. Caveat: ServiceNow scheduled jobs are coarse (minute-grain, heavy if scanning many records), so a *standing background poller over all open changes* does not scale elegantly past low/moderate volume.
- **(B) Middleware-driven (preferred at scale)** — a small stateless service holds the service-account key, does both polls (in-flight work only, with a cursor), and writes back via ServiceNow's Table/Import Set API. Scales because concurrency, backoff, and scope are under your control.

**Key point — most "polling" is not a background timer.** The orchestrator is already the active caller (it invokes `BatchRunTasks`), so the status poll is just the orchestrator **following its own action to completion**:

- **Short rollouts** (seconds–minutes): the same triggered execution loops on `GetRollout` until the stage finishes — **no standing poller needed**.
- **Long rollouts** (large DDL, hours): exceed a Flow/MID-Server execution window, so an **in-flight-only** poller is needed (scoped to active rollouts with a cursor — not a full scan).

**Scalability:** cost ∝ (in-flight changes × poll frequency). Trivial for typical change volumes (dozens/day) in either pattern. The real fix is the **generic outbound webhook**: fire `ISSUE_CREATED` and `PIPELINE_COMPLETED|FAILED` and **both polls disappear** — the integration becomes fully event-driven.

## Optional: reduce discovery latency with a push (design tradeoff)

The discovery poll (step 2) adds latency between "issue created in Bytebase" and "CHG created in ServiceNow." How much depends on the orchestrator:

- **ServiceNow-native** scheduled flows have a ~1-minute floor → minute-level gap.
- **Middleware** can poll tighter (e.g. every 10–30s) → seconds-level gap, at higher request volume.

You can collapse the gap **today, with no Bytebase code changes**, by reusing an existing webhook type as a push channel:

**Bytebase `TEAMS` webhook → Microsoft Power Automate → ServiceNow.** The `TEAMS` webhook validator allows `*.powerplatform.com` (Power Automate Workflows, current) and `*.logic.azure.com` via suffix match — so tenant URLs like `prod-NN.westus.logic.azure.com` pass (`backend/plugin/webhook/validator.go`). Wiring:

1. Subscribe a project webhook of type `TEAMS` to `ISSUE_CREATED`, pointing at a Power Automate Workflow HTTP-trigger URL.
2. On issue creation Bytebase POSTs a MessageCard immediately (async, ~instant). The card carries the issue **link** (`.../projects/{project}/issues/{uid}`), title, status, and actor.
3. The Power Automate flow parses the card, extracts the issue ref (optionally calls `GetIssue` for detail), and creates the CHG via its native **ServiceNow connector**, storing the issue ↔ CHG mapping.

**Tradeoffs (why this is optional, not the default):**

- **Reliability.** Bytebase webhooks are fire-and-forget (3s timeout, **no retry**). A slow/down endpoint means the event is **lost** — unlike polling, which self-heals on the next scan. Recommended pattern: **push for latency + keep a low-frequency reconciliation poll (every few minutes) as a backstop** to catch dropped events.
- **Dependency.** Requires Microsoft Power Platform (license + a maintained flow). Only the `TEAMS` type has both an allowed domain *and* a first-class ServiceNow connector at the far end; the other chat types (Slack, Google Chat, etc.) don't bridge cleanly.
- **Payload shape.** You're parsing a Teams **MessageCard** — a chat-notification shape, not an eventing contract. The issue link is the durable field; treat the rest as best-effort. The legacy Office 365 connector domains (`.office.com` / `.office365.com`) are being retired (2025–2026) — use **Power Automate Workflows** (`.powerplatform.com`).

This is a pragmatic interim until the **generic outbound webhook** (see Gaps) enables a clean, direct push to ServiceNow.

## Auth & operations

- **Service account** is **project-scoped** (not workspace) and least-privilege: only the SN approver role + the rollout role for gated environments.
- `AuthService.Login` (email + service key) returns a **1-hour JWT** — the orchestrator must handle login/refresh.
- **Key rotation** scheduled via `UpdateServiceAccount`; store the key in a secret manager.

## Security / compliance

- **Privileged identity.** The service account can both approve and deploy to prod; a leaked key = unauthorized prod change. Use TLS only, an IP allowlist / **MID Server** as the egress path, scoped credentials, and rotation.
- **Audit attribution (compliance-critical).** Bytebase records approvals against the **bot principal**, not the human CAB approver. Mitigate by embedding the SN approver identity + CHG number in the `ApproveIssue` comment, and document that auditors cross-reference ServiceNow. This is a known weakness, not a solved problem.
- **Connectivity.** ServiceNow (SaaS) must reach a typically self-hosted, firewalled Bytebase — MID Server (pattern A) or controlled ingress (pattern B) is a prerequisite.

## Gaps & the one fix that collapses most of this

- **No generic outbound webhook** → forces the discovery poll (step 2) and the status poll (step 7). Adding a custom webhook type firing `ISSUE_CREATED` + `ISSUE_APPROVED` + `PIPELINE_COMPLETED|FAILED` makes the integration **event-driven** and removes both polls. **Highest-leverage change.**
- **No external-ticket reference field** on the issue — link lives in a label/comment only; no structured, clickable back-link to the CHG.
- **No "external approval" state** — the SN-gated wait shows as a generic pending approval, and the approval is attributed to the bot.
- **No per-project "rollout only via API" toggle** — enforcement depends on hand-pruning `bb.taskRuns.create`.

## API reference (RPCs used)

| Step | RPC | Notes |
|------|-----|-------|
| Auth | `AuthService.Login` | service key → 1h JWT bearer |
| Create | `PlanService.CreatePlan`, `IssueService.CreateIssue` | issue references plan via `issue.plan` |
| Discover | `IssueService.ListIssues` | server-side `filter` (CEL) + `order_by` |
| Link back | `IssueService.CreateIssueComment` | write CHG number / approver into the issue |
| Approve / reject | `IssueService.ApproveIssue` / `RejectIssue` / `RequestIssue` | auth is `CUSTOM` (caller must hold the next pending role) |
| Rollout (ensure exists) | `RolloutService.GetRollout` / `CreateRollout` | poll `has_rollout`, or create synchronously (not idempotent) |
| Roll out | `RolloutService.BatchRunTasks` | per stage; optional `run_time` |
| Abort / skip | `RolloutService.BatchCancelTaskRuns` / `BatchSkipTasks` | on failure |
| Follow status | `RolloutService.GetRollout` / `ListTaskRuns` | task status PENDING→RUNNING→DONE/FAILED |

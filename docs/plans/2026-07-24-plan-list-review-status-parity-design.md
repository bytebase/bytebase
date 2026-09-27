# Plan List Review Status Parity Design

- **Date:** 2026-07-24
- **Status:** Implemented
- **Issues:** [BYT-9551](https://linear.app/bytebase/issue/BYT-9551/plan-list-shows-wrong-review-status), [BYT-9811](https://linear.app/bytebase/issue/BYT-9811/plan-list-shows-under-review-for-a-plan-whose-issue-is-closed)
- **Related PR:** [#20372](https://github.com/bytebase/bytebase/pull/20372)

## Summary

Plan List and Plan Detail must derive the Review badge from the same lifecycle
inputs. Plan Detail can do this today because it fetches the linked Issue. Plan
List cannot because the public `Plan` response exposes the linked Issue name and
approval status, but not the Issue lifecycle status.

Add an output-only `issue_status` field to `Plan`, populate it from the Issue
that `convertToPlans` already batch-loads, and pass it to the shared
`getReviewBadge` helper. Also invalidate the Plan List page cache when a linked
Issue changes, because Issue status and approval changes do not update
`Plan.update_time`.

This closes both known residual mismatches:

- A canceled Issue must show **Closed**, not an approval-derived badge.
- A done Issue without a rollout and with approval still pending must show
  **Bypassed**, not **Under review**.

## Problem

Review status is not a stored Plan field. It is a display projection over four
inputs:

```text
linked Issue + Issue status + approval status + rollout existence
```

The current Plan List projection lacks Issue status:

```text
ListPlans
  -> Plan {
       issue
       approval_status
       has_rollout
       // issue_status is missing
     }
  -> getReviewBadge(issueStatus: undefined)
```

Plan Detail follows `plan.issue` with `GetIssue`, so it has the complete input:

```text
GetPlan + GetIssue
  -> getReviewBadge(issue.status, issue.approval_status, plan.has_rollout)
```

PR #20372 centralized badge computation and fixed the BYT-9551 case where a
rollout exists while approval remains pending. It intentionally left two
categories unresolved because Plan List still cannot observe Issue status.

| Issue status | Has rollout | Approval status | Plan List today | Plan Detail |
| --- | --- | --- | --- | --- |
| `OPEN` | true | `PENDING` | Bypassed | Bypassed |
| `CANCELED` | false | `PENDING` | Under review | Closed |
| `DONE` | false | `PENDING` | Under review | Bypassed |

The second row is BYT-9811. The third row is the other residual divergence
documented by #20372.

## Current model

### Plan

`Plan` owns the proposed database changes and deployment state:

- `Plan.state` is the Plan resource state (`ACTIVE` or `DELETED`).
- `Plan.specs` are the changes to execute.
- `Plan.has_rollout` records whether deployment has started.
- `Plan.issue` points to the linked review Issue when one exists.
- `Plan.approval_status` is a denormalized summary of the linked Issue approval
  flow.

`Plan.state=DELETED` is not equivalent to a closed review. The Plan List already
shows that state beside the Plan name. It must not be reused to infer the Review
column.

### Issue

`Issue` owns review and workflow lifecycle:

- `Issue.status` is `OPEN`, `DONE`, or `CANCELED`.
- `Issue.approval_status` is `CHECKING`, `PENDING`, `APPROVED`, `REJECTED`, or
  `SKIPPED`.
- `Issue.draft` determines whether the review Issue has been submitted.
- `Issue.plan` points back to the Plan for database-change Issues.

Issue lifecycle and approval status are independent. For example, canceling an
Issue does not rewrite its approval payload, so a canceled Issue may retain
`PENDING`, `APPROVED`, or `REJECTED`.

The database enforces at most one Issue per Plan with the unique
`(project, plan_id)` index. This makes a single linked-Issue summary on `Plan`
unambiguous.

### ListPlans as a read model

The stored Plan row does not contain Issue lifecycle state. `ListPlans` already
acts as an aggregated read model:

1. List Plan rows.
2. Batch-load Issues by `(project, plan_id)`.
3. Batch-load Plan check runs.
4. Batch-load rollout task status counts.
5. Build the public `Plan` response.

The linked `IssueMessage` already contains `Status`. The value is lost only
when the public `Plan` message is built.

## Goals

1. Make the Plan List Review badge match Plan Detail for the same Plan snapshot.
2. Preserve Issue lifecycle, approval, Plan resource state, and rollout state as
   separate model dimensions.
3. Complete the existing ListPlans read model without adding frontend Issue
   requests.
4. Keep one shared badge-precedence function for Plan List and Plan Detail.
5. Prevent Back navigation from restoring a stale Plan List badge after an
   Issue action.
6. Keep the API change additive at the wire and JSON levels.

## Non-goals

- Persisting a combined Review status on the Plan row.
- Replacing Issue status or approval status with a new lifecycle enum.
- Changing when an Issue becomes `DONE` or `CANCELED`.
- Changing approval, bypass, rollout-creation, or Plan-deletion behavior.
- Redesigning the Plan List table or Plan Detail lifecycle UI.
- Adding Issue filters to `ListPlans`.
- Making Plan List poll for external Issue changes while it remains open.
- Redesigning the current draft/incomplete inference.

## Design

### Public API

Move the existing top-level `IssueStatus` enum from
`proto/v1/v1/issue_service.proto` to `proto/v1/v1/common.proto`, next to
`ApprovalStatus`.

The protobuf full name remains `bytebase.v1.IssueStatus`, and its numeric values
remain unchanged:

```proto
enum IssueStatus {
  ISSUE_STATUS_UNSPECIFIED = 0;
  OPEN = 1;
  DONE = 2;
  CANCELED = 3;
}
```

Both `Issue.status` and the new Plan field use this shared type:

```proto
message Plan {
  // Existing fields omitted.

  // The lifecycle status of the linked issue.
  // Unspecified when no linked issue exists.
  IssueStatus issue_status = 15
      [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

The field is intentionally raw Issue lifecycle state, not a backend-rendered
badge. Clients still combine it with `approval_status` and `has_rollout`
according to their presentation needs.

Moving the enum avoids two competing status types with the same values. It is
wire- and JSON-compatible because the protobuf full name and numeric values do
not change. It does change the generated module that exports `IssueStatus` in
TypeScript, so repository imports must move from `issue_service_pb` to
`common_pb`. Generated-source consumers outside this repository may need the
same import update; this source compatibility cost must be called out in the PR.
The repository's Buf configuration uses FILE-level breaking rules, so moving
the enum may be reported if breaking checks are enabled even though the current
proto workflow runs lint without a breaking check. Confirm the generated-source
compatibility decision with the API owner before implementation. If preserving
the generated module export is mandatory, use a Plan-local enum with the same
numeric semantics as the fallback; do not replace the typed status with booleans
or strings.

Also correct the `Plan.approval_status` comment. The implementation leaves it
unspecified both when no linked Issue exists and when the linked Issue is still
a draft.

### Backend projection

`convertToPlans` already builds `issueByPlanKey` using both project ID and Plan
UID. Populate `IssueStatus` in the same block that populates `Issue` and
`ApprovalStatus`:

```go
if issue := issueByPlanKey[key]; issue != nil {
    v1Plan.Issue = common.FormatIssue(issue.ProjectID, issue.UID)
    v1Plan.IssueStatus = convertToIssueStatus(issue.Status)
    if !issue.Payload.GetDraft() {
        v1Plan.ApprovalStatus = computeApprovalStatus(
            issue.Payload.GetApproval(),
        )
    }
}
```

This applies to both `GetPlan` and `ListPlans` because both use
`convertToPlans`.

No schema migration, store method, extra query, or permission check is needed.
`ListPlans` already exposes the linked Issue name and approval status to callers
with `bb.plans.list`; lifecycle status is another field in the same summary.

Draft Issues report `issue_status=OPEN` and
`approval_status=APPROVAL_STATUS_UNSPECIFIED`. The existing `plan.issue` and
approval-status logic continue to distinguish a linked draft from an
Issue-less incomplete Plan.

### Badge precedence

Keep `getReviewBadge` as the single presentation policy:

1. No linked Issue: no Review badge.
2. `issue_status=CANCELED`: Closed.
3. `has_rollout=true` or `issue_status=DONE`, with
   `approval_status=PENDING`: Bypassed.
4. Otherwise render the approval-derived badge:
   - `APPROVED`: Approved
   - `SKIPPED`: Skipped
   - `REJECTED`: Rejected
   - `PENDING`: Under review
   - `CHECKING` or unspecified: no badge

Canceled takes precedence over rollout and approval so Plan List matches Plan
Detail even for unusual or legacy state combinations.

Plan List passes `plan.issueStatus` instead of `undefined`. Remove comments and
tests that describe CANCELED and DONE-without-rollout as intentional
divergences.

### Plan List cache invalidation

The Plan List cache is keyed by project and restored only for explicit
list-to-detail Back navigation. Detail hooks currently invalidate that cache
when `Plan.update_time` changes.

Issue actions do not change `Plan.update_time`, even though they change
Issue-derived fields in the next `GetPlan` or `ListPlans` response:

- Issue close/reopen changes `Plan.issue_status`.
- Approval actions change `Plan.approval_status`.
- Draft submission changes whether approval status is populated.

Therefore, Plan List cache invalidation must also depend on the linked Issue's
`update_time`.

In both Plan Detail and Issue Detail `patchState` paths:

1. Keep invalidating the Plan List cache when the Plan changes.
2. Also invalidate the Plan List cache when the linked Issue changes.
3. Keep invalidating the Issue List cache when the Issue changes.

Expose a direct project Plan-cache invalidation helper for batch Issue actions.
After a successful batch close or reopen, invalidate the Plan List cache for
each affected project that contains a Plan-backed Issue.

Do not change `Plan.update_time` to include Issue updates. That field describes
the Plan resource and is also displayed as the Plan's Updated time; redefining
it would mix Plan edits with review activity.

### Consistency boundary

A single `ListPlans` conversion batch reads Plan rows, then Issue rows. It is not
a database snapshot spanning both queries, so a concurrent Issue transition
may appear on the next request rather than the current one. This is consistent
with the existing approval-status projection and acceptable for the list.

The frontend must not issue per-row `GetIssue` requests to close this window.
That would introduce N+1 traffic, require `bb.issues.get` in a
`bb.plans.list` surface, and create more opportunities for mixed snapshots.

## Alternatives considered

### Infer closed from approval or rollout

Rejected. Issue status is an independent dimension. The same approval and
rollout tuple can belong to an open, done, or canceled Issue.

### Fetch each Issue from Plan List

Rejected. This adds N+1 requests, creates a permission mismatch, and duplicates
data the backend already loaded.

### Use Plan state

Rejected. `Plan.state=DELETED` is Plan resource lifecycle, while
`Issue.status=CANCELED` is review lifecycle. They are displayed separately and
can change independently.

### Add `issue_canceled` and `issue_done` booleans

Rejected. Multiple booleans permit invalid combinations and duplicate an
existing enum.

### Return a backend-computed badge status

Rejected for this change. `Closed`, `Bypassed`, and `Under review` are display
semantics derived from raw domain state. The existing shared frontend helper
already centralizes that policy.

### Add a `review_summary` message

Deferred. A structured summary containing Issue name, lifecycle status,
approval status, and draft state would be reasonable if more Issue-derived
fields are added to Plan. Introducing it now would duplicate or deprecate
existing flat fields for the current narrow gap.

## Test plan

### Backend

Add Plan conversion/ListPlans coverage for:

- No linked Issue -> empty Issue and unspecified Issue status.
- Linked draft Issue -> `OPEN`, with approval status unspecified.
- Linked submitted `OPEN` Issue -> `OPEN` and its computed approval status.
- Linked `DONE` Issue -> `DONE`.
- Linked `CANCELED` Issue -> `CANCELED`.
- Plans with colliding numeric IDs in different projects do not receive each
  other's Issue status.

The collision case should continue to identify the relationship by the full
`(project, plan_id)` key.

### Frontend unit and rendering tests

Update `reviewBadge.test.ts` so Plan List and Plan Detail use the same complete
input matrix. Delete tests that intentionally expect residual divergence.

Add Plan List rendering coverage for:

- Canceled Issue renders Closed.
- Done Issue with pending approval and no rollout renders Bypassed.
- Open Issue with pending approval renders Under review.
- Rollout with pending approval remains Bypassed.
- Plan deletion and review closure remain separate visual states.

### Cache tests

Extend project paged-cache tests and detail-hook tests to verify:

- A Plan update invalidates the Plan List cache.
- An Issue status or approval update invalidates both Issue List and Plan List
  caches.
- An unchanged Issue does not invalidate either cache.
- A batch Issue close/reopen invalidates the Plan List cache for every affected
  Plan-backed project.

### Interaction regression

Cover the user journey:

1. Open Plan List with an under-review Plan.
2. Open the Plan.
3. Close the linked Issue.
4. Navigate Back.
5. Confirm the restored/refetched Plan row shows Closed without manual refresh.
6. Reopen the Issue and confirm the Plan row returns to Under review.

## File inventory

Expected source changes:

```text
proto/v1/v1/common.proto
proto/v1/v1/issue_service.proto
proto/v1/v1/plan_service.proto
backend/api/v1/plan_service.go
backend/api/v1/plan_service_test.go
frontend/src/routes/project/ProjectPlanDashboardPage.tsx
frontend/src/routes/project/utils/reviewBadge.ts
frontend/src/routes/project/utils/reviewBadge.test.ts
frontend/src/lib/projectPagedDataCache.ts
frontend/src/lib/projectPagedDataCache.test.ts
frontend/src/routes/project/plan-detail/shell/hooks/usePlanDetailPage.ts
frontend/src/routes/project/issue-detail/hooks/useIssueDetailPage.ts
frontend/src/components/IssueTable.tsx
```

Proto generation will also update generated Go, TypeScript, OpenAPI, and gRPC
documentation artifacts. Frontend imports of `IssueStatus` must move to
`common_pb` after regeneration; this affects every current frontend source file
that imports the enum, not only the feature files listed above.

## Verification

Run:

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)

gofmt -w backend/api/v1/plan_service.go \
  backend/api/v1/plan_service_test.go
golangci-lint run --allow-parallel-runners
go test -v -count=1 ./backend/api/v1 -run '^TestPlanService'

pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test

git diff --check
```

Before publication, review the generated API diff specifically for:

- `bytebase.v1.IssueStatus` retaining its full name and numeric values.
- `Plan.issue_status` using a new field number.
- No accidental change to Issue status JSON names.
- Generated TypeScript import changes caused by moving `IssueStatus`.

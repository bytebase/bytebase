# Plan Lifecycle Advance Design

- **Date:** 2026-07-27
- **Status:** Implemented
- **Issues:** [BYT-9925](https://linear.app/bytebase/issue/BYT-9925/plan-create-button-is-disabled-again-on-validation-errors-hiding),
  [BYT-9936](https://linear.app/bytebase/issue/BYT-9936/ready-for-review-always-opens-an-unnecessary-submission-form)
- **Supersedes behavior from:** #20615 (BYT-9531), #20924

## Summary

The plan-detail header keeps one primary control beside the title. Two of its
states — `create` and `ready-for-review` — disagree with the rest of the header,
and with each other, about how a primary action handles unmet conditions.

`create` disables the button and hides the reasons in a native hover tooltip
that shows only the first one. `ready-for-review` does the opposite: it always
opens a form, even when nothing needs a decision.

Both are the same missing rule, and the rule is already written down in the
deploy phase of the same header.

Replace both with one component:

- The button is **always rendered and always enabled** (except while a request
  is in flight).
- Pressing it with unmet conditions **states all of them** in a popover anchored
  to the button. The list is information — no fields, no confirm, no links away —
  and it closes itself as conditions resolve.
- Pressing it when the only thing left is a deliberate override opens **that one
  decision**, named after what it does.

## Problem

### The row speaks three dialects

| Lifecycle state | Control | Unmet conditions surfaced as |
| --- | --- | --- |
| `create` | Disabled `<Button>` | Native `title`, first reason only |
| `ready-for-review` | `<Button>` + popover | A form: labels, warning, acknowledgement, errors, Confirm/Cancel |
| `ready-for-review` (editing / no permission) | Disabled `<Button>` | Native `title` |
| `run-stage` | `<Button>` + confirm sheet | Not applicable — a real decision |
| `plan-status` | Status pill + gate panel | Click to reveal |
| not actionable | `LifecycleStamp` | Status, never a disabled action |

`DeployStageActions.tsx` states the rule the last three rows follow:

```text
not actionable → status, not a disabled action
```

`PlanStatusAction` supplies the other half: a control you click to learn why.
The two draft states predate the lifecycle refactor and never adopted either.

### Why the current `create` branch is a dead end

`PlanDetailHeader.tsx` renders a disabled button carrying
`title={createDisabledReason}`, where `createDisabledReason` is
`createPermissionReason ?? createPlanBlockingReasons[0]`.

Three separate defects fall out of that one line:

1. **Only the first reason is reachable.** With an empty title and an empty
   statement, fixing the title reveals the second blocker for the first time.
2. **Keyboard and screen-reader users get nothing.** A disabled button is out of
   the tab order, and `title` is unreliable for assistive technology even when
   it is reachable.
3. **`getCreatePlanBlockingReasons` already returns the full list.** The data is
   correct; only the presentation throws it away.

This is a regression. #20615 shipped an always-enabled button with a callout
listing every blocker; #20924 rewrote the branch and dropped it, along with the
`plan.cannot-create` key.

### Why the current `ready-for-review` branch over-asks

The popover always contains an `IssueLabelSelect` and a Confirm/Cancel pair, and
conditionally a checks warning, an acknowledgement checkbox, and an error alert.

- Labels are **already editable** in `PlanDetailMeta`, one row below. The popover
  is a second editor for the same field.
- The warning, the acknowledgement and the error only apply when checks fail.
- Every submission pays the cost of the exception.

## Design

### One rule, three tiers

The tier is derived from data, so the component never hard-codes which surface
appears. The same failing check is tier 1 or tier 2 depending on whether the
project enforces SQL review.

| Tier | Condition | Surface |
| --- | --- | --- |
| 0 | Nothing unresolved | None — the click advances |
| 1 | One or more unmet conditions | A popover listing all of them |
| 2 | A deliberate override | One confirmation, named after what it does |

Tiers 1 and 2 are mutually exclusive, so they share a single popover anchored to
the button.

Two corollaries:

- **Optional metadata is never a gate.** Labels leave the submit path. Where a
  project forces labels they become a tier-1 line; the control stays in the
  metadata row.
- **Nothing removes the action.** A running check, an unsaved edit and a missing
  permission are all tier-1 lines. The button stays where it is, and the popover
  says why. This is one fewer header state to build than a separate
  "no permission" stamp, and it keeps the header's geometry stable.

### Why a popover on the action

The explanation belongs to the control it explains. Anchoring it to the button
keeps that relationship obvious and keeps the sticky header exactly one row tall
in every state — a full-width band across the header reads as a page-level
condition rather than an answer to the press that produced it, and it pushes the
rest of the page down while it is showing.

It also matches the vocabulary the rest of the header already speaks: the
plan-status pill and the review composer both answer a press with a popover
anchored to the control.

Cost, stated plainly: the list closes when the reader clicks away, so with more
than one blocker they may need to press again after resolving the first. The
list is short, every entry names something visible on the page, and pressing
again is one click.

### Blocker sources

```text
create             → title, empty statements, create permission
ready-for-review   → empty statements, data-export unsupported, checks running,
                     checks failed (only when enforceSqlReview), labels
                     (only when forceIssueLabels), unsaved edits, update permission
```

A blocker carries a `kind`:

- `fix` — the reader resolves it.
- `wait` — it resolves itself. The row says so, so a running check does not read
  as something forgotten.

### Component shape

One component owns the surface state and renders the button plus both popover
contents:

```ts
<LifecycleAdvanceButton
  blockers={createBlockers}
  busy={updating}
  heading={t("plan.cannot-create")}
  onAdvance={() => void handleCreatePlan()}
  onBlocked={(blockers) => {
    if (blockers.some((blocker) => blocker.id === "title")) {
      titleInputRef.current?.focus();
    }
  }}
  verb={t("common.create")}
/>
```

`LifecycleAdvanceButton` owns one
`"none" | "blockers" | "decision"` surface value, so the two popovers cannot be
open at the same time. The blocker list shows only while the selected surface is
`blockers` and blockers remain, so it self-clears without an effect. A single
`dismiss` closes either surface on outside press, Escape, or Cancel.

The header keys the component on `page.pageKey`, resetting its local surface
when navigation moves to another plan while preserving it across same-plan
resource routes. `onBlocked` receives the current blocker list; the header uses
it for the one nudge that costs nothing: when the missing title is a blocker, put
the cursor in the title field, which is already in the header.

## Implementation

### `frontend/src/routes/project/plan-detail/components/lifecycle/advanceState.ts`

New. Owns the route-local blocker and decision model plus the pure resolvers that
derive it from plan and project state.

```ts
export interface AdvanceBlocker {
  id: string;
  message: string;
  kind: "fix" | "wait";
}

export interface AdvanceDecision {
  headline: string;
  body: string;
  verb: string;
}

export interface AdvanceState {
  blockers: AdvanceBlocker[];
  decision?: AdvanceDecision;
}
```

- Replace `getCreatePlanBlockingReasons` with `getCreatePlanBlockers`, which
  returns `AdvanceBlocker[]` and additionally reports a missing create
  permission.
- Replace `getCreateIssueBlockingErrors` + `getCreateIssueConfirmErrors` with a
  single `getSubmitReviewAdvance`, folding blockers and the optional failed-check
  decision into one result so the two outcomes cannot drift apart.
- Reuse `getPlanCheckSummaryWithFallback` for running and failed checks, while
  handling the backend-only `AVAILABLE` queue status explicitly.
- Export stable empty values for lifecycle states that do not render an advance.

### `frontend/src/lib/plan/workflow.ts`

- `submitDraftReview` drops its `labels` parameter and sends
  `updateMask: { paths: ["draft"] }`. Labels are already persisted by
  `PlanDetailMeta`.
- Remove the header-only blocker and warning helpers now owned beside the route.

### `frontend/src/routes/project/plan-detail/components/lifecycle/LifecycleAdvance.tsx`

New. Exports `LifecycleAdvanceButton` and its props.

Both surfaces reuse the existing `Popover` primitives, controlled and anchored to
the button rather than trigger-driven, so a press opens them only at tier 1 or 2.
`initialFocus={false}` keeps the caller's `onBlocked` nudge — the cursor in the
empty title — from being stolen by the popup. The blocker list carries
`role="alert"` so it is announced when it appears.

### `frontend/src/routes/project/plan-detail/components/PlanDetailHeader.tsx`

- Both the `create` and `ready-for-review` branches become callers of the same
  component, differing only in verb, heading and blocker source. The header row
  itself is unchanged — nothing new renders beneath it.
- Delete `showReviewPopover`, `selectedLabels`, `checksWarningAcknowledged` and
  the effect that syncs labels from `page.issue.labels`.
- Delete `ReadyForReviewPopoverContent` and the `IssueLabelSelect` / `Checkbox`
  imports.
- `handleCreatePlan` and `handleSubmitDraftReview` keep their existing
  early-return guards; the component no longer relies on `disabled` to enforce
  them.
- Resolve advance data only for the lifecycle state that currently renders it,
  and key `LifecycleAdvanceButton` on `page.pageKey` to isolate local popover
  state across plans.

### Locales

Four new keys across all five locale files:

| Key | English |
| --- | --- |
| `plan.cannot-create` | Cannot create plan |
| `plan.not-ready-for-review` | Not ready for review |
| `plan.submit-review-anyway` | Submit anyway |
| `plan.blocker.clears-on-its-own` | Clears on its own |

`issue.action-anyway` becomes orphaned when the acknowledgement checkbox goes
and is removed from all five locale files.

Everything else reuses shipping strings: `plan.title-required`,
`plan.navigator.statement-empty`, `plan.labels-required-for-review`,
`plan.editor.save-changes-before-continuing`,
`plan.draft-update-permission-required`, `common.missing-required-permission`,
`plan.lifecycle.gate-checks-failed`, `issue.checks-warning-hint`, and the two
`custom-approval.issue-review.disallow-approve-reason.*` strings.

## Testing

### `frontend/src/routes/project/plan-detail/components/lifecycle/advanceState.test.ts`

New. Covers every resolver branch:

- `getCreatePlanBlockers` — empty title, whitespace title, empty statements,
  both together in order, missing permission, and the valid case.
- `getSubmitReviewAdvance` — empty statements, data export, checks running
  (`kind: "wait"`), checks failed with and without `enforceSqlReview`, labels
  required with and without `forceIssueLabels`, unsaved edits, missing
  `bb.issues.update`, and the clean case.
- The decision is present when checks failed and SQL review is not enforced,
  absent when enforced, absent when checks pass, and absent for a plan that is
  not purely a database change.

### `frontend/src/lib/plan/workflow.test.ts`

- `submitDraftReview` — sends `draft: false` with `updateMask: ["draft"]` and no
  longer writes labels.

### `frontend/src/routes/project/plan-detail/components/lifecycle/LifecycleAdvance.test.tsx`

New:

- Tier 0 — pressing calls `onAdvance` and opens nothing.
- Tier 1 — pressing does not call `onAdvance`, reveals every blocker, and calls
  `onBlocked`.
- Tier 1 — the list is absent before the first press.
- Tier 1 — the list disappears when the blocker set empties, with no second
  press, and shortens as individual blockers resolve.
- Tier 1 — a `wait` blocker renders its clears-on-its-own affordance.
- Tier 1 — outside press dismisses the list.
- Tier 2 — pressing opens the confirmation and does not advance; confirming
  advances once; cancelling does not.
- Blockers outrank a decision when both are present.
- `busy` disables the primary button while a request is in flight.

### `frontend/src/routes/project/plan-detail/components/PlanDetailHeader.test.tsx`

Update the existing draft-advance coverage and add integration cases for the new
tiers:

- Create stays enabled, lists validation and permission blockers, focuses the
  empty title, self-clears when the title is fixed, and never calls `createPlan`
  while blocked.
- A clean draft submits in one press with no intermediate form, and the request
  updates only `draft`; label edits remain owned by the metadata row.
- Unsaved edits and a missing `bb.issues.update` permission list blockers and do
  not submit.
- Failing checks without `enforceSqlReview` reach the named override; enforced
  SQL review renders the same failure as a blocker instead.
- An open blocker list does not survive navigation to another plan.

## Out of scope

- Renaming **Ready for Review** to **Submit for review**. The control names a
  state rather than the action it performs, and the same principle that produces
  *Submit anyway* argues for it — but it is a copy change with its own blast
  radius across locales, tests and docs.
- Any change to `run-stage`, `plan-status`, or the review composer. They already
  follow the rule.
- Backend or protobuf changes. This is frontend-only.

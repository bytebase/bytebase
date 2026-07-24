# Plan Detail Resource Routing Design

**Date:** 2026-07-22
**Status:** Implemented
**Issue:** [BYT-9913](https://linear.app/bytebase/issue/BYT-9913/plan-detail-clicking-a-spec-tab-in-changes-collapses-the-section)

## Summary

Plan Detail should use one persistent route shell for a plan and nested routes
for resources owned by that plan. Resource identity belongs in the path; UI
phase disclosure stays local. Existing `?phase=changes|review|deploy` bookmarks
remain readable for compatibility, but the application does not create new
phase queries.

The design preserves Bytebase resource-name structure:

```text
projects/{project}/plans/{plan}
projects/{project}/plans/{plan}/specs
projects/{project}/plans/{plan}/specs/{spec}
projects/{project}/plans/{plan}/rollout
projects/{project}/plans/{plan}/rollout/stages/{stage}
projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}
projects/{project}/issues/{issue}
```

Navigating among a plan, its specs, rollout, stages, and tasks must not remount
the Plan Detail shell, clear its snapshot, restart its initial fetch, or reset
the user's phase disclosure state.

## Problem

The current router registers the plan root, specs collection, and spec detail as
sibling routes that each render `ProjectPlanDetailPage`. Moving from
`/plans/{plan}` to `/plans/{plan}/specs/{spec}` therefore replaces the route
component and recreates `PlanDetailStoreProvider`.

For a plan with a rollout, the new page fetches the same plan again and derives
Deploy as its default phase. This resets `activePhases` and collapses Changes,
which is the behavior reported in BYT-9913. The delayed collapse can look like a
polling update, but the underlying problem is route ownership and remounting.

The current URL model is also inconsistent:

- A spec is selected through a resource-shaped path.
- A stage or task is selected through `stageId` and `taskId` query parameters.
- Legacy rollout resource paths redirect back to the plan root.
- A plan-backed issue always redirects to Review, even after the plan lifecycle
  has advanced to Deploy.

This mixes resource identity with view state and makes selection, disclosure,
history, and polling unnecessarily coupled.

## Goals

1. Keep URLs aligned with Bytebase resource-name ownership.
2. Keep one Plan Detail shell alive for every navigation within the same plan.
3. Make spec, rollout, stage, task, and plan-backed issue entry points reliably
   deep-linkable.
4. Preserve user disclosure and per-resource UI state while the URL selection
   changes.
5. Prevent route-driven empty, stale, or flashing content during rapid
   navigation.
6. Preserve existing bookmarks through canonical redirects.

## Non-goals

- Changing backend, proto, or API resource names.
- Adding a new Plan, Spec, Rollout, Stage, Task, or Issue data model.
- Encoding every expanded phase, task card, filter, or drawer in the URL.
- Redesigning the Plan Detail visual layout.
- Changing which issue types belong on Plan Detail versus Issue Detail.

## Canonical route model

```text
/projects/{project}/plans/{plan}
├── specs
│   └── {spec}
└── rollout
    └── stages/{stage}
        └── tasks/{task}

/projects/{project}/issues/{issue}
```

| User intent | Canonical URL | Implied phase |
| --- | --- | --- |
| Open a plan using its lifecycle default | `/projects/p/plans/123` | Derived once on entry |
| Open the specs collection | `/projects/p/plans/123/specs` | Changes |
| Open a specific spec | `/projects/p/plans/123/specs/spec-2` | Changes |
| Open the rollout | `/projects/p/plans/123/rollout` | Deploy |
| Open a specific stage | `/projects/p/plans/123/rollout/stages/prod` | Deploy |
| Open a specific task | `/projects/p/plans/123/rollout/stages/prod/tasks/42` | Deploy |
| Open a specific issue | `/projects/p/issues/456` | Determined by issue workflow |
| Open an old phase bookmark | `/projects/p/plans/123?phase=review` | Legacy compatibility only |

`changes`, `review`, and `deploy` are UI phases, not resources. They must not be
introduced as path segments such as `/plans/{plan}/changes` or written as new
query state. Conversely, `specs`, `rollout`, `stages`, and `tasks` identify
plan-owned resources and must not be represented only by query parameters.

### Phase precedence

The matched resource path is authoritative:

1. A task, stage, or rollout path selects Deploy.
2. A specs collection or spec path selects Changes.
3. On the plan root, a valid legacy `phase` query selects that phase.
4. On the plan root without `phase`, the initial lifecycle state selects the
   default: a present, committed, or expected rollout -> Deploy; issue -> Review;
   otherwise Changes.

A resource path and a contradictory `phase` query must not coexist. For example:

```text
/plans/123/specs/spec-2?phase=deploy
```

The resource path wins and the router removes the redundant query with
`replace`, producing `/plans/123/specs/spec-2`.

The lifecycle default is derived only when entering a plan without an explicit
selection. Later polling updates do not change resource selection or refocus the
page. When a rollout materializes for a plan that is already open, Deploy is
revealed once without collapsing other phases, writing the URL, or scrolling.

## Route ownership

`plans/{plan}` is the route boundary that owns the Plan Detail page. Its nested
routes describe selection only.

Conceptually:

```tsx
{
  path: "plans/:planId",
  element: <PlanDetailRouteShell />,
  children: [
    { index: true },
    { path: "specs" },
    { path: "specs/:specId" },
    {
      path: "rollout",
      children: [
        { index: true },
        { path: "stages/:stageId" },
        { path: "stages/:stageId/tasks/:taskId" },
      ],
    },
  ],
}
```

The exact route configuration is an implementation detail, but the ownership
contract is not:

- `PlanDetailRouteShell` owns `PlanDetailStoreProvider`, the plan snapshot,
  polling, editing scopes, leave guards, phase disclosure, and the page layout.
- The shell identity is `{projectId}/{planId}`.
- Child matches only update `{phase, specId, stageId, taskId}`.
- Only a project or plan identity change may recreate the shell or run the
  initial plan fetch.
- Child routes must not introduce loaders or lazy page components that delay the
  selection commit or replace the shell.

Because Plan Detail renders all phases in one page, nested leaf routes do not
own separate page bodies. They provide route selection to the persistent shell.

## Issue entry

Issue is a project-level resource:

```text
projects/{project}/issues/{issue}
```

It must not be duplicated under a plan as
`/plans/{plan}/issues/{issue}`. The current Plan model has at most one linked
issue, so carrying both IDs after resolution would create invalid combinations
such as a plan paired with an unrelated issue.

When `/projects/{project}/issues/{issue}` is opened:

- A plan-backed draft review issue redirects with `replace` to
  `/projects/{project}/plans/{plan}`. Its snapshot naturally defaults to Review.
- A submitted schema/data change issue that belongs on Plan Detail redirects to
  `/projects/{project}/plans/{plan}`. The plan snapshot selects its current
  lifecycle phase.
- A valid phase already present on an old issue URL is preserved through the
  redirect. New issue links do not add one.
- Create-database, export, grant, and other workflows owned by Issue Detail stay
  on the issue route.
- A route-neutral activity anchor, such as a review comment ID, may be preserved
  through the redirect as a query or fragment. Spec, stage, and task selectors
  from the issue URL are not preserved.

This keeps the issue URL valid as an entry point while making the plan the
canonical identity of the Plan Detail surface. Generic list and shared issue
links follow lifecycle progress, while existing phase bookmarks and resource
links keep their narrower focus.

## Selection and disclosure contract

The URL identifies the user's primary focus. It does not serialize the entire
expanded/collapsed state of the page.

- Arriving at a spec route ensures Changes is expanded and selects that spec.
- Arriving at rollout, stage, or task ensures Deploy and the selected resource
  are visible.
- Arriving with a legacy `?phase=review` ensures Review is expanded.
- Selecting a resource may expand its owning phase but must not collapse other
  phases the user opened.
- Manually selecting, showing, or hiding a phase only changes local disclosure;
  it does not write the URL.
- Refreshing or externally opening a focused URL expands its implied phase
  again.

Within one plan, disclosure state and per-resource UI state remain owned by the
persistent store. Navigating to another plan starts with a new disclosure state.

Disclosure synchronization is keyed by semantic resource identity, not only by
the implied phase: `plan + phase`, `spec + specId`, `rollout`, `stage + stageId`,
or `task + taskId`. Legacy query selectors and their canonical resource paths
produce the same key, while secondary state such as line, task run, or fragment
does not affect it.

### History writes

- Explicit user selection of a spec, stage, or task uses `push`.
- User phase selection is local and creates no history or query state.
- Alias redirects, legacy URL migration, contradictory-query cleanup, and
  invalid-selection fallback use `replace`.
- Automatic default selection, automatic frontier-stage selection, and polling
  do not write the URL.
- Back and Forward restore the resource selection and ensure its phase is
  expanded without resetting the rest of the same-plan store.
- Same-plan resource writes prevent the router's default scroll reset so the
  persistent shell keeps its live position. Plan Detail does not save or replay
  scroll positions; history-based scroll restoration remains opt-in for list
  pages such as Issues and Plans.

### Editing guards

- A spec change that would discard an unsaved statement remains guarded. The
  tab and statement stay on the current spec if navigation is canceled.
- After navigation is accepted, the URL selection, tab highlight, and statement
  body update in one render commit.
- Stage navigation does not discard stage-local state because stage contents
  remain mounted; it must not show an unnecessary leave confirmation.

## Rendering and flicker contract

Changing the URL does not inherently cause flicker. Flicker occurs when a route
change remounts ownership, clears data, temporarily renders the wrong entity, or
lets asynchronous work settle out of order.

Plan Detail navigation must satisfy the following:

- Initial disclosure is committed before the first ready render. Same-plan
  resource arrivals expand their destination in a layout effect before paint;
  disclosure synchronization never writes the URL.
- Route-driven spec selection resets during render, not in a post-paint effect,
  so the previous spec cannot appear for an intermediate frame.
- The page header, phase rail, snapshot, and polling loop stay mounted for every
  same-plan child navigation.
- The page never returns to its initializing/invisible state for a spec, stage,
  or task selection.
- Spec tab highlight and spec body change together.
- A cached sheet statement is available synchronously on the first render of the
  selected spec. An uncached statement shows a stable loading state, never an
  empty "No data" flash or the previous spec's SQL.
- The spec body may be keyed by stable spec ID to reset spec-local state, but
  the surrounding Changes phase remains mounted and dimensionally stable.
- Stage contents remain mounted and are shown/hidden by selection so task-card
  expansion, filters, Monaco instances, and fetched logs survive stage switches.
- Rapid A -> B -> A navigation cannot allow an earlier route transition or
  asynchronous fetch to overwrite the latest selection.
- Polling preserves unchanged object identities and never changes selection.
  A newly materialized rollout may expand Deploy once, while subsequent polls
  preserve the user's disclosure choices.

## Edge cases

### Plan root

- A plan root with no explicit phase derives its default once on entry.
- If approval and plan-check gates have passed but rollout creation is still
  pending, the initial lifecycle default is Deploy. This state is committed
  before the first ready render and does not require a follow-up navigation.
- A newly appearing issue updates lifecycle data without changing the user's
  current focus. A newly materialized rollout reveals Deploy once without
  changing focus, URL, scroll, or other phase disclosure.
- A deleted but readable plan keeps its canonical URL and renders its deleted
  state.
- Plan-level not-found and permission-denied errors are handled by the parent
  shell.

### Spec

- A legacy `?phase=changes` bookmark shows the first available spec or the
  Changes empty state.
- An unknown spec ID is evaluated only after the plan snapshot is ready. The
  page reports that the spec no longer exists and replaces the URL with the plan
  root while retaining the already-visible Changes disclosure.
- Same-target specs remain distinguishable by stable spec ID; target is not a
  substitute for route identity.
- An unsaved spec uses client-local selection and has no fake resource path.
  After persistence, the route is replaced with the real spec path.
- Plan creation remains an action route such as `/plans/create`; it must not use
  a placeholder spec ID in the URL.

### Rollout

- If `plan.hasRollout`, issue state, or another committed signal proves that a
  rollout is expected but the rollout snapshot is not visible yet, a rollout,
  stage, or task URL is retained and shows a pending state while active polling
  continues.
- If no rollout exists and none is expected, `/plans/{plan}/rollout` is replaced
  with the plan root while retaining the already-visible Deploy disclosure.

### Stage and task

- Stage and task IDs are resolved within the current plan's rollout; they are not
  treated as globally unique.
- An unknown stage falls back with `replace` to the rollout route after the
  rollout has finished loading.
- An unknown task falls back to its valid stage route. If neither is valid, it
  falls back to the rollout route.
- If a task exists under a different stage from the URL, its full task resource
  name determines the correct stage and the URL is replaced with the canonical
  path.
- Task-run history remains secondary UI state, for example `?taskRunId={run}` or
  a drawer. It becomes a path resource only if Task Run receives a dedicated
  page-level contract.

### Query and fragment state

Queries and fragments may encode view details that do not replace resource
identity, such as:

```text
/plans/123/specs/spec-2?line=42&planCheckResult=abc
/plans/123?phase=review#comment-456  (legacy)
/plans/123/rollout/stages/prod/tasks/42?taskRunId=7
```

Navigation preserves only parameters owned by the destination view. It must not
blindly carry incompatible spec, stage, task, or phase selection across resource
boundaries.

## Compatibility and canonicalization

Existing links remain valid and normalize without committing the legacy URL.
Direct entries use `replace`; in-app navigation keeps the referring history
entry and commits only the canonical destination:

```text
/plans/{plan}/specs
-> unchanged compatibility resource-collection URL

/plans/{plan}/specs/{spec}
-> unchanged URL, matched beneath the persistent plan shell

/plans/{plan}?stageId={stage}
-> /plans/{plan}/rollout/stages/{stage}

/plans/{plan}?stageId={stage}&taskId={task}
-> /plans/{plan}/rollout/stages/{stage}/tasks/{task}

/plans/{plan}/rollout
-> unchanged URL, rendered by the persistent plan shell

/plans/{plan}/rollout/stages/{stage}
-> unchanged URL, rendered by the persistent plan shell

/plans/{plan}/rollout/stages/{stage}/tasks/{task}
-> unchanged URL, rendered by the persistent plan shell
```

Existing route-name constants for plan, spec, rollout, stage, and task should
remain stable where their resource meaning is unchanged. Compatibility concerns
must not preserve the current sibling-page ownership or query-only stage/task
selection.

## Rejected alternatives

### Phase paths

```text
/plans/{plan}/changes/{spec}
/plans/{plan}/review
/plans/{plan}/deploy/stages/{stage}
```

Rejected because Changes, Review, and Deploy are UI phases rather than Bytebase
resources, and the paths obscure the actual Spec and Rollout resource names.

### Query-only resource selection

```text
/plans/{plan}?specId={spec}&stageId={stage}&taskId={task}
```

Rejected because it permits contradictory selections, loses parent-resource
scope, and treats real resource identity as presentation state.

### Duplicate issue identity under a plan

```text
/plans/{plan}/issues/{issue}
```

Rejected because Issue is owned by Project, not Plan, and the current model does
not require two identities to select Review.

### Full sibling page routes

Rejected because rendering the complete Plan Detail page independently at the
plan, spec, and rollout routes recreates page ownership during an internal
selection and is the direct cause of BYT-9913.

## Acceptance criteria

1. Opening a plan root with an existing rollout, expanding Changes, and selecting
   a non-selected spec updates the URL to the spec resource while Changes remains
   expanded and the selected SQL is visible.
2. Spec, stage, and task navigation within a plan preserves the Plan Detail shell
   and does not repeat the initial plan fetch.
3. A generic plan-backed issue entry opens the current lifecycle phase; an
   existing `?phase=review` bookmark continues to open Review.
4. Rollout, stage, and task resource URLs are directly shareable and survive
   refresh.
5. Rapid resource switching and Back/Forward never show stale selection,
   collapse unrelated phases, or snap back to an older route.
6. Spec switching never flashes empty or previous-spec SQL; stage switching
   preserves task-card and editor state.
7. Invalid child resources degrade to the nearest valid route without turning a
   valid plan into a page-level 404.
8. Existing spec and rollout bookmarks remain valid, and query-based stage/task
   links canonicalize without adding extra browser history entries.
9. No runtime interaction, redirect, or invalid-resource fallback creates a new
   `phase` query.

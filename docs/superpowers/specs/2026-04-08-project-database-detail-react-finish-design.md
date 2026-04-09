# Project Database Detail React Finish

## Summary

Finish BYT-9154 by migrating the project-scoped database detail page itself to React, not just the changelog and revision leaf pages. The route will remain the same, but the page shell, tab navigation, header actions, and all tab panels will become React-owned.

The target is a React-first database detail page with behavior parity where it matters, plus targeted cleanup of state ownership and page structure. A thin compatibility wrapper is acceptable only for a stubborn leaf widget or drawer. Whole-panel Vue embedding is out of scope for the finished state.

## Goals

- Replace the project database detail route component with a React page.
- Keep the existing route path, route name, query behavior, and hash-driven tab navigation where feasible.
- Make React own:
  - page bootstrap and loading state
  - canonical project redirect behavior
  - warning banner, title, metadata, and header actions
  - tab state and hash synchronization
  - all five panels: overview, changelog, revision, catalog, and settings
- Migrate the smaller panels directly to React.
- Migrate the heavier overview and catalog panels into React-owned panels, allowing only leaf-level compatibility wrappers if a sub-widget is too expensive to port in one pass.
- Remove the old Vue route shell from active routing once the React page is in place.

## Non-Goals

- Do not preserve the exact Vue internal structure.
- Do not introduce a generic React-to-Vue bridging framework.
- Do not redesign the route shape or rename tabs.
- Do not broaden this work into unrelated schema-editor or sensitive-data migrations outside the database detail surface.

## Current State

The current route in `frontend/src/router/dashboard/projectV1.ts` still mounts `frontend/src/views/DatabaseDetail/DatabaseDetail.vue` for:

- `/projects/:projectId/instances/:instanceId/databases/:databaseName`

That Vue page currently owns:

- no-environment warning banner
- header title and metadata
- SQL editor, schema diagram, sync, export, transfer, and change-database actions
- hash-synced tabs:
  - `overview`
  - `changelog`
  - `revision`
  - `catalog`
  - `setting`

The changelog detail and revision detail leaf pages are already React, and `useProjectDatabaseDetail()` already exists as a shared React bootstrap hook. The unfinished work is the main database detail page and its tab surfaces.

## Design

### Route Ownership

Keep the existing route path and route name for project database detail, but change the route target from the Vue page to `ReactPageMount` with:

- `page: "ProjectDatabaseDetailPage"`
- the existing route params forwarded as props

This keeps upstream navigation stable:

- breadcrumbs and links that already target `PROJECT_V1_ROUTE_DATABASE_DETAIL`
- existing redirects and callers
- hash links such as `#revision` and `#catalog`

### Shared Bootstrap Layer

`useProjectDatabaseDetail()` remains the bootstrap entry point for project database detail surfaces. The React page should use it for:

- database fetch
- metadata prefetch
- readiness/loading state
- canonical project redirect
- derived flags such as `allowAlterSchema` and `isDefaultProject`

If the main page needs additional shared route state helpers, extend the hook narrowly rather than duplicating fetch or redirect logic in the page.

The React page becomes the only top-level owner of page orchestration. Panel components should receive resolved props and avoid reimplementing route normalization.

### Page Composition

Create `frontend/src/react/pages/project/ProjectDatabaseDetailPage.tsx` and split it into focused React units rather than a single large page file.

Recommended structure:

- `ProjectDatabaseDetailPage`
  - fetch/bootstrap
  - route/hash state
  - modal and drawer state
- database detail header section
  - title
  - environment and instance metadata
  - release metadata when present
  - SQL editor and schema diagram actions
  - sync/export/transfer/change-database actions
- tab model helpers
- one React component per panel:
  - overview
  - changelog
  - revision
  - catalog
  - settings

This keeps page concerns separate from panel concerns and gives the route a clear React-owned boundary.

### Tab and Hash Behavior

Keep the current tab identifiers:

- `overview`
- `changelog`
- `revision`
- `catalog`
- `setting`

Behavior:

- initialize selected tab from `location.hash`
- default to `overview` when the hash is empty or invalid
- update the hash when the selected tab changes
- preserve query parameters while updating the hash
- preserve the selected hash when canonical project redirects occur

Use the existing React tabs primitives in `frontend/src/react/components/ui/tabs.tsx`.

### Header and Action Behavior

The React page must preserve the current user-facing actions and gating behavior:

- no-environment warning banner that points users toward settings
- database name and resource name display
- environment and instance metadata
- release badge/value when present
- SQL editor button
- schema diagram button when alter-schema capability exists
- sync database action
- export schema action
- transfer project action
- change database action

Permission and feature checks should stay behaviorally equivalent to the Vue page, but the control flow should be simplified so the page, not the tabs, owns top-level action gating and modal state.

### Panel Strategy

#### Overview

The overview tab must become a React panel. React should own:

- schema selection state
- search/filter state
- section layout
- loading and empty handling
- permission gating

Do not accept a whole-panel Vue embed here.

Because this panel fans out into many existing table-oriented components, allow a narrow compatibility wrapper for a specific leaf widget if direct migration is disproportionately expensive. That wrapper must stay below the panel boundary. The panel itself remains React-owned and testable from React.

#### Changelog

Rewrite the changelog tab panel directly in React. It is small enough that a compatibility wrapper would add debt without saving meaningful effort.

Expected behavior:

- paged changelog list
- preserve the existing per-database session key behavior for pagination/table state
- same project permission gating for listing changelogs

#### Revision

Rewrite the revision tab panel directly in React.

Expected behavior:

- paged revision list
- import/create revision action
- refresh after create/delete
- same permission and workflow behavior as the current Vue panel

#### Catalog

The catalog tab must also become a React-owned panel. React should own:

- search state
- selected sensitive-column state
- grant-access button state
- permission and licensing checks
- grant-access drawer orchestration

As with overview, do not accept a whole-panel Vue embed. A narrow compatibility wrapper is acceptable only for a specific deeply-coupled child when direct migration of that child would materially expand scope beyond this issue.

#### Settings

Rewrite the settings tab directly in React.

Expected behavior:

- environment editing
- labels editing surface
- same update permissions and success notification behavior

This panel is small enough to migrate fully without a compatibility layer.

### Compatibility Rule

The accepted finish bar is:

- zero Vue page shell on the project database detail route
- zero Vue whole-panel embedding
- thin Vue compatibility wrappers allowed only for isolated leaf widgets or drawers that are too costly to port immediately

If a leaf wrapper is used, it should be implemented as a temporary compatibility seam with tightly scoped props. It should not become a reusable cross-framework abstraction.

### Cleanup

After the React page is wired in:

- stop routing the project database detail route to `frontend/src/views/DatabaseDetail/DatabaseDetail.vue`
- delete dead route glue if it is no longer used
- keep only the Vue code that still serves other non-migrated surfaces

The cleanup target is route ownership and obvious dead glue, not broad unrelated component deletion.

## Testing Strategy

Run the standard frontend verification flow after implementation:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- relevant `pnpm --dir frontend test` coverage for the new React page and panels

React-focused tests should cover:

- route bootstrap and loading behavior
- canonical project redirect behavior with hash and query preservation
- tab initialization from hash and hash updates on tab switch
- header action visibility and gating
- changelog panel paging behavior
- revision import flow and list refresh behavior
- catalog permission/licensing edge cases and grant-access enablement
- settings environment update behavior

For the heavier overview and catalog migrations, prefer behavior tests over implementation-detail tests. The main regression risk is user workflow parity, not internal hook structure.

## Alternatives Considered

### Full rewrite of every sub-widget into React in one pass

Rejected as the default approach because the overview and catalog surfaces fan out into many existing Vue components. That path is the cleanest theoretically, but it materially increases scope and risk for this issue.

### React page shell with several Vue whole-panel embeds

Rejected because it does not satisfy the chosen finish bar. It would move the route into React while leaving most of the real page behavior owned by Vue.

### Keep the Vue page and stop after changelog/revision detail migration

Rejected because it leaves BYT-9154 incomplete. The issue is about the database detail and catalog surface, not only the nested detail routes.

## Risks

- The overview panel may uncover more Vue-only dependencies than expected.
- The catalog panel has licensing, permission, and drawer flows that are easy to regress if migrated mechanically.
- Hash synchronization can regress deep links if routing logic is split across the page and panels.
- Header actions currently mix feature checks, permissions, and modal state; moving them into React needs careful parity checks.

## Rollout

Implement in one focused migration branch:

1. Add the React `ProjectDatabaseDetailPage`.
2. Swap the route to `ReactPageMount`.
3. Migrate header, tab, and hash behavior into React.
4. Port changelog, revision, and settings panels fully to React.
5. Port overview and catalog panels to React ownership, using leaf-level compatibility seams only for sub-widgets whose direct migration would materially expand scope beyond this issue.
6. Add regression tests for routing, tabs, header actions, and panel workflows.
7. Remove dead Vue route glue that is no longer needed.

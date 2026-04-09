# Database Detail React Migration Design

Date: 2026-04-09

## Summary

Migrate the project database detail page to a fully React-owned implementation by removing all page-local Vue bridge mounts and replacing them with native React panels and interactions.

This migration is driven by a production failure on the database detail page caused by provider mismatches inside Vue islands mounted from React. The target state is not a repaired bridge layer. The target state is a bridge-free database detail page.

## Problem

The current React page at `frontend/src/react/pages/project/ProjectDatabaseDetailPage.tsx` still mounts several legacy Vue components via `createApp(...)` bridge wrappers. These wrappers do not inherit the provider tree from the main Vue app root, which leads to runtime failures such as missing `n-config-provider` and missing `OverlayStackManager` context.

The immediate crash occurs in the overview tab, but the underlying issue affects the page architecture more broadly:

- React owns the route and tab shell
- Vue islands own meaningful page content
- each island mounts its own app boundary
- provider assumptions leak across those boundaries

This creates fragility during migration and slows removal of Vue from the page.

## Goals

- Remove all Vue bridge mounts from the project database detail page
- Keep the route, tabs, and top-level page ownership in React
- Preserve core user workflows for overview, changelog, revision, catalog, settings, and schema diagram entry
- Keep permission behavior and route synchronization intact
- Reuse existing stores, services, and router integration patterns already used by React pages

## Non-Goals

- Full app-wide removal of Vue
- Rewriting shared Vue stores during this task
- Creating a new generic Vue bridge helper
- Perfect visual parity where the old implementation adds migration cost without user value

## Scope

### In Scope

- `frontend/src/react/pages/project/ProjectDatabaseDetailPage.tsx`
- `frontend/src/react/pages/project/database-detail/DatabaseDetailHeader.tsx`
- `frontend/src/react/pages/project/database-detail/panels/DatabaseOverviewPanel.tsx`
- `frontend/src/react/pages/project/database-detail/panels/DatabaseCatalogPanel.tsx`
- `frontend/src/react/pages/project/database-detail/panels/DatabaseChangelogPanel.tsx`
- `frontend/src/react/pages/project/database-detail/panels/DatabaseRevisionPanel.tsx`
- schema diagram interaction on the database detail page
- page-specific tests for database detail React flows

### To Delete

- `frontend/src/react/pages/project/database-detail/legacy/DatabaseObjectExplorerBridge.tsx`
- `frontend/src/react/pages/project/database-detail/legacy/DatabaseOverviewInfoBridge.tsx`
- `frontend/src/react/pages/project/database-detail/legacy/SensitiveColumnTableBridge.tsx`
- `frontend/src/react/pages/project/database-detail/legacy/GrantAccessDrawerBridge.tsx`
- page-local `createApp(...)` mounts in database detail panels

## Existing Vue-backed Areas

The database detail page still depends on Vue-backed islands in these areas:

- Overview info
- Object explorer
- Sensitive column catalog table
- Grant access drawer
- Changelog table
- Revision table
- Create revision drawer
- Schema diagram button dialog content

Settings is already React and does not need migration.

## Proposed Approach

Use a full React rewrite for the database detail page with acceptable simplification for secondary flows.

This approach explicitly avoids introducing a better bridge layer. Any effort spent improving bridge composition would increase migration debt and preserve the same failure mode class.

### Why This Approach

- fixes the current crash by removing the failing boundary rather than patching it
- aligns with the migration direction
- reduces duplicated runtime stacks inside one page
- makes page testing simpler and more deterministic
- avoids future provider-context bugs for this page

## Component Design

### 1. Database Overview Panel

Replace the current bridge-driven overview panel with a React-native implementation.

Responsibilities:

- render overview info fields now provided by `DatabaseOverviewInfo.vue`
- render schema selector
- render object sections by engine:
  - tables
  - views
  - external tables
  - extensions
  - functions
  - sequences
  - streams
  - tasks
  - packages
- keep route query sync for selected schema
- keep local search state for tables and external tables

Implementation notes:

- use `useVueState` with existing stores rather than creating a new React data layer
- replace Vue data tables with React tables or list views using existing React UI primitives
- implement table-detail interactions in React instead of reusing `TableDetailDrawer.vue`

### 2. Database Catalog Panel

Replace the sensitive data table and grant-access bridge flow with React-native equivalents.

Responsibilities:

- flatten database catalog into display rows
- support search filtering
- support row selection
- support delete/removal actions where permitted
- support opening grant-access flow for selected rows

Implementation notes:

- preserve existing masking and exemption policy semantics
- allow a simpler visual table design if selection and actions remain clear
- use existing React dialogs and buttons instead of Vue drawers

### 3. Database Changelog Panel

Replace the Vue changelog table mount with a native React table.

Responsibilities:

- display paged changelog rows
- keep `usePagedData` behavior
- keep pagination footer behavior
- keep current changelog row navigation behavior

### 4. Database Revision Panel

Replace both the Vue revision table mount and the Vue create-revision drawer mount.

Responsibilities:

- display paged revision rows
- support deletion refresh behavior
- support opening an import/create revision flow
- refresh list after create/import

Implementation notes:

- the import flow may use a React dialog rather than reproducing the exact Vue drawer
- functional parity matters more than visual parity

### 5. Schema Diagram Interaction

Replace `SchemaDiagramButtonBridge.tsx` with a React-owned interaction.

Preferred outcome:

- open a React dialog
- render a React-native schema visualization or schema browser

Acceptable fallback for this migration:

- ship a simpler React representation of schema structure if the existing schema diagram is too expensive to port immediately

The schema diagram must no longer depend on a Vue app mounted inside the React page.

## State and Data Flow

Top-level page responsibilities remain in `ProjectDatabaseDetailPage.tsx` and `useProjectDatabaseDetail.ts`:

- fetch database
- fetch schema metadata
- normalize canonical project route
- maintain selected tab from route hash

Panel-level responsibilities:

- derive their own display state from shared stores
- manage local search, selection, and dialog state
- call store actions and service clients directly

This preserves the current repo pattern used by React pages:

- direct store access
- `useVueState` for reactive Vue store reads
- no extra bridge store layer

## Permission Model

Keep existing permission checks:

- route-level permissions remain unchanged
- `ComponentPermissionGuard` continues to guard changelog, revision, and catalog sections where used
- project-level checks inside panels remain based on `hasProjectPermissionV2`

No permission logic should move into ad hoc UI-only checks that diverge from the current page behavior.

## Routing Behavior

Must preserve:

- tab state encoded in route hash
- schema selection encoded in route query
- canonical project redirect when database project differs from route project
- existing database detail route name and params

## Simplifications Allowed

These are acceptable in this migration if they reduce complexity without breaking primary workflows:

- use a simpler schema diagram experience
- use a simpler catalog table layout
- use a React dialog instead of a drawer for revision import or grant-access flows
- reduce non-essential visual polish inherited from Vue components

These are not acceptable:

- losing schema selection
- losing ability to inspect database objects
- losing changelog or revision listing
- losing catalog search and grant-access entry
- reintroducing page-local Vue mounts

## Risks

### Functional Risk

The overview panel covers many engine-specific branches. Reimplementing it risks silent regressions for engines not covered by the immediate reproduction case.

Mitigation:

- keep logic close to current Vue implementation structure
- derive section lists from the same stores and utility functions
- add focused tests for engine-driven rendering conditions where feasible

### Migration Risk

Trying to preserve exact Vue component behavior can expand scope rapidly.

Mitigation:

- prefer acceptable simplification for secondary flows
- preserve behaviorally important outcomes, not internal component parity

### Test Risk

Current tests mock bridge components. Those tests will need to move up to actual React panel assertions.

Mitigation:

- rewrite tests around rendered React behavior rather than bridge call assertions

## Implementation Phases

### Phase 1

Replace overview tab and remove:

- `DatabaseOverviewInfoBridge`
- `DatabaseObjectExplorerBridge`

This fixes the current page crash.

### Phase 2

Replace changelog and revision panels with native React tables and dialogs.

### Phase 3

Replace catalog sensitive-column table and grant-access flow.

### Phase 4

Replace schema diagram bridge and remove remaining page-local Vue mounts.

### Phase 5

Delete unused `legacy/` files and update tests.

## Verification

Minimum verification for this migration:

- database detail overview tab renders without console provider errors
- schema selection updates route query correctly
- changelog tab renders and paginates
- revision tab renders and import/create flow opens and refreshes
- catalog tab renders, filters, selects rows, and opens grant-access flow
- settings tab still works
- no remaining imports of page-local database-detail bridge files

Required commands after implementation:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- targeted `pnpm --dir frontend test`

## Success Criteria

The migration is complete when:

- the database detail page is fully usable
- the page no longer mounts Vue apps from React
- the original provider-context crash is impossible because the failing architecture has been removed
- tests cover the new React-native behavior rather than bridge invocation details

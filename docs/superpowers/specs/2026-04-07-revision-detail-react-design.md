# Revision Detail React Migration

## Summary

Migrate the database revision detail route from Vue to React with a dependency-first sequence: convert the shared components the page depends on first, then migrate the revision detail panel, then switch the route page to React.

## Motivation

`RevisionDetail.vue` is only a thin route wrapper. The actual page behavior lives in Vue shared components, especially `RevisionDetailPanel.vue` and `TaskRunLogViewer.vue`. Migrating only the route wrapper would leave the page effectively Vue-backed and make later cleanup harder.

The migration should follow the same rule we use elsewhere in the React transition: when a page depends on shared Vue components that are on the migration path, migrate those shared pieces first so the page can consume React-native dependencies instead of introducing temporary bridges.

## Design

### Migration Order

1. Migrate shared React primitives needed by revision detail.
2. Migrate the shared revision detail panel onto those React primitives.
3. Migrate the route page and router wiring last.

This keeps dependency direction clean and avoids a React page that still embeds Vue-owned UI.

### Shared Components To Migrate First

#### Task Run Log Viewer

Migrate the shared task run log viewer used by revision detail from:

- `frontend/src/components/RolloutV1/components/TaskRun/TaskRunLogViewer/TaskRunLogViewer.vue`
- `frontend/src/components/RolloutV1/components/TaskRun/TaskRunLogViewer/useTaskRunLogData.ts`

to a React implementation under the React component tree. The React version should preserve the current behavior:

- fetch task run log entries from the rollout API
- fetch associated sheet data for regular tasks and release tasks
- support single-replica, multi-replica, and release-file grouping views
- keep expand/collapse behavior and summary metadata

The underlying data-fetching logic can stay close to the current composable, but should be exposed through React hooks instead of Vue refs/watchers.

#### Read-Only Monaco Wrapper

Revision detail needs a reusable read-only SQL viewer in React. Add a small shared React wrapper around the existing Monaco editor infrastructure for this pattern instead of inlining editor setup into the page.

Responsibilities:

- mount a Monaco editor instance in React
- support read-only content display
- support auto-height with the same constraints used by the current page
- be reusable by other detail pages that still rely on Vue-only Monaco wrappers today

#### Shared Utility-Level Reuse

Reuse existing non-UI utilities directly instead of rewriting them:

- revision store access via `useRevisionStore`
- sheet fetching via `sheetServiceClientConnect`
- formatting helpers such as `bytesToString`, `formatAbsoluteDateTime`, and revision/task link utilities

No new data layer should be introduced for this migration.

### Revision Detail Panel

Replace the shared Vue panel:

- `frontend/src/components/Revision/RevisionDetailPanel.vue`

with a React-native shared panel component that keeps the current behavior:

- fetch the revision by name
- fetch raw sheet content for the statement when present
- show loading state while revision/sheet data is being resolved
- show revision version, type, and created time
- show task run logs through the new React `TaskRunLogViewer`
- show the SQL statement in the shared read-only Monaco wrapper
- preserve copy-to-clipboard behavior for the statement

The React version should stay close to the existing UI, but cleanup is acceptable where it improves clarity without changing the page workflow.

The `frontend/src/components/Revision/index.ts` entrypoint should stop exporting the Vue panel for revision detail usage. If the shared revision table remains Vue for now, keep only the exports that are still valid and move the React panel to a React-facing export surface.

### Route Page Migration

Replace the current route page:

- `frontend/src/views/DatabaseDetail/RevisionDetail.vue`

with a React page mounted through `frontend/src/react/ReactPageMount.vue`, following the existing route migration pattern used elsewhere in `projectV1.ts`.

The new page should:

- accept the same route-derived props: `project`, `instance`, `database`, `revisionId`
- load the database resource by name
- render the breadcrumb trail for databases -> database -> revisions -> current revision
- render a loading spinner until the database is ready
- render the shared React revision detail panel once ready

### Router Wiring

Update `frontend/src/router/dashboard/projectV1.ts` so the revision detail child route uses `ReactPageMount.vue` with a page name such as `DatabaseRevisionDetailPage`.

The route path, route name, permissions, and prop mapping should remain unchanged. Only the mounted implementation changes.

### File Structure

Expected additions:

- `frontend/src/react/pages/project/DatabaseRevisionDetailPage.tsx`
- React task run log viewer files under `frontend/src/react/components/`
- a shared React read-only Monaco viewer component
- a React revision detail panel component

Expected updates:

- `frontend/src/router/dashboard/projectV1.ts`
- `frontend/src/components/Revision/index.ts` or adjacent exports, depending on final ownership split

No `frontend/src/react/mount.ts` change should be required as long as the new page follows the existing `frontend/src/react/pages/project/*.tsx` naming and export convention.

Expected removals after migration is complete and no longer referenced:

- `frontend/src/views/DatabaseDetail/RevisionDetail.vue`
- `frontend/src/components/Revision/RevisionDetailPanel.vue`

The shared Vue task run log viewer should only be removed if all remaining consumers are migrated or can be switched safely in the same change. Otherwise, keep the Vue implementation temporarily and introduce the React version alongside it.

### Behavior Changes

User-facing behavior should remain substantially the same:

- same route and breadcrumb flow
- same revision metadata and statement display
- same embedded task run logs
- same copy action

Acceptable cleanup:

- modest internal component reshaping
- extracting shared React pieces instead of keeping page-local logic
- small presentational adjustments needed to match existing React patterns

Not acceptable:

- dropping embedded task run logs
- replacing the shared-component dependency chain with a page-local shortcut
- introducing a Vue bridge inside the React page

### Testing

Verify the migration at three levels.

#### Shared Components

- task run log viewer renders populated logs for a task run with entries
- task run log viewer handles missing sheet content without crashing
- read-only Monaco wrapper renders statement content and remains non-editable

#### Revision Detail Panel

- loading state appears while fetching
- populated revision renders version, metadata, task log viewer, and statement
- revision without sheet content shows an empty statement area without throwing

#### Route Page

- route mounts from `projectV1.ts` through `ReactPageMount.vue`
- breadcrumb links navigate to databases list, database detail, and revisions anchor
- page waits for database readiness before rendering the panel

Required frontend verification commands after code changes:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

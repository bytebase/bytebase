# Project Database Detail React Migration

## Summary

Migrate the project-scoped database changelog detail route from Vue to React, remove the Vue-only `ProjectDatabaseLayout.vue` parent route, and introduce a shared React database-detail bootstrap layer that both changelog detail and revision detail use.

This does **not** migrate the main database detail page in the same change. That page remains Vue temporarily, but it must stop depending on the removed parent layout.

## Motivation

The current project database-detail route tree has an awkward boundary:

- `frontend/src/views/project/ProjectDatabaseLayout.vue` is a Vue parent route that only exists to:
  - pass route params through `<router-view>`
  - provide `DatabaseDetailContext`
  - prefetch schema metadata
- `frontend/src/views/DatabaseDetail/ChangelogDetail.vue` is a Vue route shell around `frontend/src/components/Changelog/ChangelogDetail.vue`
- `frontend/src/react/pages/project/DatabaseRevisionDetailPage.tsx` is already React, but it reimplements database fetch and breadcrumb bootstrapping on its own

That split causes three problems:

1. **React migration is blocked by a Vue parent route.** As long as `ProjectDatabaseLayout.vue` owns the route subtree, any child page migration is incomplete.
2. **Database-detail bootstrap logic is fragmented.** Database fetch, schema warmup, project normalization, and derived flags are spread across Vue layout code, Vue injection context, and individual pages.
3. **Future migration cost stays high.** The later React migration of the main database detail page would either duplicate this logic again or inherit a temporary architecture.

## Goals

- Remove `frontend/src/views/project/ProjectDatabaseLayout.vue` from active routing.
- Migrate the changelog detail route to React.
- Refactor revision detail to use the same React database-detail bootstrap path.
- Move database-detail bootstrap concerns into a reusable React unit:
  - canonical resource naming
  - database fetch
  - schema metadata warmup
  - derived detail flags
  - project-route normalization
- Keep the existing main database detail page working while it remains Vue.

## Non-Goals

- Do not migrate `frontend/src/views/DatabaseDetail/DatabaseDetail.vue` to React in this change.
- Do not redesign database detail UX.
- Do not attempt cross-framework context sharing between React and Vue.
- Do not migrate unrelated database panels.

## Design

### Route Structure

Replace the nested database-detail route subtree in `frontend/src/router/dashboard/projectV1.ts` with explicit leaf routes:

| Route | Target |
|-------|--------|
| `/projects/:projectId/instances/:instanceId/databases/:databaseName` | existing Vue database detail page |
| `/projects/:projectId/instances/:instanceId/databases/:databaseName/changelogs/:changelogId` | new React changelog detail page |
| `/projects/:projectId/instances/:instanceId/databases/:databaseName/revisions/:revisionId` | existing React revision detail page, refactored onto shared hook |

The current Vue parent route entry that points at `ProjectDatabaseLayout.vue` is removed. Each leaf route becomes directly responsible for its own page component and props.

### Shared React Database-Detail Layer

Add a focused React-only unit for project-scoped database-detail bootstrapping. A hook is preferred over a global provider:

`useProjectDatabaseDetail({ projectId, instanceId, databaseName })`

Responsibilities:

- Build canonical resource names for:
  - project: `projects/${projectId}`
  - instance: `instances/${instanceId}`
  - database: `instances/${instanceId}/databases/${databaseName}`
- Fetch the database record from the v1 database store.
- Warm database metadata via the schema store, matching the current layout behavior.
- Ignore schema metadata permission failures so pages can still render.
- Derive:
  - `database`
  - `project`
  - `allowAlterSchema`
  - `isDefaultProject`
  - `loading`
  - `ready`
- Normalize the route when the fetched database belongs to a different project than the URL.

This hook replaces the responsibilities currently split between:

- `frontend/src/views/project/ProjectDatabaseLayout.vue`
- `frontend/src/components/Database/context.ts`
- ad hoc fetch logic in `frontend/src/react/pages/project/DatabaseRevisionDetailPage.tsx`

### Route Normalization

The shared React layer owns project-route correction for React pages.

Behavior:

- If the fetched database belongs to a project different from `projectId` in the URL, redirect to the canonical route that matches `database.project`.
- Preserve the current leaf intent:
  - database detail stays database detail
  - changelog detail stays changelog detail
  - revision detail stays revision detail
- Preserve relevant route state:
  - hash for database detail
  - query for database detail
  - dynamic IDs like `changelogId` and `revisionId`

This matches the existing correction behavior in `frontend/src/views/DatabaseDetail/DatabaseDetail.vue`, but moves the responsibility into the new React bootstrap path for React pages.

### React Changelog Detail Page

Add a React page parallel to `DatabaseRevisionDetailPage.tsx`:

- `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`

The page owns:

- breadcrumb rendering
- loading shell
- changelog fetch
- previous changelog fetch for diff source
- rollback button routing
- readonly schema snapshot rendering
- task run log rendering

The page must preserve the current Vue behavior in `frontend/src/views/DatabaseDetail/ChangelogDetail.vue` and `frontend/src/components/Changelog/ChangelogDetail.vue`:

- show changelog title when present
- show status and created time
- show task run logs for `DONE` and `FAILED` changelogs
- link to the full task route when `taskRun` is present
- show schema snapshot
- show diff by default using the previous changelog schema as the original content
- allow rollback only when:
  - project is not default
  - user has alter-schema capability
  - database engine supports schema rollback
  - changelog status is `DONE`

This is a functional port, not a redesign.

### Revision Detail Refactor

Refactor `frontend/src/react/pages/project/DatabaseRevisionDetailPage.tsx` to consume `useProjectDatabaseDetail(...)` instead of directly fetching the database itself.

What changes:

- Database fetch and metadata warmup come from the shared hook.
- Route normalization comes from the shared hook.
- Breadcrumb props continue to be derived from the same route params.

What does not change:

- The revision detail page remains a React leaf page.
- The revision detail panel implementation and user-visible behavior stay the same.

### Vue Database Detail Coexistence

The main database detail page remains Vue in this PR, but it can no longer depend on `ProjectDatabaseLayout.vue`.

Coexistence rule:

- React pages use the new React bootstrap layer.
- Vue database detail becomes self-bootstrapping for its own needs.

Concretely:

- Route the database detail leaf directly to the existing Vue component.
- Move the minimum required bootstrap logic into the Vue database detail path so it still has:
  - access to database detail context
  - metadata warmup
  - current project normalization

The temporary duplication is acceptable because it is isolated to the still-Vue page and disappears when that page migrates.

### File Structure

Expected new or modified files:

- Create: `frontend/src/react/pages/project/DatabaseChangelogDetailPage.tsx`
- Create: `frontend/src/react/pages/project/database-detail/useProjectDatabaseDetail.ts`
- Modify: `frontend/src/react/pages/project/DatabaseRevisionDetailPage.tsx`
- Modify: `frontend/src/router/dashboard/projectV1.ts`
- Modify: `frontend/src/views/DatabaseDetail/DatabaseDetail.vue`
- Delete or stop routing to: `frontend/src/views/project/ProjectDatabaseLayout.vue`

### Testing Strategy

For frontend verification in this repo, run:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- targeted frontend tests for any new React logic with meaningful pure or component-level coverage

Recommended test focus:

- shared hook handles successful bootstrap
- shared hook tolerates metadata permission failure
- shared hook computes canonical redirect correctly
- changelog detail diff uses previous changelog schema
- revision detail still renders correctly through the shared hook

## Alternatives Considered

### Keep the Vue parent layout and migrate only the changelog leaf

Rejected because it leaves `ProjectDatabaseLayout.vue` in the critical path and preserves the Vue injection boundary that future React work needs to remove anyway.

### Wrap the existing Vue changelog component in a React shell

Rejected because it creates a temporary mixed-framework composition for a page that must become fully React later. It saves little and leaves the hard boundary unresolved.

### Migrate the entire database-detail subtree now

Rejected for scope. The main database detail page is much larger and touches more workflows than the routes targeted here.

## Risks

- Flattening the route tree can accidentally change route props or navigation behavior.
- Moving bootstrap logic out of `ProjectDatabaseLayout.vue` can break the still-Vue database detail page if its replacement bootstrap path is incomplete.
- Redirect behavior must preserve hashes, queries, and dynamic IDs carefully to avoid subtle UX regressions.
- The changelog detail port includes Monaco/diff/log rendering, so visual parity needs attention.

## Rollout

Land this as one focused migration PR:

1. Introduce shared React database-detail hook.
2. Migrate changelog detail page to React.
3. Refactor revision detail onto the shared hook.
4. Flatten the routes and remove the Vue parent layout from active routing.
5. Make the Vue database detail page self-sufficient until its later migration.

# React Zustand Resource Store Migration Phase 2 Design

## Context

Phase 1 moved the first protobuf resource stores into the React app store and added a guard that prevents React code from re-importing those legacy Pinia stores. The next useful batch is the remaining resource stores with meaningful React usage but bounded API surfaces:

- `User`
- `Role`
- `Release`
- `Revision`
- `Changelog`
- `ProjectWebhook`

This phase keeps the Vue Pinia stores in place for Vue consumers. The goal is to migrate React consumers to `useAppStore` and add guard coverage so React does not regress.

## Goals

- Add Zustand slices under `frontend/src/react/stores/app/` for the six resources.
- Preserve the observable behavior of the current Pinia stores for React callers.
- Migrate all React non-test consumers of the six legacy stores.
- Update affected tests to mock `@/react/stores/app` instead of legacy Pinia stores.
- Extend `frontend/src/react/no-legacy-vue-deps.test.ts` to block these stores from React code.

## Non-Goals

- Do not delete legacy Pinia stores.
- Do not migrate Vue consumers.
- Do not include `Issue`, `Plan`, `Rollout`, `Policy`, `ProjectIamPolicy`, worksheet, or SQL editor legacy store migrations in this phase.
- Do not refactor UI flows beyond the minimum needed for store migration.

## Current React Consumer Surface

Approximate non-test React consumer counts on current `main`:

- `useUserStore`: 20 files
- `useRoleStore`: 10 files
- `useReleaseStore`: 5 files
- `useRevisionStore`: 3 files
- `useChangelogStore`: 3 files
- `useProjectWebhookV1Store`: 3 files

The batch is large enough to be a single focused resource-store PR, but not mixed with lifecycle resources such as `Issue`, `Plan`, and `Rollout`.

## Architecture

Add these files:

- `frontend/src/react/stores/app/user.ts`
- `frontend/src/react/stores/app/role.ts`
- `frontend/src/react/stores/app/release.ts`
- `frontend/src/react/stores/app/revision.ts`
- `frontend/src/react/stores/app/changelog.ts`
- `frontend/src/react/stores/app/projectWebhook.ts`

Update these shared app-store files:

- `frontend/src/react/stores/app/types.ts`
- `frontend/src/react/stores/app/index.ts`
- `frontend/src/react/stores/app/index.test.ts`
- `frontend/src/react/hooks/useAppState.ts`

Each slice should follow the existing app-store style:

- Store resource maps as plain objects keyed by resource name.
- Store request maps for deduped fetches where the Pinia store currently caches requests.
- Keep list functions as async actions that return API response payloads and upsert entities into cache.
- Use `useAppStore(selector)` for reactive reads in React components.
- Use selected actions from `useAppStore(selector)` for component event handlers.
- Reserve `useAppStore.getState()` for imperative contexts where a hook is not legal.

## Slice Responsibilities

### User

The user slice should cover:

- current app user compatibility where needed by user consumers
- `listUsers`
- `fetchUser`
- `batchGetOrFetchUsers`
- `getOrFetchUserByIdentifier`
- `getUserByIdentifier`
- `createUser`
- `updateUser`
- `updateEmail`
- `archiveUser`
- `restoreUser`

It should preserve the special `allUsersUser()` cache entry and unknown-user fallback behavior used by selectors and tables. It should also invalidate permission cache behavior through app-store permission state if needed, rather than importing the legacy permission Pinia store.

### Role

The role slice should cover:

- `listRoles`
- `getRoleByName`
- `upsertRole`
- `deleteRole`

It should keep the current `allowMissing: true` update behavior and remove deleted roles from the cached list.

### Release

The release slice should cover:

- `listReleasesByProject`
- `fetchRelease`
- `getReleasesByProject`
- `getReleaseByName`
- `updateRelease`
- `deleteRelease`
- `undeleteRelease`

It should preserve deleted release state updates and the unknown-release fallback expected by current UI.

### Revision

The revision slice should cover:

- `listRevisionsByDatabase`
- `listAllRevisionsByDatabase`
- `fetchRevision`
- `getRevisionsByDatabase`
- `getRevisionByName`
- `deleteRevision`

It should keep database-scoped list behavior and support the import-revision and revision panel flows.

### Changelog

The changelog slice should cover:

- `clearChangelogCache`
- `listChangelogs`
- `getOrFetchChangelogListOfDatabase`
- `changelogListByDatabase`
- `fetchChangelog`
- `getOrFetchChangelogByName`
- `getChangelogByName`
- `fetchPreviousChangelog`

The cache key must include `ChangelogView`, preserving the current BASIC/FULL lookup behavior. `getChangelogByName(name)` without a view should continue to prefer FULL and fall back to BASIC.

### ProjectWebhook

The project webhook slice should cover:

- `getProjectWebhookFromProjectById`
- `createProjectWebhook`
- `updateProjectWebhook`
- `deleteProjectWebhook`
- `testProjectWebhook`

The mutation APIs return updated `Project` responses from the backend. Consumer migration should keep the current project refresh/update behavior unchanged.

## Consumer Migration

Migrate all React non-test consumers of these stores:

- User consumers in account selectors, member/user settings, auth profile/setup/reset flows, issue and audit tables, approval flows, SQL editor display labels, masking exemption, access grant pages, and plan/data export dashboards.
- Role consumers in role selectors, roles settings, request-role sheets, custom approval utilities, environment settings, role grant details, setup, and SQL editor request query actions.
- Release consumers in release dashboard/detail, import revision, issue statement section, and task-run log data.
- Revision consumers in revision panel, database revision panel, and import revision.
- Changelog consumers in sync schema, database changelog list/detail, and changelog panels.
- Project webhook consumers in webhook list/detail/form pages.

Tests that currently mock the legacy stores should be updated to mock `@/react/stores/app`.

## Guardrails

Extend `frontend/src/react/no-legacy-vue-deps.test.ts` with a Phase 2 guard that fails if React code imports or references:

- `useUserStore`
- `useRoleStore`
- `useReleaseStore`
- `useRevisionStore`
- `useChangelogStore`
- `useProjectWebhookV1Store`
- direct module imports for the corresponding Pinia store files

The guard should exclude itself and can allow Vue-side Pinia stores to remain untouched.

## Error Handling

Keep current behavior:

- Silent fetches should continue to use `silentContextKey`.
- User not-found paths should preserve existing unknown-user fallback semantics.
- Role deletion should keep graceful-request behavior, either by preserving the same wrapper behavior or by matching its notification/error semantics in callers.
- Changelog invalid or unknown IDs should return `undefined` instead of issuing invalid backend requests.

## Testing

Use focused tests while developing, then run the standard frontend verification:

- `pnpm --dir frontend exec vitest run src/react/stores/app/index.test.ts src/react/no-legacy-vue-deps.test.ts`
- Focused tests for touched consumers that already have coverage.
- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

## Implementation Notes

- Migrate the slices first, then migrate consumers by resource group.
- Prefer small mechanical patches and grep after each group to ensure no React consumer remains.
- Use `useAppStore(selector)` for reactive values; do not wrap `useAppStore.getState()` in `useVueState`.
- Keep generated artifacts out of the commit unless verification commands intentionally update them and they are required.

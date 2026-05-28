# React Zustand Resource Migration Phase 1 Design

## Summary

Migrate the first high-value set of React-used API resources from legacy
Vue/Pinia stores into the React `useAppStore` Zustand store:

- `Group`
- `ServiceAccount`
- `WorkloadIdentity`
- `IdentityProvider`
- `AccessGrant`

This phase targets resources that already have React consumers but still import
Pinia stores directly. The goal is to establish the resource-slice pattern and
remove the most common access/account Pinia dependencies from React without
forcing a full store migration.

## Motivation

The React application already uses `useAppStore` for core resources such as
workspace, project, instance, database, sheet, IAM, and subscription state.
Several React pages still depend on legacy Pinia stores for access/account
resources:

- Settings pages: users, groups, service accounts, workload identities, IDPs.
- Auth pages: sign-in and account settings IDP reads.
- Project access surfaces: access grants and SQL Editor access panes.
- Shared account utilities and selectors used by React components.

Keeping those imports in React makes migration progress uneven and keeps the
React tree coupled to Vue reactivity. Phase 1 moves the most actively used
access/account resources into Zustand while leaving legacy Pinia stores in place
for any Vue or not-yet-migrated consumers.

## Scope

### In Scope

Add React app-store slices for:

1. `Group`
2. `ServiceAccount`
3. `WorkloadIdentity`
4. `IdentityProvider`
5. `AccessGrant`

Switch React consumers of those stores to `useAppStore` where the behavior is
equivalent.

Add tests for cache behavior, request de-duplication, filter construction, and
mutation cache updates.

Add regression coverage so migrated React surfaces do not re-import these five
Pinia stores.

### Out of Scope

Do not migrate these resources in phase 1:

- `Issue`
- `IssueComment`
- `Plan`
- `Rollout`
- `Release`
- `DatabaseCatalog`
- `DBSchema`
- `Policy`
- `Setting`
- `WorkspaceApprovalSetting`
- `SQLReview`
- `ProjectWebhook`
- `AuditLog`
- SQL execution/export helpers
- AI conversation plugin store

Do not delete the legacy Pinia modules in this phase. Deletion should happen
only after a follow-up audit confirms no Vue or React consumers remain.

Do not convert unrelated UI, route, or design-system code while migrating the
store calls.

## Architecture

Use the existing `frontend/src/react/stores/app/` structure. Each migrated
resource gets its own slice file and type section:

```text
frontend/src/react/stores/app/
  accessGrant.ts
  group.ts
  identityProvider.ts
  serviceAccount.ts
  workloadIdentity.ts
  types.ts
  index.ts
```

`index.ts` composes the new slices into `createAppStore()` alongside existing
slices.

The store remains a bounded React app store. It does not become a generic API
client or a global resource framework. Resource-specific behavior stays in the
resource slice, while only small repeatable utilities may be shared.

## Store Contract

Each resource slice follows the current app-store pattern:

- Store entities in plain records keyed by full resource name.
- Store in-flight get requests by resource name when the resource supports
  get-by-name.
- Return API responses from list/search methods and also upsert returned
  entities into the cache.
- Update the cache after successful create/update/undelete operations.
- Remove or replace cached entries after successful delete/revoke operations
  according to the API response semantics.
- Keep pagination tokens in page components, not in the shared store.

React components should use:

- `useAppStore(selector)` for reactive reads.
- `useAppStore(selector)` for actions used inside components.
- `useAppStore.getState().action()` only for imperative code outside render
  subscriptions.

React code must not use `useVueState(() => useAppStore.getState()...)` as a
subscription bridge.

## Resource Details

### Group

Port behavior from `frontend/src/store/modules/v1/group.ts`.

State:

- `groupsByName: Record<string, Group>`
- `groupRequests: Record<string, Promise<Group | undefined>>`
- `groupErrorsByName: Record<string, Error | undefined>`

Actions and helpers:

- `listGroups(params)`
- `fetchGroup(name, silent?)`
- `batchFetchGroups(names)`
- `batchGetOrFetchGroups(names)`
- `getGroupByIdentifier(id)`
- `createGroup(group)`
- `updateGroup(group)`
- `deleteGroup(name)`
- `ensureGroupIdentifier(id)`
- `buildGroupFilter(filter)`

Behavior to preserve:

- Normalize identifiers to `groups/{email}`.
- Batch fetch unique group names.
- Keep list results sorted at the selector/consumer level when needed.
- Preserve permission-sensitive empty-list behavior if the existing caller
  depends on it.

### ServiceAccount

Port behavior from `frontend/src/store/modules/serviceAccount.ts`.

State:

- `serviceAccountsByName: Record<string, ServiceAccount>`
- `serviceAccountRequests: Record<string, Promise<ServiceAccount | undefined>>`

Actions and helpers:

- `listServiceAccounts(params)`
- `fetchServiceAccount(name, silent?)`
- `getServiceAccount(name)`
- `createServiceAccount(parent, serviceAccount)`
- `updateServiceAccount(serviceAccount, updateMask?)`
- `deleteServiceAccount(name)`
- `undeleteServiceAccount(name)`
- `ensureServiceAccountFullName(identifier)`
- `buildAccountListFilter(filter)`

Behavior to preserve:

- Return a stable fallback object from `getServiceAccount()` for unknown names.
- Normalize identifiers before cache lookup.
- Update cache after create, update, and undelete.
- Mark deleted service accounts according to the API response or remove cache
  entries only when the API does not return a usable updated resource.

### WorkloadIdentity

Port behavior from `frontend/src/store/modules/workloadIdentity.ts`.

State:

- `workloadIdentitiesByName: Record<string, WorkloadIdentity>`
- `workloadIdentityRequests: Record<string, Promise<WorkloadIdentity | undefined>>`

Actions and helpers:

- `listWorkloadIdentities(params)`
- `fetchWorkloadIdentity(name, silent?)`
- `getWorkloadIdentity(name)`
- `createWorkloadIdentity(parent, workloadIdentity)`
- `updateWorkloadIdentity(workloadIdentity, updateMask?)`
- `deleteWorkloadIdentity(name)`
- `undeleteWorkloadIdentity(name)`
- `ensureWorkloadIdentityFullName(identifier)`

Behavior to preserve:

- Reuse the account-list filter shape shared with service accounts.
- Return a stable fallback user-shaped object where existing React account
  utilities expect one.
- Preserve subject-pattern parsing and form behavior in UI code; the store only
  owns API resource state.

### IdentityProvider

Port behavior from `frontend/src/store/modules/idp.ts`.

State:

- `identityProvidersByName: Record<string, IdentityProvider>`
- `identityProviderList: IdentityProvider[]` or a selector over
  `identityProvidersByName`
- `identityProviderRequests: Record<string, Promise<IdentityProvider | undefined>>`

Actions:

- `listIdentityProviders(parent?)`
- `fetchIdentityProvider(name, silent?)`
- `getIdentityProvider(name)`
- `createIdentityProvider(identityProvider)`
- `updateIdentityProvider(identityProvider, updateMask?)`
- `deleteIdentityProvider(name)`

Behavior to preserve:

- Listing replaces the known list for IDP pages while also populating by-name
  cache.
- Patch/update should preserve existing field-mask semantics.
- Delete removes the cached provider after success.
- Sign-in page reads must stay reactive so IDP buttons update after the list
  load completes.

### AccessGrant

Port behavior from `frontend/src/store/modules/accessGrant.ts`.

State:

- `accessGrantsByName: Record<string, AccessGrant>`
- `accessGrantRequests: Record<string, Promise<AccessGrant | undefined>>`

Actions and helpers:

- `fetchAccessGrant(name)`
- `searchMyAccessGrants(params)`
- `listAccessGrants(params)`
- `createAccessGrant(parent, accessGrant)`
- `activateAccessGrant(name)`
- `revokeAccessGrant(name)`
- `buildAccessGrantFilter(filter)`

Behavior to preserve:

- Preserve ACTIVE versus EXPIRED filter semantics based on `expire_time`.
- Upsert search/list/get results into cache.
- Update cache after create/activate/revoke based on returned resources.
- Keep page-level pagination outside the shared store.

## Consumer Migration

Switch React consumers in this order:

1. Settings/member pages:
   - `frontend/src/react/pages/settings/GroupsPage.tsx`
   - `frontend/src/react/pages/settings/UsersPage.tsx`
   - `frontend/src/react/pages/settings/ServiceAccountsPage.tsx`
   - `frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx`
2. IDP surfaces:
   - `frontend/src/react/pages/auth/SigninPage.tsx`
   - `frontend/src/react/pages/settings/IDPsPage.tsx`
   - `frontend/src/react/pages/settings/IDPDetailPage.tsx`
   - `frontend/src/react/pages/settings/general/AccountSection.tsx`
3. Access-grant surfaces:
   - `frontend/src/react/pages/project/ProjectAccessGrantsPage.tsx`
   - `frontend/src/react/pages/project/issue-detail/components/IssueDetailAccessGrantDetails.tsx`
   - `frontend/src/react/components/sql-editor/AccessPane.tsx`
4. Shared account components and helpers:
   - `frontend/src/react/components/AccountMultiSelect.tsx`
   - `frontend/src/react/components/CreateWorkloadIdentitySheet.tsx`
   - React-used member/IAM helper paths that currently read these Pinia stores.

If a shared utility is used by both Vue and React, avoid forcing it to depend on
`useAppStore`. Prefer either a React-specific helper or a small pure helper that
accepts resource lookup functions as arguments.

## Testing Strategy

### Store Tests

Extend `frontend/src/react/stores/app/index.test.ts` unless the file becomes too
large. If it becomes hard to read, split tests by slice while keeping the same
mock style.

Required coverage:

- Slice is composed into `createAppStore()`.
- Get-by-name request de-duplication.
- List/search upserts returned resources into cache.
- Create/update/undelete writes returned resource into cache.
- Delete/revoke updates cache consistently.
- Filter builder output for meaningful combinations.
- Name normalizers handle full resource names and bare identifiers.

### React Import Guard

Add or extend no-legacy tests so migrated React files do not import:

- `useGroupStore`
- `useServiceAccountStore`
- `useWorkloadIdentityStore`
- `useIdentityProviderStore`
- `useAccessGrantStore`

The guard should target only files migrated in phase 1 so unrelated legacy
surfaces do not block the PR.

### Verification Commands

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend exec vitest run frontend/src/react/stores/app/index.test.ts
pnpm --dir frontend test
git diff --check
```

If CI static checks fail while local checks pass, reproduce with:

```bash
pnpm --dir frontend exec eslint src --no-cache
```

## Rollout Strategy

Use multiple focused PRs if implementation gets large:

1. Store slices and tests.
2. Settings/member consumer migration.
3. IDP consumer migration.
4. Access-grant consumer migration.
5. Shared helper cleanup and no-legacy guards.

Do not delete Pinia stores until a later cleanup confirms no Vue or React
consumers remain.

## Risks

- Shared utilities imported by both Vue and React may accidentally become
  coupled to `useAppStore`. Keep shared utilities pure or split React-specific
  wrappers.
- IDP list behavior affects sign-in. The sign-in page must remain reactive after
  async list loading.
- Service account and workload identity stores return fallback user-shaped
  resources today. Preserve this where account selectors rely on it.
- Access grant filters include time-sensitive ACTIVE/EXPIRED logic. Test these
  filters with fixed time inputs or deterministic assertions.
- A broad consumer migration can touch many React pages. Keep each PR narrowly
  staged if review size grows.

## Success Criteria

- The five phase 1 resources are available through `useAppStore`.
- Migrated React consumers no longer import the five legacy Pinia stores.
- Existing behavior for list/get/create/update/delete flows remains unchanged.
- Store tests cover cache, request, mutation, and filter behavior.
- Frontend fix, check, type-check, focused tests, full tests, and `git diff
  --check` pass before implementation is considered complete.

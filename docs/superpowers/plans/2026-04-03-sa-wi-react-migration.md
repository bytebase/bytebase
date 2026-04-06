# Service Account & Workload Identity React Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate ServiceAccountPanel and WorkloadIdentityPanel from Vue to React, following the same patterns used in the UsersPage/MembersPage/GroupsPage migration.

**Architecture:** Create two new React pages (`ServiceAccountsPage.tsx`, `WorkloadIdentitiesPage.tsx`) that reuse the existing `shared/usePagedData`, `shared/UserAvatar`, and `shared/RoleMultiSelect`. Each page contains its own table and drawer components inline. Update routes to point to React pages. Delete old Vue components.

**Tech Stack:** React, TypeScript, Vue Router, Pinia stores via `useVueState`

---

### Task 1: Create ServiceAccountsPage.tsx

**Files:**
- Create: `frontend/src/react/pages/settings/ServiceAccountsPage.tsx`

- [ ] **Step 1: Create the page**

Port `ServiceAccountPanel.vue` (231 lines) and `CreateServiceAccountDrawer.vue` (317 lines) into a single React page file. Follow the exact same pattern as `UsersPage.tsx`:

- Import and use `usePagedData`, `PagedTableFooter`, `UserAvatar`, `RoleMultiSelect` from `shared/`
- `ServiceAccountTable` — renders active service accounts with avatar, title, email, operations (view/deactivate/restore)
- `CreateServiceAccountDrawer` — form with title, email (with domain suffix), roles (create mode only)
- Main `ServiceAccountsPage` function — action bar with "Add Service Account" button, active table, inactive toggle, inactive table, drawer

Key stores:
- `useServiceAccountStore()` — `listServiceAccounts`, `createServiceAccount`, `updateServiceAccount`, `deleteServiceAccount`, `undeleteServiceAccount`, `getServiceAccount`
- `useWorkspaceV1Store()` — `patchIamPolicy` for role assignment
- `useActuatorV1Store()` — `workspaceResourceName`

Permissions:
- `bb.serviceAccounts.create` — Add button
- `bb.serviceAccounts.update` / `bb.serviceAccounts.delete` / `bb.serviceAccounts.undelete` — row operations
- Route-level: `bb.serviceAccounts.list`

Email suffix: `getServiceAccountSuffix()` from `@/types` (returns `service.bytebase.com` or project-specific)

The panel currently receives a `project` prop but in the workspace settings context it's always undefined (workspace-level). Only handle workspace-level for this migration.

- [ ] **Step 2: Type check**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/settings/ServiceAccountsPage.tsx
git commit -m "feat(frontend): migrate ServiceAccountPanel to React"
```

---

### Task 2: Create WorkloadIdentitiesPage.tsx

**Files:**
- Create: `frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx`

- [ ] **Step 1: Create the page**

Port `WorkloadIdentityPanel.vue` (233 lines) and `CreateWorkloadIdentityDrawer.vue` (743 lines) into a single React page file. Same pattern as Task 1 but with a much more complex drawer.

`WorkloadIdentityTable` — same pattern as ServiceAccountTable.

`CreateWorkloadIdentityDrawer` — complex form with:
- Title, Email (with domain suffix)
- Platform select (GitHub Actions / GitLab CI) — `WorkloadIdentityConfig_ProviderType`
- Owner/Group field (label changes per platform)
- Repository/Project field (label/hint changes per platform)
- Allowed Branches/Tags dropdown (GitLab only, options: all/branch/tag)
- Branch/Tag field (GitHub always, GitLab when specific ref selected)
- Advanced Settings (collapsible):
  - Issuer URL / GitLab URL
  - Audience
  - Subject Pattern
- Roles (create mode only)

**Critical: Bidirectional subject pattern binding:**
- When owner/repo/branch change → auto-compute `subjectPattern` via `computedSubjectPattern`
- When `subjectPattern` changes → reverse-parse into owner/repo/branch via `parseWorkloadIdentitySubjectPattern` from `@/utils`
- Use refs as flags to prevent circular updates (same pattern as Vue's `isUpdatingFromPattern`/`isUpdatingFromFields`)

GitHub pattern: `repo:owner/repo:ref:refs/heads/branch`
GitLab pattern: `project_path:owner/project:ref_type:branch:ref:name`

Platform change resets issuerUrl to preset, clears branch, resets refType to "all".

Key stores:
- `useWorkloadIdentityStore()` — `listWorkloadIdentities`, `createWorkloadIdentity`, `updateWorkloadIdentity`, `deleteWorkloadIdentity`, `undeleteWorkloadIdentity`, `getWorkloadIdentity`
- `useWorkspaceV1Store()` — `patchIamPolicy`
- `useActuatorV1Store()` — `workspaceResourceName`

Import `parseWorkloadIdentitySubjectPattern`, `getWorkloadIdentityProviderText` from `@/utils`.
Import `WorkloadIdentityConfig_ProviderType`, `WorkloadIdentityConfigSchema`, `WorkloadIdentitySchema` from proto types.

Permissions: same pattern as ServiceAccounts but with `bb.workloadIdentities.*`

- [ ] **Step 2: Type check**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx
git commit -m "feat(frontend): migrate WorkloadIdentityPanel to React"
```

---

### Task 3: Update routes

**Files:**
- Modify: `frontend/src/router/dashboard/workspace.ts:320-340`

- [ ] **Step 1: Update Service Accounts route**

Change `component` from Vue `ServiceAccountPanel.vue` to React `ReactPageMount.vue` with `page: "ServiceAccountsPage"`:

```typescript
{
  path: "service-accounts",
  name: WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
  meta: {
    title: () => t("settings.members.service-accounts"),
    requiredPermissionList: () => ["bb.serviceAccounts.list"],
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: () => ({ page: "ServiceAccountsPage" }),
},
```

- [ ] **Step 2: Update Workload Identities route**

Same pattern:

```typescript
{
  path: "workload-identities",
  name: WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
  meta: {
    title: () => t("settings.members.workload-identities"),
    requiredPermissionList: () => ["bb.workloadIdentities.list"],
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: () => ({ page: "WorkloadIdentitiesPage" }),
},
```

- [ ] **Step 3: Type check and commit**

```bash
pnpm --dir frontend type-check
git add frontend/src/router/dashboard/workspace.ts
git commit -m "feat(frontend): update routes for React SA and WI pages"
```

---

### Task 4: Lint, fix, verify, and cleanup

- [ ] **Step 1: Run frontend fix and check**

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
```

- [ ] **Step 2: Commit any fixes**

```bash
git add -u
git commit -m "fix(frontend): lint and i18n fixes for SA/WI migration"
```

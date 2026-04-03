# Users Page Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the 3,489-line `UsersPage.tsx` monolith into three independent pages (Users, Members, Groups) with separate routes and sidebar entries.

**Architecture:** Extract shared utilities (`usePagedData`, `PagedTableFooter`, `AADSyncDrawer`) into a `shared/` directory, then create `MembersPage.tsx` and `GroupsPage.tsx` as standalone pages. Update routes and sidebar to expose all three as independent entries. Members becomes visible in both SaaS and non-SaaS modes.

**Tech Stack:** React, TypeScript, Vue Router (routes/sidebar are still Vue), Pinia stores via `useVueState`

**Spec:** `docs/superpowers/specs/2026-04-03-users-page-split-design.md`

---

### Task 1: Extract shared utilities

**Files:**
- Create: `frontend/src/react/pages/settings/shared/usePagedData.ts`
- Create: `frontend/src/react/pages/settings/shared/AADSyncDrawer.tsx`
- Modify: `frontend/src/react/pages/settings/UsersPage.tsx`

- [ ] **Step 1: Create `shared/usePagedData.ts`**

Extract `usePagedData` hook (lines 118-241), `PagedTableFooter` component (lines 247-296), and their imports from `UsersPage.tsx`. The file should export both `usePagedData` and `PagedTableFooter`.

Required imports to include:
```typescript
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { getPageSizeOptions, useSessionPageSize } from "@/react/hooks/useSessionPageSize";
import { cn } from "@/react/lib/utils";
```

Also export the `PagedDataResult` type (the return type of `usePagedData`) so consumers can reference it.

- [ ] **Step 2: Create `shared/AADSyncDrawer.tsx`**

Extract `AADSyncDrawer` (lines 2098-2284) and its imports. This drawer is used by both Users and Groups pages.

Key imports it needs:
```typescript
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { useActuatorV1Store, useSettingV1Store } from "@/store";
```

Read lines 2098-2289 carefully for the full set of imports needed.

- [ ] **Step 3: Update `UsersPage.tsx` to import from shared**

Replace the inline `usePagedData`, `PagedTableFooter`, and `AADSyncDrawer` definitions with imports:
```typescript
import { usePagedData, PagedTableFooter } from "./shared/usePagedData";
import { AADSyncDrawer } from "./shared/AADSyncDrawer";
```

Remove the extracted code blocks (lines 114-296 and 2094-2284) and their now-unused imports.

- [ ] **Step 4: Verify UsersPage still works**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: No type errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/settings/shared/
git add frontend/src/react/pages/settings/UsersPage.tsx
git commit -m "refactor(frontend): extract shared usePagedData and AADSyncDrawer from UsersPage"
```

---

### Task 2: Create `MembersPage.tsx`

**Files:**
- Create: `frontend/src/react/pages/settings/MembersPage.tsx`
- Modify: `frontend/src/react/pages/settings/UsersPage.tsx` (remove members code)

- [ ] **Step 1: Create `MembersPage.tsx`**

Extract these components from `UsersPage.tsx` into a new standalone page:
- `MemberTable` (lines 2304-2472)
- `MemberTableByRole` (lines 2478-2627)
- `EditMemberRoleDrawer` (lines 2633-2815)

**Important:** The exported function name must be `MembersPage` (matching the filename) for `mount.ts` auto-discovery via `import.meta.glob("./pages/settings/*.tsx")`.

Create a new `MembersPage` export function containing the members-specific state and UI from the main `UsersPage` function. This includes:
- `memberSearchText`, `memberViewTab`, `selectedMembers`, `showEditMemberDrawer`, `editingMember` state
- `memberBindings` computed value (lines 2868-2879)
- `canSetIamPolicy` permission check (lines 2881-2883)
- `handleRevokeSelected` (lines 2885-2913)
- `handleMemberUpdateBinding` (lines 2915-2918)
- `handleMemberRevokeBinding` (lines 2920-2933)
- The Members tab content (lines 3229-3260 for action bar, lines 3363-3404 for tab panel)

The page should NOT have tabs — it directly renders the member view with sub-tabs ("View by Members" / "View by Roles").

Required imports — pull from the UsersPage imports list:
```typescript
import type { MemberBinding } from "@/components/Member/types";
import { getMemberBindings } from "@/components/Member/utils";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/react/components/ui/tabs";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useCurrentUserV1, useRoleStore, useWorkspaceV1Store } from "@/store";
import { AccountType, ALL_USERS_USER_EMAIL, getAccountTypeByEmail, getUserEmailInBinding, userBindingPrefix } from "@/types";
import { PRESET_WORKSPACE_ROLES, PresetRoleType } from "@/types/iam";
import { displayRoleDescription, displayRoleTitle, hasWorkspacePermissionV2, sortRoles } from "@/utils";
```

Read the extracted components carefully for any additional imports (lucide icons, etc.).

- [ ] **Step 2: Remove members code from `UsersPage.tsx`**

Remove from `UsersPage.tsx`:
- `MemberTable`, `MemberTableByRole`, `EditMemberRoleDrawer` function definitions
- Members-related state variables from the main `UsersPage` function (`memberSearchText`, `memberViewTab`, `selectedMembers`, `showEditMemberDrawer`, `editingMember`)
- `memberBindings`, `canSetIamPolicy`, `handleRevokeSelected`, `handleMemberUpdateBinding`, `handleMemberRevokeBinding`
- The MEMBERS tab trigger (lines 3111-3114)
- The MEMBERS tab action bar (lines 3229-3260)
- The MEMBERS tab panel (lines 3363-3404)
- The `EditMemberRoleDrawer` render at bottom of UsersPage
- The `TabValue` type — change to just `"USERS" | "GROUPS"` (or remove tabs entirely if only USERS remains after Task 3)
- Remove `isSaaSMode` conditional around the USERS tab trigger (it's always shown now since Members is separate)
- Remove unused imports (`getMemberBindings`, `MemberBinding`, `useWorkspaceV1Store`, etc.)

- [ ] **Step 3: Verify type checking passes**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/settings/MembersPage.tsx
git add frontend/src/react/pages/settings/UsersPage.tsx
git commit -m "refactor(frontend): extract MembersPage from UsersPage"
```

---

### Task 3: Create `GroupsPage.tsx`

**Files:**
- Create: `frontend/src/react/pages/settings/GroupsPage.tsx`
- Modify: `frontend/src/react/pages/settings/UsersPage.tsx` (remove groups code)

- [ ] **Step 1: Create `GroupsPage.tsx`**

Extract these components from `UsersPage.tsx`:
- `GroupTable` (lines 673-845)
- `GroupRow` (lines 847-984)
- `RoleMultiSelect` (lines 990-1186)
- `extractUserTitle` (lines 1187-1190)
- `normalizeMemberIdentifier` (lines 1600-1605)
- `deduplicateMembers` (lines 1607-1622)
- `CreateGroupDrawer` (lines 1624-2092)

**Important:** The exported function name must be `GroupsPage` (matching the filename) for `mount.ts` auto-discovery.

Create a new `GroupsPage` export function containing the groups-specific state and UI:
- `groupSearchText` state
- `showCreateGroupDrawer`, `editingGroup` state
- `showAadSyncDrawer` state
- `hasUserGroupFeature`, `workspaceDomains`, `hasDirectorySyncFeature`, `canAccessSettings` computed values
- `groupPaged` data (using imported `usePagedData`)
- Deep-link support for `?name=groups/...` query param (lines 3002-3024)
- The GROUPS tab action bar (lines 3157-3228)
- The GROUPS tab panel content (lines 3406-3433)

Import shared utilities:
```typescript
import { usePagedData, PagedTableFooter } from "./shared/usePagedData";
import { AADSyncDrawer } from "./shared/AADSyncDrawer";
```

The page should NOT have tabs — it directly renders the groups table with action bar.

- [ ] **Step 2: Remove groups code from `UsersPage.tsx`**

Remove from `UsersPage.tsx`:
- `GroupTable`, `GroupRow`, `RoleMultiSelect`, `extractUserTitle`, `normalizeMemberIdentifier`, `deduplicateMembers`, `CreateGroupDrawer` function definitions
- Groups-related state from main function (`groupSearchText`, `showCreateGroupDrawer`, `editingGroup`)
- `hasUserGroupFeature`, `workspaceDomains` computed values (if only used by groups)
- `groupPaged` data
- Deep-link `?name=groups/...` effect (lines 3002-3024)
- The GROUPS tab trigger, action bar, and panel
- The `CreateGroupDrawer` render at bottom
- The `TabValue` type, `getInitialTab` function, and `handleTabChange` — no more tabs
- The hash sync effects (lines 3029-3043) — `window.location.hash = tab` and `hashchange` listener
- Remove all tab-related JSX (`<Tabs>`, `<TabsList>`, `<TabsTrigger>`, `<TabsPanel>`)
- Remove now-unused imports (`Tabs`, `TabsList`, `TabsPanel`, `TabsTrigger`, etc.)

**Keep in `UsersPage`:**
- Its own `showAadSyncDrawer` state and `<AADSyncDrawer>` render (imported from shared) — the Directory Sync button on the Users page still needs it
- `hasDirectorySyncFeature`, `canAccessSettings` — still used by the Directory Sync button

**Update cross-page navigation:**
- The `UserTable` `onGroupSelected` callback (currently calls `setTab("GROUPS")` + opens drawer) must instead navigate to the Groups page: `router.push({ name: WORKSPACE_ROUTE_GROUPS, query: { name: group.name } })`. Import `WORKSPACE_ROUTE_GROUPS` from `workspaceRoutes`.

After this, `UsersPage.tsx` should contain only:
- Imports (from shared + own deps)
- `UserTable` (lines 302-667) and its helpers (`UserGroupsCell`, `HighlightText`)
- `CreateUserDrawer` (lines 1192-1592)
- The main `UsersPage` function with only user-related state and UI
- No tabs — just the user table, search, add user button, directory sync button, inactive users section

- [ ] **Step 3: Verify type checking passes**

Run:
```bash
pnpm --dir frontend type-check
```

Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/settings/GroupsPage.tsx
git add frontend/src/react/pages/settings/UsersPage.tsx
git commit -m "refactor(frontend): extract GroupsPage from UsersPage"
```

---

### Task 4: Add routes and route constant

**Files:**
- Modify: `frontend/src/router/dashboard/workspaceRoutes.ts:28`
- Modify: `frontend/src/router/dashboard/workspace.ts:310-354`

- [ ] **Step 1: Add `WORKSPACE_ROUTE_GROUPS` constant**

In `frontend/src/router/dashboard/workspaceRoutes.ts`, add after line 28 (`WORKSPACE_ROUTE_MEMBERS`):

```typescript
export const WORKSPACE_ROUTE_GROUPS = "workspace.groups";
```

- [ ] **Step 2: Update Users route with SaaS guard**

In `frontend/src/router/dashboard/workspace.ts`, update the Users route (lines 310-318) to add permission, title, and a `beforeEnter` guard that redirects SaaS users to Members:

```typescript
{
  path: "users",
  name: WORKSPACE_ROUTE_USERS,
  meta: {
    title: () => t("common.users"),
    requiredPermissionList: () => ["bb.users.list"],
  },
  beforeEnter: (_to, _from, next) => {
    const actuatorStore = useActuatorV1Store();
    if (actuatorStore.isSaaSMode) {
      next({ name: WORKSPACE_ROUTE_MEMBERS, replace: true });
    } else {
      next();
    }
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: () => ({ page: "UsersPage" }),
},
```

Import `useActuatorV1Store` at the top of the file if not already imported.

- [ ] **Step 3: Update Members route**

Update the Members route (lines 342-354) to use React instead of Vue:

```typescript
{
  path: "members",
  name: WORKSPACE_ROUTE_MEMBERS,
  meta: {
    title: () => t("settings.sidebar.members"),
    requiredPermissionList: () => ["bb.workspaces.getIamPolicy"],
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: () => ({ page: "MembersPage" }),
},
```

- [ ] **Step 4: Add Groups route**

Add a new route after the Members route:

```typescript
{
  path: "groups",
  name: WORKSPACE_ROUTE_GROUPS,
  meta: {
    title: () => t("settings.members.groups.self"),
    requiredPermissionList: () => ["bb.groups.list"],
  },
  component: () => import("@/react/ReactPageMount.vue"),
  props: () => ({ page: "GroupsPage" }),
},
```

Import `WORKSPACE_ROUTE_GROUPS` at the top of `workspace.ts` alongside the other route constants.

- [ ] **Step 5: Verify type checking passes**

Run:
```bash
pnpm --dir frontend type-check
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/router/dashboard/workspaceRoutes.ts
git add frontend/src/router/dashboard/workspace.ts
git commit -m "feat(frontend): add routes for separate Members and Groups pages"
```

---

### Task 5: Update sidebar navigation

**Files:**
- Modify: `frontend/src/utils/useDashboardSidebar.ts:128-152`

- [ ] **Step 1: Update sidebar entries**

In `frontend/src/utils/useDashboardSidebar.ts`, replace lines 128-152 (the Users and Members entries in the IAM and Admin section):

**Users entry** (line 128-134) — add `hide: isSaaSMode` and fix title:
```typescript
{
  title: t("common.users"),
  name: WORKSPACE_ROUTE_USERS,
  type: "route",
  hide: actuatorStore.isSaaSMode,
},
```

**Members entry** (lines 147-152) — remove `hide`:
```typescript
{
  title: t("settings.sidebar.members"),
  name: WORKSPACE_ROUTE_MEMBERS,
  type: "route",
},
```

**Groups entry** — add new entry after Members:
```typescript
{
  title: t("settings.members.groups.self"),
  name: WORKSPACE_ROUTE_GROUPS,
  type: "route",
},
```

Import `WORKSPACE_ROUTE_GROUPS` at the top of the file alongside the other route constants.

- [ ] **Step 2: Verify type checking passes**

Run:
```bash
pnpm --dir frontend type-check
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/utils/useDashboardSidebar.ts
git commit -m "feat(frontend): update sidebar with separate Users, Members, Groups entries"
```

---

### Task 6: Delete dead Vue components

**Files:**
- Delete: `frontend/src/views/SettingWorkspaceMembers.vue`

- [ ] **Step 1: Verify no other imports of `SettingWorkspaceMembers.vue`**

Search for imports of `SettingWorkspaceMembers` across the codebase. The only reference should be the old route in `workspace.ts` which was already updated in Task 4.

```bash
rg "SettingWorkspaceMembers" frontend/src/
```

If there are other references, update them before deleting.

- [ ] **Step 2: Delete the file**

```bash
rm frontend/src/views/SettingWorkspaceMembers.vue
```

- [ ] **Step 3: Check if `WorkspaceMembers.vue` can be deleted**

Search for imports of `WorkspaceMembers` from `@/components/Member/Views/`:
```bash
rg "WorkspaceMembers" frontend/src/
```

If `SettingWorkspaceMembers.vue` was its only consumer, delete `frontend/src/components/Member/Views/WorkspaceMembers.vue` too.

- [ ] **Step 4: Verify type checking passes**

Run:
```bash
pnpm --dir frontend type-check
```

- [ ] **Step 5: Commit**

```bash
git add -u
git commit -m "chore(frontend): remove dead Vue members components"
```

---

### Task 7: Lint, fix, and verify

**Files:** All modified/created files

- [ ] **Step 1: Run frontend fix (ESLint + Biome)**

```bash
pnpm --dir frontend fix
```

Fix any reported issues.

- [ ] **Step 2: Run frontend check**

```bash
pnpm --dir frontend check
```

Expected: No errors.

- [ ] **Step 3: Run type check**

```bash
pnpm --dir frontend type-check
```

Expected: No errors.

- [ ] **Step 4: Run frontend tests**

```bash
pnpm --dir frontend test
```

Expected: All tests pass.

- [ ] **Step 5: Commit any fixes**

```bash
git add -u
git commit -m "fix(frontend): lint and format fixes for users page split"
```

---

### Task 8: Manual smoke test checklist

No automated tests exist for this page. Verify manually:

- [ ] **Non-SaaS mode:**
  - Sidebar shows "Users", "Members", "Groups" as separate entries under IAM and Admin
  - `/settings/users` — shows user table with add/search/inactive users
  - `/settings/members` — shows member list with grant/revoke, view by members/roles sub-tabs
  - `/settings/groups` — shows groups table with add/search, expandable member lists
  - Directory Sync button works on both Users and Groups pages

- [ ] **SaaS mode:**
  - Sidebar does NOT show "Users" entry
  - Sidebar shows "Members" and "Groups"
  - `/settings/members` — works same as non-SaaS
  - `/settings/groups` — works same as non-SaaS
  - `/settings/users` (direct URL) — redirects to `/settings/members`

- [ ] **Cross-page navigation:**
  - On Users page, clicking a group badge in the user table navigates to `/settings/groups` and opens the group

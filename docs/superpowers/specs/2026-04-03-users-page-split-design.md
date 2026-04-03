# Users Page Split: Three Independent Pages

## Summary

Split the monolithic `UsersPage.tsx` (3,489 lines) into three independent pages with separate routes and sidebar entries: Users, Members, and Groups.

## Motivation

PR #19829 migrated the `/users` page from Vue to React but consolidated three distinct concepts (user accounts, workspace IAM, groups) into a single tabbed page. This created two problems:

1. **Incorrect tab visibility**: The React version shows the Members tab only in SaaS mode, but the Vue version had a separate "Members" sidebar entry visible in non-SaaS mode (and hidden in SaaS). The Members functionality (workspace IAM policy management) should be available in both modes.
2. **Monolith file**: A single 3,489-line file mixing unrelated concerns.

## Design

### Route & Sidebar Structure

| Route | Sidebar Entry | Component | Visibility |
|-------|--------------|-----------|------------|
| `/settings/users` | "Users" | `UsersPage.tsx` | Non-SaaS only |
| `/settings/members` | "Members" | `MembersPage.tsx` | Always |
| `/settings/groups` | "Groups" | `GroupsPage.tsx` | Always |

- Reuse existing route constants `WORKSPACE_ROUTE_USERS` and `WORKSPACE_ROUTE_MEMBERS`
- Add new `WORKSPACE_ROUTE_GROUPS` constant
- Sidebar in `useDashboardSidebar.ts`:
  - Users entry: **add** `hide: isSaaSMode` (currently always visible with a dynamic title)
  - Members entry: **remove** `hide: isSaaSMode` (currently hidden in SaaS; make always visible)
  - Groups entry: **add** new sidebar entry (always visible)

### File Structure

```
frontend/src/react/pages/settings/
  UsersPage.tsx          # User account CRUD
  MembersPage.tsx        # Workspace IAM policy
  GroupsPage.tsx         # Group management
  shared/
    usePagedData.ts      # Pagination hook + PagedTableFooter component
    AADSyncDrawer.tsx    # Entra/AAD directory sync drawer (used by Users + Groups)
```

### Page Contents

**UsersPage.tsx** (non-SaaS only, ~800 lines):
- User count limit alert (shown when remaining count <= 3)
- Active users table with search + pagination
- "Add User" button, "Directory Sync" button
- Expandable "Show Inactive Users" section with its own search + pagination
- `CreateUserDrawer` for create/edit user accounts

**MembersPage.tsx** (always shown, ~700 lines):
- Two sub-tabs: "View by Members" / "View by Roles"
- Member search input
- Bulk "Revoke Access" button (in Members view), "Grant Access" button
- `MemberTable` — flat list with checkboxes for bulk operations
- `MemberTableByRole` — expandable sections grouped by role
- `EditMemberRoleDrawer` for granting/updating member roles
- No SaaS-specific logic; identical behavior in both modes

**GroupsPage.tsx** (always shown, ~800 lines):
- Groups table with search + pagination + expandable member lists
- "Add Group" button (with workspace domain restriction tooltip)
- "Directory Sync" button
- `CreateGroupDrawer` for create/edit groups
- Deep-link support for `?name=groups/...` query parameter

### Shared Code

**`usePagedData.ts`**: Extract the `usePagedData` hook and `PagedTableFooter` component. Used by all three pages for consistent pagination behavior.

**`AADSyncDrawer.tsx`**: Extract the Entra/AAD directory sync drawer. Used by both Users and Groups pages.

### Route Configuration Changes

In `frontend/src/router/dashboard/workspace.ts`:
- Keep existing Users route, update component to `ReactPageMount.vue` with `UsersPage`
- Keep existing Members route (line 342-354), update component from Vue `SettingWorkspaceMembers.vue` to React `ReactPageMount.vue` with `MembersPage`
- Add new Groups route at path `"groups"` with `WORKSPACE_ROUTE_GROUPS` constant, component `ReactPageMount.vue` with `GroupsPage`

In `frontend/src/router/dashboard/workspaceRoutes.ts`:
- Add `WORKSPACE_ROUTE_GROUPS` constant

### Route Guards

- `/settings/users`: Add route guard to redirect SaaS users to `/settings/members` (sidebar hides it, but direct URL access should redirect)
- `/settings/members` and `/settings/groups`: No guard needed, always accessible

### Route Permissions

- Users route: `requiredPermissionList: ["bb.users.list"]`
- Members route: `requiredPermissionList: ["bb.workspaces.getIamPolicy"]` (simplify from current `["bb.workspaces.getIamPolicy", "bb.users.list", "bb.groups.list"]` since those are no longer on this page)
- Groups route: `requiredPermissionList: ["bb.groups.list"]`

### Legacy URL Handling

The current implementation uses hash fragments (`#USERS`, `#MEMBERS`, `#GROUPS`) for tab navigation. After the split, these become separate routes. Old bookmarked URLs like `/settings/users#GROUPS` will land on the Users page without a Groups tab. No redirect is needed — the hash will simply be ignored and the user sees the Users page content.

### Vue Component Cleanup

After migration, these Vue components become dead code and should be deleted:
- `frontend/src/views/SettingWorkspaceMembers.vue`
- `frontend/src/components/Member/Views/WorkspaceMembers.vue` (verify no other consumers first)

### Mount System

All three pages use `ReactPageMount.vue` with lazy loading via `mount.ts`. Export names must match file names: `UsersPage`, `MembersPage`, `GroupsPage`.

### Behavior Changes

Each page works exactly as the current tabs do. The user-visible changes are:
- Members page becomes accessible in **both** modes (previously: separate sidebar entry in non-SaaS Vue, tab in SaaS React)
- Users page hidden in SaaS mode (same as current)
- Groups gets its own sidebar entry and route (previously a tab under Users)

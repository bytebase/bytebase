# Users Page Split: Three Independent Pages

## Summary

Split the monolithic `UsersPage.tsx` (3,489 lines) into three independent pages with separate routes and sidebar entries: Users, Members, and Groups. This restores the separation that existed in the previous Vue implementation, where Members had its own sidebar entry in non-SaaS mode.

## Motivation

PR #19829 migrated the `/users` page from Vue to React but consolidated three distinct concepts (user accounts, workspace IAM, groups) into a single tabbed page. This created two problems:

1. **Lost functionality**: Non-SaaS mode lost the Members page for managing workspace IAM policy (the Vue version had a separate "Members" sidebar entry for non-SaaS)
2. **Monolith file**: A single 3,489-line file mixing unrelated concerns

## Design

### Route & Sidebar Structure

| Route | Sidebar Entry | Component | Visibility |
|-------|--------------|-----------|------------|
| `/settings/users` | "Users" | `UsersPage.tsx` | Non-SaaS only |
| `/settings/members` | "Members" | `MembersPage.tsx` | Always |
| `/settings/groups` | "Groups" | `GroupsPage.tsx` | Always |

- Reuse existing route constants `WORKSPACE_ROUTE_USERS` and `WORKSPACE_ROUTE_MEMBERS`
- Add new `WORKSPACE_ROUTE_GROUPS` constant
- Sidebar: Users gets `hide: isSaaSMode`, Members and Groups are always visible

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

### Behavior Changes

None. Each page works exactly as the current tabs do. The only user-visible change is:
- Members page becomes accessible in non-SaaS mode (restoring previous Vue behavior)
- Users page is hidden in SaaS mode (same as current behavior)
- Groups gets its own sidebar entry (previously a tab under Users)

### Route Configuration Changes

In `frontend/src/router/dashboard/workspace.ts`:
- Keep existing Users route, update component to new `UsersPage.tsx`
- Keep existing Members route (line 342-354), update component from Vue `SettingWorkspaceMembers.vue` to React `MembersPage.tsx` via `ReactPageMount.vue`
- Add new Groups route with `WORKSPACE_ROUTE_GROUPS` constant

In `frontend/src/router/dashboard/workspaceRoutes.ts`:
- Add `WORKSPACE_ROUTE_GROUPS` constant

In `frontend/src/utils/useDashboardSidebar.ts`:
- Users entry: `hide: isSaaSMode` (already exists)
- Members entry: remove `hide: isSaaSMode` (make always visible)
- Groups entry: add new sidebar entry (always visible)

### Mount System

All three pages use `ReactPageMount.vue` with lazy loading via `mount.ts`. Export names must match file names: `UsersPage`, `MembersPage`, `GroupsPage`.

# SQL Editor React Migration — Stage 10 Design

**Date:** 2026-04-22
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Fully retire `AccessGrantRequestDrawer.vue` by migrating its 2 remaining Vue callers (`RequestQueryButton.vue` and `MaskingReasonPopover.vue`) to React with full feature parity. This requires building a React `RoleGrantPanel` (since `RequestQueryButton`'s non-JIT flow uses the Vue `RoleGrantPanel`), reusing the existing React `AccessGrantRequestDrawer` (created in Stage 9), and swapping all upstream Vue parents that embed these components.

**Non-goals (Stage 10):**
- Deleting Vue `RoleGrantPanel.vue` — multi-caller pattern, retained for its 4 other Vue callers (`SensitiveData/GrantAccessForm`, `DataExportButton`, `AddProjectMember/AddProjectMemberForm`, `Permission/ComponentPermissionGuard`).
- Deleting Vue `RoleGrantPanel/DatabaseResourceForm/*` — same rationale; React `DatabaseResourceSelector` already exists separately.
- Migrating any other Vue parent (e.g., `VirtualDataBlock`, `ResultViewV1`, `VirtualDataTable`, `DatabaseNode`) — only the 1-line swaps inside them.
- AI plugin bridge, OpenAIButton migration, FolderForm port, TabList migration.
- Refactoring or extracting shared code between the new React `RoleGrantPanel` and the existing `RequestRoleSheet.tsx` (MembersPage flow).

## 2. Architecture

After Stage 9: React `AccessGrantRequestDrawer.tsx` is private to `AccessPane.tsx`. After Stage 10: it remains at the same location (`react/components/sql-editor/AccessGrantRequestDrawer.tsx`) but is also imported by the new React `RequestQueryButton.tsx` and `MaskingReasonPopover.tsx`.

### Component tree

```
[Vue parents — unchanged]                  [React components]
VirtualDataBlock.vue (Vue)         ─►       <RequestQueryButton>
ResultViewV1.vue (Vue)             ─►       <RequestQueryButton>
VirtualDataTable.vue (Vue)         ─►       <RequestQueryButton>
DatabaseNode.vue (Vue)             ─►       <RequestQueryButton>
                                            <MaskingReasonPopover>

[Each React component renders its own children]
RequestQueryButton.tsx       ─► RoleGrantPanel.tsx (new)
                             OR  AccessGrantRequestDrawer.tsx (Stage 9)
MaskingReasonPopover.tsx     ─► AccessGrantRequestDrawer.tsx (Stage 9)
AccessPane.tsx (Stage 9)     ─► AccessGrantRequestDrawer.tsx (Stage 9)
```

Each Vue parent embeds the React component via `<ReactPageMount page="RequestQueryButton" ... />` etc.

## 3. New React files

### 3.1 `RoleGrantPanel.tsx` (~250 lines)

**Location:** `frontend/src/react/components/sql-editor/RoleGrantPanel.tsx`

(Placed under `sql-editor/` because Stage 10 needs it specifically for the SQL editor request-query flow. If/when the other 4 Vue `RoleGrantPanel` callers migrate, the React version can be lifted to a shared location.)

**Props:**
```tsx
type Props = {
  readonly projectName: string;
  readonly databaseResources: DatabaseResource[];
  readonly placement?: "right" | "bottom" | "top" | "left";
  readonly role: string; // PresetRoleType.SQL_EDITOR_USER for SQL editor flow
  readonly requiredPermissions: Permission[];
  readonly onClose: () => void;
};
```

**Shape:**
- shadcn `Sheet` + `SheetContent` (width="wide" or similar large width)
- Title: `t("issue.title.request-role")`
- Body: form with role select (pre-filled, disabled), database resource selector, expiration picker, reason textarea, issue labels
- Submit creates an issue via `issueServiceClientConnect.createIssue(...)`
- On success: navigate to issue detail page in new tab + close

**Reuses:**
- React `DatabaseResourceSelector` from `@/react/components/DatabaseResourceSelector`
- React `IssueLabelSelect` from `@/react/components/IssueLabelSelect`
- React `RoleSelect` from `@/react/components/RoleSelect` (already exists)
- React `ExpirationPicker` from `@/react/components/ui/expiration-picker`
- shadcn `Sheet`, `Textarea`, `Button`
- `issueServiceClientConnect` from `@/connect`
- `useCurrentUserV1`, `useProjectV1Store`, `pushNotification` from `@/store`

**Mirrors Vue source's:**
- Issue creation logic (CEL expression building, time-bounded role, etc.)
- Project's `enforceIssueTitle` flag enforcement
- `requiredPermissions` propagation

### 3.2 `RequestQueryButton.tsx` (~140 lines)

**Location:** `frontend/src/react/components/sql-editor/RequestQueryButton.tsx`

**Props:**
```tsx
type Props = {
  readonly size?: "sm" | "default";
  readonly text: boolean;
  readonly statement?: string;
  readonly permissionDeniedDetail: PermissionDeniedDetail;
};
```

**Shape:**
- Uses `usePermissionCheck(["bb.accessGrants.create" | "bb.issues.create"], project)` based on JIT detection
- Conditional rendering of Button + state-managed drawer/panel
- JIT flow → opens `<AccessGrantRequestDrawer>` with pre-filled query + targets
- Non-JIT flow → opens `<RoleGrantPanel>` with `role=SQL_EDITOR_USER` + missing resources + required permissions

**Reuses:**
- `AccessGrantRequestDrawer` from `./AccessGrantRequestDrawer`
- `RoleGrantPanel` from `./RoleGrantPanel`
- `FeatureBadge` from `@/react/components/FeatureBadge`
- `PermissionGuard` from `@/react/components/PermissionGuard`
- `useVueState` for `useSQLEditorStore.project`, `useProjectV1Store.getProjectByName`
- `parseStringToResource` from `@/components/RoleGrantPanel/DatabaseResourceForm/common` (shared utility — verify it works in React; it should be pure)

### 3.3 `MaskingReasonPopover.tsx` (~140 lines)

**Location:** `frontend/src/react/components/sql-editor/MaskingReasonPopover.tsx`

**Props:**
```tsx
type Props = {
  readonly reason: MaskingReason;
  readonly statement?: string;
  readonly database?: string;
  readonly onClick?: () => void;
};
```

**Shape:**
- Uses Stage 6 `Popover` primitive with hover trigger
- Trigger: small `EyeOff` icon + optional semantic-type icon
- Content: title, semantic type, algorithm, context, classification level, optional "Request JIT" button
- "Request JIT" button → opens `<AccessGrantRequestDrawer>` with pre-filled query + targets + `unmask=true`

**Reuses:**
- `AccessGrantRequestDrawer` from `./AccessGrantRequestDrawer`
- Stage 6 `Popover` primitive
- `useVueState` for project store + JIT feature check

### 3.4 Test files (one per component)

3 new test files; ~3-5 tests each. Mock pattern follows existing sql-editor tests.

## 4. Vue caller swaps

5 Vue parent files will have 1-line `<RequestQueryButton>` or `<MaskingReasonPopover>` swap to `<ReactPageMount page="..." ... />`:

### 4.1 `EditorCommon/ResultView/VirtualDataBlock.vue`
Find `<RequestQueryButton ... />`. Replace with `<ReactPageMount page="RequestQueryButton" :size :text :statement :permissionDeniedDetail />`. Drop the Vue import.

### 4.2 `EditorCommon/ResultView/ResultViewV1.vue`
Same pattern — find Vue usage, swap to ReactPageMount, drop import.

### 4.3 `EditorCommon/ResultView/DataTable/VirtualDataTable.vue`
Same.

### 4.4 `ConnectionPanel/ConnectionPane/TreeNode/DatabaseNode.vue`
Same — has `<RequestQueryButton>` and possibly `<MaskingReasonPopover>` (verify both during implementation).

### 4.5 Any additional callers — verify via grep

The 5 files above came from earlier audits. Implementer must `rg "RequestQueryButton|MaskingReasonPopover"` first to confirm the complete caller list and adjust scope if a 6th caller surfaces.

## 5. Vue file deletions

After all caller swaps verified:

- `frontend/src/views/sql-editor/EditorCommon/ResultView/RequestQueryButton.vue`
- `frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/common/MaskingReasonPopover.vue`
- `frontend/src/views/sql-editor/AsidePanel/AccessPane/AccessGrantRequestDrawer.vue`

Verify zero remaining callers of each via `rg` before deleting. The `AsidePanel/AccessPane/` directory will then be fully empty (its `index.ts` was deleted in Stage 9, `AccessPane.vue` and `AccessGrantItem.vue` already deleted in Stage 9). Remove the directory entirely.

**Retained Vue files** (multi-caller, out of scope for deletion):
- `components/RoleGrantPanel/RoleGrantPanel.vue` — 4 other callers
- `components/RoleGrantPanel/DatabaseResourceForm/*` — used by Vue RoleGrantPanel and others
- `components/RoleGrantPanel/MaxRowCountSelect.vue` — used by Vue RoleGrantPanel and others (React port already exists from Stage 6)

## 6. i18n keys

Verify in all 5 React locales; add missing using byte-exact Vue values; convert `{var}` → `{{var}}`:

For `RoleGrantPanel`:
- `issue.title.request-role`
- `common.cancel`, `common.submit`, `common.reason`
- (other keys depending on form fields — implementer audits during implementation)

For `RequestQueryButton`:
- `sql-editor.request-jit`
- `sql-editor.request-query`

For `MaskingReasonPopover`:
- `masking.reason.title`
- `masking.reason.semantic-type`
- `masking.reason.algorithm`
- `masking.reason.context`
- `masking.reason.classification`
- `sql-editor.request-jit` (shared with RequestQueryButton)
- `masking.algorithms.*` (template-style — `formatAlgorithm` in Vue does dynamic key lookup; check if these are in React `dynamic.*` keys or static. May require fallback handling.)

## 7. Verification

### 7.1 Per-stage

- `pnpm fix && check && type-check && test` all green
- ~10-15 new tests across 3 component test files
- No regressions

### 7.2 Manual UX

For RequestQueryButton (mounted via VirtualDataBlock / ResultViewV1 / VirtualDataTable / DatabaseNode):
1. Run a query with insufficient permissions → "Request Query" button appears in error result
2. Click → if project allows JIT and only `bb.sql.select` is missing → AccessGrantRequestDrawer opens with statement + missing databases pre-filled
3. Click in non-JIT scenario → RoleGrantPanel opens with role=SQL_EDITOR_USER + missing resources + required permissions pre-filled
4. Submit either flow → success notification + drawer closes

For MaskingReasonPopover (mounted via DatabaseNode):
5. Hover masked column → popover shows masking reason details
6. If JIT available and statement present → "Request JIT" button → drawer opens with `unmask=true`

For AccessPane (Stage 9, already React):
7. Click Run on grant — still works (uses execute-sql event bridge)

### 7.3 Bridge integrity

Test that the "Request" buttons inside still-Vue ResultView and ConnectionPane work end-to-end:
- Permission denied → button shown
- Click → drawer/panel opens
- Submit → API call succeeds
- Drawer closes properly

## 8. Practical checklist

- [ ] React `RoleGrantPanel.tsx` + test
- [ ] React `RequestQueryButton.tsx` + test
- [ ] React `MaskingReasonPopover.tsx` + test
- [ ] i18n keys verified/added (with `{var}` → `{{var}}` conversion as needed)
- [ ] 5 Vue caller files swapped (`VirtualDataBlock.vue`, `ResultViewV1.vue`, `VirtualDataTable.vue`, `DatabaseNode.vue` for RequestQueryButton; `DatabaseNode.vue` for MaskingReasonPopover — verify caller list during implementation)
- [ ] Vue `RequestQueryButton.vue` deleted
- [ ] Vue `MaskingReasonPopover.vue` deleted
- [ ] Vue `AccessGrantRequestDrawer.vue` deleted (and the now-empty `AsidePanel/AccessPane/` directory)
- [ ] Vue `RoleGrantPanel.vue` retained (4 other callers)
- [ ] `pnpm fix && check && type-check && test` all pass
- [ ] Manual UX from §7.2 verified

## 9. Out of scope (deferred)

- Migrating Vue `RoleGrantPanel.vue` and its DatabaseResourceForm subtree — multi-caller, requires future stage targeting all 4 remaining Vue callers (SensitiveData/GrantAccessForm, DataExportButton, AddProjectMember/AddProjectMemberForm, Permission/ComponentPermissionGuard)
- Refactoring shared code between React `RoleGrantPanel` and `RequestRoleSheet.tsx` — both create issues with similar shape; extraction risks scope creep into MembersPage flow
- Vue parent migrations (`VirtualDataBlock`, `ResultViewV1`, `VirtualDataTable`, `DatabaseNode`) — only the inline embedding lines change

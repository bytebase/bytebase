# SQL Editor React Migration — Stage 7 Design

**Date:** 2026-04-21
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate `EditorCommon/SharePopover.vue` (228 lines, 2 callers) to React. Both Vue callers (`EditorAction.vue` and `SheetTree.vue`) swap to `<ReactPageMount page="SharePopoverBody" ... />`. Vue file is deletable after both swaps. Reuses the `Popover` primitive built in Stage 6 for the nested visibility-selector.

**Non-goals:**
- Porting `CopyButton.vue` (10+ Vue callers — out of scope). Use inline `navigator.clipboard.writeText` pattern already established in multiple React files.
- Emittery or async-action bridges (deferred to a later stage).
- Migrating `SheetTree.vue` (851 lines) or `EditorAction.vue` themselves — only the one-line swaps where they embed `<SharePopover>`.
- Moving the outer `NPopover` click-trigger (the Share button in the toolbar) — only the popover *content* migrates.
- Deprecating CopyButton — still heavily used elsewhere.

## 2. Architecture

**One React component** (`SharePopoverBody.tsx`, ~200 lines) replaces the body of the Vue `SharePopover`. Mounted as the content slot of Vue's outer `NPopover` in both caller files:

```vue
<!-- Before (EditorAction and SheetTree both use this pattern) -->
<NPopover trigger="click" ...>
  <template #trigger>...share button...</template>
  <template #default>
    <SharePopover :worksheet="..." @on-updated="..." />
  </template>
</NPopover>

<!-- After -->
<NPopover trigger="click" ...>
  <template #trigger>...share button...</template>
  <template #default>
    <ReactPageMount page="SharePopoverBody"
      :worksheet="..." :onUpdated="..." />
  </template>
</NPopover>
```

Note: `@on-updated` (Vue event emit) → `:onUpdated` (explicit prop-callback) per the Stage 1 kebab-case-prop learning. Vue's `attrs` doesn't auto-camelCase kebab-case bindings.

## 3. The React component

**File:** `frontend/src/react/components/sql-editor/SharePopoverBody.tsx`

**Props:**
```tsx
type Props = {
  readonly worksheet?: Worksheet;
  readonly onUpdated?: () => void;
};
```

**Structure (mirrors Vue source):**

- **Header section:** "Share" title + visibility selector.
  - Visibility selector is a nested Popover using Stage 6's `Popover` primitive.
  - Selector trigger shows the current access level label + a `ChevronDown` icon.
  - Nested popover content lists 3 options: Private, Project Read, Project Write.
  - Each option: icon (`LockKeyhole` for Private, `Users` for the two Project options) + label + description + `Check` mark if currently selected.
  - Non-creators see the selector styled as read-only (disabled).

- **Link display:** input field showing the shareable URL with `Link2` icon prefix + copy button.
  - Copy button calls `navigator.clipboard.writeText` directly (no CopyButton component).
  - Copy button disabled unless the current tab's status is `"CLEAN"` (mirrors the Vue predicate).

**Stores (top-level hooks):**
- `useActuatorV1Store` — for `workspaceExternalURL`
- `useCurrentUserV1` — for `me` (creator check)
- `useWorkSheetStore` — for `patchWorksheet`
- `pushNotification` — for success toasts (imported from `@/store`)
- `useSQLEditorTabStore` — for `currentTab` (copy button disabled gate)

**Router:**
- `router.resolve({ name: SQL_EDITOR_WORKSHEET_MODULE, params: { project, sheet } }).href` — builds the shareable link, then constructs absolute URL via `new URL(href, workspaceExternalURL || window.location.origin)`.

**Icons (lucide-react swaps):**
- heroicons-solid `chevron-down` → `ChevronDown` from lucide-react
- heroicons-solid `check` → `Check` from lucide-react
- heroicons-solid `link` → `Link2` from lucide-react
- `LockKeyhole`, `Users` — same lucide names (Vue also uses lucide-vue-next)

**Access change flow (`handleChangeAccess`):**
1. Guard: user isn't the creator or no worksheet → no-op.
2. Call `worksheetStore.patchWorksheet(worksheet, { visibility }, ["visibility"])`.
3. If clipboard is available, `navigator.clipboard.writeText(sharedTabLink)`; push notification "URL copied to clipboard". Else push "Updated".
4. Call `onUpdated?.()` prop callback (Vue source emits `on-updated`).
5. Close the nested visibility selector popover.

## 4. Vue caller swaps

### 4.1 `EditorAction.vue`

Find the `<SharePopover :worksheet="..." />` occurrence (around line 107 in current file). Replace:

```vue
<SharePopover :worksheet="sheetAndTabStore.currentSheet" />
```

with:

```vue
<ReactPageMount page="SharePopoverBody"
  :worksheet="sheetAndTabStore.currentSheet" />
```

Drop the `import SharePopover from "./SharePopover.vue";` line. `ReactPageMount` is already imported from Stage 4. No `onUpdated` prop needed — the Vue original didn't pass one.

### 4.2 `SheetTree.vue`

Find the caller via `grep -n SharePopover frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/SheetTree.vue`. Replace:

```vue
<SharePopover
  :worksheet="worksheetEntity"
  @on-updated="handleContextMenuClickOutside"
/>
```

with:

```vue
<ReactPageMount page="SharePopoverBody"
  :worksheet="worksheetEntity"
  :onUpdated="handleContextMenuClickOutside" />
```

Drop the `import SharePopover from "@/views/sql-editor/EditorCommon/SharePopover.vue";` line. Add `import ReactPageMount from "@/react/ReactPageMount.vue";` if not already present.

## 5. Vue file deletion

Post-swap verification:
```bash
grep -rn "SharePopover" frontend/src/ | grep -v "frontend/src/react/"
```

Expected: zero Vue references outside the file being deleted and the `EditorCommon/index.ts` barrel.

Then:
```bash
rm frontend/src/views/sql-editor/EditorCommon/SharePopover.vue
```

And remove `SharePopover` from both the import block and the export block in `frontend/src/views/sql-editor/EditorCommon/index.ts`.

This is the first multi-caller migration where the Vue file IS cleanly deletable (because both callers migrate together). Sets the precedent for future multi-caller cleanup.

## 6. i18n keys

Verify in all 5 React locales; add missing using Vue locale values byte-exact:
- `sql-editor.link-access`
- `sql-editor.private`
- `sql-editor.private-desc`
- `sql-editor.project-read`
- `sql-editor.project-read-desc`
- `sql-editor.project-write`
- `sql-editor.project-write-desc`
- `sql-editor.url-copied-to-clipboard`
- `common.share` (probably exists)
- `common.updated` (probably exists)

## 7. Verification

- `pnpm fix && check && type-check && test` all green
- New tests (~6) in `SharePopoverBody.test.tsx`:
  - Renders "Share" title and link input
  - Shows 3 visibility options (Private, Project Read, Project Write) with correct icons
  - Disables the visibility selector when user is not creator
  - `handleChangeAccess` calls `patchWorksheet` and `pushNotification` and invokes `onUpdated` callback
  - Copy button writes to clipboard and pushes notification
  - Copy button disabled when `currentTab?.status !== "CLEAN"`

**Manual UX:**
1. EditorAction share button: click opens outer popover → see Share title + visibility selector + URL + copy button
2. Copy button click → URL copied + toast notification
3. Change visibility (creator only) → worksheet updated, URL auto-copied, toast shown
4. Non-creator sees visibility selector disabled
5. SheetTree context menu share: click opens popover → same behavior + parent closes on `onUpdated`

## 8. Out of scope (deferred)

- CopyButton React port (10+ Vue callers — separate concern).
- Emittery events bridge.
- Async-actions bridge (maybeSwitchProject / createWorksheet / abortAutoSave etc.).
- Vue-in-React mount infrastructure.

## 9. Practical checklist

- [ ] `react/components/sql-editor/SharePopoverBody.tsx` + test created
- [ ] 8 i18n keys verified/added
- [ ] `EditorAction.vue` swap (1 template line + 1 import line)
- [ ] `SheetTree.vue` swap (1 template line + 1 import line)
- [ ] Vue `SharePopover.vue` deleted
- [ ] `EditorCommon/index.ts` barrel import + export removed
- [ ] `pnpm fix && check && type-check && test` all pass
- [ ] Manual UX verified from both call sites (EditorAction share button + SheetTree context-menu share)

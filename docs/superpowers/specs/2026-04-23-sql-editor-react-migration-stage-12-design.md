# SQL Editor React Migration — Stage 12 Design

**Date:** 2026-04-23
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Retire the `WorksheetPane` Vue subtree (`WorksheetPane.vue` + `SheetTree.vue` + `TreeNodeSuffix.vue` + `dropdown.ts` + `common.ts`) by porting it to React end-to-end. After this stage, the SQL editor's worksheet sidebar is fully React, and the SQL editor's `AsidePanel.vue` mounts it via `ReactPageMount`.

This builds on Stage 11's infrastructure: the `Tree` primitive, the `useSQLEditorWorksheetStore` Pinia store (delegated via `useSheetContextByView`), the React `TreeNodePrefix.tsx`, and the React `FolderForm.tsx` (used by the move-to-folder modal).

**Non-goals:**
- Migrating `AsidePanel.vue` itself — it stays Vue. Only the inline `<WorksheetPane>` mount swaps to `<ReactPageMount page="WorksheetPane" />`.
- Migrating `SchemaPane.vue`, `ConnectionPane.vue`, `HistoryPane.vue` (already React), `AccessPane.tsx` (already React) — other AsidePanel tabs are out of scope.
- Refactoring the existing `useSQLEditorWorksheetStore` (Stage 11 finished it).
- Adding any new sidebar feature — visual + behavioral parity only.

## 2. Architecture overview

```
Phase 1 — context-menu primitive
  frontend/src/react/components/ui/context-menu.tsx          (new, ~100 lines + tests)

Phase 2 — helpers
  frontend/src/react/components/sql-editor/useDropdown.ts    (new, ~250 lines, ports dropdown.ts)
  frontend/src/react/components/sql-editor/TreeNodeSuffix.tsx (new, ~150 lines)
  frontend/src/react/components/sql-editor/filterNode.ts     (new, ~25 lines, ports common.ts)

Phase 3 — SheetTree
  frontend/src/react/components/sql-editor/SheetTree.tsx     (new, ~700 lines)

Phase 4 — WorksheetPane
  frontend/src/react/components/sql-editor/WorksheetPane.tsx (new, ~300 lines)
  frontend/src/react/components/sql-editor/FilterMenuItem.tsx (new, ~30 lines)

Phase 5 — Vue caller swap & cleanup
  frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue    (modify: 1-line swap)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/WorksheetPane.vue        (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/SheetTree.vue  (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodeSuffix.vue (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/dropdown.ts    (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/common.ts      (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/index.ts       (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/index.ts                 (delete)
  frontend/src/views/sql-editor/AsidePanel/WorksheetPane/FilterMenuItem.vue       (delete if unused outside scope)
```

Each phase is commit-able independently.

## 3. Phase 1 — `context-menu.tsx` primitive

### 3.1 File

`frontend/src/react/components/ui/context-menu.tsx`

### 3.2 Wraps

`@base-ui/react/context-menu` (Base UI ships this — confirmed via `node_modules/@base-ui/react/context-menu/`).

### 3.3 Exports

Following the shadcn pattern used by `dropdown-menu.tsx`:

```ts
export {
  ContextMenu,        // BaseContextMenu.Root
  ContextMenuTrigger, // BaseContextMenu.Trigger — wraps a target element
  ContextMenuContent, // Portal + Positioner + Popup + styling
  ContextMenuItem,    // BaseContextMenu.Item — styled with hover/disabled states
  ContextMenuSeparator,
  ContextMenuLabel,   // optional section label
};
```

### 3.4 Styling

Same shadcn aesthetic as `dropdown-menu.tsx`. Uses semantic tokens (`bg-background`, `border-control-border`, `text-control`), `LAYER_SURFACE_CLASS` z-index, portal to `getLayerRoot("overlay")`.

### 3.5 Tests

`context-menu.test.tsx` — 4 scenarios:
- Right-click on trigger opens the content
- Clicking an item fires its `onClick`
- ESC closes
- `disabled` items don't fire onClick

## 4. Phase 2 — Helpers

### 4.1 `filterNode.ts`

Direct port of `common.ts`'s `filterNode` helper. Pure function, no UI, no Vue. Used by `useDropdown` to filter visible nodes during multi-select.

### 4.2 `useDropdown.ts`

React port of Vue `dropdown.ts` (~230 lines). Returns the same shape:
```ts
{
  context: { showDropdown, x, y, currentNode, … },
  options: ContextMenuOption[],   // computed from currentNode + view + multiSelectMode + worksheetFilter
  worksheetEntity,                // for the share popover
  handleSharePanelShow,
  handleMenuShow,                 // imperative open at coordinates
  handleClickOutside,             // close handler
}
```

Key differences from Vue:
- Vue uses `naive-ui`'s `NDropdown` with manual `x`/`y` positioning. React equivalent is `ContextMenu` from Phase 1, which Base UI handles positioning internally on right-click. Adjust the API: drop `x`/`y` and the `handleMenuShow(e: MouseEvent, node)` becomes the trigger element's `onContextMenu`.
- All naive-ui imports replaced with React equivalents.
- Vue refs become React `useState`.
- `t()` from `@/plugins/i18n` becomes `useTranslation` from `react-i18next`.

### 4.3 `TreeNodeSuffix.tsx`

React port of Vue `TreeNodeSuffix.vue` (147 lines). Renders the right-side controls of a tree row:
- Star/unstar button (heart icon, debounced toggle via `worksheetV1Store.upsertWorksheetOrganizer`)
- Visibility badge (private / project-read / project-write)
- "More" button that triggers the context menu

Props:
```ts
{
  readonly node: WorksheetFolderNode;
  readonly view: SheetViewMode;
  readonly onSharePanelShow: (...) => void;
  readonly onContextMenuShow: (e: MouseEvent, node) => void;
  readonly onToggleStar: ({ worksheet, starred }) => void;
}
```

i18n keys verified in all 5 React locales; add missing keys with byte-exact Vue values.

## 5. Phase 3 — `SheetTree.tsx`

### 5.1 Props

```ts
{
  readonly view: SheetViewMode;
  readonly multiSelectMode?: boolean;
  readonly checkedNodes?: WorksheetFolderNode[];
  readonly onMultiSelectModeChange: (next: boolean) => void;
  readonly onCheckedNodesChange: (nodes: WorksheetFolderNode[]) => void;
}
```

### 5.2 Feature parity audit (the 11 features)

1. **Tree display** — `Tree<WorksheetFolderNode>` from Phase 1 of Stage 11. Data via `useSheetContextByView(view).sheetTree` from the Pinia store.
2. **Click worksheet → open** — row `onClick`. cmd/ctrl-click → `forceNewTab=true`. Calls `openWorksheetByName({ worksheet, forceNewTab })` from `Sheet/context.ts`.
3. **Click folder → expand/collapse** — chevron toggle (matches FolderForm pattern).
4. **Multi-select mode (checkbox column)** — when `multiSelectMode`, `renderNode` shows a checkbox before the prefix icon. Checkbox state derived from `checkedNodes`. Toggling fires `onCheckedNodesChange`.
5. **Drag-and-drop** — react-arborist's built-in `onMove` callback. Handler ports `handleDrop` from Vue: validates target, calls `batchUpdateWorksheetFolders` for moved worksheets and `folderContext.moveFolder` for moved folders. Disabled when `view === 'draft'`, when editing a node, or when in multi-select mode (matches `:draggable` Vue binding).
6. **In-place rename** — when `editingNode === node`, `renderNode` swaps the label `<span>` for an `<Input>` with `autofocus`, `onBlur` + Enter handlers calling `handleRenameNode` (ports the Vue logic line-for-line, including the folder-name "/", "." restriction). Auto-focus via `useEffect` + `inputRef.focus()`.
7. **Context menu (right-click)** — wrap each row in `ContextMenu` from Phase 1. `ContextMenuContent` items computed from `useDropdown(view, worksheetFilter)` (Phase 2). Menu items: Open, Open in new tab, Copy link, Star/Unstar, Share, Rename, Delete.
8. **Star/unstar** — handled by `TreeNodeSuffix` from Phase 2.
9. **Delete (with confirm)** — `handleDeleteFolders` (ports the Vue logic) opens a shadcn `AlertDialog` instead of naive-ui `useDialog`. Same content: warning, list of affected folders/worksheets, Cancel + Delete buttons. On confirm: deletes worksheets via `worksheetV1Store.deleteWorksheetByName` and folders via `folderContext.removeFolder`, closes affected tabs.
10. **Highlighted matching label** — port the existing React `HighlightLabelText` if it exists; else inline a small helper (we have one in `ClassificationTree.tsx` from the recent refactor — but since that's not exported, port the same approach).
11. **Loading spinner** — port `BBSpin` use during fetch. Use the existing React `Spinner` if it exists; else use `Loader2` from `lucide-react` with `animate-spin`.

### 5.3 Tests

`SheetTree.test.tsx` — 6 scenarios:
- Renders tree from store data
- Click worksheet → fires `openWorksheetByName`
- Click folder → toggles expand
- Multi-select mode renders checkboxes; toggle fires `onCheckedNodesChange`
- Right-click → opens context menu (mock the menu primitive)
- Delete confirm → fires `worksheetV1Store.deleteWorksheetByName`

Mock the Tree primitive, ContextMenu primitive, AlertDialog, and Pinia stores following existing test patterns.

## 6. Phase 4 — `WorksheetPane.tsx`

### 6.1 Layout

Ports `WorksheetPane.vue` (262 lines):
- View tabs at top (My / Shared / Draft) — uses Tabs primitive (already in shadcn lib at `tabs.tsx` if present; otherwise build a thin wrapper)
- Search input below tabs
- Filter menu (FunnelIcon dropdown) — uses `DropdownMenu` from `dropdown-menu.tsx` (existing primitive)
- Multi-select mode toolbar (FolderInputIcon "move", TrashIcon "delete", XIcon "exit") visible when `multiSelectMode === true`
- `<SheetTree>` (from Phase 3) below

### 6.2 State

- `view: SheetViewMode` — local React state, defaults to `useSheetContext().view.value` if persisted
- `multiSelectMode: boolean` — local
- `checkedNodes: WorksheetFolderNode[]` — local
- Move-to-folder modal (uses existing React `FolderForm` via direct import + shadcn `Dialog`)
- Filter (search keyword, only-starred, etc.) — read from Pinia store via `useVueState`

### 6.3 Move modal

The existing flow: WorksheetPane.vue has a BBModal containing `<ReactPageMount page="FolderForm" :folder :onFolderChange />`. In React, this becomes a shadcn `Dialog` containing `<FolderForm folder onFolderChange />` (direct import, no ReactPageMount needed since both are React).

Same submit behavior: `batchUpdateWorksheetFolders` from Pinia store.

### 6.4 i18n

Verify all keys exist in React locales. Likely: `sql-editor.sheet.view.my`, `sql-editor.sheet.view.shared`, `sql-editor.sheet.view.draft`, `sql-editor.sheet.move-to-folder`, `sql-editor.sheet.delete-selected`, etc.

### 6.5 Tests

`WorksheetPane.test.tsx` — 5 scenarios:
- Renders all three view tabs
- Switching view updates `<SheetTree>` view prop
- Filter menu toggle fires search
- Multi-select toolbar appears when checked nodes exist
- Move modal opens + submits

## 7. Phase 5 — Caller swap & cleanup

### 7.1 `AsidePanel.vue` swap

Replace:
```vue
<WorksheetPane v-if="asidePanelTab === 'WORKSHEET'" />
```

With:
```vue
<ReactPageMount v-if="asidePanelTab === 'WORKSHEET'" page="WorksheetPane" />
```

Drop the `import WorksheetPane from "./WorksheetPane";` line. Add `ReactPageMount` import if not present.

### 7.2 Delete

After verifying zero remaining references:
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/WorksheetPane.vue`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/SheetTree.vue`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodeSuffix.vue`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/dropdown.ts`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/common.ts`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/SheetList/index.ts`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/index.ts`
- `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/FilterMenuItem.vue` (only if unused elsewhere — verify via grep)

After deletion the directory `frontend/src/views/sql-editor/AsidePanel/WorksheetPane/` should be empty. Remove it.

### 7.3 Verify

- `rg "WorksheetPane\.vue|SheetTree\.vue|TreeNodeSuffix\.vue"` → zero results
- `rg "from.*SheetList"` → zero results
- `pnpm fix && type-check && test --run && check` all green

## 8. Per-phase checklist

### Phase 1
- [ ] `context-menu.tsx` + 4 tests
- [ ] `fix && type-check && test --run` green

### Phase 2
- [ ] `filterNode.ts` + (unit test if non-trivial)
- [ ] `useDropdown.ts` (no separate test — covered by SheetTree)
- [ ] `TreeNodeSuffix.tsx` + 3 tests (renders icons by view, star toggle fires callback, share button fires)
- [ ] i18n keys verified
- [ ] `fix && type-check && test --run` green

### Phase 3
- [ ] `SheetTree.tsx` + 6 tests
- [ ] All 11 features parity-verified
- [ ] `fix && type-check && test --run` green

### Phase 4
- [ ] `WorksheetPane.tsx` + 5 tests
- [ ] `FilterMenuItem.tsx` (small)
- [ ] Move modal works with React `FolderForm`
- [ ] `fix && type-check && test --run` green

### Phase 5
- [ ] `AsidePanel.vue` swapped to `ReactPageMount`
- [ ] All Vue files deleted
- [ ] Empty directory removed
- [ ] `rg` confirms zero references
- [ ] `fix && type-check && test --run && check` green

## 9. Manual UX verification (after all phases)

1. SQL Editor open → click "Worksheets" tab in aside panel → tree of folders + worksheets renders
2. Switch view (My / Shared / Draft) → tree refetches and re-renders
3. Type in search → tree filters with highlighted matches
4. Click worksheet → opens in editor; cmd-click → opens in new tab
5. Click folder chevron → expands / collapses
6. Right-click row → context menu opens; click items work
7. Hover → star button appears; click → toggles starred state
8. Drag worksheet onto folder → moves it; refresh confirms
9. Filter menu → toggle "only starred" works
10. Toggle multi-select → checkboxes appear; check multiple → toolbar shows count
11. Toolbar "Move to folder" → modal opens with React `FolderForm`; pick folder → submit moves all checked
12. Toolbar "Delete" → AlertDialog confirms → deletes selected
13. Click row label twice (or via context menu Rename) → in-place input; Enter saves new name
14. Loading spinner appears briefly on tab switch / project switch

## 10. Out of scope (deferred)

- AsidePanel.vue React migration — this stage only swaps one mount inside it.
- SchemaPane / ConnectionPane / other AsidePanel tabs.
- AI plugin context bridge (deferred from earlier discussions).
- Refactor of `useSQLEditorWorksheetStore` (already shaped the way React needs it).
- New features in the worksheet pane.

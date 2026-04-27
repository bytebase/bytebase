# SQL Editor React Migration — Stage 14 Design

**Date:** 2026-04-24
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Port the two remaining sidebar-adjacent Vue subtrees — `TabList/` and `ConnectionPanel/` — to React end-to-end. After this stage, the **top-of-editor tab bar** and the **Connect-to-Database drawer** are 100% React; the only Vue surfaces left around the editor are the root `SQLEditorHomePage.vue` (deferred), the editor panels themselves (`EditorMain`/`StandardPanel`/`TerminalPanel` — host-only), the `SchemaPane` tree, and the `ResultView` subtree.

**Non-goals:**
- Migrating `SQLEditorHomePage.vue`, `EditorPanel.vue`, `StandardPanel.vue`, `EditorMain.vue`, `TerminalPanel.vue` — still Vue; they host these React mounts.
- Migrating `SchemaPane/*` or `ResultView/*` — separate stages.
- Refactoring `useSQLEditorTabStore`, `useSQLEditorContext`, or the connection picker's `useConnectionTreeContext` — the migration reads/writes them via `useVueState` / direct method calls.
- Visual or behavioral changes — parity only.
- Porting deep Vue dependencies (`RichDatabaseName`, `FeatureBadge`, `DatabaseIcon`, etc.) beyond the minimum needed. When a child Vue component has no React equivalent and isn't worth lifting alone, inline a local React display helper or mount it via `ReactPageMount` where feasible.

## 2. Inventory

### 2.1 TabList subtree (7 files, ~963 LOC)

| File | LOC | Notes |
|---|---|---|
| `TabList.vue` | 287 | Horizontal tab bar; `VueDraggable` for drag-reorder; `NScrollbar` for horizontal scroll; "+" button to add a tab; keyboard shortcut wiring. |
| `TabItem/TabItem.vue` | 133 | One tab; composes Prefix + Label + Suffix + AdminLabel. |
| `TabItem/Label.vue` | 161 | Tab title + in-place rename; hover tooltip. |
| `TabItem/Prefix.vue` | 48 | Leading icons (draft pencil, shared users, admin wrench, `SheetConnectionIcon` — already React via ReactPageMount). |
| `TabItem/Suffix.vue` | 104 | Close button + dirty indicator. |
| `TabItem/AdminLabel.vue` | 62 | "ADMIN" badge overlay. |
| `ContextMenu.vue` | 118 | Right-click: Close, Close others, Close to the right, Close all, Rename. |
| `context.ts` | 47 | Vue inject context (scrollbar ref, focus-tab helper). |

### 2.2 ConnectionPanel subtree (~14 files, ~1557 LOC)

| File | LOC | Notes |
|---|---|---|
| `ConnectionPanel.vue` | 79 | Shell: Drawer wrapping ConnectionPane. |
| `ConnectionPane/ConnectionPane.vue` | 853 | The big one — instance/database NTree + batch-mode + database groups + "select data source" radio + search. |
| `ConnectionPane/DatabaseGroupTable.vue` | 75 | Read-only table listing groups. |
| `ConnectionPane/DatabaseGroupTag.vue` | 35 | Small closable tag for selected groups. |
| `ConnectionPane/TreeNode/DatabaseNode.vue` | 72 | Database leaf rendering — mounted by arborist. |
| `ConnectionPane/TreeNode/InstanceNode.vue` | 40 | Instance internal node. |
| `ConnectionPane/TreeNode/LabelNode.vue` | 27 | Environment label group node. |
| `ConnectionPane/TreeNode/Label.vue` | 51 | Shared label renderer (name + tags + icons). |
| `ConnectionPane/DatabaseHoverPanel/DatabaseHoverPanel.vue` | 160 | Popover that shows on hover over a database node. |
| `ConnectionPane/DatabaseHoverPanel/hover-state.ts` | 19 | Hover-timer state. |
| `ConnectionPane/tree.ts` | 138 | Builds the NTree data model from pinia. |
| Index/re-exports | ~8 total | Plumbing. |

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `VueDraggable` (`vue-draggable-plus`) | `@dnd-kit/core` + `@dnd-kit/sortable` | Already installed |
| `NScrollbar` | Native `overflow-x-auto` + our custom scrollbar CSS | No primitive needed; inline styles |
| `NDrawer` / `DrawerContent` | shadcn `Sheet` (existing in `src/react/components/ui/sheet.tsx`) | ✓ |
| `NTree` | `Tree` primitive (`src/react/components/ui/tree.tsx`) | ✓ from Stage 11 |
| `NTooltip` | `Tooltip` primitive | ✓ |
| `NPopover` | `Popover` primitive | ✓ |
| `NDivider` | `Separator` primitive | ✓ |
| `NTag` with `closable` | `Badge` + close icon composed manually | ✓ |
| `useDialog` (naive-ui imperative) | shadcn `AlertDialog` via local state | ✓ |
| `useResizeObserver` (@vueuse) | `useEffect` + `ResizeObserver` | Inline |
| `scroll-into-view-if-needed` | Same library (framework-agnostic) | ✓ |
| `usePreventBackAndForward` | Same utility (framework-agnostic) | ✓ |
| `RichDatabaseName`, `FeatureBadge`, `DatabaseIcon` | Inline minimal React versions; mount via `ReactPageMount` only if already ported | Case-by-case |

## 4. Architecture & phases

Five phases. Each commit-able independently, each green on `pnpm fix / type-check / test / check`.

### Phase 1 — TabList leaves

**Files ported:** `TabItem/AdminLabel.vue`, `TabItem/Prefix.vue`, `TabItem/Suffix.vue`, `TabItem/Label.vue`.

**Approach:** Presentational components. Each becomes a React component under `src/react/components/sql-editor/TabItem/`. They compose existing React primitives (Tooltip, Button, Input). `SheetConnectionIcon` (already React from Stage 13) is reused directly.

**Deliverables:** 4 `.tsx` + 3 tests (AdminLabel has no state worth covering; combine with Label test).

### Phase 2 — TabList shell + context menu

**Files ported:** `TabItem.vue` (composer), `TabList.vue` (draggable list), `ContextMenu.vue`, `context.ts`.

**Approach:**
- `TabList.tsx` wraps `@dnd-kit/sortable`'s `SortableContext` + `useSortable` around each tab row. Drag reorders `tabStore.openTabList` (Pinia array) via `tabStore.setOrder(...)` (existing method).
- Horizontal scrolling is a native `overflow-x-auto` container; sticky "+" button on the right.
- Context menu uses the existing `DropdownMenu` primitive; right-click on a tab fires `onContextMenuShow` to pick the target tab.
- Replaces `useTabListContext` Vue inject with direct store calls + `useVueState`.

**Deliverables:** `TabList.tsx`, `TabItem.tsx`, `TabContextMenu.tsx`, plus tests for (a) tab render from store, (b) click selects tab, (c) close fires store.closeTab, (d) drag reorders tab array, (e) context menu "Close others" fires correct store calls.

**Caller swap:** `SQLEditorHomePage.vue` — `<TabList />` → `<ReactPageMount page="TabList" container-class="w-full" />`.

### Phase 3 — ConnectionPanel tree nodes + hover panel

**Files ported:** `TreeNode/Label.vue`, `TreeNode/LabelNode.vue`, `TreeNode/InstanceNode.vue`, `TreeNode/DatabaseNode.vue`, `DatabaseHoverPanel.vue`, `hover-state.ts`.

**Deferred to Phase 4:** `tree.ts` — reading the source revealed it's a Vue composable (uses `useDebounceFn`, `ref`, `computed`, `watch`), not pure TS. Its port is coupled to `ConnectionPane` consumption, so it rides with Phase 4.

**Approach:**
- Each *Node.vue* becomes a small presentational React component under `src/react/components/sql-editor/ConnectionPane/TreeNode/`. `Label.tsx` is the shared renderer that dispatches on `node.meta.type`.
- `hover-state.ts` becomes a React context + `useProvideHoverState()` hook mirroring Vue's `useDelayedValue` semantics (1000ms open, 350ms close).
- `DatabaseHoverPanel.tsx` portals a `position: fixed` panel into the overlay layer, clamping y into the viewport and listening for mouseenter/leave + click-outside. Matches Vue visuals (environment / instance / project / labels grid).
- Inlined display logic (engine icon, environment tint, instance / project names) instead of creating full React ports of `EnvironmentV1Name` / `InstanceV1Name` / `RichDatabaseName` / `ProjectV1Name`. The `tree-node-database` class is preserved so existing CSS hooks still apply.

**Deliverables:** Per-node `.tsx` files + `Label.tsx` + `DatabaseHoverPanel.tsx` + `hover-state.ts` + tests for LabelNode (2), InstanceNode (3), DatabaseNode (4), Label dispatch (4), DatabaseHoverPanel render + clamp (3).

### Phase 4 — ConnectionPane + batch mode (sliced)

Reading the source turned up more scope than the original framing assumed. `ConnectionPane.vue` is 853 lines and depends on a context menu (`actions.tsx`, 225L), a no-React-equivalent `FeatureModal`, naive-ui `NTree`/`NTabs`/`NRadio`/`NCheckbox`/`NDivider`/`MaskSpinner`, and `DatabaseGroupDataTable.vue` (an independent multi-selection DataTable used outside SQL Editor too). We slice Phase 4 into four shippable chunks:

#### Phase 4a — Cheap leaves

**Files ported:** `DatabaseGroupTag.vue` → `DatabaseGroupTag.tsx`; Vue `tree.ts` composable → React hook `tree.ts` (`useSQLEditorTreeByEnvironment`).

Both are consumer-less at Phase 4a ship — they're staged for Phase 4d's `ConnectionPane.tsx`. Tests: DatabaseGroupTag (4).

#### Phase 4b — Context menu + FeatureModal

**Files ported:** `actions.tsx` → React `useConnectionMenu()` hook + `setConnection()` plain function + `ConnectionContextMenu.tsx` (imperative handle, mirrors the TabContextMenu invisible-trigger pattern).

**New:** `src/react/components/ui/feature-modal.tsx` — shadcn Dialog wrapping the subscription paywall strings from `dynamic.subscription.features.*`. Used by ConnectionPane to gate batch-query + database-group features.

Both are standalone — Phase 4b doesn't swap any callers.

#### Phase 4c — DatabaseGroup data table

**Files ported:** `DatabaseGroupDataTable.vue` → `DatabaseGroupDataTable.tsx` (shared component, used by SQL Editor + elsewhere); then `DatabaseGroupTable.vue` → `DatabaseGroupTable.tsx` (thin wrapper: search box + data table + controlled selection).

Consumer-less — Phase 4d's `ConnectionPane.tsx` picks it up. The broader migration of non-SQL-Editor consumers of the Vue DataTable is tracked separately.

#### Phase 4d — ConnectionPane shell + caller swap

**Files ported:** `ConnectionPane.vue` → `ConnectionPane.tsx` (uses Phase 4a/b/c artifacts, Phase 3 leaves, Stage 11 Tree primitive, React AdvancedSearch).

**Caller swap:** `ConnectionPanel.vue` (still Vue) replaces `<ConnectionPane>` with `<ReactPageMount page="ConnectionPane" :show="isOpen" />`. Phase 5 then swaps the outer `ConnectionPanel` shell.

**Deliverables:** `ConnectionPane.tsx` + 5 tests + delete Vue `ConnectionPane/*.vue` + `actions.tsx` + `tree.ts` + `DatabaseHoverPanel/*.vue` + `TreeNode/*.vue` + `index.ts`.

### Phase 5 — Shell + caller swap + cleanup

**Files ported:** `ConnectionPanel.vue` (the Drawer shell) → `ConnectionPanel.tsx` (uses shadcn `Sheet`).

**Approach:**
- `ConnectionPanel.tsx` wraps `ConnectionPane` in `<Sheet open onOpenChange>`. Width tier from the sheet primitive; override with `width="wide"` (832px). Reads the Vue-side `showConnectionPanel` ref through props passed by `SQLEditorHomePage.vue`'s `ReactPageMount`.

**Caller swap:**
- `SQLEditorHomePage.vue`: `<ConnectionPanel v-model:show="showConnectionPanel" />` → `<ReactPageMount page="ConnectionPanel" :open="showConnectionPanel" :onOpenChange="(v) => (showConnectionPanel = v)" />`.

**Delete:** All ported Vue files + their `index.ts`.

**Verify:**
- `rg "TabList/|ConnectionPanel/|ConnectionPane"` → zero live imports outside the React tree.
- `pnpm fix && type-check && test --run && check` — all green.

## 5. Per-phase checklist

### Phase 1 — TabList leaves
- [ ] `TabItem/AdminLabel.tsx`
- [ ] `TabItem/Prefix.tsx`
- [ ] `TabItem/Suffix.tsx`
- [ ] `TabItem/Label.tsx` + test
- [ ] i18n keys ported (rename prompt, close confirm)
- [ ] Gates green

### Phase 2 — TabList
- [ ] `TabList.tsx` + dnd-kit wiring + test
- [ ] `TabItem.tsx` + test
- [ ] `TabContextMenu.tsx` + test
- [ ] `SQLEditorHomePage.vue` swap to `<ReactPageMount page="TabList" ... />`
- [ ] Delete Vue `TabList/*` files + `TabList/index.ts`
- [ ] Gates green

### Phase 3 — Connection tree nodes + HoverPanel
- [x] Per-node `.tsx` files + shared `Label.tsx` + test
- [x] `DatabaseHoverPanel.tsx` + `hover-state.ts` + test
- [x] Gates green
- [-] `ConnectionPane/tree.ts` — deferred to Phase 4 (Vue composable, not pure TS; coupled to ConnectionPane consumer)

### Phase 4a — Cheap leaves
- [x] `DatabaseGroupTag.tsx` + test
- [x] `tree.ts` React hook (`useSQLEditorTreeByEnvironment`)
- [x] Gates green

### Phase 4b — Context menu + FeatureModal
- [x] `actions.tsx` React port — `setConnection()` + `useConnectionMenu()`
- [x] `ConnectionContextMenu.tsx` imperative handle
- [x] `feature-modal.tsx` UI primitive
- [x] Tests + 6 i18n keys across 5 locales
- [x] Gates green

### Phase 4c — DatabaseGroup data table
- [x] `DatabaseGroupDataTable.tsx` (shared React component; `showActions` dropdown variant deferred to when a non-SQL-Editor caller ports)
- [x] `DatabaseGroupTable.tsx` (SQL Editor wrapper, inline `useDBGroupListByProject` via React hooks)
- [x] Tests (DataTable 6, Table 2)
- [x] Gates green

### Phase 4d — ConnectionPane shell + caller swap
- [x] `ConnectionPane.tsx` + 5 tests (+ 8 new i18n keys)
- [x] Gates green
- [-] AdvancedSearch scope chips (label / engine / instance / project filters) — deferred; React port uses `SearchInput` free-text only for now. Tracked as Phase 4e follow-up.
- [ ] Swap `ConnectionPanel.vue`'s `<ConnectionPane>` → `<ReactPageMount page="ConnectionPane" />` (Phase 5)
- [ ] Delete Vue `ConnectionPane/*.vue` + `actions.tsx` + `tree.ts` + `DatabaseHoverPanel/*` + `TreeNode/*` + `index.ts` (Phase 5)

### Phase 5 — Shell + cleanup
- [x] `ConnectionPanel.tsx` + 5 tests (shadcn Sheet, wide tier)
- [x] Swap `SQLEditorHomePage.vue` caller
- [x] Delete entire Vue `ConnectionPanel/` tree (ConnectionPanel.vue, ConnectionPane/*, TreeNode/*, DatabaseHoverPanel/*, actions.tsx, tree.ts, DatabaseGroupTag.vue, DatabaseGroupTable.vue, index.ts files)
- [x] `grep` confirms zero live Vue callers
- [x] Gates green + 1 new i18n key (`sql-editor.manage-connections`) across 5 locales

## 6. Manual UX verification (after all phases)

1. Open SQL editor → tabs render; the active tab is highlighted.
2. Click a tab → it becomes active; the editor content swaps.
3. Drag a tab → order persists in pinia store.
4. Right-click a tab → context menu shows; "Close others" leaves only the clicked tab.
5. Click the "+" button → a new unconnected tab opens with focus.
6. Mid-click a tab → it closes.
7. Click "Select a database to start" / "Connect to database" → ConnectionPanel drawer slides in from the right at ~800px width.
8. Panel renders instance tree grouped by environment label.
9. Hover on a database row for ~300ms → DatabaseHoverPanel appears with metadata.
10. Click a database → drawer closes, editor switches to that connection.
11. Enable batch-mode → multi-select checkboxes appear; selected databases surface as closable tags at the top.
12. Click a DatabaseGroupTag's close → group is removed from the tab's `batchQueryContext`.
13. Keyboard: Esc closes the drawer.

## 7. Out of scope (deferred)

- `SQLEditorHomePage.vue` migration (root container; depends on all children being React).
- `SchemaPane/*` — separate stage.
- `ResultView/VirtualDataTable/*` — separate stage; needs perf spike first.
- `EditorPanel.vue`, `Panels/*` (TablesPanel / ViewsPanel / etc.) — separate stage.

## 8. Risks & open questions

- **`@dnd-kit` perf at 30+ tabs:** users routinely open many tabs. The dnd-kit sortable is well-optimized but we should spot-check in Phase 2 with 30–50 tabs.
- **`NTree` behavior gaps:** our React `Tree` primitive was tuned for worksheet-size trees (hundreds of rows). ConnectionPane could surface 10k+ databases. Phase 3 will benchmark; if react-arborist's virtualization isn't sufficient, fall back to a simpler virtualization. Note: only the database-level leaves need virtualization; instances/environments are ≤200 nodes.
- **`usePreventBackAndForward`:** TabList binds this globally; verify only-one-installer wins so the React mount doesn't duplicate listeners.
- **`vue-draggable-plus` attribute parity:** the Vue component has `ghost-class="ghost"` styling during drag. Need equivalent ghost-state styling in dnd-kit's DragOverlay.
- **`RichDatabaseName`:** Vue-only. Options: (a) port a minimal React `RichDatabaseName` helper now (since ConnectionPanel and probably SchemaPane both need it), (b) inline a smaller display (engine icon + title) locally for Stage 14 and refactor later. Prefer (a) — lands as a small shared component in `src/react/components/RichDatabaseName.tsx`, reusable in Stage 15+.
- **`FeatureBadge`:** Used for the batch-query paywall. If a React version doesn't exist, mount via ReactPageMount OR port a minimal React version. Decide in Phase 4.

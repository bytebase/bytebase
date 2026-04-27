# SQL Editor React Migration — Stage 15 Design

**Date:** 2026-04-26
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Port the entire `SchemaPane/` subtree (the schema browser shown when `asidePanelTab === "SCHEMA"`) to React. After this stage, all four AsidePanel tabs (Worksheet, **Schema**, History, Access) are React; the only Vue surface left in the SQL Editor sidebar is `AsidePanel.vue` itself, which simply mounts the four React children.

**Non-goals:**
- Migrating `AsidePanel.vue` — host shell, still Vue; mounts the React panes.
- Migrating `EditorPanel/*`, `ResultView/*`, or `SQLEditorHomePage.vue` — separate stages.
- Refactoring `useDBSchemaV1Store`, `useSQLEditorTabStore`, `useSQLEditorContext`, or `useCurrentTabViewStateContext` — read/written via `useVueState` and direct method calls.
- Visual or behavioral changes — parity only.
- Changing the `NodeType` / `NodeTarget` discriminated unions or the wire shape of `tab.treeState.keys` (expanded/selected key persistence). The React port must produce **byte-identical** key strings so users' persisted tree state survives the swap.
- Porting the schema viewer modal body (the surface that opens from `actions.tsx` "View schema") — only the trigger is in scope.

## 2. Inventory

### 2.1 SchemaPane root + utilities (7 files, ~2,369 LOC)

| File | LOC | Notes |
|---|---|---|
| `SchemaPane.vue` | 546 | Root orchestrator: search input, NTree ↔ FlatTableList toggle, NDropdown context menu, hover-state provider, expand/select wiring. |
| `FlatTableList.vue` | 198 | Alt-UI used when a database has more than ~1000 tables; NVirtualList of expandable rows. |
| `SyncSchemaButton.vue` | 78 | Popover button calling `dbSchemaStore.syncDatabase`. |
| `tree.ts` | 862 | Pure TS: `NodeType` (16 variants), `NodeTarget`, `TreeNode` types + `keyForNodeTarget` (large switch) + `buildDatabaseSchemaTree(metadata)` recursive builder. |
| `actions.tsx` | 640 | Right-click menu: `useActions` (selectAll, viewDetail, openNewTab, copyName, …) + `useDropdown` (NDropdown imperative show/position). Heavy on Vue composables + JSX icon rendering for naive-ui DropdownOption. Despite the `.tsx` extension, this is a Vue module. |
| `click.ts` | 42 | 250ms single/double-click discriminator built on Emittery. |
| `index.ts` | 3 | Default-export re-export. |

### 2.2 TreeNode/ (20 files, ~750 LOC)

| File | LOC | Notes |
|---|---|---|
| `Label.vue` | 86 | Dispatcher — switches on `node.meta.type` to render the right `*Node`; passes through `HighlightLabelText` for search highlighting. |
| `CommonNode.vue` | 41 | Shared template: icon slot + text + suffix slot. |
| `ColumnNode.vue` | 96 | Reads `dbSchemaStore.getTableMetadata` for PK/index badges; renders `NTag` for the column type. |
| `IndexNode.vue` | 44 | Unique / primary badge logic. |
| `CheckNode.vue` | 59 | Reads metadata for the constraint expression. |
| `TextNode.vue` | 40 | Expandable section headers ("Tables", "Views", …). |
| `DummyNode.vue` | 34 | Error / empty placeholder. |
| 13 thin variants | ~380 | `TableNode` 21, `ViewNode` 26, `SchemaNode` 24, `DatabaseNode` 39, `ProcedureNode` 39, `FunctionNode` 37, `SequenceNode` 26, `TriggerNode` 26, `PackageNode` 26, `ExternalTableNode` 23, `ForeignKeyNode` 28, `PartitionTableNode` 28, `DependencyColumnNode` 30 — each is essentially `<CommonNode icon={X} label={node.label} />`. |
| `index.ts` | 3 | Re-exports. |

### 2.3 HoverPanel/ (10 files, ~440 LOC)

| File | LOC | Notes |
|---|---|---|
| `HoverPanel.vue` | 109 | Fixed-position popover; mouseenter/leave restart the timer; viewport-clamping math. |
| `TableInfo.vue` | 78 | name / engine / row-count / size / collation / comment. |
| `ColumnInfo.vue` | 85 | type / nullability / default / comment. |
| `ViewInfo.vue` | 25 | name / engine / row-count / comment. |
| `ExternalTableInfo.vue` | 35 | location / data format. |
| `TablePartitionInfo.vue` | 48 | partition type / key / value. |
| `InfoItem.vue` | 23 | Reusable `(label, value)` row. |
| `CommonText.vue` | 11 | Text wrapper. |
| `hover-state.ts` | 25 | Typed wrapper around the shared `EditorCommon/hover-state.ts` context (key = `"schema-pane"`). |
| `index.ts` | 6 | Re-exports. |

### 2.4 Caller integration

`AsidePanel.vue:59` mounts `<SchemaPane v-if="asidePanelTab === 'SCHEMA'" />`. The other three tabs (`WorksheetPane`, `HistoryPane`, `AccessPane`) and the gutter (`GutterBar`) are already mounted via `<ReactPageMount …>`, so the harness is in place. SchemaPane is the last Vue child of `AsidePanel.vue`.

### 2.5 Stores & context the subtree depends on

| Source | What it provides |
|---|---|
| `useSQLEditorTabStore` | currentTab; `currentTab.treeState.keys` (persisted expand/select state); `openTabList`. |
| `useDBSchemaV1Store` | `getOrFetchDatabaseMetadata`, `getTableMetadata`, `getSchemaMetadata`, `syncDatabase`. |
| `useDatabaseV1Store` | engine detection, permissions. |
| `useConnectionOfCurrentSQLEditorTab` | reactive `{ database, schema }` for the active tab. |
| `useSQLEditorContext` (provide/inject) | `schemaViewer` ref, `events` (Emittery), `pendingInsertAtCaret`. |
| `useCurrentTabViewStateContext` | `updateViewState`, `viewState` (used by actions to flip detail panes). |
| `provideHoverStateContext("schema-pane")` | local provide/inject for `{ state, position, update }` consumed by HoverPanel. |

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NTree` | `Tree` primitive (`src/react/components/ui/tree.tsx`) | ✓ from Stage 11; reused in Stage 14 ConnectionPane |
| `NTree.renderLabel` (search highlight) | Render `<HighlightLabelText>` inside our `Tree` row renderer | New small component if not already in tree (verify reuse from Stage 14) |
| `NDropdown` (imperative) | `ContextMenu` primitive (`src/react/components/ui/context-menu.tsx`) | ✓ added Stage 14b |
| `NPopover` | `Popover` primitive | ✓ |
| `NVirtualList` (FlatTableList) | `react-window` (already a dep) — decide between `FixedSizeList` and `VariableSizeList` in Phase 4d | Decision in 4d |
| `NEmpty` | Inline `<div>` with i18n empty text | n/a |
| `NTag` (column-type badge) | `Badge` primitive | ✓ |
| `NIcon` | Direct `lucide-react` icon | ✓ |
| `useDelayedValue` (`src/composables/useDelayedValue.ts`, 47 LOC) | New `useDelayedValue` React hook (`src/react/hooks/useDelayedValue.ts`); ref-stored timer; identical `update(value, "before"\|"after", overrideDelay?)` signature | New util |
| `provide/inject` HoverStateContext | React `HoverStateProvider` + `useHoverState` (scoped to `SchemaPane.tsx`; not a global context) | New |
| `@vueuse/core useElementSize` | `useResizeObserver` hook (already inline-used in Stage 14) | Reuse |
| `@vueuse/core useEventListener` | `useEffect + addEventListener` | Inline |
| `@vueuse/core useMounted` | `useState(false) + useEffect(()=>setTrue,[])` | Inline |
| `@vueuse/core computedAsync` | `useEffect + useState` async pattern; cancel via AbortController where appropriate | Inline |
| `@vueuse/core refDebounced` | Tiny `useDebouncedValue(value, ms)` hook (or reuse if Stage 14 landed one) | Reuse / inline |
| `@vueuse/core useClipboard` | `navigator.clipboard.writeText` (already used in Stage 14b actions) | Inline |
| `Emittery` (click.ts) | Keep — framework-agnostic | ✓ |
| `protobuf-es` metadata types | Same imports | ✓ |
| `lodash-es` | Same imports | ✓ |
| `RichDatabaseName` (Vue) | Inline minimal display, or reuse the React helper if Stage 14 landed one (`src/react/components/RichDatabaseName.tsx`) | Reuse if present |

## 4. Architecture & phases

Five phases. Each commit-able independently, each green on `pnpm fix / type-check / test / check`.

### Phase 1 — Pure utilities + leaf TreeNodes

**Files ported:** `tree.ts` (copy as-is into `src/react/components/sql-editor/SchemaPane/schemaTree.ts` — already pure TS, no Vue deps), `click.ts` (likewise), and the React equivalents of `useDelayedValue` + `EditorCommon/hover-state.ts` (`useDelayedValue.ts` + `HoverStateProvider`/`useHoverState` exported from `src/react/components/sql-editor/SchemaPane/hover-state.tsx`). Plus the 16 thin leaf TreeNodes that just compose `CommonNode` + an icon + label, plus `CommonNode.tsx`, `DummyNode.tsx`, `TextNode.tsx`.

Add or reuse `HighlightLabelText.tsx`. (Search Stage 14 output first — ConnectionPane likely already has one. If yes, lift to a shared location; if no, write a small new one and use it from both surfaces in a follow-up.)

**Rename note:** The new tree builder is renamed `schemaTree.ts` (not `tree.ts`) to avoid colliding with `src/react/components/sql-editor/ConnectionPane/tree.ts` from Stage 14.

**Approach:** No UI rewiring yet — these compile but aren't called. `schemaTree.ts` gets a unit test (`schemaTree.test.ts`) that asserts `buildDatabaseSchemaTree(fixtureMetadata)` produces the expected key paths for representative MySQL + Postgres + Oracle metadata fixtures. **This test is the regression net for the rest of the migration** — every subsequent phase keeps it green.

**Deliverables:** ~16 `.tsx` + 3 `.ts/.tsx` (utils) + 1 test.

### Phase 2 — Heavy TreeNodes + Label dispatcher

**Files ported:** `ColumnNode.tsx` (reads `dbSchemaStore.getTableMetadata` via `useVueState`), `IndexNode.tsx`, `CheckNode.tsx`, and `Label.tsx` (the dispatcher that switches on `NodeType`).

**Approach:** Each node is a presentational component with a stable signature `{ node: TreeNode; keyword: string }`. `Label.tsx` is a `switch (node.meta.type)` returning the right child. Verify rendered output against Vue side-by-side for the same `NodeTarget` fixtures.

**Deliverables:** 4 `.tsx` + 1 `Label.test.tsx` covering all 16 dispatch branches via fixtures.

### Phase 3 — HoverPanel

**Files ported:** `HoverPanel.tsx`, `TableInfo.tsx`, `ColumnInfo.tsx`, `ViewInfo.tsx`, `ExternalTableInfo.tsx`, `TablePartitionInfo.tsx`, `InfoItem.tsx`, `CommonText.tsx`.

**Approach:** Use the `Popover` primitive in **controlled** mode — the consumer (`SchemaPane.tsx`) drives `open`, `anchor`, and `state` via the React `useHoverState()` context from Phase 1. Viewport clamping uses `useResizeObserver` + bounds math (matches Vue's clamp at `HoverPanel.vue:90–100`). Render whichever `*Info` matches `state.column ?? state.partition ?? state.table ?? state.externalTable ?? state.view`. Mount via `getLayerRoot("overlay")` — non-modal hover surface goes into the standard overlay family.

**Deliverables:** 8 `.tsx` + `HoverPanel.test.tsx` (mock dbSchema metadata, verify the right pane renders for each state shape, verify clamp math).

### Phase 4 — actions + ContextMenu wiring + SyncSchemaButton + FlatTableList (sliced)

The largest phase. Slice into 4a–4d so each can land cleanly.

#### 4a — `useSchemaPaneActions` (logic only, no UI)
Rewrite Vue `actions.tsx` as a React module. Two exports:
- `useSchemaPaneActions()` returns invokable handlers: `selectAllFromTableOrView`, `viewDetail`, `openNewTab`, `copyName`, `insertColumnAtCaret`, etc. These call into `useSQLEditorTabStore`, `useSQLEditorContext().events`, `useCurrentTabViewStateContext().updateViewState`.
- `useSchemaPaneContextMenu(target)` returns the `ContextMenuItem[]` array keyed by the `NodeTarget` discriminator. Mirror Vue branching exactly — **do not reorder items**; the order is the user-visible menu.

Snapshot test: feed an exhaustive set of `NodeTarget` fixtures and assert the menu shape. This is the easiest place to introduce subtle regressions.

#### 4b — `SchemaContextMenu`
Imperative-handle wrapper around the `ContextMenu` primitive (matches `ConnectionContextMenu.tsx` from Stage 14b). Exposes `open(event, target)`. Internally renders items from `useSchemaPaneContextMenu(target)`.

#### 4c — `SyncSchemaButton`
Popover (last-sync timestamp + tooltip) + Button (RefreshCcw icon). On click, calls `dbSchemaStore.syncDatabase` then `dbSchemaStore.getOrFetchDatabaseMetadata`. State machine: idle → syncing → idle, with disabled button while syncing.

#### 4d — `FlatTableList`
Virtual list (`react-window` `VariableSizeList` if rows can expand inline, else `FixedSizeList`). Receives the same `tables[]` shape Vue uses. Emits `select`, `selectAll`, `contextmenu` callbacks. Verify smoothness with a synthetic 5k-row fixture before merging.

**Deliverables:** ~5 `.tsx` + 4 tests.

### Phase 5 — SchemaPane shell + caller swap + cleanup

**Files ported:** `SchemaPane.tsx` (the orchestrator).

**Approach:**
- Internal state: `searchPattern` (debounced 200ms via local hook), `expandedKeys`/`selectedKeys` (synced **bidirectionally** with `tabStore.currentTab.treeState.keys` via `useVueState({deep:true})` — same pattern as TabList from Stage 14), `flatMode` (computed: `tableCount > 1000`).
- Compose: `<HoverStateProvider>` → mount `<SchemaContextMenu ref={menuRef}>` → conditional `<Tree>` or `<FlatTableList>` → `<HoverPanel>` (anchored via context).
- Subscribe to `dbSchemaStore.getOrFetchDatabaseMetadata(currentDatabase)` via `useEffect`; rebuild via `buildDatabaseSchemaTree` when metadata version bumps.
- Caller swap: `AsidePanel.vue:59` from `<SchemaPane v-if="..." />` to `<ReactPageMount page="SchemaPane" v-if="..." container-class="h-full" />`.
- Register `SchemaPane` in `src/react/mount.ts` page registry.
- Delete the entire Vue `SchemaPane/` directory once `grep` confirms zero callers.
- Verify `EditorCommon/hover-state.ts` is still used by other Vue surfaces; delete if not.

**Deliverables:** 1 `.tsx` + `SchemaPane.test.tsx` + caller swap + Vue tree deletion + i18n key audit (no new keys expected — schema-pane keys are display-only and ported as-is).

## 5. Per-phase checklist

### Phase 1 — Utilities + leaf TreeNodes
- [ ] `schemaTree.ts` ported (`src/react/components/sql-editor/SchemaPane/schemaTree.ts`)
- [ ] `schemaTree.test.ts` covering each `NodeType` branch
- [ ] `click.ts` ported (or imported from existing location if framework-agnostic)
- [ ] `useDelayedValue.ts` React hook + test
- [ ] `HoverStateProvider` / `useHoverState` context + types
- [ ] 16 leaf `TreeNode/*.tsx` files + shared `CommonNode.tsx`, `TextNode.tsx`, `DummyNode.tsx`
- [ ] `HighlightLabelText.tsx` (or reuse if Stage 14 landed one)
- [ ] Gates green

### Phase 2 — Heavy nodes + Label dispatcher
- [ ] `ColumnNode.tsx`, `IndexNode.tsx`, `CheckNode.tsx`
- [ ] `Label.tsx` dispatcher
- [ ] `Label.test.tsx` (16 branches)
- [ ] Gates green

### Phase 3 — HoverPanel
- [ ] `HoverPanel.tsx` + 6 `*Info.tsx` + `InfoItem.tsx` + `CommonText.tsx`
- [ ] `HoverPanel.test.tsx`
- [ ] Viewport clamp math verified
- [ ] Gates green

### Phase 4a — Actions logic
- [ ] `useSchemaPaneActions.ts` + test
- [ ] `useSchemaPaneContextMenu(target)` snapshot test across all NodeTarget shapes
- [ ] Gates green

### Phase 4b — Context menu
- [ ] `SchemaContextMenu.tsx` + imperative ref handle + test
- [ ] Gates green

### Phase 4c — Sync button
- [ ] `SyncSchemaButton.tsx` + test
- [ ] Gates green

### Phase 4d — Flat table list
- [ ] `FlatTableList.tsx` + virtualization decision documented inline
- [ ] Test (synthetic 5k-row fixture for scroll smoothness)
- [ ] Gates green

### Phase 5 — Shell + caller swap + cleanup
- [ ] `SchemaPane.tsx` + test
- [ ] `mount.ts` registers page
- [ ] `AsidePanel.vue` swapped to `<ReactPageMount page="SchemaPane" />`
- [ ] `grep` confirms zero live Vue callers of `SchemaPane.vue` and the TreeNode/HoverPanel files
- [ ] Delete entire Vue `SchemaPane/` directory
- [ ] Verify `EditorCommon/hover-state.ts` is still used; delete if not
- [ ] Gates green

## 6. Manual UX verification (after all phases)

1. Open SQL Editor → connect to a MySQL database with ~10 tables → schema tree renders, expanded to schema level.
2. Click a table → row highlights; tree state persists across tab switches and survives page reload.
3. Type `users` in the search box → matches highlight; non-matching subtrees collapse; clear search → tree restores prior expand state.
4. Hover a column for ~1s → HoverPanel appears with type / nullability / default; move mouse off → panel hides after 350ms; quick re-hover → panel re-appears immediately within the after-window.
5. Right-click a table → context menu shows Copy / Select * / View schema / Open in new tab; click Select * → SQL is inserted at editor caret.
6. Right-click a column → menu shows column-specific actions (Insert at caret, Copy name).
7. Click Sync icon → spinner; metadata refetches; tree re-renders without losing expand/select state.
8. Connect to a database with >1000 tables → SchemaPane renders FlatTableList instead of NTree → scroll smoothly, expand a row → columns appear inline.
9. Repeat 1–6 across MySQL, Postgres (schemas), and Oracle (packages) to cover engine-specific node types.
10. Switch the AsidePanel tab Worksheet → Schema → History → Schema → tree re-renders without losing expand/select state.
11. Disconnect (clear connection) → AsidePanel hides SchemaPane (`v-if`); reconnect → fresh fetch.

## 7. Out of scope (deferred)

- `AsidePanel.vue` migration (host shell — Vue-only after this stage; mounts the four React panes via `ReactPageMount`).
- `EditorPanel.vue`, `Panels/*` (TablesPanel / ViewsPanel / IndexesPanel / etc.) — separate stage.
- `ResultView/VirtualDataTable/*` — separate stage; needs perf spike first.
- `SQLEditorHomePage.vue` — root container; depends on EditorPanel.
- Schema viewer modal body (the modal that opens from `actions.tsx` "View schema" — only the trigger is ported; modal body remains Vue).

## 8. Risks & open questions

- **`buildDatabaseSchemaTree` regression risk.** 862 LOC of recursive metadata→tree code with 16 type branches. Tree-key paths must match exactly so `tabStore.currentTab.treeState.keys` (persisted expand/select state) keeps working across the Vue→React swap. **Mitigation:** Phase 1 lands a unit test fixture per engine before any orchestrator changes.

- **`Tree` primitive at 10k+ rows.** Stage 14 risks list flagged this for ConnectionPane (≤10k databases). SchemaPane can hit 10k+ columns when a single big table is fully expanded. **Mitigation:** FlatTableList already handles >1000-table DBs; for column-explosion within a single table, apply the same threshold — fall back to a virtualized leaf list. Benchmark in Phase 5.

- **Hover-state timing.** 1000ms-before / 350ms-after is fragile in React because effect cleanup races setTimeout callbacks. **Mitigation:** ref-stored timer (not state); cancel + reschedule unconditionally in the same effect; test with rapid mouseenter/leave bursts. Mirror `useDelayedValue.ts` line-for-line so the semantics are identical.

- **Search-highlight + virtual scroll.** Vue uses NTree's `renderLabel` callback so the highlight DOM is rebuilt only for visible rows. Naive React port could re-render all rows on every keystroke. **Mitigation:** memoize the row renderer; key the highlight `useMemo` on `(keyword, node.id)`. Benchmark with a 5k-column fixture.

- **`useSQLEditorContext` provide/inject.** Many `actions.tsx` paths call `context.events.emit(...)` to drive caret-insert and pendingInsert state. The React port reads state via `useVueState` and calls methods directly for emits. Need to verify `events` (Emittery) is reachable as a stable reference from a Pinia setup store. It should be — typically held in a `ref(emittery)` — but confirm in Phase 4a.

- **`provideHoverStateContext` is per-key.** Other surfaces may use the shared `EditorCommon/hover-state.ts` with a different key. Phase 5 cleanup must check before deleting the shared module. If still in use, leave it; the React port has its own scoped provider.

- **`actions.tsx` has 640 LOC of branchy menu logic.** The easiest place to introduce subtle 1:1 regressions. **Mitigation:** snapshot test in 4a covering every `NodeTarget` discriminator. Sit Vue and React versions side-by-side during 4b smoke testing.

- **Naming collision: `tree.ts`.** A file at `src/react/components/sql-editor/SchemaPane/tree.ts` would shadow `…/ConnectionPane/tree.ts` in fuzzy searches. Renamed to `schemaTree.ts` for clarity.

- **`FlatTableList` virtualization library choice.** `react-window` is already a dep used by the worksheet table; prefer it over introducing `react-virtualized` or `@tanstack/virtual`. Confirm `react-window`'s expandable-row support meets parity with `NVirtualList`'s; if not, fall back to a non-virtualized list with a 1k-row hard cap matching Vue's existing threshold.

- **HighlightLabelText source of truth.** If Stage 14 landed a React equivalent for ConnectionPane search, lift it to a shared location (`src/react/components/sql-editor/HighlightLabelText.tsx`) before Stage 15 imports it; otherwise we'll end up with two near-identical copies.

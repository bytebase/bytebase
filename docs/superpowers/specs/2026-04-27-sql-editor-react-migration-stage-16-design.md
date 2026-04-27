# SQL Editor React Migration — Stage 16 Design

**Date:** 2026-04-27
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Port the leaf panels under `EditorPanel/Panels/*` to React so the React `SchemaPane` "View detail" menu actions land in real React surfaces. After this stage the per-view drill-downs (Tables / Views / Functions / Procedures / Sequences / External Tables / Packages / Triggers / Info) all render as React; only `Panels.vue` (the small `viewState.view` switch) and `DiagramPanel` remain Vue under `EditorPanel/Panels/`.

**Non-goals:**
- Migrating `EditorPanel.vue`, `StandardPanel`, `TerminalPanel`, `ResultPanel` — separate stage.
- Migrating `Panels.vue` itself — keep it as the Vue orchestrator that switches on `viewState.view` and mounts the right React panel via `<ReactPageMount>`. Same split we used for `AsidePanel.vue` in Stage 15.
- Porting `DiagramPanel` (cytoscape-backed `SchemaDiagram` Vue tree) — kept Vue inside `Panels.vue`'s switch; a future stage handles it.
- Porting `MonacoEditor` for `CodeViewer` — wrap the existing React Monaco host (used elsewhere in the editor) rather than migrate Monaco itself.
- Refactoring `useCurrentTabViewStateContext` — read/write via `useVueState` + direct `tabStore.updateTab` calls (same pattern used in Stage 15's SchemaPane).
- Refactoring the `availableActions` computation (already ported in Stage 15 inside `SchemaPane.tsx`'s `useAvailableActions` hook — Phase 1 of this stage extracts it to a shared util).
- Visual / behavioral changes — parity only.

## 2. Inventory

### 2.1 Panel roots (11 panels, ~4,857 LOC, 39 files)

| Panel | Files | LOC | Pattern |
|---|---|---|---|
| `Panels.vue` | 1 | 149 | Switch on `viewState.view`. **Stays Vue** — host-only. |
| `TablesPanel/` | 9 | 1,522 | List → multi-tab detail (Columns / Indexes / FKs / Checks / Triggers / Partitions). |
| `ViewsPanel/` | 7 | 884 | List → multi-tab detail (Definition / Columns / Dependency Columns). |
| `ExternalTablesPanel/` | 4 | 454 | List → single-tab detail (Columns). |
| `SequencesPanel/` | 3 | 297 | List-only. |
| `TriggersPanel/` | 3 | 253 | List → CodeViewer. |
| `ProceduresPanel/` | 3 | 256 | List → CodeViewer. |
| `FunctionsPanel/` | 3 | 246 | List → CodeViewer. |
| `PackagesPanel/` | 3 | 253 | List → CodeViewer. |
| `InfoPanel/` | 2 | 264 | Dashboard composing mini-tables of every schema-object kind. |
| `DiagramPanel/` | 2 | 26 | Thin wrapper around `SchemaDiagram` Vue. **Stays Vue.** |
| `common/` (shared) | 3 | 250 | `SchemaSelectToolbar.vue`, `CodeViewer.vue`, `EllipsisCell.vue`. |

### 2.2 Shared sub-tables (reused across detail panels)

| Component | Used by | Notes |
|---|---|---|
| `ColumnsTable` | TablesPanel detail, ViewsPanel detail, ExternalTablesPanel detail | Most-reused; column-header set varies slightly per parent. |
| `IndexesTable` | TablesPanel detail | |
| `ForeignKeysTable` | TablesPanel detail | |
| `ChecksTable` | TablesPanel detail (when `checkConstraints.length > 0`) | |
| `TriggersTable` | TablesPanel detail + standalone TriggersPanel | |
| `PartitionsTable` | TablesPanel detail | |
| `DependencyColumnsTable` | ViewsPanel detail (DEPENDENCY-COLUMNS tab) | |
| `DefinitionViewer` | ViewsPanel detail (DEFINITION tab) | Monaco read-only + Format checkbox; same shape as `CodeViewer`. |

### 2.3 Mounting & state

`Panels.vue` (kept Vue) switches on `viewState.view`:
```html
<InfoPanel v-if="viewState.view === 'INFO'" />
<TablesPanel v-if="viewState.view === 'TABLES'" />
…
<DiagramPanel v-if="viewState.view === 'DIAGRAM'" />
```
Each panel re-mounts on tab change via `:key="tab?.id"`. Stage 16 swaps every `<XPanel />` (except `DiagramPanel`) to `<ReactPageMount page="XPanel" />` — no other Vue surface needs changing.

`useCurrentTabViewStateContext()` exposes `{ viewState, selectedSchemaName, availableActions, updateViewState }`. The React side reads/writes via direct store access (`tabStore.currentTab.viewState`, `tabStore.updateTab(...)`) — matches Stage 15 SchemaPane.

### 2.4 Stores read

`useSQLEditorTabStore`, `useDBSchemaV1Store`, `useDatabaseV1Store`, `useConnectionOfCurrentSQLEditorTab`, `useSQLEditorContext` (CodeViewer only — for AI panel size + emit). No `useDialog` / `useMessage` / `useNotification` imperative APIs — toasts and modals are upstream.

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NDataTable` (16 uses) + `useAutoHeightDataTable` | shadcn `DataTable` (existing in `src/react/components/ui/`) wrapping TanStack Table + native overflow scroll for virtualization at the cell level. Auto-height via `useResizeObserver` (same as Stage 14). | ✓ DataTable exists; Phase 1 adds a thin `useAutoHeightDataTable` React util |
| `NSelect` (schema dropdown) | shadcn `Select` | ✓ |
| `NButton` (back / search trigger) | shadcn `Button` | ✓ |
| `NCheckbox` (Format toggle, column toggles) | shadcn `Checkbox` | ✓ |
| `NTabs` / `NTab` (detail sub-tabs) | shadcn `Tabs` / `TabsTrigger` / `TabsPanel` | ✓ |
| `NSplit` (CodeViewer editor + AI pane) | Reuse the existing React resizable container pattern (already used by `WorksheetPane` / `ConnectionPanel`) — no new primitive | Reuse |
| `MonacoEditor` (CodeViewer + DefinitionViewer) | The existing React Monaco wrapper (used by `EditorAction`, etc.) | ✓ |
| `RichEngineName`, `EngineIcon` | Same engine-icon-path approach used in Stage 14/15 (`EngineIconPath` map + lucide icons) | Reuse |
| `<i18n-t>` (interpolated counter labels) | `<Trans>` from `react-i18next` | ✓ |
| `useAutoHeightDataTable` composable | New React util `useTableContainerHeight` — `ResizeObserver` + viewport math, returns `{ ref, maxHeight }` | New small util |
| `<ReactPageMount page="DatabaseChooser">` (in `Panels.vue` toolbar) | Already React — unchanged | ✓ |
| `<ReactPageMount page="OpenAIButton">` (in `CodeViewer`) | Replaced by direct React import inside the React `CodeViewer` | ✓ |
| `availableActions` (Vue context) | Lift `SchemaPane.tsx`'s local `useAvailableActions` to a shared util in `src/react/components/sql-editor/availableActions.ts` so both Stage 15 SchemaPane and Stage 16 panel toolbar can consume it | Lift in Phase 1 |

## 4. Architecture & phases

Four phases. Each commit-able independently, each green on `pnpm fix / type-check / test / check`.

### Phase 1 — Shared infrastructure

**Files added:**
- `src/react/components/sql-editor/Panels/common/SchemaSelectToolbar.tsx` — schema dropdown (Select) bound to `tabStore.currentTab.viewState.schema`. Mirrors Vue's bidirectional sync with `connection.schema` via a small effect.
- `src/react/components/sql-editor/Panels/common/CodeViewer.tsx` — header (back button + format checkbox + OpenAIButton) + Monaco read-only pane. Receives `{ statement, language, schema, onBack }` props.
- `src/react/components/sql-editor/Panels/common/DefinitionViewer.tsx` — narrower variant of CodeViewer for the ViewsPanel DEFINITION tab.
- `src/react/components/sql-editor/Panels/common/EllipsisCell.tsx` — cell renderer with truncate + `HighlightLabelText` keyword.
- `src/react/components/sql-editor/Panels/common/useViewStateNav.ts` — small hook: `{ viewState, setDetail, clearDetail }` reading/writing `tab.viewState` via `useVueState({deep:true})` + `tabStore.updateTab`. Centralizes the list↔detail navigation.
- `src/react/components/sql-editor/Panels/common/useTableContainerHeight.ts` — ResizeObserver-based hook returning `{ ref, maxHeight }` for filling the panel's available area.
- `src/react/components/sql-editor/availableActions.ts` — extracted from Stage 15's `SchemaPane.tsx`. SchemaPane.tsx swaps to import from here (small refactor).
- Reusable sub-tables (presentational, no list-detail navigation):
  - `Panels/common/tables/ColumnsTable.tsx` — driven by a generic `columns: ColumnMetadata[]` prop + a `flavor: "table" | "view" | "external-table"` switch for column-header variations.
  - `Panels/common/tables/IndexesTable.tsx`
  - `Panels/common/tables/ForeignKeysTable.tsx`
  - `Panels/common/tables/ChecksTable.tsx`
  - `Panels/common/tables/TriggersTable.tsx` — accepts an `onSelect(trigger)` for the standalone TriggersPanel use.
  - `Panels/common/tables/PartitionsTable.tsx`
  - `Panels/common/tables/DependencyColumnsTable.tsx`

**Approach:** Land the shared building blocks before any panel consumes them. Each gets a small unit test (header rendering + filter behavior).

**Deliverables:** ~14 `.tsx` + 7 tests.

### Phase 2 — CodeViewer-based panels + the simplest list

**Files ported:**
- `Panels/SequencesPanel.tsx` — list-only, simplest panel. Validates the table + toolbar primitives.
- `Panels/FunctionsPanel.tsx` — list → CodeViewer (read function signature + body).
- `Panels/ProceduresPanel.tsx` — list → CodeViewer.
- `Panels/PackagesPanel.tsx` — list → CodeViewer.
- `Panels/TriggersPanel.tsx` — list → CodeViewer; the row-action wires through to `TriggersTable`'s `onSelect` from Phase 1.

For each: top-level `.tsx` + a top-level mount-shim `Panels/<Name>Panel.tsx` re-exporting from the subdirectory — same pattern Stage 15 used for `SchemaPane.tsx` to satisfy the non-recursive mount-registry glob.

**Caller swap:** `Panels.vue` line 6–46 — replace `<SequencesPanel />` etc. with `<ReactPageMount page="SequencesPanel" />`. One commit can do all five panels at once or split per panel.

**Deliverables:** 5 panels × (`.tsx` + shim + test) + caller swap = ~15 files.

### Phase 3 — Multi-tab detail panels

**Files ported:**
- `Panels/TablesPanel.tsx` — list view + `TableDetail.tsx` sub-component (multi-tab Columns / Indexes / FKs / Checks / Triggers / Partitions). Tab state lives in `viewState.detail` (existing schema — `detail.table` plus a `detail.tab` enum for the active sub-tab).
- `Panels/ViewsPanel.tsx` — list + `ViewDetail.tsx` (Definition / Columns / Dependency Columns).
- `Panels/ExternalTablesPanel.tsx` — list + `ExternalTableDetail.tsx` (Columns).

These reuse every Phase 1 sub-table directly; the detail components are mostly orchestration (tab state + back button + which sub-table to show).

**Caller swap:** swap the three remaining `<XPanel />` invocations in `Panels.vue`.

**Deliverables:** 3 panels × (root + detail + shim + test) = ~12 files.

### Phase 4 — InfoPanel + cleanup

**Files ported:**
- `Panels/InfoPanel.tsx` — dashboard with mini-tables for Tables / Views / Functions / Procedures (and engine-conditional Sequences / External Tables / Packages). Each mini-table reuses the Phase 1 components in a compact "first 5 rows + see all" mode. Drill-down click sets `viewState.view` to the corresponding detail panel.

**Cleanup:**
- Verify all 9 ported panels compile cleanly into the production bundle.
- Delete the Vue files for every ported panel (root, sub-components, common helpers consumed only by the ported panels).
  - **Keep:** `Panels.vue` (host shell), `DiagramPanel/*` (cytoscape passthrough), `common/SchemaSelectToolbar.vue` and `common/CodeViewer.vue` only if any non-ported Vue consumer remains — verify with `grep` first; expect to delete both.
- `grep` confirms zero live Vue callers of every deleted file.
- Update Stage 15's `SchemaPane.tsx` to import `useAvailableActions` from the new shared `availableActions.ts` (Phase 1 added it; this final cleanup wires the import).

**Deliverables:** 1 panel + Vue tree deletion + small Stage-15 refactor.

## 5. Per-phase checklist

### Phase 1 — Shared infrastructure
- [ ] `SchemaSelectToolbar.tsx` + test
- [ ] `CodeViewer.tsx` + test (renders Monaco + Format checkbox + OpenAIButton mount)
- [ ] `DefinitionViewer.tsx`
- [ ] `EllipsisCell.tsx`
- [ ] `useViewStateNav.ts` + test
- [ ] `useTableContainerHeight.ts` + test
- [ ] `availableActions.ts` shared util + Stage-15 SchemaPane refactor
- [ ] 7 sub-table components (`ColumnsTable`, `IndexesTable`, `ForeignKeysTable`, `ChecksTable`, `TriggersTable`, `PartitionsTable`, `DependencyColumnsTable`) + tests
- [ ] Gates green

### Phase 2 — CodeViewer-based + simplest list
- [ ] `SequencesPanel.tsx` + shim + test
- [ ] `FunctionsPanel.tsx` + shim + test
- [ ] `ProceduresPanel.tsx` + shim + test
- [ ] `PackagesPanel.tsx` + shim + test
- [ ] `TriggersPanel.tsx` + shim + test
- [ ] `Panels.vue` swapped to 5 `<ReactPageMount>`s
- [ ] Delete the 5 Vue panel directories
- [ ] Gates green

### Phase 3 — Multi-tab detail panels
- [ ] `TablesPanel.tsx` + `TableDetail.tsx` + shim + test
- [ ] `ViewsPanel.tsx` + `ViewDetail.tsx` + shim + test
- [ ] `ExternalTablesPanel.tsx` + `ExternalTableDetail.tsx` + shim + test
- [ ] `Panels.vue` swapped to 3 more `<ReactPageMount>`s
- [ ] Delete the 3 Vue panel directories + their detail components
- [ ] `viewState.detail.tab` enum extension persisted (so a refresh restores the active sub-tab)
- [ ] Gates green

### Phase 4 — InfoPanel + cleanup
- [ ] `InfoPanel.tsx` + shim + test
- [ ] `Panels.vue` swapped (`<ReactPageMount page="InfoPanel" />`); only `<DiagramPanel />` left as a Vue child
- [ ] Delete `InfoPanel.vue`
- [ ] Delete `common/SchemaSelectToolbar.vue` + `common/CodeViewer.vue` + `common/EllipsisCell.vue` if unreferenced
- [ ] `availableActions.ts` consumers verified — Stage 15 SchemaPane refactor lands
- [ ] `grep` confirms `DiagramPanel` is the only remaining Vue panel under `EditorPanel/Panels/`
- [ ] Gates green

## 6. Manual UX verification (after all phases)

1. Open SQL Editor → connect to a database → right-click a table in SchemaPane → "View detail" → editor flips to TablesPanel detail with that table preselected. Repeat for views / procedures / functions / sequences / packages / external tables / triggers.
2. From Panels' toolbar, switch the schema dropdown → list rebinds to the selected schema.
3. In TablesPanel detail, click each sub-tab (Columns / Indexes / FKs / Checks / Triggers / Partitions) → content swaps; the row count badge updates.
4. Search box at the top of every list → keyword filter applies in <50ms with no flicker.
5. ProceduresPanel → click a row → CodeViewer shows the procedure body. Toggle Format checkbox → SQL is reformatted. Click "Explain" → React OpenAIButton renders.
6. ViewsPanel → click a view → DEFINITION tab shows the Monaco read-only body; switch to COLUMNS / DEPENDENCY COLUMNS tabs.
7. InfoPanel → click "See all" on each mini-table → switches to the matching panel.
8. DiagramPanel → still renders the existing Vue cytoscape canvas (unchanged).
9. Tab switch → all panel internal state resets (because Panels.vue's `:key="tab?.id"` is preserved on the React mounts via `key`).
10. Refresh the page after navigating to a Tables sub-tab → the same sub-tab is restored from `viewState.detail.tab`.
11. Repeat 1–7 across MySQL, Postgres, Oracle to cover engine-specific column header variations.

## 7. Out of scope (deferred)

- `DiagramPanel` migration — cytoscape-backed; large effort; ship separately.
- `EditorPanel.vue`, `StandardPanel`, `TerminalPanel`, `ResultPanel` — separate stage. The Stage 14 risks list flagged ResultView's `VirtualDataTable` as needing a perf spike first; that probably becomes Stage 17.
- `SQLEditorHomePage.vue` — root container; depends on EditorPanel.
- `Panels.vue` host migration — kept as the Vue switch in Stage 16; absorbed by EditorPanel migration later.
- `MonacoEditor` improvements — wrap, don't migrate.

## 8. Risks & open questions

- **`viewState.detail.tab` field addition.** TablesPanel and ViewsPanel detail need to remember the active sub-tab across refreshes. Vue persists the tab as local state in the detail component (lost on refresh); for parity we have two options: (a) match Vue and let refreshes reset to the first tab, (b) extend `viewState.detail` with a `tab?` field. **Recommendation:** match Vue behavior in Phase 3 to keep the persistence schema unchanged; revisit if QA flags the reset as a regression. (Updates the per-phase checklist accordingly.)

- **NDataTable virtualization parity.** Vue's `useAutoHeightDataTable` enables `:virtual-scroll="true"` for tables with hundreds of rows (e.g. a table with 500 columns). The shadcn `DataTable` we ship today uses CSS `overflow-y-auto` without virtualization. **Mitigation:** for column counts likely to exceed ~500 rows we cap the rendered window via `slice(0, N)` + a "Load more" footer (same trick FlatTableList uses for >1000 tables). Benchmark in Phase 3 with a synthetic 1k-column table fixture; if scroll latency is noticeable, lift the worksheet table's virtualization helper into a shared primitive.

- **i18n key fan-out.** Reusable sub-tables source ~50+ column-header keys from `schema-editor.column.*`, `db.*`, etc. Most already exist (Stage 15 added many). Any newly-needed keys land via the Phase-N scripts that ran for prior stages. Plan one batch sweep per Phase to avoid scattering edits.

- **Panels.vue + React mounts at scale.** Eleven `v-if` branches each potentially mounting React adds churn on `viewState.view` flips. Each `ReactPageMount` is its own React root; rapid view switching could create/destroy roots needlessly. **Mitigation:** confirm `Panels.vue`'s existing `:key` pattern doesn't force re-mounts on every render; if it does, lift the React roots to a single `<ReactPageMount page="PanelsHost" />` — but defer to actual measurement before adding that complexity.

- **CodeViewer's NSplit → React resizable.** The Vue `NSplit` allows users to drag the editor / AI-pane divider. We don't have a generic resizable primitive in `src/react/components/ui/`; shadcn ships one but we haven't pulled it in. **Mitigation:** start with a fixed 60/40 split and add the primitive only if QA reports the divider-drag as missing. Tracked as a follow-up.

- **`availableActions` extraction touches Stage 15.** Stage 15's `SchemaPane.tsx` defines `useAvailableActions` locally. Phase 1 lifts it to `src/react/components/sql-editor/availableActions.ts`; SchemaPane gets a small import diff. Low-risk but requires re-running the SchemaPane test suite.

- **InfoPanel "see all" navigation.** The Vue InfoPanel's mini-tables embed a header link that flips `viewState.view` to the detail panel. The same React panel needs to render under both `viewState.view === "INFO"` (mini-table mode) and `viewState.view === "TABLES"` (full-list mode). Two options: (a) make the React InfoPanel an aggregator that renders multiple ported panels in compact mode (introducing a `compact?: boolean` prop on each), or (b) duplicate the mini-table rendering logic locally in InfoPanel.tsx. **Recommendation:** option (b) — the mini-tables differ enough (5-row preview, no header toolbar) that a separate, simpler component is clearer than a `compact` flag everywhere. Phase 4 owns the call.

- **DiagramPanel kept Vue.** Stage 17/18 owns its migration. As long as Panels.vue's `v-if="viewState.view === 'DIAGRAM'" → <DiagramPanel />` path still renders a Vue component, no bridge work is needed in Stage 16. Confirm in Phase 4 manual UX.

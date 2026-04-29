# SQL Editor React Migration — Stage 18 Design

**Date:** 2026-04-28
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the four remaining `ResultPanel/` Vue surfaces that
don't transitively render `ResultView` (so Stage 20's perf spike on
`ResultView` itself can land cleanly), plus the one chain dep that
`BatchQuerySelect` pulls in (`DataExportButton`) and a small
`DatabaseTable` extension. Single-track stage; `DiagramPanel`/`SchemaDiagram`
have their own stage (19) since cytoscape is its own subsystem (~2,265 LOC).

**Non-goals:**
- `EditorCommon/ResultView/*` — Stage 20 (perf-spike headline).
- `ResultPanel/DatabaseQueryContext.vue` — directly mounts `ResultViewV1`,
  co-migrates with Stage 20.
- `Panels/DiagramPanel/DiagramPanel.vue` + `components/SchemaDiagram/*`
  — Stage 19, its own focused stage.
- The five host shells (`Panels`, `EditorPanel`, `StandardPanel`,
  `TerminalPanel`, `ResultPanel`, `SQLEditorHomePage`) — Stage 21.
- Visual / behavioral changes — parity only.

## 2. Inventory

**In scope (~444 LOC of `sql-editor/` Vue + 414 LOC `DataExportButton` chain dep + small `DatabaseTable` extension):**

| File | LOC | Notes |
|---|---|---|
| `EditorPanel/ResultPanel/BatchQuerySelect.vue` | 359 | Tab strip + batch-export button. The trickiest part is `<DataExportButton>` with its `:database-list` + `:show-selection` + `:pagination` form slot. |
| `EditorPanel/ResultPanel/ContextMenu.vue` | 85 | Imperative `defineExpose({ show, hide })` API that `BatchQuerySelect` calls via `contextMenuRef.value?.show(index, $event)`. Convert to a Pinia/module store during the port to break the imperative coupling. |
| `components/DataExportButton.vue` | 414 | Outside `sql-editor/` but a hard dep. Drawer + format picker (CSV/JSON/SQL/XLSX) + password toggle + validate hook + multi-file zip download flow. No React equivalent today (`DataExportPrepSheet.tsx` is a different surface — project-level export wizard, doesn't fit). |

**`DatabaseTable` extension** (small, no separate file):
- React `DatabaseTable.tsx` currently takes `filter: DatabaseFilter` and fetches its own list from the server.
- Add an optional `databases?: Database[]` prop that, when present, skips the fetch and renders the provided list directly.
- `BatchQuerySelect` consumes it that way (Vue does the same: `:database-list="databaseList"`).

### Out of scope (later stages)

```
Panels/DiagramPanel/DiagramPanel.vue          → Stage 19 (cytoscape subsystem)
components/SchemaDiagram/* (44 files)         → Stage 19
EditorCommon/ResultView/*                     → Stage 20 (perf spike)
ResultPanel/DatabaseQueryContext.vue          → Stage 20 (mounts ResultViewV1)
ResultPanel/ResultPanel.vue                   → Stage 21 (host shell)
StandardPanel/StandardPanel.vue               → Stage 21
TerminalPanel/TerminalPanel.vue               → Stage 21
Panels/Panels.vue                             → Stage 21
EditorPanel/EditorPanel.vue                   → Stage 21
SQLEditorHomePage.vue                         → Stage 21
SQLEditorPage.vue                             → Stage 21
```

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NDrawer` (Vue Naive UI drawer) for `DataExportButton` | shadcn `Sheet` (`@/react/components/ui/sheet`) — already used by other React drawers in the codebase | ✓ |
| `NButton` (DataExportButton trigger) | `Button` | ✓ |
| `NSelect` (export format picker) | `Select` | ✓ |
| `NCheckbox` (password toggle) | native `<input type="checkbox">` (matches Stage 16/17 panel checkboxes) | ✓ |
| `NDropdown` (`ContextMenu`) | shadcn `DropdownMenu` — already used by `OpenAIButton`, `EditorAction` | ✓ |
| `NTooltip` (`BatchQuerySelect` "show empty" toggle, batch-export tooltip) | shared `Tooltip` (passes through children when content is falsy — Stage 17 lesson) | ✓ |
| `NScrollbar` (`BatchQuerySelect` horizontal scroll) | native `overflow-x-auto` | ✓ |
| `DatabaseV1Table` (Vue) | extended React `DatabaseTable` with `databases?` opt-out | New small extension |
| `useEmitteryEventListener` (Vue composable) | `useEffect` + direct `.on()` (pattern from Stage 16+17) | ✓ |
| `provideResultTabListContext` (Vue provide/inject for `ContextMenu` ↔ `BatchQuerySelect`) | Pinia store + module-level emittery (mirrors `aiContextEvents` from Stage 16) | New small store |

## 4. Architecture & phases

Single track. Order matters.

### Phase 1 — Foundations (`DatabaseTable` extension + `DataExportButton`)

1. Extend React `DatabaseTable.tsx` with `databases?: Database[]` opt-out. When provided, skip the fetch path entirely; render the given list with the existing selection/highlight/sort UI. Verify the existing settings page consumer still works through the unchanged filter path.
2. Port `DataExportButton.tsx`. Substantial — drawer with form slot, format picker, password input, "Maximum N rows" warning, `validate()` predicate, multi-file zip download via the existing `sqlStore.exportData` API. Reuse `Sheet` + `Select` + `Button`. The `form` slot becomes a React `children` or `formContent` prop.

**Deliverables:** 2 new React surfaces + 1 small extension + 3 tests.

### Phase 2 — `ContextMenu` + Pinia state replacing the imperative coupling

1. Build `Panels/common/resultTabContext.ts` — Pinia store + emittery singleton replacing Vue's `provideResultTabListContext`. Holds `contextMenu` state (`{ x, y, index } | undefined`) and a `close-tab` event channel.
2. Port `ContextMenu.tsx` reading from the Pinia store + emitting on `close-tab`. No imperative `defineExpose` API — driven entirely by store state.

**Deliverables:** 1 store + 1 React surface + 1 mount shim + 1 test.

### Phase 3 — `BatchQuerySelect` + caller swap

1. Port `BatchQuerySelect.tsx`:
   - Tab strip with environment-tinted buttons (env color logic same as `TabItem`)
   - "Show/hide empty results" toggle
   - `DataExportButton` mount (form slot = the extended `DatabaseTable`)
   - Right-click on a tab fires `contextMenuStore.open(x, y, index)` — no more imperative `ref.show()`
2. `ResultPanel.vue` swaps `<BatchQuerySelect>` and `<ContextMenu>` to `<ReactPageMount>` calls. `ResultPanel.vue` itself stays Vue.
3. Delete the Vue files (`BatchQuerySelect.vue`, `ContextMenu.vue`, `ResultPanel/context.ts`).
4. `components/DataExportButton.vue` — keep around if any non-SQL-editor Vue surface still imports it; otherwise delete.

**Deliverables:** 1 React surface + 1 mount shim + 1 test + Vue file deletions.

## 5. Per-phase checklist

### Phase 1 — Foundations (`DatabaseTable` extension + `DataExportButton`)
- [ ] React `DatabaseTable` `databases?: Database[]` opt-out + test (verify settings-page consumer through filter path still works)
- [ ] `DataExportButton.tsx` + shim + tests (open drawer, validate predicate, format pick CSV/JSON/SQL/XLSX, password toggle, multi-file zip download)
- [ ] i18n keys synced for export drawer

### Phase 2 — `ContextMenu` + Pinia state
- [ ] `Panels/common/resultTabContext.ts` Pinia store (state: `{ x, y, index } | undefined`) + emittery (`close-tab` channel)
- [ ] `ContextMenu.tsx` + shim + test (right-click → `contextMenuStore.open(x, y, index)`, item select → `events.emit("close-tab", { kind })`, ESC/outside-click closes)

### Phase 3 — `BatchQuerySelect` + caller swap
- [ ] `BatchQuerySelect.tsx` + shim + test (env-color tinting, tab close, show-empty toggle, batch-export selection table via extended `DatabaseTable`)
- [ ] `ResultPanel.vue` swaps `<BatchQuerySelect>` and `<ContextMenu>` to `<ReactPageMount>` calls
- [ ] Delete `BatchQuerySelect.vue`, `ContextMenu.vue`, and Vue `ResultPanel/context.ts`
- [ ] Delete `components/DataExportButton.vue` if no remaining Vue caller; otherwise leave it for a follow-up
- [ ] Gates green

## 6. Manual UX verification

- **DataExportButton** — open drawer, switch formats, pick "ZIP password", verify disabled state when validate predicate fails, click Export → multi-file zip downloads with one entry per selected database.
- **BatchQuerySelect** — run `SELECT 1` against a batch of 3 databases. Tab strip renders 3 env-tinted tabs, one with "(empty)" if the query returns 0 rows, click to select, right-click → context menu close actions, Export button opens drawer with pre-filtered database list.
- **ContextMenu** — Close, Close others, Close to the right, Close all behave the same as Vue.

## 7. Out of scope (deferred)

- `Panels/DiagramPanel/DiagramPanel.vue` + `components/SchemaDiagram/*` — Stage 19
- `EditorCommon/ResultView/*` and `ResultPanel/DatabaseQueryContext.vue` — Stage 20 (perf spike)
- The host shells (`Panels`, `EditorPanel`, `StandardPanel`, `TerminalPanel`, `ResultPanel`, `SQLEditorHomePage`, `SQLEditorPage`) — Stage 21
- AI plugin port — separate effort (currently hosted by Vue at `Panels.vue`/`StandardPanel.vue` via the Stage 16/17 hoist)

## 8. Risks & open questions

- **`DataExportButton` is the headline of this stage.** 414 LOC of drawer + form-slot + zip download is non-trivial. The React port has multi-stage value (any "export to file" surface can reuse it), so the LOC is justified even though it sits outside `sql-editor/`.

- **`ContextMenu` imperative API removal.** Currently `BatchQuerySelect` calls `contextMenuRef.value?.show(index, $event)` — this is a `defineExpose` imperative API. Replacing it with a Pinia store + emittery is a clean refactor but slightly changes the abstraction (state-driven instead of imperative). Mirrors how Stage 17 replaced `MonacoEditor.getActiveStatement()` ref API with the `activeStatementRef` shared shallowRef.

- **`DatabaseV1Table`'s pre-fetched-list mode and React `DatabaseTable`'s server-fetch mode coexisting.** The `databases?: Database[]` opt-out is straightforward but needs careful interaction with the existing pagination/sort UI. Test both paths in Phase 1.

- **AI plugin still Vue.** The Vue host at `Panels.vue` currently wraps its content in an `<NSplit>` with `<AIChatToSQL>` when `showAIPanel && isShowingCode`. `BatchQuerySelect`/`ContextMenu` don't host code surfaces, so this hoist stays untouched in Stage 18. (Stage 21 will revisit when `Panels.vue` itself migrates to React.)

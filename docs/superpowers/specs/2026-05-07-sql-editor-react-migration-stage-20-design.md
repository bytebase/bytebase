# SQL Editor React Migration — Stage 20 Design

**Date:** 2026-05-07
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the SQL Editor's `ResultView` subsystem (the result-set
display) plus its two direct hosts (`DatabaseQueryContext.vue` and the
`<ResultViewV1>` mount inside `TerminalPanel.vue`) from Vue to React.
This is the perf-critical headline of the migration: `VirtualDataTable`
is the piece users notice when running 10k+ row queries. After this
stage, only the host shells remain (`Stage 21`), and the Vue
`DataExportButton.vue` becomes deletable.

**Non-goals:**
- Host shells (`StandardPanel`, `TerminalPanel`, `ResultPanel`,
  `Panels`, `EditorPanel`, `SQLEditorHomePage`, `SQLEditorPage`) — Stage 21.
- Visual / behavioral changes — parity port only. No new features.
- Restoring the result-pane test coverage to 100%; the Vue tree has
  zero tests today, so we add a perf seed suite but don't backfill
  comprehensive unit tests.
- AI plugin port — separate effort.

## 2. Inventory

**In scope (~2,771 LOC):**

| File | LOC | Notes |
|---|---|---|
| `EditorCommon/ResultView/SingleResultViewV1.vue` | 833 | Heaviest. Single-result orchestrator: tabs (TABLE / VERTICAL), search box + scroll-to-row, export toolbar, selection summary, copy-tooltips host, vertical/table mode toggle, detail-panel drawer trigger. |
| `EditorCommon/ResultView/VirtualDataTable.vue` | 458 | Perf-critical. Sticky-header table layout with column resizing, binary-format buttons per column, row/column selection. Uses Naive UI `NVirtualList` with fixed 35px row height. Exposes `scrollTo(index)` via `defineExpose`. |
| `EditorCommon/ResultView/ResultViewV1.vue` | 359 | Multi-result tab router (a query can return multiple result sets). Mounts `SingleResultViewV1` per result. Routes errors to `ErrorView`. |
| `EditorCommon/ResultView/DetailPanel.vue` | 303 | Modal/drawer viewer for the full content of a single cell (long string, JSON pretty-print, binary hex/utf8/base64). |
| `EditorCommon/ResultView/TableCell.vue` | 233 | Per-cell formatter: string (escape + truncate), JSON (inline / collapse), binary (hex / utf8 / base64), sensitive-data masked icon, NULL placeholder. |
| `EditorCommon/ResultView/ErrorView.vue` | 146 | Generic error display with copy / Postgres-specific delegate. |
| `EditorCommon/ResultView/VirtualDataBlock.vue` | 144 | Vertical "card" mode: each row renders as a stacked block. Dynamic row height (`60 + columns.length * 28`), same `scrollTo(index)` API. |
| `EditorCommon/ResultView/BinaryFormatButton.vue` | 72 | Three-button group (hex / utf8 / base64) per binary column. |
| `EditorCommon/ResultView/SelectionCopyTooltips.vue` | 59 | Floating toolbar over the selected range — copy as TSV / JSON. |
| `EditorCommon/ResultView/PrettyJSON.vue` | 56 | JSON pretty-printer (used by `DetailPanel` and `TableCell`). |
| `EditorCommon/ResultView/SensitiveDataIcon.vue` | 41 | Lock-icon trigger that opens the React `MaskingReasonPopover` via `ReactPageMount`. |
| `EditorCommon/ResultView/PostgresError.vue` | 41 | PG-specific error message formatter (line/position highlight). |
| `EditorCommon/ResultView/ColumnSortedIcon.vue` | 19 | Sort-direction caret SVG. |
| `EditorCommon/ResultView/EmptyView.vue` | 14 | "No rows" placeholder. |
| `EditorCommon/ResultView/{context,binary-format-store,selection-logic,index}.ts` | ~190 | Shared state: SQLResultViewContext (provide/inject), binary format toggles, selection ranges, barrel exports. |
| `EditorPanel/ResultPanel/DatabaseQueryContext.vue` | 125 | Direct host of `<ResultViewV1>`. Wraps it with a status overlay (PENDING / EXECUTING / DONE / CANCELLED). Co-migrates because the imperative coupling is tight. |

**Total**: ~2,771 LOC across ~17 files.

### Out of scope (later stages)

```
StandardPanel.vue                                 → Stage 21
TerminalPanel.vue (the wrapping shell)            → Stage 21 (the inner ResultViewV1 mount migrates here)
ResultPanel.vue                                   → Stage 21
Panels.vue                                        → Stage 21
EditorPanel.vue                                   → Stage 21
SQLEditorHomePage.vue                             → Stage 21
SQLEditorPage.vue                                 → Stage 21
components/DataExportButton.vue                   → deletable after this stage
```

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NVirtualList` (Naive UI virtualization) | `@tanstack/react-virtual` `useVirtualizer` | ✓ Already a peer (used by other React tables) |
| `NTabs` / `NTabPane` (multi-result tab strip) | shadcn `Tabs` (`@/react/components/ui/tabs`) | ✓ |
| `NSwitch` (table/vertical toggle) | shadcn `Switch` | ✓ |
| `NButton` / `NTooltip` | shadcn `Button` + shared `Tooltip` | ✓ |
| Naive UI `darkTheme` (terminal mode) | Tailwind `dark:` variants on the React components | ✓ |
| `NConfigProvider` (dark theme scope) | not needed — Tailwind variants travel with React tree | ✓ |
| Vue `provide` / `inject` for `SQLResultViewContext` | React context (one provider per `<ResultViewV1>` mount) | ✓ |
| `binary-format-store` (Vue reactive Map) | React state in the same context, mutators via callbacks | ✓ |
| `selection-logic` (custom Vue composable for row/col selection) | React hook + ref-driven imperative range management | ✓ |
| `useLocalStorage` (`@vueuse/core`) | shared React hook (or `useSyncExternalStore` over `localStorage`) — Stage 16/17 already has a pattern | ✓ |
| `useElementSize` | `ResizeObserver` in a `useEffect`, or a shared `useElementSize` React hook (peer Stage 17 pattern) | ✓ |
| `lucide-vue-next` icons | `lucide-react` | ✓ |
| `@bufbuild/protobuf` | same package | ✓ |
| `lodash-es`, `dayjs`, `uuid` | same packages | ✓ |
| Vue `defineExpose({ scrollTo })` on `VirtualDataTable`/`VirtualDataBlock` | `useImperativeHandle` + ref forwarding, OR direct `useVirtualizer` ref consumed by parent | Needs decision (§4) |

**Key call-out**: the React `DataExportButton.tsx` already exists (built in Stage 18). The two Vue `<DataExportButton>` mounts inside `ResultViewV1.vue` and `SingleResultViewV1.vue` swap to the React component as part of this stage, which finally lets us delete the Vue `DataExportButton.vue`.

## 4. Architecture & phases

### Pre-Stage — Perf baseline spike (~1 day, no code merge)

Before porting `VirtualDataTable`, capture a Vue baseline so we have
something to compare against:

- Run a query that returns **10,000 rows × 12 columns** (mix of string,
  number, JSON, NULL, binary).
- Capture: scroll FPS (Chrome devtools perf trace), heap size at idle,
  time-to-first-paint after results arrive, time to render after
  scrolling 5k rows down, memory delta after toggling table↔vertical
  mode 5 times.
- Repeat with **100,000 rows × 12 columns** to find the breaking point.
- Document the numbers in `docs/superpowers/specs/2026-05-XX-stage-20-perf-baseline.md`.

Spike output is the **acceptance criteria for the React port**: parity
or better on every metric, with no regression beyond noise (±5%) at the
10k row scale.

### Phase 1 — Foundations (types, context, leaves)

1. Port `types/*.ts` and the shared state modules:
   - `EditorCommon/ResultView/context.ts` → `react/components/sql-editor/ResultView/context.tsx`
     — React context (per `<ResultViewV1>` mount). Holds: `dark` flag,
     `disallowCopyingData` flag, `detail` (current cell open in detail
     panel), `dataIsTrimmed` flag, the binary-format Map, the selection
     ranges. Mirrors the per-instance pattern from Stage 19.
   - `binary-format-store.ts` → React state inside the context.
   - `selection-logic.ts` → React hook (`useResultSelection`) returning
     `{ ranges, addRange, clearRanges, isSelected(row, col) }`.

2. Port the small leaves 1:1. They're pure presentation:
   - `EmptyView.tsx`, `ColumnSortedIcon.tsx`, `SensitiveDataIcon.tsx`,
     `PostgresError.tsx`, `BinaryFormatButton.tsx`,
     `SelectionCopyTooltips.tsx`, `PrettyJSON.tsx`, `ErrorView.tsx`.

**Deliverables:** 8 React surfaces + 1 context + 2 hooks + 1 unit test (segment-overlap-style for `selection-logic`'s range merging math).

### Phase 2 — Cell rendering (`TableCell.tsx`)

`TableCell.vue` is 233 LOC of pure formatting logic — string escape +
truncate, JSON pretty-print toggle, binary format dispatch, sensitive
data lock icon, NULL placeholder. Most of this is straight 1:1.

The **only React-specific concern**: HTML escaping. The Vue version
uses `v-html` + a sanitizer. The React version uses
`dangerouslySetInnerHTML` + the same sanitizer (the shared
`getHighlightHTMLByRegExp` already does DOMPurify; reuse via the
`HighlightLabelText` rule from memory).

**Deliverables:** 1 React surface + 1 unit test (cell formatter dispatch by column type).

### Phase 3 — `DetailPanel.tsx`

The modal viewer for full cell content. Reuses `PrettyJSON` and
`BinaryFormatButton` from Phase 1. Mounts via the shared `Dialog`
primitive (not `Sheet` — this is read-only content viewing, per the
shadcn-skill rule: dialog for non-form interactions).

**Deliverables:** 1 React surface.

### Phase 4 — Virtualization (`VirtualDataTable.tsx` + `VirtualDataBlock.tsx`)

The headline. Both components use `@tanstack/react-virtual`'s
`useVirtualizer`. Decision points:

**Imperative API surface (`scrollTo(index)`):**
- Vue exposed `scrollTo(index)` via `defineExpose`. Callers (`SingleResultViewV1`) used it for search-result navigation and mode-toggle restore.
- React equivalent: forward the `useVirtualizer` ref directly via `useImperativeHandle`. Simpler than building a wrapper imperative API — `useVirtualizer` already exposes `scrollToIndex(index, { align })` natively.

**Row height:**
- `VirtualDataTable` — fixed 35px, `estimateSize: () => 35` is enough.
- `VirtualDataBlock` — dynamic. Use `measureElement()` per row + `estimateSize: () => 60 + columns.length * 28` as a starting hint.

**Column resizing:**
- Vue uses pointer-event handlers + a width Map. React port reuses the `useColumnWidths` hook from Stage 18's `DatabaseTableView` work (already proven).

**Selection (row/column checkboxes):**
- Driven by the `useResultSelection` hook from Phase 1.
- `SelectionCopyTooltips` (Phase 1) renders the floating toolbar.

**Deliverables:** 2 React surfaces + 1 unit test (virtualizer mount + scrollToIndex math) + perf benchmark comparing 10k-row scroll vs. Vue baseline.

### Phase 5 — `SingleResultViewV1.tsx` (the orchestrator)

The 833-LOC heavy. It composes everything:
- Mode toggle (table / vertical)
- Search input + match navigation (highlights cells, calls `scrollToIndex`)
- Export toolbar — mounts the React `DataExportButton` (already built in Stage 18)
- Selection summary + copy tooltips
- Detail panel mount
- Sensitive-data masking — already React via `ReactPageMount`; in the React tree this becomes a direct mount (no bridge)

**Risks:** the search-highlight loop runs over the full row set. On 10k rows this is O(rows × cols) per keystroke. The Vue version debounces; the React port must match (debounce 200ms + only highlight visible-window rows; full-table match is computed once and stored as a `Set<rowIndex>`).

**Deliverables:** 1 React surface + 1 unit test (search match index computation).

### Phase 6 — `ResultViewV1.tsx`

Multi-result tab router. Lightweight — wraps a `Tabs` primitive over an array of result sets. Each tab renders a `<SingleResultView>`. Errors route to `<ErrorView>`.

**Deliverables:** 1 React surface.

### Phase 7 — `DatabaseQueryContext.tsx` + `TerminalPanel` mount swap

1. Port `DatabaseQueryContext.vue` (125 LOC) → `DatabaseQueryContext.tsx`.
   The status-overlay state machine (PENDING / EXECUTING / DONE / CANCELLED) is read from `tabStore.currentTab.databaseQueryContexts` via `useVueState`. The execute / re-execute / cancel buttons call into `useExecuteSQL()` and the `events.cancel-query` channel.
2. Add a top-level page shim at `react/components/sql-editor/DatabaseQueryContext.tsx` for `mount.ts` lookup.
3. Swap the two Vue host points:
   - `ResultPanel.vue`'s `<DatabaseQueryContext>` mount → `<ReactPageMount page="DatabaseQueryContext" :page-props="{ database, context }" />`.
   - `TerminalPanel.vue`'s `<ResultViewV1>` mount → `<ReactPageMount page="TerminalResultView" ... />` (a thin wrapper that adapts the dark / loading props).

**Deliverables:** 1 React surface + 1 page shim + 1 wrapper + 2 host-side swaps.

### Phase 8 — Vue tree deletion

Delete the entire `frontend/src/views/sql-editor/EditorCommon/ResultView/` Vue tree (~17 files). Delete `frontend/src/views/sql-editor/EditorPanel/ResultPanel/DatabaseQueryContext.vue`. Delete `frontend/src/components/DataExportButton.vue` (last consumers gone). Update barrels and route imports. Run gates.

**Deliverables:** ~19 file deletions + barrel cleanups + green gates.

## 5. Per-phase checklist

### Phase 1 — Foundations
- [ ] `ResultView/context.tsx` (React context + provider)
- [ ] `useResultSelection` hook + segment-merge unit test
- [ ] `useBinaryFormat` hook (per-column format toggle)
- [ ] `EmptyView.tsx`, `ColumnSortedIcon.tsx`, `SensitiveDataIcon.tsx`,
  `PostgresError.tsx`, `BinaryFormatButton.tsx`,
  `SelectionCopyTooltips.tsx`, `PrettyJSON.tsx`, `ErrorView.tsx`

### Phase 2 — Cell
- [ ] `TableCell.tsx` (string / JSON / binary / sensitive / NULL formatter)
- [ ] Unit test for cell formatter dispatch

### Phase 3 — DetailPanel
- [ ] `DetailPanel.tsx` (Dialog primitive, JSON pretty-print + binary toggle)

### Phase 4 — Virtualization
- [ ] `VirtualDataTable.tsx` (`useVirtualizer`, fixed 35px rows, sticky header)
- [ ] `VirtualDataBlock.tsx` (`useVirtualizer` + `measureElement`, dynamic row height)
- [ ] `useImperativeHandle({ scrollToIndex })` on both
- [ ] Reuse `useColumnWidths` (Stage 18) for column drag-resize
- [ ] Unit test for `scrollToIndex` math
- [ ] **Perf benchmark** vs. Vue baseline at 10k and 100k rows

### Phase 5 — Single result
- [ ] `SingleResultView.tsx` (mode toggle, search, export toolbar, detail trigger, selection summary)
- [ ] Search-match index computation memoized; only visible-window cells receive highlight DOM
- [ ] Unit test for search-match index

### Phase 6 — Multi result
- [ ] `ResultView.tsx` (`Tabs` primitive over result sets)

### Phase 7 — Caller swap
- [ ] `DatabaseQueryContext.tsx` (status state machine + cancel)
- [ ] Page shim for `mount.ts`
- [ ] `ResultPanel.vue` swap to `<ReactPageMount page="DatabaseQueryContext">`
- [ ] `TerminalPanel.vue` swap (terminal-mode wrapper)

### Phase 8 — Cleanup
- [ ] Delete `EditorCommon/ResultView/` Vue tree
- [ ] Delete `ResultPanel/DatabaseQueryContext.vue`
- [ ] Delete `components/DataExportButton.vue` (last live caller gone)
- [ ] Gates green
- [ ] Manual UX verification (§6)

## 6. Manual UX verification

- **Run a 10k-row query** → result paints in < 500 ms, scroll FPS ≥ 55 (Chrome desktop, no extensions).
- **Run a 100k-row query** → result paints, scroll FPS ≥ 30. Memory at idle within 20% of Vue baseline.
- **Toggle table ↔ vertical** mode 10× quickly → no memory growth, no visible flicker.
- **Search** for a substring with 200+ matches → debounced; UI doesn't lag while typing.
- **Click a cell** → DetailPanel opens; toggle JSON / binary modes; close.
- **Multi-cell selection**: click + shift-click → selection range highlights, copy tooltips appear, copy-as-TSV / JSON write the right content.
- **Export** → DataExportButton drawer opens (now React, same behavior); CSV / JSON / SQL / XLSX downloads; password-protected ZIP works for batch query mode.
- **Multi-result set** (procedure that returns 3 result sets) → Tabs strip renders, switching tabs preserves scroll position per tab.
- **Error result** → `ErrorView` renders message + copy; PG errors show line/position highlight.
- **Sensitive data** → masked cells show lock icon; clicking opens the React `MaskingReasonPopover`.
- **Terminal mode** (`TerminalPanel`) → dark theme applies to the result view; result table renders correctly under dark colors.

## 7. Out of scope (deferred)

- The five host shells (`Panels`, `EditorPanel`, `StandardPanel`,
  `TerminalPanel`, `ResultPanel`, `SQLEditorHomePage`, `SQLEditorPage`)
  — Stage 21.
- AI plugin port — separate effort.
- Result-pane test coverage beyond the perf seed + targeted unit tests.

## 8. Risks & open questions

- **`@tanstack/react-virtual` dynamic-height differences vs. NVirtualList.**
  NVirtualList's `estimateRowHeight` callback is per-render. react-virtual
  uses a per-item measure via `measureElement`, which means heights
  stabilize after one render-and-measure round. For `VirtualDataBlock`'s
  card mode this means the first scroll past unrendered rows might
  shift — typical react-virtual behavior. Acceptable; widely accepted
  trade-off vs. a wrapper.

- **Perf parity.** The Vue NVirtualList is well-tuned; react-virtual is
  also well-tuned. Risk areas: search highlight loop, cell formatter
  cost, column-resize re-layout. Mitigations: debounce search; memoize
  cell formatter results per cell value (LRU); column-resize uses
  CSS `width` only on the colgroup, no row re-render.

- **Selection-range UX.** Vue's `selection-logic.ts` has subtle behavior
  for shift-click to extend, ctrl-click to toggle, drag-select. Port
  exactly — write the unit test against the Vue rules first to lock in
  behavior, then make the React hook pass.

- **Detail panel layering.** `DetailPanel` is a Dialog. It must mount
  via `getLayerRoot("overlay")` (per the React layering policy). Verify
  the AI side pane stacking still makes sense when the detail dialog is
  open + AI panel is open simultaneously.

- **Search highlighting on 100k rows.** Vue debounces but still loops
  over all rows once on commit. React port must do the same; the
  match-index `Set<number>` is then read-only at render time, only the
  windowed rows render highlights. No regression expected.

- **DataExportButton swap is the unlock.** Once `ResultViewV1` and
  `SingleResultViewV1` are React, both consumers of the Vue
  `DataExportButton.vue` are gone. Verify with `find-dead-vue.mjs`
  before deleting. Should be the last live caller.

- **TerminalPanel's dark mode.** Currently uses `NConfigProvider` +
  `darkTheme` at the Vue level. The React port relies on Tailwind
  `dark:` variants — but these are class-based, not theme-context-
  based. Need to either (a) toggle a `dark` class on the terminal's
  ResultView wrapper, or (b) pass a `dark` prop down through the
  context. Option (b) matches Vue's current `dark` prop flow and is
  the recommended path.

- **No existing tests.** Every test added in this stage is new. Don't
  block on full coverage — seed targeted suites for the math-heavy
  pieces (selection ranges, search index, scroll math), let the rest
  ride on manual UX verification.

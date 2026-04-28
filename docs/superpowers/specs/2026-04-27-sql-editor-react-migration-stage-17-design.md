# SQL Editor React Migration — Stage 17 Design

**Date:** 2026-04-27
**Author:** d@bytebase.com
**Status:** Done (2026-04-27)

## 1. Goal & non-goals

**Goal:** Port every Vue surface inside `EditorPanel/StandardPanel/`,
`EditorPanel/TerminalPanel/`, and the non-`ResultView` leaves under
`EditorPanel/ResultPanel/` to React. After this stage every Monaco
editor surface in the SQL Editor (worksheet + terminal) is React, and
the Vue surface area shrinks to: the host shells (`Panels.vue`,
`StandardPanel.vue`, `TerminalPanel.vue`, `ResultPanel.vue`,
`EditorPanel.vue`), `DatabaseQueryContext.vue`, and everything under
`EditorCommon/ResultView/`.

**Non-goals:**
- `EditorCommon/ResultView/*` — the `VirtualDataTable` (~458 LOC) and
  its sub-components stay Vue. Performance characteristics warrant a
  dedicated spike; deliberately deferred to the *final* migration
  stage.
- The five host shells — they each directly host a Vue ResultView
  descendant, so they fall out for free with the final stage.
- `DiagramPanel.vue` — cytoscape passthrough; separate stage.
- `DatabaseQueryContext.vue` — wraps `ResultViewV1`; defers with
  ResultView.
- `SQLEditorHomePage.vue` and `EditorPanel.vue` — depend on the host
  shells; defer.
- New features or behavioral changes — parity only.

## 2. Inventory

### 2.1 In scope (~960 LOC migrated)

| File | LOC | Migration target | Status |
|---|---|---|---|
| `StandardPanel/SQLEditor.vue` | 401 | `react/components/sql-editor/StandardPanel/SQLEditor.tsx` | ✓ |
| `StandardPanel/EditorMain.vue` | 167 | `react/components/sql-editor/StandardPanel/EditorMain.tsx` | ✓ |
| `StandardPanel/UploadFileButton.vue` | 44 | `react/components/sql-editor/StandardPanel/UploadFileButton.tsx` | ✓ |
| `TerminalPanel/CompactSQLEditor.vue` | 312 | `react/components/sql-editor/TerminalPanel/CompactSQLEditor.tsx` | ✓ |

**Bonus migrations** discovered during the upload-button chain port:

| File | LOC | Migration target | Status |
|---|---|---|---|
| `components/misc/SQLUploadButton.vue` | 99 | `react/components/sql-editor/StandardPanel/SQLUploadButton.tsx` | ✓ |
| `components/FileContentPreviewModal.vue` | 100 | `react/components/sql-editor/StandardPanel/FileContentPreviewModal.tsx` | ✓ |

### 2.2 Moved out of scope (deferred to the ResultPanel stage)

| File | LOC | Reason for deferral |
|---|---|---|
| `ResultPanel/BatchQuerySelect.vue` | 359 | Hard dependency on `DataExportButton.vue` (414 LOC, no React equivalent — its drawer + form-slot + multi-file-zip download flow is its own port) and on `DatabaseV1Table`'s pre-fetched-list mode (the React `DatabaseTable` only fetches via server filter; would require a new `databases?: Database[]` opt-out path). Both naturally co-migrate with `ResultPanel.vue` since `BatchQuerySelect` is a child of that host shell. |
| `ResultPanel/ContextMenu.vue` | 85 | Coupled to `BatchQuerySelect` via an imperative `ref.show()` API. Migrates as a unit with the parent. |

### 2.2 Stays Vue (this stage)

```
SQLEditorHomePage.vue           — still hosts EditorPanel
└── EditorPanel.vue             — still hosts Panels
    └── Panels.vue              — Stage 16 host
        └── #code-panel slot
            ├── StandardPanel.vue
            │   ├── #1: ReactPageMount("EditorMain")        ← swapped
            │   └── #2: ResultPanel.vue
            │       ├── ReactPageMount("BatchQuerySelect")  ← swapped
            │       ├── ReactPageMount("ContextMenu")       ← swapped
            │       └── DatabaseQueryContext.vue (uses ResultView)
            └── TerminalPanel.vue
                └── loop body:
                    ├── ReactPageMount("CompactSQLEditor")  ← swapped
                    └── ResultViewV1 (Vue, untouched)
```

The five Vue containers gain `ReactPageMount` calls but keep their NSplit / loop / dropdown wiring.

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `MonacoEditor` (Vue) | `MonacoEditor.tsx` (React, 659 LOC) — the React Monaco wrapper used by `ReadonlyMonaco`, `EditorAction`, etc. | ✓ exists |
| `formatSQL` from `@/components/MonacoEditor/sqlFormatter` | Same module, framework-neutral | ✓ |
| `useExecuteSQL` (Vue composable) | Either lift the Vue composable to a JS module (it already runs on Pinia stores) or replicate the body in a React hook | Lift — Phase 1 |
| `AIChatToSQL` (Vue Suspense pane) | Hoisted to a Vue host wrapper (Stage 16 pattern) — for `EditorMain`, the host is `StandardPanel.vue` | Reuse |
| `useAIActions` Monaco command | Already React — `Panels/common/useAIActions.ts` (Stage 16) | ✓ |
| `OpenAIButton` | Already React | ✓ |
| `BBSpin` (Vue spinner) | Same `Loader2` pattern used by other React leaves (`<Loader2 className="animate-spin" />`) | Reuse |
| `NDropdown` (right-click context menu) | shadcn `DropdownMenu` | ✓ |
| `NTabPane` / `NTabs` (BatchQuerySelect tab strip) | shadcn `Tabs` | ✓ |
| `NSplit` (EditorMain editor / AI pane) | Hoisted to `StandardPanel.vue` (stays Vue) | ✓ |
| `useEmitteryEventListener` (Vue composable) | `useEffect` + direct emittery `.on()` — pattern already used in `SchemaPane.tsx` | ✓ |
| `defineAsyncComponent(() => import("./SQLEditor.vue"))` | React lazy via `React.lazy()` + `<Suspense>` boundary | ✓ |

## 4. Architecture & phases

Three phases. Each is independently commit-able and gate-green.

### Phase 1 — `useExecuteSQL` portability + small leaves

**Files added:**
- React `UploadFileButton.tsx` + top-level shim (44 LOC)
- React `ContextMenu.tsx` + top-level shim (85 LOC) — small `DropdownMenu` wrapper around the context menu actions

**Refactor:**
- Verify `useExecuteSQL` is callable from React. If it currently uses
  Vue refs in its return shape, lift the body to a framework-neutral
  function and have the Vue composable be a thin wrapper. The React
  side gets a `useExecuteSQL` React hook that calls the same core.

**Caller swap:**
- `StandardPanel/SQLEditor.vue` line N: replace the Vue
  `<UploadFileButton />` import + render with
  `<ReactPageMount page="UploadFileButton" />` *before* migrating
  `SQLEditor.vue` itself, so the swap is incremental. (Optional —
  could also wait for Phase 3 since `SQLEditor` migrates there.)
- `ResultPanel.vue`: replace the Vue `<ContextMenu />` with
  `<ReactPageMount page="ContextMenu" />`.

**Deliverables:** ~3 `.tsx` + 2 mount shims + 2 tests.

### Phase 2 — `BatchQuerySelect` + `CompactSQLEditor`

**Files added:**
- React `BatchQuerySelect.tsx` + top-level shim (359 LOC). Uses
  shadcn `Tabs`; the close-tab action wires through to Pinia store
  helpers the Vue version already calls (`tabStore.removeTab`,
  `closeOtherBatchQueryItems`, etc.).
- React `CompactSQLEditor.tsx` + top-level shim (312 LOC). Wraps
  `MonacoEditor` with terminal-specific options (single-line, dark
  theme, no minimap, history nav keybindings).

**Caller swaps:**
- `ResultPanel.vue`: replace `<BatchQuerySelect />` with
  `<ReactPageMount page="BatchQuerySelect" />`.
- `TerminalPanel.vue`: replace the inline `<CompactSQLEditor>` inside
  the `v-for` loop with `<ReactPageMount page="CompactSQLEditor" :pageProps="{ ... }" />`.
  Per-query props (`content`, `readonly`, `onExecute`, `onHistory`,
  `onClearScreen`) flow through `pageProps`.

**Deliverables:** 2 `.tsx` + 2 mount shims + 2 tests + 2 caller swaps.

### Phase 3 — `SQLEditor` + `EditorMain`

**Files added:**
- React `SQLEditor.tsx` + top-level shim (401 LOC). Worksheet Monaco
  editor with autocomplete, multi-cursor, Ctrl+Enter execute,
  Cmd+S save, format-on-save, etc. Reuses the existing
  `MonacoEditor.tsx` for the editor instance.
- React `EditorMain.tsx` + top-level shim. Renders:
  - `<EditorAction>` (already React)
  - `<SQLEditor>` (just-migrated)
  - `<Welcome>` (already React) when `tab` is null
  - `<ExecutingHintModal>` + `<SaveSheetModal>` (already React, hidden mounts)
  No AI pane inside (hoisted — see below).

**AI pane hoist (mirrors Stage 16 pattern):**
- `StandardPanel.vue` wraps the React `EditorMain` in a horizontal
  `<NSplit>` when `showAIPanel`:
  ```html
  <NSplit
    v-if="showAIPanel"
    direction="horizontal"
    :size="editorPanelSize.size"
    ...
  >
    <template #1>
      <ReactPageMount page="EditorMain" />
    </template>
    <template #2>
      <Suspense>
        <AIChatToSQL key="ai-chat-to-sql" />
      </Suspense>
    </template>
  </NSplit>
  ```
- React `EditorMain.tsx` sets `uiStore.isShowingCode = true` on mount
  (same flag introduced in Stage 16) so `Panels.vue` doesn't
  *also* try to host the AI pane in CODE view (Panels.vue's gate is
  `view !== 'CODE'`, but we'll add the explicit
  `&& view !== 'CODE'` guard if it isn't there already).

**Caller swap:**
- `StandardPanel.vue`: replace `<EditorMain />` with
  `<ReactPageMount page="EditorMain" />` inside the new horizontal
  NSplit.

**Deliverables:** 2 `.tsx` + 2 mount shims + 2 tests + 1 caller swap +
small `StandardPanel.vue` refactor.

## 5. Per-phase checklist

### Phase 1 — Portability + leaves
- [x] `UploadFileButton.tsx` + shim (with chain: `SQLUploadButton.tsx` + `FileContentPreviewModal.tsx`)
- [x] Delete `StandardPanel/UploadFileButton.vue` + `components/misc/SQLUploadButton.vue` + `components/FileContentPreviewModal.vue`
- [x] Gates green
- [—] `useExecuteSQL` lift: not needed — the Vue composable's body is
  Pinia-store-driven. `reactive({})` and `markRaw()` are no-ops outside
  Vue's reactivity system; the React `EditorMain` calls
  `useExecuteSQL()` directly.
- [↗] `ContextMenu.tsx` migration deferred to the ResultPanel stage (see § 2.2).

### Phase 2 — CompactSQLEditor
- [x] `CompactSQLEditor.tsx` + shim
- [x] `TerminalPanel.vue` swaps `CompactSQLEditor` (per-query `pageProps` flow)
- [x] Delete `TerminalPanel/CompactSQLEditor.vue`
- [x] Gates green
- [↗] `BatchQuerySelect.tsx` deferred to the ResultPanel stage (see § 2.2).

### Phase 3 — SQLEditor + EditorMain
- [x] `SQLEditor.tsx` + shim (worksheet Monaco editor with Cmd+Enter run / Cmd+Shift+Enter run-in-new-tab / Cmd+S save / Cmd+E explain — engine-conditional / AI Monaco actions / format / append-content / set-selection / pendingInsertAtCaret)
- [x] `EditorMain.tsx` + shim (mounts `EditorAction`, `SQLEditor`, `Welcome`, hidden `ExecutingHintModal` + `SaveSheetModal`)
- [x] AI pane hoist landed in `StandardPanel.vue` (Vue `<NSplit>` + `<AIChatToSQL>` wrapping the React `EditorMain`)
- [x] `StandardPanel.vue` swaps `EditorMain` to `ReactPageMount`
- [x] Delete `StandardPanel/EditorMain.vue` + `StandardPanel/SQLEditor.vue`
- [x] Cross-framework `getActiveStatement`: replaced the Vue `defineExpose` ref API with a shared `activeStatementRef` shallowRef in `state.ts` — React `SQLEditor` writes via `onActiveContentChange`, Vue `EditorMain` (now React) reads it from the toolbar Run handler. Mirrors the existing `activeSQLEditorRef` pattern.
- [x] Gates green

## 6. Manual UX verification (after each phase)

- **Phase 1:** Upload a `.sql` file via the editor button; right-click
  a result tab → close / close-others / close-saved actions still
  work.
- **Phase 2:** Run a batch query against multiple databases → tab
  strip renders one tab per database, errors flagged, close-tab and
  keyboard nav (`⌘+W`, `⌘+⇧+W`) behave the same. Switch a tab to ADMIN
  mode → CompactSQLEditor renders dark, executes commands, history
  nav (Ctrl+P / Ctrl+N) works.
- **Phase 3:** WORKSHEET tab → run query (`⌘+Enter`); save sheet
  (`⌘+S`); format SQL; toggle AI panel (now hosted by Vue
  StandardPanel) → `AIChatToSQL` renders to the right; explain code
  via OpenAIButton or Monaco context menu. Welcome screen renders
  when no tab is connected.
- **All:** Switch between WORKSHEET and ADMIN tabs; tab key change
  doesn't leak Monaco instances; no stale event listeners.

## 7. Out of scope (deferred)

- `EditorCommon/ResultView/*` — the perf-sensitive VirtualDataTable.
  Final stage covers this together with the host shells.
- The five host shells (`Panels.vue`, `StandardPanel.vue`,
  `TerminalPanel.vue`, `ResultPanel.vue`, `EditorPanel.vue`) — they
  each transitively host a Vue ResultView descendant; migrating them
  before ResultView would force a React→Vue mount bridge.
- `DiagramPanel.vue` — cytoscape; separate stage.
- `SQLEditorHomePage.vue` — depends on EditorPanel; final stage.
- **`ResultPanel/BatchQuerySelect.vue` + `ResultPanel/ContextMenu.vue`**
  — moved out of Stage 17 mid-stage. `BatchQuerySelect` would require
  porting `DataExportButton.vue` (414 LOC of drawer + form-slot +
  multi-file zip download flow with no React equivalent) and
  extending the React `DatabaseTable` with a pre-fetched-list opt-out
  path; `ContextMenu` is coupled to it via an imperative `ref.show()`
  API. Both naturally co-migrate with `ResultPanel.vue`'s host shell,
  so they roll into the ResultPanel stage rather than expanding
  Stage 17's scope. See § 2.2.

## 8. Risks & open questions

- **`useExecuteSQL` portability.** The Vue composable returns Vue
  refs (loading state, etc.) and uses `watch`/`computed`. Likely
  needs a React-friendly variant. **Mitigation:** if the body is
  pure Pinia + emittery, lift it to a JS function and wrap with a
  React `useState` hook — same pattern as `useAIActions`. If it has
  heavier Vue lifecycle ties, defer the React `EditorMain` for one
  phase and stub a thin event-driven wrapper instead.

- **Async SQLEditor mount.** Vue uses `defineAsyncComponent` to
  lazy-load `SQLEditor.vue`. The React lazy equivalent is
  `React.lazy()` + Suspense boundary. The mount registry already
  lazy-loads via `import.meta.glob`, so a top-level `SQLEditor.tsx`
  shim is automatically lazy. **No new infrastructure needed.**

- **AI pane hoist behavior change.** Today AI pane is inside
  `EditorMain.vue` (next to SQLEditor). After hoist it's inside
  `StandardPanel.vue` (sibling of EditorMain). Layout-wise identical;
  but the hoist creates an extra NSplit boundary. Confirm
  `editorPanelSize` resize math still works — it should, since the
  same Pinia store drives both.

- **Terminal mode `Suspense` per-row.** Vue wraps each
  `CompactSQLEditor` in `<Suspense>` because of its async mount.
  React equivalent: each `<ReactPageMount>` is its own React root,
  so each row is independently mounted. **Pre-existing behavior.**

- **Cross-mount key stability in TerminalPanel loop.** Today Vue
  re-mounts when `query.id` changes. We need to pass `:key="query.id"`
  to `ReactPageMount` and confirm the underlying React root unmounts
  cleanly. **Verify in Phase 2.**

- **`pageProps` for terminal queries.** Each terminal-mode
  `CompactSQLEditor` mount needs the per-query
  `{ content, readonly, onExecute, onHistory, onClearScreen }`. The
  existing `ReactPageMount` `pageProps` mechanism already supports
  this. Vue→React event handlers are normal function props.

- **History keybindings (Ctrl+P / Ctrl+N).** Vue version registers
  Monaco actions for history navigation. Same pattern as Stage 16's
  `useAIActions` — register `editor.addAction` in a `useEffect`.

## 9. Outcomes (2026-04-27)

**Migrated:** `SQLEditor.vue`, `EditorMain.vue`, `UploadFileButton.vue`,
`SQLUploadButton.vue` (chain), `FileContentPreviewModal.vue` (chain),
`CompactSQLEditor.vue` — ~960 LOC of Vue→React.

**Vue surface remaining inside `EditorPanel/`:** `EditorPanel.vue`
(host), `Panels/Panels.vue` (host with `AIChatToSQL`),
`Panels/DiagramPanel/*` (cytoscape, deferred), `StandardPanel.vue`
(host with AI pane hoist), `TerminalPanel.vue` (host),
`ResultPanel/*` (host + `BatchQuerySelect` + `ContextMenu` +
`DatabaseQueryContext`).

**Architectural decisions:**

1. **`useExecuteSQL` reused as-is.** The Vue composable's body is
   Pinia-store-driven; `reactive({})` and `markRaw()` are no-ops
   outside Vue's reactivity system. The React `EditorMain` calls it
   directly — no port required.
2. **`activeStatementRef` shallowRef bridge.** When the toolbar
   Run button fires from React, it can't access the previous Vue
   `sqlEditorRef.value?.getActiveStatement()` because `ReactPageMount`
   doesn't pass refs across frameworks. Added a module-level
   `shallowRef<string>` in `state.ts` (parallel to the existing
   `activeSQLEditorRef`); React `SQLEditor` writes via
   `onActiveContentChange`; Vue / React readers consume `.value`.
3. **AI pane hoisted to `StandardPanel.vue`.** `AIChatToSQL` is
   Vue-only; embedding it in React `EditorMain` would force a
   React→Vue mount bridge. The Vue `<NSplit>` + `<AIChatToSQL>`
   moved one level up so the React panel only owns the editor
   surface — same pattern Stage 16 applied to the schema-object
   panels.
4. **i18n parity.** 3 missing React-locale keys synced from Vue
   (`common.preview`, `sql-editor.select-encoding`,
   `sql-editor.upload-file`).

**Validation:** `pnpm fix` / `pnpm check` / `pnpm type-check` all
green. 285 React tests passing.

**Rolls into next stage:** `BatchQuerySelect.vue` + `ContextMenu.vue`
(see § 7).

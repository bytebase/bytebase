# SQL Editor React Migration — Stage 13 Design

**Date:** 2026-04-24
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Retire the top editor toolbar (`EditorAction.vue` + its sub-components) and the remaining SQL-Editor-level placeholder surfaces (`SQLEditorHomePage.vue`, `ReadonlyModeNotSupported.vue`, `SheetConnectionIcon.vue`, orphaned placeholders) by porting them to React end-to-end.

After this stage, every in-scope file listed below is React; only Vue parents that still host React via `ReactPageMount` remain temporarily.

**Non-goals:**
- Migrating `EditorPanel.vue`, `StandardPanel.vue`, `EditorMain.vue`, `TerminalPanel.vue`, or `TabList/*`. They stay Vue — they will mount the new React `EditorAction` via `ReactPageMount`.
- Migrating `SQLEditor.vue` (Monaco wrapper), the `ResultView` subtree, the Panels (`TablesPanel`, `ViewsPanel`, etc.), or the `SchemaPane` tree — separate stages.
- Refactoring the existing Pinia stores (`useSQLEditorStore`, `useSQLEditorTabStore`, `useWorkSheetStore`, `useSQLEditorContext`). We read/write them via `useVueState` / direct method calls as we already do in Stages 11/12.
- Adding new features — behavioral + visual parity only.

## 2. Inventory (what's in scope)

| # | File | LOC | Complexity | Notes |
|---|---|---|---|---|
| 1 | `frontend/src/views/sql-editor/Disconnected.vue` | 1 | trivial | Orphan. Delete. |
| 2 | `frontend/src/views/sql-editor/GettingStarted.vue` | 1 | trivial | Orphan. Delete. |
| 3 | `frontend/src/views/sql-editor/EditorPanel/ReadonlyModeNotSupported.vue` | 24 | low | Uses Vue `AdminModeButton.vue`. |
| 4 | `frontend/src/views/sql-editor/EditorCommon/SheetConnectionIcon.vue` | 24 | low | Consumed by `TabList/TabItem/Prefix.vue`. |
| 5 | ~~`SQLEditorHomePage.vue`~~ | ~~197~~ | — | **Deferred** — turned out to be the root container that hosts `AsidePanel`, `TabList`, `EditorPanel`, `ConnectionPanel`, `Drawer`, `Quickstart`, `IAMRemindModal`, all still Vue. Migrating it requires those children to be React first. Late-stage target. |
| 6 | `frontend/src/views/sql-editor/EditorCommon/OpenAIButton/Button.vue` | 55 | low | Presentational; inline into parent. |
| 7 | `frontend/src/views/sql-editor/EditorCommon/OpenAIButton/OpenAIButton.vue` | 164 | medium | Consumed by `EditorAction`, `CodeViewer`, `ViewDetail`. |
| 8 | `frontend/src/views/sql-editor/EditorCommon/ExecuteHint.vue` | 188 | medium | Modal body. |
| 9 | `frontend/src/views/sql-editor/EditorCommon/ExecutingHintModal.vue` | 24 | low | Wraps `ExecuteHint` in a BBModal. Currently rendered inline in `EditorMain.vue`. |
| 10 | `frontend/src/views/sql-editor/EditorCommon/EditorAction.vue` | 316 | medium-high | Primary target. Consumed by `EditorMain.vue` + `TerminalPanel.vue`. Emits `execute`. |
| 11 | `frontend/src/views/sql-editor/EditorCommon/AdminModeButton.vue` | 63 | low | React version already exists (`AdminModeButton.tsx`); Vue version is dead after callers migrate. |

Already-React pieces this stage reuses without modification:
- `QueryContextSettingPopover.tsx` (Stage 10)
- `ChooserGroup.tsx` (Stage 9-ish)
- `AdminModeButton.tsx` (already React)
- `SharePopoverBody.tsx` (existing; we use it directly now, no `ReactPageMount` round-trip)

## 3. Phase breakdown

Each phase is commit-able and lint/type-check/test green on its own.

### Phase 1 — Placeholder cleanup (quick wins)

**Deliverables:**
- Delete `Disconnected.vue`, `GettingStarted.vue` (confirmed unused).
- Port `ReadonlyModeNotSupported.vue` → `frontend/src/react/components/sql-editor/ReadonlyModeNotSupported.tsx`.
  - Composes React `NoPermissionPlaceholder` (create if missing — tiny; otherwise inline the "no permission" styling), `InstanceV1Name` equivalent (already used elsewhere in React — reuse), and `AdminModeButton.tsx`.
  - Swap `EditorMain.vue`'s `<ReadonlyModeNotSupported v-else />` → `<ReactPageMount v-else page="ReadonlyModeNotSupported" />`.
- Port `SheetConnectionIcon.vue` → `SheetConnectionIcon.tsx`.
  - Swap the one caller (`TabList/TabItem/Prefix.vue`) to `<ReactPageMount page="SheetConnectionIcon" :tab="tab" />`.
- Delete Vue originals after swapping.

**Tests (new):** `ReadonlyModeNotSupported.test.tsx`, `SheetConnectionIcon.test.tsx`, `SQLEditorHomePage.test.tsx`. Pattern: mock stores/router and assert core strings + key interactions.

**Dependencies needed:** Any missing Base UI / shadcn primitives should already exist (`Button`, `Tooltip`). No new primitives required.

### Phase 2 — OpenAIButton

**Deliverables:**
- Port `OpenAIButton/OpenAIButton.vue` + `Button.vue` → `frontend/src/react/components/sql-editor/OpenAIButton.tsx` (merge both into one file — `Button.vue` is purely presentational).
- Wraps the existing AI plugin state (pinia store `useAIContextStore` / similar — check the source for exact store).
- Swap callers one at a time:
  - `EditorAction.vue` — keep Vue until Phase 4, but swap the import to `<ReactPageMount page="OpenAIButton" :statement="..." />`. Or defer the swap to Phase 4 (cleaner).
  - `EditorPanel/Panels/common/CodeViewer.vue` — swap to `ReactPageMount`.
  - `EditorPanel/Panels/ViewsPanel/ViewDetail.vue` — swap to `ReactPageMount`.

**Tests (new):** `OpenAIButton.test.tsx` — covers: feature-flag gating, button disabled when disconnected, click opens the AI chat panel (mock emit).

**Delete:** `OpenAIButton/Button.vue`, `OpenAIButton/OpenAIButton.vue`, `OpenAIButton/index.ts`, and the dir afterwards.

### Phase 3 — ExecuteHint + ExecutingHintModal

**Deliverables:**
- Port `ExecuteHint.vue` → `frontend/src/react/components/sql-editor/ExecuteHint.tsx`. Body of the confirmation modal: "do you want to run admin mode?" + `AdminModeButton` action.
- Port `ExecutingHintModal.vue` → `frontend/src/react/components/sql-editor/ExecutingHintModal.tsx`.
  - Replaces `BBModal` with the shadcn `Dialog` primitive.
  - Reads `sqlEditorStore.isShowExecutingHint` / `executingHintDatabase` via `useVueState`.
- Swap `EditorMain.vue`'s `<ExecutingHintModal />` → `<ReactPageMount page="ExecutingHintModal" />`.
- Delete Vue `ExecuteHint.vue` + `ExecutingHintModal.vue`.

**Tests (new):** `ExecutingHintModal.test.tsx` — renders when flag true, closes on OK, delegates to `AdminModeButton`.

### Phase 4 — EditorAction toolbar (primary target)

**Deliverables:**
- Port `EditorAction.vue` → `frontend/src/react/components/sql-editor/EditorAction.tsx`.

**Layout (parity with Vue):**
```
action-left:
  - [ADMIN mode?]  Exit-Admin-Mode button
  - [WORKSHEET mode?] ButtonGroup: Run-button (with query limit) + QueryContextSettingPopover
  - AdminModeButton (icon-only)
  - [showSheetsFeature?] Save button (with Save hover tooltip + Cmd/Ctrl+S shortcut hint)
  - [showSheetsFeature?] Share button (hover tooltip "Share" + click popover → SharePopoverBody)
action-right:
  - ChooserGroup (already React)
  - OpenAIButton (from Phase 2)
```

**Primitives used (all exist):**
- `Button` / `ButtonGroup` (our shadcn lib — check if we need to add a `ButtonGroup` wrapper; otherwise compose with `flex gap-0`)
- `Tooltip` (for hover titles)
- `Popover` (for Save hover + Share click)
- `Dialog` (via `FeatureModal` — already React)
- `ReactPageMount` is gone from this subtree — direct imports only

**State reads** (via `useVueState`):
- `currentTab`, `isDisconnected` (tab store)
- `instance` (connection)
- `resultRowsLimit` (sql-editor store)
- `worksheet` (via `getWorksheetByName`)
- `showConnectionPanel`, `asidePanelTab` (sql-editor context for share handlers — unchanged)

**Props:**
```ts
type Props = {
  readonly onExecute: (params: SQLEditorQueryParams) => void;
};
```

**Caller swaps:**
- `EditorMain.vue`: `<EditorAction @execute="handleExecuteFromActionBar" />` → `<ReactPageMount page="EditorAction" :onExecute="handleExecuteFromActionBar" />`
- `TerminalPanel.vue`: `<EditorAction />` → `<ReactPageMount page="EditorAction" :onExecute="noop" />` (or an admin-mode-safe no-op; verify original behaviour — the `TerminalPanel` doesn't pass `@execute`, so it runs in ADMIN mode where the Run button is not rendered anyway).

**Share popover wiring:**
- Use React `Popover` + `SharePopoverBody` directly (the existing React component already works; no `ReactPageMount` round-trip needed inside React code).
- Hover tooltip + click popover are two distinct interactions — use `Tooltip` outside / inside `Popover` per the Vue structure.

**Tests (new):** `EditorAction.test.tsx` — 5 scenarios:
1. Renders Run button when connected + non-empty statement; disabled otherwise.
2. Clicking Run fires `onExecute` with the tab's statement + connection.
3. Save button disabled when status === 'CLEAN'; enabled when dirty; click emits `save-sheet` event.
4. Share popover opens on click when `allowShare` is true.
5. In ADMIN mode, the Exit-Admin button renders (not the Run button).

### Phase 5 — Cleanup

- Delete Vue originals:
  - `EditorAction.vue`
  - `AdminModeButton.vue` (all callers now React after Phases 1, 3, 4)
  - Empty directories if any.
- Update `frontend/src/views/sql-editor/EditorCommon/index.ts` to drop the re-exports.
- `rg "EditorAction\.vue|OpenAIButton\.vue|ExecuteHint\.vue|ExecutingHintModal\.vue|SheetConnectionIcon\.vue|ReadonlyModeNotSupported\.vue|SQLEditorHomePage\.vue|Disconnected\.vue|GettingStarted\.vue"` — confirm zero results.
- Gates: `pnpm fix && pnpm type-check && pnpm test --run src/react/components/sql-editor src/views/sql-editor && pnpm check` all green.

## 4. Per-phase checklist

### Phase 1
- [ ] `ReadonlyModeNotSupported.tsx` + test
- [ ] `SheetConnectionIcon.tsx` + test
- [ ] `SQLEditorHomePage.tsx` + test
- [ ] Delete the four Vue files
- [ ] Swap `EditorMain.vue` + `TabItem/Prefix.vue` + `SQLEditorPage.vue` (or router entry) to `ReactPageMount`
- [ ] Gates green

### Phase 2
- [ ] `OpenAIButton.tsx` + test
- [ ] Swap 3 callers to `ReactPageMount` (or inline direct import when the caller is already React)
- [ ] Delete Vue `OpenAIButton/*`
- [ ] Gates green

### Phase 3
- [ ] `ExecuteHint.tsx` (no separate test if trivial)
- [ ] `ExecutingHintModal.tsx` + test
- [ ] Swap `EditorMain.vue` to `ReactPageMount page="ExecutingHintModal"`
- [ ] Delete 2 Vue files
- [ ] Gates green

### Phase 4
- [ ] `EditorAction.tsx` + 5 tests
- [ ] Swap `EditorMain.vue` + `TerminalPanel.vue`
- [ ] `OpenAIButton` imported directly (not via `ReactPageMount`) since we're in React
- [ ] `SharePopoverBody` imported directly
- [ ] Gates green

### Phase 5
- [ ] Delete Vue `EditorAction.vue`, `AdminModeButton.vue`
- [ ] Prune `EditorCommon/index.ts`
- [ ] `rg` search confirms zero remaining references
- [ ] `pnpm fix && type-check && test && check` all green

## 5. Manual UX verification (after all phases)

1. Empty editor (no tab): Welcome + Run button disabled; saves a draft on Cmd/Ctrl+S.
2. Connected + non-empty SQL: Run button enabled; click executes; result panel appears.
3. Run button shows limit indicator; QueryContextSettingPopover opens next to Run.
4. Switch to ADMIN mode (via AdminModeButton) → Run button replaced by "Exit Admin Mode".
5. Save button — disabled when status=CLEAN; Cmd/Ctrl+S triggers save; toast confirms.
6. Share button — hover shows "Share" tooltip; click opens `SharePopoverBody`; toggling visibility persists.
7. ChooserGroup on the right — selects environment/database/schema.
8. OpenAIButton — disabled when disconnected; click toggles the AI side panel.
9. ExecutingHintModal — run a DML on a restricted DB → modal appears → confirm opens Admin Mode.
10. ReadonlyModeNotSupported — connect to instance without readonly → placeholder renders with AdminModeButton.
11. Tab bar — each tab's SheetConnectionIcon renders correct state glyph.
12. Landing page — open SQL Editor with no project selected → SQLEditorHomePage renders.

## 6. Out of scope (deferred)

- `SQLEditorPage.vue` itself — the top-level route. Stays Vue; only hosts React inside.
- `EditorMain.vue`, `StandardPanel.vue`, `EditorPanel.vue`, `TerminalPanel.vue` — stay Vue.
- `TabList/*` — separate stage.
- `SchemaPane` tree + node types — separate stage.
- `ResultView` + `VirtualDataTable` — separate stage.
- `SQLEditor.vue` Monaco wrapper — complex; separate stage.

## 7. Risks & open questions

- **OpenAIButton AI store:** the exact pinia store driving the AI chat panel needs verification. Use `useVueState` to read it. If the AI panel is controlled from a React context we've already introduced, prefer that.
- **FeatureModal:** currently imported from `@/components/FeatureGuard` (Vue). Check if a React equivalent exists; if not, mount via `<ReactPageMount>` OR port a thin React wrapper in Phase 4.
- **Keyboard shortcut:** `Cmd/Ctrl+S` binding is currently registered globally via Vue's `useKeyboardShortcut` (or similar). Ensure we don't double-register from React; likely the binding lives in a composable invoked by the parent Vue tree, unchanged.
- **TerminalPanel execute handler:** the Vue `<EditorAction />` in `TerminalPanel.vue` doesn't pass `@execute`. In React, `onExecute` must be an explicit required prop, so callers pass a no-op. Document this in the component comment.
- **SharePopoverBody** is already React and mounted elsewhere (in SheetTree via `ReactPageMount` historically, now directly). Phase 4 uses it directly — verify no props-API drift from its last use site.

# SQL Editor React Migration — Stage 8 Design (Revised)

**Date:** 2026-04-21
**Author:** d@bytebase.com
**Status:** Implemented as infrastructure-only (UI migration reverted mid-stage per the no-partial-regression principle)

## Revision note

The original Stage 8 plan included migrating `SaveSheetModal.vue` as the first consumer of the new bridges, with a "ship without folder picker (F2)" feature regression to avoid porting the heavyweight `FolderForm.vue` sub-dependency.

**That plan was wrong.** After the React SaveSheetModal shipped, the missing folder picker was flagged as a visible UX regression. The user established a firm rule: *during migration, either don't migrate the Vue file at all, or migrate the entire UX with full feature parity — no middle way.* (See `feedback_migration_no_partial_regression.md` in memory.)

The SaveSheetModal portion was reverted. Stage 8 ships as infrastructure only. SaveSheetModal migration is deferred until `FolderForm` + its tree primitive + `useSheetContextByView` inject bridge are ready (a future stage).

## 1. Goal & non-goals

**Goal:** Build the Emittery `events` bus bridge and the async-actions bridge so that future stages (starting with Stage 9+) can migrate Vue files that consume editor events or call the async worksheet methods, without having to design the bridges each time.

**Non-goals (Stage 8):**
- Migrating any Vue UI component. No `.vue` file is swapped or deleted.
- Porting `FolderForm.vue` or a tree primitive.
- Bridging the AI plugin context (`useAIContext`).
- Vue-in-React mount infrastructure.

## 2. What shipped

### 2.1 Emittery singleton

**File:** `frontend/src/views/sql-editor/events.ts`

Extracted the `SQLEditorEvents` type union from `context.ts` and created a module-level singleton Emittery instance. Both Vue and React code import the same instance directly; emit/listen is symmetric across both.

```ts
import Emittery from "emittery";
import type { IRange } from "monaco-editor";
import type { SQLEditorTab } from "@/types";

export type SQLEditorEvents = {
  "save-sheet": { tab: SQLEditorTab; editTitle?: boolean };
  "alter-schema": { databaseName: string; schema: string; table: string };
  "format-content": undefined;
  "tree-ready": undefined;
  "project-context-ready": { project: string };
  "set-editor-selection": IRange;
  "append-editor-content": { content: string; select: boolean };
  "insert-at-caret": { content: string };
};

export const sqlEditorEvents: Emittery<SQLEditorEvents> =
  new Emittery<SQLEditorEvents>();
```

**Test:** `frontend/src/views/sql-editor/events.test.ts` — 2 tests: singleton identity, emit/on roundtrip.

### 2.2 React subscription hook

**File:** `frontend/src/react/hooks/useSQLEditorEvent.ts`

```ts
export function useSQLEditorEvent<E extends keyof SQLEditorEvents>(
  event: E,
  handler: (data: SQLEditorEvents[E]) => void
): void;
```

Uses a ref pattern so handler identity changes don't cause resubscribe churn. Unsubscribes on unmount. React consumers that need to emit import `sqlEditorEvents` directly and call `.emit(...)`.

**Test:** `frontend/src/react/hooks/useSQLEditorEvent.test.tsx` — 3 tests: subscribe on mount, unsubscribe on unmount, handler ref updates on rerender.

### 2.3 Async-actions Pinia store

**File:** `frontend/src/store/modules/sqlEditor/worksheet.ts`

Created `useSQLEditorWorksheetStore` — a Pinia setup-store that lifted 4 methods and the auto-save AbortController out of `provideSQLEditorContext()`:

- `autoSaveController: Ref<AbortController | null>`
- `abortAutoSave()` — aborts controller and clears it
- `maybeSwitchProject(projectName)` — fetches project + IAM policy, sets `editorStore.project`, emits `project-context-ready`
- `maybeUpdateWorksheet({ tabId, worksheet, title, database, statement, folders, signal })` — patches existing worksheet
- `createWorksheet({ tabId, title, statement, folders, database })` — creates new worksheet, optionally opens a new tab

Follows the same pattern as Stage 2's `useSQLEditorUIStore` lift. Closures over 5 sibling Pinia stores are moved into the setup function.

Exported from the `sqlEditor/index.ts` barrel.

**Test:** `frontend/src/store/modules/sqlEditor/worksheet.test.ts` — 4 smoke tests: initial controller state, abortAutoSave no-op and active paths, maybeSwitchProject with invalid name.

### 2.4 `context.ts` refactor

Changed `provideSQLEditorContext()` to delegate:
- `events` field: changed from `new Emittery()` to the `sqlEditorEvents` singleton
- The 4 async action fields: delegated to `useSQLEditorWorksheetStore()`
- The module-level `autoSaveController` and `abortAutoSave` function: removed; `watchDebounced` auto-save composable now uses `worksheetStore.abortAutoSave()` and `worksheetStore.maybeUpdateWorksheet(...)` directly, and reads/writes `worksheetStore.autoSaveController` for controller management

**Invariant preserved:** The `SQLEditorContext` type shape is identical. All 30+ Vue consumers of `useSQLEditorContext()` continue to work without edits.

Type re-export compatibility: `SQLEditorEvents` is re-exported from `context.ts` as `Emittery<SQLEditorEventsMap>` (Emittery-wrapped type) so that Vue consumers typing parameters as `SQLEditorEvents` work unchanged. The new `events.ts` exports the plain event-name map.

## 3. What this unblocks for future stages

Any Vue file that uses `useSQLEditorContext().events.on(...)`, `.emit(...)`, or the 4 async methods can now have a React equivalent that uses:
- `useSQLEditorEvent("event-name", handler)` for listening
- `sqlEditorEvents.emit("event-name", data)` for emitting
- `useSQLEditorWorksheetStore()` for the async methods

Future stage candidates that consume these bridges:
- `HistoryPane.vue` — emits `append-editor-content`
- `ErrorView.vue`, `AdviceItem.vue` — emit `set-editor-selection`
- `TabList.vue` — consumes `createWorksheet`
- `SaveSheetModal.vue` — when `FolderForm` is ported (full parity restoration)
- `AsidePanel.vue` — consumes `maybeSwitchProject`

## 4. What was reverted (and why)

The original plan (now understood to violate the no-partial-regression principle):
- Created React `SaveSheetModal.tsx` that passed `folders: []` on both save paths.
- Swapped `StandardPanel.vue`'s caller to `<ReactPageMount page="SaveSheetModal" />`.
- Deleted the Vue `SaveSheetModal.vue`.
- Added `sql-editor.save-sheet` + `save-sheet-input-placeholder` i18n keys.

Problem: users lost the folder-picker UI when saving new worksheets. Existing folder assignments were preserved by backend but could not be changed through the React modal. This was a real regression, not acceptable per the no-partial-regression principle.

Revert actions:
- Deleted React `SaveSheetModal.tsx` + test
- Restored Vue `SaveSheetModal.vue` (via `git checkout`)
- Restored `EditorCommon/index.ts` barrel (import + export for `SaveSheetModal`)
- Restored `StandardPanel.vue:64` to `<SaveSheetModal />`
- Removed the 2 i18n keys from all 5 React locales (no React consumer now)

## 5. Verification of what shipped

- `pnpm fix && check && type-check && test` all green
- Tests: 1286 passing — includes the +9 Stage 8 infra tests (2 events + 3 hook + 4 store)
- Type-check baseline only (6 pre-existing `SchemaEditorLite` errors, no new)
- No Vue UI changed; no Vue files deleted
- `useSQLEditorContext()` returns identical shape; all existing Vue flows (save-sheet via Ctrl+S, auto-save, project switch, worksheet create from Welcome) work unchanged — exercised via the refactored `watchDebounced` composable and the store-delegated methods

## 6. Practical checklist

### Infrastructure (completed)

- [x] `frontend/src/views/sql-editor/events.ts` created
- [x] `frontend/src/views/sql-editor/events.test.ts` created
- [x] `frontend/src/react/hooks/useSQLEditorEvent.ts` created
- [x] `frontend/src/react/hooks/useSQLEditorEvent.test.tsx` created
- [x] `frontend/src/store/modules/sqlEditor/worksheet.ts` created
- [x] `frontend/src/store/modules/sqlEditor/worksheet.test.ts` created
- [x] `modules/sqlEditor/index.ts` barrel exports new store
- [x] `context.ts` refactored (type re-export, events singleton, async-method delegation, auto-save composable uses store)
- [x] `SQLEditorContext` shape preserved; Vue consumers unchanged
- [x] `pnpm fix && check && type-check && test` all pass
- [ ] ~~React SaveSheetModal + swap + deletion~~ — REVERTED per §4

### HistoryPane migration (new addition — full feature parity)

- [ ] `frontend/src/react/components/sql-editor/HistoryPane.tsx` created
- [ ] `frontend/src/react/components/sql-editor/HistoryPane.test.tsx` created
- [ ] React locale keys added (see §7.3)
- [ ] `AsidePanel.vue:57` swapped: `<HistoryPane v-if="asidePanelTab === 'HISTORY'" />` → `<ReactPageMount v-if="asidePanelTab === 'HISTORY'" page="HistoryPane" />`
- [ ] `AsidePanel.vue:82` import updated: remove `import HistoryPane from "./HistoryPane";`
- [ ] Vue directory `frontend/src/views/sql-editor/AsidePanel/HistoryPane/` deleted after `rg` confirms zero remaining callers (sub-file `HistoryConnectionIcon.vue` usage audit needed)

### Orphan cleanup

- [ ] Delete `frontend/src/views/sql-editor/EditorCommon/ResultView/ErrorView/AdviceItem.vue` (confirmed zero callers — dead code)

## 7. HistoryPane React migration — full feature parity

### 7.1 New React file

`frontend/src/react/components/sql-editor/HistoryPane.tsx`

**Props:** zero. Mounted via `<ReactPageMount page="HistoryPane" />` from `AsidePanel.vue`.

**Store reads (via `useVueState`):**
- `useSQLEditorTabStore()` → `currentTab?.connection.database`
- `useSQLEditorStore()` → `project`
- `useSQLEditorQueryHistoryStore()` → `getQueryHistoryList(filter)`, `fetchQueryHistoryList(filter)`, `resetPageToken(filter)`

**State:**
- `searchText: string` — user's search input
- `loading: boolean` — fetch-in-progress flag

**Effects:**
- Initial fetch + refetch when historyQuery changes (i.e., when tab/database/project/searchText changes)
- Debounced search (300ms via setTimeout in a ref)

**Feature parity points:**

| Vue feature | React mapping |
|---|---|
| `SearchBox` from v2 | `SearchInput` from `@/react/components/ui/search-input` |
| `CopyButton` from v2 | Inline button calling `navigator.clipboard.writeText(history.statement)` + push "Copied" notification (pattern established in Stage 7 SharePopoverBody) |
| `MaskSpinner` loading overlay | `Loader2` from `lucide-react` with `animate-spin` (established React loader pattern, used in `IssueTable.tsx`, etc.) |
| `v-html` + `getHighlightHTMLByKeyWords` | `dangerouslySetInnerHTML={{ __html: getHighlightHTMLByKeyWords(statement, searchText) }}` — the utility is already exported from `@/utils/util.ts` and returns pre-escaped HTML |
| `NButton quaternary size="small"` load-more | shadcn `Button variant="ghost" size="sm"` |
| `useDebounceFn` (vueuse) | `useEffect` with `setTimeout` + ref, mirroring the Stage 1 debounce pattern |
| `dayjs` timestamp formatting | Same `dayjs` — already imported elsewhere in React |
| `pushNotification` on copy | Same — imported from `@/store` in React |
| Empty state: "No history found" | Same |
| Click history row → emit `append-editor-content` | `sqlEditorEvents.emit("append-editor-content", { content, select: true })` |
| If no current tab: `tabStore.addTab(...)` | Same — direct Pinia call |

**Styling parity:**
- Outer: `relative w-full h-full flex flex-col justify-start items-start`
- Search row: `w-full px-1`
- List container: `w-full flex flex-col justify-start items-start overflow-y-auto`
- Each row: `w-full p-2 gap-y-1 border-b flex flex-col ... cursor-pointer hover:bg-gray-50` — or `hover:bg-control-bg` to use semantic tokens
- Timestamp span: `text-xs text-gray-500` — or `text-control-placeholder` for semantic parity
- Statement preview: `max-w-full text-xs wrap-break-word font-mono line-clamp-3`
- Load-more button centered in `w-full flex flex-col items-center my-2`
- Loading overlay: full-cover flex-center with spinner
- Empty state: `w-full flex items-center justify-center py-8 textinfolabel`

### 7.2 Test file

`HistoryPane.test.tsx` — 5 tests:
1. Renders with no history → empty state "No history found"
2. Renders history list with 2 entries → both timestamps and statement previews visible
3. Typing in search → debounced store call with updated filter
4. Click history row → `sqlEditorEvents.emit("append-editor-content", { content, select: true })` called (mock the singleton)
5. Click copy button → `navigator.clipboard.writeText(history.statement)` called + notification pushed

Use the established mock pattern: `vi.hoisted` for store factory stubs, mock the Emittery singleton via `vi.mock("@/views/sql-editor/events", ...)`, mock the icons/primitives as pass-throughs where needed.

### 7.3 i18n keys

Verify these in all 5 React locales; add missing using Vue values byte-exact:
- `sql-editor.search-history-by-statement`
- `sql-editor.no-history-found`
- `common.load-more`
- `common.copy` (for the copy-button aria-label) — verify existence

### 7.4 Caller swap

`AsidePanel.vue:57`:
```vue
<!-- Before -->
<HistoryPane v-if="asidePanelTab === 'HISTORY'" />

<!-- After -->
<ReactPageMount
  v-if="asidePanelTab === 'HISTORY'"
  page="HistoryPane"
/>
```

Line 82: remove `import HistoryPane from "./HistoryPane";`. (`ReactPageMount` already imported in Stage 3.)

### 7.5 Vue file deletion

`frontend/src/views/sql-editor/AsidePanel/HistoryPane/` directory — check contents. The spec in Stage 3 already noted this directory has additional sub-files beyond `HistoryPane.vue`:
- `HistoryPane.vue` — target
- `HistoryConnectionIcon.vue` (27 lines) — small helper icon used inside HistoryPane

If `HistoryConnectionIcon.vue` is only used by `HistoryPane.vue`, the whole directory gets deleted. If it's used elsewhere, only `HistoryPane.vue` + its index file get deleted; `HistoryConnectionIcon.vue` retained.

Verify via `rg` before deletion.

**Note:** Checking the Vue `HistoryPane.vue` source, I don't see a `HistoryConnectionIcon` import — it may be dead code in the same directory, separately cleanup-able. Implementer will verify and handle during migration.

## 8. AdviceItem cleanup (orphan deletion)

`frontend/src/views/sql-editor/EditorCommon/ResultView/ErrorView/AdviceItem.vue` (92 lines) has **zero callers** anywhere in the codebase. Grep confirmed no file imports or references it.

Simple cleanup:
- Verify zero callers one more time during implementation via `rg "AdviceItem"`
- Delete the file

No React migration needed — the file is dead code.

## 9. Deferred (explicitly future stages)

- **SaveSheetModal React migration with full parity** — requires `FolderForm` + tree primitive + `useSheetContextByView` inject bridge as prerequisites. Not in scope until those are ready.
- **Tree primitive + `FolderForm` port** — separate stage. Prerequisites: decide on a virtualized tree library (react-arborist, react-window, react-virtuoso) and port `TreeNodePrefix` + `useSheetContextByView` bridge.
- **AI plugin context bridge** — needed for future OpenAIButton migration.

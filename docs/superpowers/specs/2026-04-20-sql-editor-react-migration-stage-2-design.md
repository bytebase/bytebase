# SQL Editor React Migration — Stage 2 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue` (33 lines) to React, and establish the `useSQLEditorContext` → Pinia bridge pattern by lifting the context's 6 UI state refs + `editorPanelSize` computed + `handleEditorPanelResize` action into a new `useSQLEditorUIStore`. Vue consumers keep the same `useSQLEditorContext()` API; React leaves use the Pinia store directly.

**Non-goals (Stage 2):**
- Migrating any `EditorPanel/Panels/*` panel, Monaco, AsidePanel, or TabList.
- Moving the `Emittery` event bus (`events`) out of `context.ts`.
- Moving the async actions (`maybeSwitchProject`, `createWorksheet`, `maybeUpdateWorksheet`, `abortAutoSave`) or the auto-save `watchDebounced` composable — they stay in `provideSQLEditorContext`.
- Changing any of the ~30 Vue files that call `useSQLEditorContext()`.

## 2. Architecture

Create `frontend/src/store/modules/sqlEditor/uiState.ts` exporting `useSQLEditorUIStore` — a Pinia setup-store holding the 6 UI state refs, the `editorPanelSize` computed, and the `handleEditorPanelResize` action. Refactor `frontend/src/views/sql-editor/context.ts` so `provideSQLEditorContext`'s returned `SQLEditorContext` object gets those 7 fields from the store instead of constructing them inline — the shape Vue consumers see stays identical. React leaves skip the inject entirely and call `useSQLEditorUIStore()` directly, subscribing to reactive fields via `useVueState` and writing via direct store mutations.

**Single source of truth:** one Pinia store instance per tab (SQL Editor mount). Vue's `provideSQLEditorContext` wraps the store call; React's `useSQLEditorUIStore()` resolves to the same singleton via Pinia's standard global instance. No state duplication.

**The `aiPanelSize` LocalStorage ref** (currently `useLocalStorage(STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE, 0.3)` inside `provideSQLEditorContext`) moves into the store. The store uses `useLocalStorage` internally to preserve the persistence key and default value — byte-for-byte compatible with existing user preferences.

## 3. The `useSQLEditorUIStore` shape

```ts
// frontend/src/store/modules/sqlEditor/uiState.ts
import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils";
import type { AsidePanelTab } from "@/views/sql-editor/context";

const minimumEditorPanelSize = 0.5;

export const useSQLEditorUIStore = defineStore("sqlEditorUI", () => {
  const asidePanelTab = ref<AsidePanelTab>("WORKSHEET");
  const showConnectionPanel = ref(false);
  const showAIPanel = ref(false);
  const schemaViewer = ref<
    | {
        schema?: string;
        object?: string;
        type?: GetSchemaStringRequest_ObjectType;
      }
    | undefined
  >(undefined);
  const pendingInsertAtCaret = ref<string | undefined>();
  const highlightAccessGrantName = ref<string | undefined>();

  const aiPanelSize = useLocalStorage(
    STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
    0.3
  );

  const editorPanelSize = computed(() => {
    if (!showAIPanel.value) return { size: 1, max: 1, min: 1 };
    return {
      size: Math.max(1 - aiPanelSize.value, minimumEditorPanelSize),
      max: 0.9,
      min: minimumEditorPanelSize,
    };
  });

  const handleEditorPanelResize = (size: number) => {
    if (size >= 1) return;
    aiPanelSize.value = 1 - size;
  };

  return {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    editorPanelSize,
    handleEditorPanelResize,
  };
});
```

**`AsidePanelTab` type:** stays exported from `context.ts` (it's a shared alias). The store imports it from there; the consumer surface doesn't change.

**`STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE`:** already exported from `@/utils`. Same key → same user preference.

**Setup-store vs. options-store:** setup-style matches the project's existing SQL editor stores (e.g. `useSQLEditorStore` in `modules/sqlEditor/editor.ts`, `useSQLEditorTabStore` in `modules/sqlEditor/tab.ts`) and supports `ref` / `computed` composition cleanly. Must return all exposed fields (Pinia requirement).

**Not in the store:** `events` (Emittery), the async action methods, the auto-save `watchDebounced` — all stay in `provideSQLEditorContext` per §1.

## 4. `useSQLEditorContext` wrapper — preserving Vue API

In `frontend/src/views/sql-editor/context.ts`, `provideSQLEditorContext()` currently constructs all 12 fields of the `SQLEditorContext` object inline. The refactor: delegate the 7 state/action fields to the store, keep the remaining 5 inline.

```ts
// frontend/src/views/sql-editor/context.ts  (relevant diff only)
import { storeToRefs } from "pinia";
import { useSQLEditorUIStore } from "@/store";
// ... other imports unchanged ...

export const provideSQLEditorContext = () => {
  const editorStore = useSQLEditorStore();
  const tabStore = useSQLEditorTabStore();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const worksheetStore = useWorkSheetStore();
  const uiStore = useSQLEditorUIStore();

  // Unchanged: async actions, auto-save watcher, Emittery events
  const maybeSwitchProject = async (projectName: string) => {
    /* unchanged body */
  };
  const maybeUpdateWorksheet = async (/* unchanged */) => {
    /* unchanged body */
  };
  const createWorksheet = async (/* unchanged */) => {
    /* unchanged body */
  };
  let autoSaveController: AbortController | null = null;
  const abortAutoSave = () => {
    /* unchanged body */
  };

  // storeToRefs preserves reactivity when destructuring Pinia setup-stores
  const {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    editorPanelSize,
  } = storeToRefs(uiStore);

  const context: SQLEditorContext = {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    editorPanelSize,
    handleEditorPanelResize: uiStore.handleEditorPanelResize,
    events: new Emittery(),
    maybeSwitchProject,
    createWorksheet,
    maybeUpdateWorksheet,
    abortAutoSave,
  };

  // Auto-save watchDebounced — unchanged, still in Vue setup phase
  watchDebounced(
    () => currentTab.value?.statement,
    async () => {
      /* unchanged body */
    },
    { debounce: 2000 }
  );

  provide(KEY, context);
  return context;
};
```

**Key invariants:**

1. **The returned `SQLEditorContext` shape is unchanged** — same 12 fields, same types. The 30 Vue consumer files keep working without edits.
2. **`storeToRefs` preserves reactivity.** Destructuring a Pinia setup-store directly loses reactivity; `storeToRefs` keeps the fields as `Ref<T>`, matching the existing `Ref<boolean>` etc. signatures in `SQLEditorContext`.
3. **The local `aiPanelSize` LocalStorage binding moves with `editorPanelSize` into the store.** The old inline `useLocalStorage` call in `context.ts` is removed — it's now in the store's setup function and initializes lazily on first `useSQLEditorUIStore()` call.
4. **Pinia store export.** Add `export * from "./uiState"` to `frontend/src/store/modules/sqlEditor/index.ts` following the sibling files' export pattern. The root `store/index.ts` already re-exports everything under `modules/` so no further barrel edits are needed.

## 5. ConnectionHolder migration

### 5.1 New React files

| File | Purpose |
|---|---|
| `frontend/src/react/components/sql-editor/ConnectionHolder.tsx` | React leaf replacing the Vue `ConnectionHolder.vue`. Renders a single button (shadcn `Button` variant matching the Vue ghost-primary visual) with a `LinkIcon` and the `sql-editor.connect-to-a-database` label (already in React locales from Stage 1). On click: `useSQLEditorUIStore().showConnectionPanel = true`. |
| `frontend/src/react/components/sql-editor/ConnectionHolder.test.tsx` | Unit tests: renders button with correct label + icon; click sets `showConnectionPanel` via the store. |

### 5.2 Data flow

| What ConnectionHolder needs | Source | How React gets it |
|---|---|---|
| i18n label `sql-editor.connect-to-a-database` | already in React locales (Stage 1) | `useTranslation()` |
| `LinkIcon` | `lucide-react` | direct import |
| Open connection panel | `showConnectionPanel.value = true` | `useSQLEditorUIStore().showConnectionPanel = true` — direct Pinia write, NO prop callback needed. This is the whole point of Stage 2's bridge. |

**No `onConnect` prop** — the React leaf reaches directly into the Pinia store. This is the pattern every future `useSQLEditorContext`-consuming React leaf will use.

### 5.3 Vue caller modified

`frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue` — one call site at line 75 (`<ConnectionHolder v-else />`) with the import at line 98 (`import { ConnectionHolder, ... } from "../../EditorCommon"`).

Swap pattern mirrors Stage 1's `StandardPanel.vue` swap:

```vue
<!-- template -->
<div v-else class="flex-1 flex flex-col min-h-0">
  <ReactPageMount page="ConnectionHolder" />
</div>

<!-- script -->
import ReactPageMount from "@/react/ReactPageMount.vue";
// Remove ConnectionHolder from the "../../EditorCommon" named import, keeping
// the other named imports on that line.
```

The `flex-1 flex flex-col min-h-0` wrapper div comes from the Stage 1 learning — inline `ReactPageMount` needs a flex-grow wrapper when the Vue original used `h-full` inside a flex column parent. ConnectionHolder's Vue original uses `h-full` so the same wrapper applies.

### 5.4 Vue file deletion

After the swap, verify with `rg ConnectionHolder` scoped to `frontend/src`. Expected remaining references after the swap: only `EditorCommon/index.ts` (the barrel re-export) and the file itself. Remove the `ConnectionHolder` entry from `EditorCommon/index.ts`'s named exports and delete `EditorCommon/ConnectionHolder.vue`.

### 5.5 Mount infrastructure

No new infrastructure — `frontend/src/react/mount.ts`'s `./components/sql-editor/*.tsx` glob from Stage 1 Task 5 already picks up `ConnectionHolder.tsx`.

## 6. Verification & UX parity

### 6.1 Per-leaf verification (run before Stage 2 PR)

- `pnpm --dir frontend fix` — auto-format + lint
- `pnpm --dir frontend check` — CI-equivalent (ESLint + Biome + React i18n consistency)
- `pnpm --dir frontend type-check` — Vue + React. No new errors beyond the 6 pre-existing `SchemaEditorLite` ones.
- `pnpm --dir frontend test` — existing 1152 tests pass, plus new tests:
  - `sqlEditorUI.test.ts` — store smoke test: initial values, `handleEditorPanelResize` behavior, `editorPanelSize` computed transitions when `showAIPanel` toggles.
  - `ConnectionHolder.test.tsx` — renders button with correct label and icon; click sets `showConnectionPanel` via the store.

### 6.2 UX parity verification (manual, dev server)

ConnectionHolder renders inside the admin-mode Terminal panel as the v-else fallback — when the terminal panel has no connection.

1. Open SQL Editor, switch current tab to Admin mode, and disconnect → React ConnectionHolder should render: a primary ghost button with a LinkIcon and label "Connect to database".
2. Button click → connection panel opens — exactly as the Vue original.
3. Locale switch (English ↔ Chinese) → label updates.
4. Hover, focus ring, active style should match the Vue naive-ui `type="primary" ghost` visual. Take side-by-side screenshots vs. the pre-change Vue version.

### 6.3 Bridge integration verification (the critical check for Stage 2)

Vue consumers of `useSQLEditorContext()` must still work after the refactor. Specifically verify these read+write flows still function after the lift:

- Opening the connection panel from any Vue caller (e.g. `Welcome`'s `changeConnection` in `StandardPanel.vue`, which sets `showConnectionPanel.value = true` + `asidePanelTab.value = "SCHEMA"`) still opens the panel and switches tabs.
- AI panel toggle (via the OpenAI button) still animates `editorPanelSize` and persists `aiPanelSize` to LocalStorage.
- Aside tab switching (Worksheet ↔ Schema ↔ History ↔ Access) still persists via `asidePanelTab`.
- Schema viewer (when a user clicks a table/view) still opens via `schemaViewer`.

If any of these breaks, the lift broke reactivity — likely a `storeToRefs` miss. Fix before proceeding.

## 7. Out of scope for Stage 2 (explicit deferrals)

- **Migrating the `events` Emittery bus to React consumers.** First needed when a React leaf emits or listens to an editor event — none yet.
- **Async action bridge** (`maybeSwitchProject`, `createWorksheet`, `maybeUpdateWorksheet`, `abortAutoSave`). These stay Vue-side until a React leaf needs them.
- **Auto-save composable move.** The `watchDebounced` stays in `provideSQLEditorContext` because it needs Vue setup phase.
- **Full-lift C3 refactor.** Per-consumer Vue edits come when/if the remaining context fields need bridging.
- **Vue-in-React mount infrastructure.** Deferred until a panel migration actually requires it.

## 8. Future stage sketch (informational only — each gets its own brainstorm + spec + plan)

| Stage | Scope | Why next |
|---|---|---|
| **1 (DONE)** | `Welcome` leaf | Smallest leaf; established React-in-Vue mount + Pinia-from-React + permission-check patterns. |
| **2 (this spec)** | `ConnectionHolder` leaf + `useSQLEditorUIStore` bridge | Establishes the Pinia bridge for `useSQLEditorContext` UI state; sized to a 33-line write-only consumer. |
| **3** | `AsidePanel/GutterBar/TabItem.vue` (57 lines) + `GutterBar.vue` (79 lines) | First real READ consumer of `useSQLEditorUIStore().asidePanelTab` from React (reactive subscription, active-state rendering). |
| **4** | `EditorCommon/DatabaseChooser.vue`, `SaveSheetModal.vue`, similar small `EditorCommon/*` leaves | Continues bridge exercise; each is a single responsibility. |
| **5** | Small panels under `EditorPanel/Panels/*` (e.g. `TablesPanel.vue` + its `TablesTable.vue`) | Forces the Vue-in-React or cascade-migrate question to be answered. |
| **6** | Remaining panels, `AsidePanel/*`, `ConnectionPanel/*`, `TabList/*`, `EditorCommon/ResultView/*` | Bulk middle of the migration. |
| **7** | Monaco wrapper | Behind a stable integration seam per playbook. |
| **8** | `SQLEditorPage` shell flip | React owns the route. |
| **9** | Delete dead Vue. | After `rg` confirms zero callers. |

## 9. Practical checklist for Stage 2

- [ ] `frontend/src/store/modules/sqlEditor/uiState.ts` created with `useSQLEditorUIStore` + test.
- [ ] Store re-exported from `frontend/src/store/modules/sqlEditor/index.ts` (add `export * from "./uiState"` alongside existing sibling exports).
- [ ] `frontend/src/views/sql-editor/context.ts` refactored — 7 fields delegated to the store via `storeToRefs`; rest unchanged.
- [ ] `frontend/src/react/components/sql-editor/ConnectionHolder.tsx` + test created.
- [ ] `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue` swapped to `<ReactPageMount page="ConnectionHolder" />` inside a `flex-1 flex flex-col min-h-0` wrapper; `ConnectionHolder` removed from the `"../../EditorCommon"` named import.
- [ ] `frontend/src/views/sql-editor/EditorCommon/index.ts` — `ConnectionHolder` export removed.
- [ ] `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue` deleted after `rg ConnectionHolder` confirms zero remaining callers.
- [ ] `pnpm --dir frontend fix && check && type-check && test` all pass.
- [ ] Manual UX parity verified: admin terminal no-connection state, click opens connection panel, locale switch, visual style matches ghost-primary naive button.
- [ ] Bridge integration verified: all existing Vue flows using `showConnectionPanel`, `asidePanelTab`, `showAIPanel`, `editorPanelSize`, `schemaViewer` still work.

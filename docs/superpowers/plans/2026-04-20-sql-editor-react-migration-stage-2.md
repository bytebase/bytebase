# SQL Editor React Migration — Stage 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue` (33 lines) to React and establish the `useSQLEditorContext` → Pinia bridge by extracting 6 UI state refs + `editorPanelSize` computed + `handleEditorPanelResize` setter into a new `useSQLEditorUIStore`. Vue's `useSQLEditorContext()` API shape stays unchanged — React leaves read/write the Pinia store directly.

**Architecture:** A new Pinia setup-store (`modules/sqlEditor/uiState.ts`) owns the 7 fields; `provideSQLEditorContext` delegates to it via `storeToRefs` while keeping its existing return shape; React `ConnectionHolder.tsx` writes `showConnectionPanel` directly through the store with no prop callback.

**Tech Stack:** Pinia setup-stores, Vue 3 `storeToRefs`, `@vueuse/core` `useLocalStorage`, React 18, `@base-ui/react`, shadcn `Button`, `lucide-react`, vitest.

**Reference spec:** `docs/superpowers/specs/2026-04-20-sql-editor-react-migration-stage-2-design.md`

**Workflow note:** **Do not auto-commit** — user commits manually. Each task ends with "Stop for user review."

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/store/modules/sqlEditor/uiState.ts` | Create | New `useSQLEditorUIStore` Pinia setup-store: 6 reactive UI state refs + `editorPanelSize` computed + `handleEditorPanelResize` setter + internal `aiPanelSize` LocalStorage ref. |
| `frontend/src/store/modules/sqlEditor/uiState.test.ts` | Create | Store smoke tests: initial values, `handleEditorPanelResize` behavior, `editorPanelSize` computed transitions when `showAIPanel` toggles. |
| `frontend/src/store/modules/sqlEditor/index.ts` | Modify | Append `export * from "./uiState"` to match sibling export pattern. |
| `frontend/src/views/sql-editor/context.ts` | Modify | Refactor `provideSQLEditorContext`: delegate 7 fields to `useSQLEditorUIStore` via `storeToRefs`, remove inline `aiPanelSize` / `editorPanelSize` / `handleEditorPanelResize` / `showConnectionPanel` / `showAIPanel` / `schemaViewer` / `pendingInsertAtCaret` / `highlightAccessGrantName` / `asidePanelTab` definitions. `SQLEditorContext` return shape unchanged. |
| `frontend/src/react/components/sql-editor/ConnectionHolder.tsx` | Create | React leaf replacing the Vue `ConnectionHolder.vue`. Primary ghost button with `LinkIcon` + `sql-editor.connect-to-a-database` label. Click → `useSQLEditorUIStore().showConnectionPanel = true`. |
| `frontend/src/react/components/sql-editor/ConnectionHolder.test.tsx` | Create | Unit tests: renders button with expected label + icon; click invokes the store setter. |
| `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue` | Modify | Line 75: replace `<ConnectionHolder v-else />` with `<div v-else class="flex-1 flex flex-col min-h-0"><ReactPageMount page="ConnectionHolder" /></div>`. Line 97-101: remove `ConnectionHolder` from the `"../../EditorCommon"` named import; add `import ReactPageMount from "@/react/ReactPageMount.vue"`. |
| `frontend/src/views/sql-editor/EditorCommon/index.ts` | Modify | Remove `import ConnectionHolder from "./ConnectionHolder.vue"` (line 1) and the `ConnectionHolder,` entry in the `export` block (line 12). |
| `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue` | Delete | Orphaned after the swap and barrel removal. |

---

## Task 1: Create `useSQLEditorUIStore` (TDD)

**Goal:** New Pinia setup-store owning the 7 fields that React leaves need to bridge. Test-first to lock the public surface.

**Files:**
- Create: `frontend/src/store/modules/sqlEditor/uiState.ts`
- Create: `frontend/src/store/modules/sqlEditor/uiState.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/store/modules/sqlEditor/uiState.test.ts`:

```ts
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test } from "vitest";

let useSQLEditorUIStore: typeof import("./uiState").useSQLEditorUIStore;

beforeEach(async () => {
  // Fresh pinia + fresh module each test so LocalStorage reactivity doesn't leak.
  localStorage.clear();
  setActivePinia(createPinia());
  ({ useSQLEditorUIStore } = await import("./uiState"));
});

describe("useSQLEditorUIStore", () => {
  test("initial state has documented defaults", () => {
    const store = useSQLEditorUIStore();
    expect(store.asidePanelTab).toBe("WORKSHEET");
    expect(store.showConnectionPanel).toBe(false);
    expect(store.showAIPanel).toBe(false);
    expect(store.schemaViewer).toBeUndefined();
    expect(store.pendingInsertAtCaret).toBeUndefined();
    expect(store.highlightAccessGrantName).toBeUndefined();
  });

  test("editorPanelSize returns full-width when AI panel is hidden", () => {
    const store = useSQLEditorUIStore();
    expect(store.showAIPanel).toBe(false);
    expect(store.editorPanelSize).toEqual({ size: 1, max: 1, min: 1 });
  });

  test("editorPanelSize returns clamped split when AI panel is shown", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    // Default aiPanelSize LocalStorage value is 0.3 → editor gets 1 - 0.3 = 0.7
    // which is above the 0.5 minimum, so 0.7 is used.
    expect(store.editorPanelSize).toEqual({ size: 0.7, max: 0.9, min: 0.5 });
  });

  test("editorPanelSize clamps to minimum 0.5 when AI panel is huge", () => {
    // Pre-seed LocalStorage with an oversized AI panel so the editor would be < 0.5.
    localStorage.setItem(
      "bb.sql-editor.ai-panel-size",
      JSON.stringify(0.8)
    );
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    // 1 - 0.8 = 0.2, clamped up to the 0.5 minimum.
    expect(store.editorPanelSize.size).toBe(0.5);
  });

  test("handleEditorPanelResize writes complement to aiPanelSize LocalStorage", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    store.handleEditorPanelResize(0.6);
    // editor=0.6 → aiPanelSize=0.4 → editor size = max(1 - 0.4, 0.5) = 0.6
    expect(store.editorPanelSize.size).toBe(0.6);
  });

  test("handleEditorPanelResize no-ops when size is >= 1", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    store.handleEditorPanelResize(1);
    // aiPanelSize stays at 0.3 default → editor size = 0.7
    expect(store.editorPanelSize.size).toBe(0.7);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
pnpm --dir frontend test -- uiState.test --run
```

Expected: FAIL with `Cannot find module './uiState'`.

- [ ] **Step 3: Verify the LocalStorage key value**

The test uses the string `"bb.sql-editor.ai-panel-size"` for one seed. Before writing the store, confirm that matches `STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE`:

Run:
```bash
grep -n "STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE" /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/utils/*.ts
```

Expected output identifies the constant's value. If the actual key string differs from `"bb.sql-editor.ai-panel-size"`, update the test's `localStorage.setItem(...)` call to use the actual value.

- [ ] **Step 4: Write the implementation**

Create `frontend/src/store/modules/sqlEditor/uiState.ts`:

```ts
import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils";
import type { AsidePanelTab } from "@/views/sql-editor/context";

const minimumEditorPanelSize = 0.5;

/**
 * UI state for the SQL Editor shell.
 *
 * Extracted from `useSQLEditorContext` so React leaves can access the same
 * reactive state that Vue consumers read via inject. Vue's
 * `useSQLEditorContext()` wraps this store via `storeToRefs` and preserves
 * its existing API shape.
 */
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
    if (!showAIPanel.value) {
      return { size: 1, max: 1, min: 1 };
    }
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

- [ ] **Step 5: Run the test to verify it passes**

Run:
```bash
pnpm --dir frontend test -- uiState.test --run
```

Expected: PASS (6 tests).

If the tests fail with "Cannot set property asidePanelTab... (setter only)" or similar, the test is accessing a `storeToRefs` wrapped object. The direct `store.xxx = value` pattern works against a raw Pinia store instance — which is what `useSQLEditorUIStore()` returns. Confirm the test file calls the store directly (not `storeToRefs`).

- [ ] **Step 6: Stop for user review**

Report: "`useSQLEditorUIStore` store + 6 tests created and passing. Ready for commit."

---

## Task 2: Export the store from the sqlEditor barrel

**Goal:** Make `useSQLEditorUIStore` reachable via `@/store` so both Vue `context.ts` and React `ConnectionHolder.tsx` can import it with a clean path.

**Files:**
- Modify: `frontend/src/store/modules/sqlEditor/index.ts`

- [ ] **Step 1: Read current state**

Current file content (5 lines):

```ts
export * from "./editor";
export * from "./tab";
export * from "./tree";
export * from "./queryHistory";
export * from "./webTerminal";
```

- [ ] **Step 2: Add the uiState export**

Change to:

```ts
export * from "./editor";
export * from "./tab";
export * from "./tree";
export * from "./queryHistory";
export * from "./uiState";
export * from "./webTerminal";
```

(Alphabetical ordering; `uiState` sits between `queryHistory` and `webTerminal`.)

- [ ] **Step 3: Verify the import path resolves**

Run:
```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. No new "cannot find export" errors.

- [ ] **Step 4: Stop for user review**

Report: "Store barrel updated. Ready for commit."

---

## Task 3: Refactor `context.ts` to delegate to the store

**Goal:** Replace the inline UI-state construction in `provideSQLEditorContext` with delegation to `useSQLEditorUIStore` via `storeToRefs`, preserving the `SQLEditorContext` shape so the ~30 Vue consumers keep working unchanged.

**File:**
- Modify: `frontend/src/views/sql-editor/context.ts`

- [ ] **Step 1: Read the current file end-to-end**

Open and read `frontend/src/views/sql-editor/context.ts`. Identify the three blocks to modify:

1. **Imports** (lines 1-32): `useLocalStorage` is imported; `STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE` is imported. Both become unused after the refactor. `storeToRefs` from `pinia` needs to be added; `useSQLEditorUIStore` from `@/store` needs to be added.
2. **`provideSQLEditorContext` top** (around lines 122-148): the `showConnectionPanel = ref(false)`, `aiPanelSize = useLocalStorage(...)`, `showAIPanel = ref(false)`, `editorPanelSize = computed(...)` blocks and the `minimumEditorPanelSize` constant all move away.
3. **`context` object construction** (around lines 304-324): the 7 inlined fields (`asidePanelTab: ref("WORKSHEET")`, `showConnectionPanel`, `showAIPanel`, `editorPanelSize`, `schemaViewer: ref(undefined)`, `pendingInsertAtCaret: ref()`, `highlightAccessGrantName: ref<string | undefined>()`, `handleEditorPanelResize: (size) => { ... }`) are replaced with the `storeToRefs`-destructured fields and `uiStore.handleEditorPanelResize`.

- [ ] **Step 2: Update the imports**

Change:
```ts
import { create } from "@bufbuild/protobuf";
import { useLocalStorage, watchDebounced } from "@vueuse/core";
import Emittery from "emittery";
import { isUndefined } from "lodash-es";
```

To:
```ts
import { create } from "@bufbuild/protobuf";
import { watchDebounced } from "@vueuse/core";
import Emittery from "emittery";
import { isUndefined } from "lodash-es";
import { storeToRefs } from "pinia";
```

Remove `useLocalStorage` from the `@vueuse/core` import; add the `storeToRefs` import from `pinia`.

Find the existing `@/store` import:
```ts
import {
  pushNotification,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
```

Add `useSQLEditorUIStore` alphabetically:
```ts
import {
  pushNotification,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
  useWorkSheetStore,
} from "@/store";
```

Find the `@/utils` import. Remove `STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE` from it (the store now owns the LocalStorage key). Keep every other named import on that line. If `STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE` was the only import, remove the entire import line — but check first: the current line in `context.ts` is:

```ts
import {
  extractWorksheetConnection,
  isSimilarDefaultSQLEditorTabTitle,
  isWorksheetWritableV1,
  NEW_WORKSHEET_TITLE,
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
```

Remove the `STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,` line; leave the others.

- [ ] **Step 3: Remove the top-of-function UI-state construction**

Find the block starting at `export const provideSQLEditorContext = () => {`. Remove:

```ts
  const showConnectionPanel = ref(false);

  const aiPanelSize = useLocalStorage(
    STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
    0.3 /* panel size should in [0.1, 1-minimumEditorPanelSize]*/
  );
  const showAIPanel = ref(false);
  const editorPanelSize = computed(() => {
    if (!showAIPanel.value) {
      return {
        size: 1,
        max: 1,
        min: 1,
      };
    }
    return {
      size: Math.max(1 - aiPanelSize.value, minimumEditorPanelSize),
      max: 0.9,
      min: minimumEditorPanelSize,
    };
  });
```

Also remove the module-scope `const minimumEditorPanelSize = 0.5;` declaration (currently above `provideSQLEditorContext`) — it's now a constant inside the store.

- [ ] **Step 4: Add store destructure near the other store calls**

At the top of `provideSQLEditorContext` where the existing `const editorStore = useSQLEditorStore();` etc. appear, add:

```ts
  const uiStore = useSQLEditorUIStore();
  const {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    editorPanelSize,
  } = storeToRefs(uiStore);
```

Place it right after the last existing store call (e.g. after `const worksheetStore = useWorkSheetStore();`).

- [ ] **Step 5: Replace the `context` object fields**

Find the `const context: SQLEditorContext = { ... };` block (around lines 304-324). The current shape:

```ts
  const context: SQLEditorContext = {
    asidePanelTab: ref("WORKSHEET"),
    showConnectionPanel,
    showAIPanel,
    editorPanelSize,
    schemaViewer: ref(undefined),
    pendingInsertAtCaret: ref(),
    highlightAccessGrantName: ref<string | undefined>(),
    events: new Emittery(),

    maybeSwitchProject,
    handleEditorPanelResize: (size: number) => {
      if (size >= 1) {
        return;
      }
      aiPanelSize.value = 1 - size;
    },
    createWorksheet,
    maybeUpdateWorksheet,
    abortAutoSave,
  };
```

Replace with:

```ts
  const context: SQLEditorContext = {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    editorPanelSize,
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    events: new Emittery(),

    maybeSwitchProject,
    handleEditorPanelResize: uiStore.handleEditorPanelResize,
    createWorksheet,
    maybeUpdateWorksheet,
    abortAutoSave,
  };
```

Note: `asidePanelTab`, `showConnectionPanel`, `showAIPanel`, `editorPanelSize`, `schemaViewer`, `pendingInsertAtCaret`, `highlightAccessGrantName` are all the `Ref<T>` values from `storeToRefs` — they match the existing `Ref<T>` / `ComputedRef<T>` types in `SQLEditorContext`.

- [ ] **Step 6: Verify unused imports are gone**

Run:
```bash
pnpm --dir frontend exec eslint src/views/sql-editor/context.ts 2>&1 | tail -15
```

Expected: no errors. If ESLint flags `ref` or `computed` as unused imports (because their uses moved into the store), remove those from the `vue` import list — but only remove `ref` / `computed` if they're genuinely no longer used in `context.ts` (they might still be used elsewhere in the file for non-UI refs). Grep within the file first to verify:

```bash
grep -n "\bref\b\|\bcomputed\b" /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/views/sql-editor/context.ts
```

- [ ] **Step 7: Type-check**

Run:
```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. Zero new errors. If the `SQLEditorContext` type mismatch shows up (e.g. `Ref<AsidePanelTab>` expected but received `WritableComputedRef` or similar), it likely means `storeToRefs` wasn't applied correctly — re-check Step 4.

- [ ] **Step 8: Run the full test suite**

Run:
```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: 1152+ tests pass; no regressions from the context refactor.

- [ ] **Step 9: Stop for user review**

Report: "`context.ts` refactored — 7 fields delegated to `useSQLEditorUIStore`, `SQLEditorContext` shape unchanged, type-check clean, tests pass. Ready for commit."

---

## Task 4: Create React `ConnectionHolder.tsx` (TDD)

**Goal:** React leaf that renders a primary ghost button (visual parity with Vue naive-ui `NButton type="primary" ghost`) + `LinkIcon` + the migrated `sql-editor.connect-to-a-database` label. Click reaches directly into `useSQLEditorUIStore().showConnectionPanel`.

**Files:**
- Create: `frontend/src/react/components/sql-editor/ConnectionHolder.tsx`
- Create: `frontend/src/react/components/sql-editor/ConnectionHolder.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/components/sql-editor/ConnectionHolder.test.tsx`:

```tsx
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useSQLEditorUIStore: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/store", () => ({
  useSQLEditorUIStore: mocks.useSQLEditorUIStore,
}));

let ConnectionHolder: typeof import("./ConnectionHolder").ConnectionHolder;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ ConnectionHolder } = await import("./ConnectionHolder"));
});

describe("ConnectionHolder", () => {
  test("renders the Connect-to-database label", () => {
    const store = { showConnectionPanel: false };
    mocks.useSQLEditorUIStore.mockReturnValue(store);
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionHolder />
    );
    render();
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    expect(container.querySelector("button")).not.toBeNull();
    unmount();
  });

  test("click sets showConnectionPanel to true on the store", () => {
    const store = { showConnectionPanel: false };
    mocks.useSQLEditorUIStore.mockReturnValue(store);
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionHolder />
    );
    render();
    const button = container.querySelector("button");
    act(() => {
      button?.click();
    });
    expect(store.showConnectionPanel).toBe(true);
    unmount();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
pnpm --dir frontend test -- ConnectionHolder.test --run
```

Expected: FAIL with `Cannot find module './ConnectionHolder'`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/react/components/sql-editor/ConnectionHolder.tsx`:

```tsx
import { LinkIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useSQLEditorUIStore } from "@/store";

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue.
 * Rendered as the v-else fallback inside the admin-mode Terminal panel when
 * there is no active database connection. Click opens the connection panel
 * via the SQL Editor UI store.
 */
export function ConnectionHolder() {
  const { t } = useTranslation();
  const uiStore = useSQLEditorUIStore();

  const handleClick = () => {
    uiStore.showConnectionPanel = true;
  };

  return (
    <div className="flex items-center justify-center w-full h-full">
      <Button variant="outline" onClick={handleClick}>
        <LinkIcon className="size-5" />
        {t("sql-editor.connect-to-a-database")}
      </Button>
    </div>
  );
}
```

Note on variant: the Vue original uses naive-ui `NButton type="primary" ghost` which renders as a bordered-accent button (accent border + text, transparent fill). The shadcn `outline` variant is the closest match in the project's current button surface (`border border-control-border bg-transparent text-control hover:bg-control-bg`). This is functionally equivalent — the exact hover/focus hue may differ slightly; confirm in the manual UX check (Task 7) and iterate if the difference is visible.

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
pnpm --dir frontend test -- ConnectionHolder.test --run
```

Expected: PASS (2 tests).

- [ ] **Step 5: Stop for user review**

Report: "`ConnectionHolder.tsx` + 2 tests created and passing. Ready for commit."

---

## Task 5: Swap `TerminalPanel.vue` to use `<ReactPageMount>`

**Goal:** Replace the Vue `<ConnectionHolder v-else />` with an inline-mounted React version at line 75 of `TerminalPanel.vue`. Remove the Vue `ConnectionHolder` named import; add the `ReactPageMount` import.

**File:**
- Modify: `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue`

- [ ] **Step 1: Read current state**

Open the file and confirm:
- Line 75: `<ConnectionHolder v-else />`
- Lines 97-101: 
```ts
import {
  ConnectionHolder,
  EditorAction,
  ResultViewV1,
} from "../../EditorCommon";
```

If line numbers differ slightly (whitespace changes), adjust but make the same semantic edits.

- [ ] **Step 2: Replace the template**

Change line 75 from:
```vue
<ConnectionHolder v-else />
```

To:
```vue
<div v-else class="flex-1 flex flex-col min-h-0">
  <ReactPageMount page="ConnectionHolder" />
</div>
```

(The `flex-1 flex flex-col min-h-0` wrapper comes from the Stage 1 learning — inline `ReactPageMount` needs a flex-grow wrapper so its `h-full` root has a height to fill.)

- [ ] **Step 3: Update the script imports**

Change:
```ts
import {
  ConnectionHolder,
  EditorAction,
  ResultViewV1,
} from "../../EditorCommon";
```

To:
```ts
import ReactPageMount from "@/react/ReactPageMount.vue";
import { EditorAction, ResultViewV1 } from "../../EditorCommon";
```

Keep the existing `@/react/ReactPageMount.vue` import placed in the `@/...` absolute-path group (Vue file convention: absolute paths before relative paths).

- [ ] **Step 4: Type-check**

Run:
```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. If you see `Cannot find name 'ConnectionHolder'` anywhere, there's a stray reference — search for it in the file and remove.

- [ ] **Step 5: Run the test suite**

Run:
```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: all tests pass.

- [ ] **Step 6: Stop for user review**

Report: "`TerminalPanel.vue` swapped to `<ReactPageMount page=\"ConnectionHolder\" />` with flex-wrapper; imports updated; type-check clean. Ready for commit."

---

## Task 6: Remove the Vue `ConnectionHolder` barrel export and delete the Vue file

**Goal:** The Vue file and its barrel entry are orphaned after Task 5's swap. Verify zero callers, then delete.

**Files:**
- Modify: `frontend/src/views/sql-editor/EditorCommon/index.ts`
- Delete: `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue`

- [ ] **Step 1: Search for remaining callers**

Run these three patterns from `/Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/`:

```bash
grep -rn "from.*EditorCommon/ConnectionHolder" frontend/src/
grep -rn "import.*ConnectionHolder.*from.*EditorCommon" frontend/src/
grep -rn "<ConnectionHolder" frontend/src/
```

**Expected:** Each pattern should return zero matches OUTSIDE the files being deleted (`EditorCommon/index.ts` and `EditorCommon/ConnectionHolder.vue` themselves). The React `ConnectionHolder.tsx` at `frontend/src/react/components/sql-editor/ConnectionHolder.tsx` is the replacement, not a caller — its own imports don't match any of these patterns.

If any match appears outside the files to delete, STOP and report BLOCKED with the caller details.

- [ ] **Step 2: Remove the barrel entry**

Open `frontend/src/views/sql-editor/EditorCommon/index.ts`. Current content:

```ts
import ConnectionHolder from "./ConnectionHolder.vue";
import DisconnectedIcon from "./DisconnectedIcon.vue";
import EditorAction from "./EditorAction.vue";
import ExecutingHintModal from "./ExecutingHintModal.vue";
import OpenAIButton from "./OpenAIButton/OpenAIButton.vue";
import { ResultViewV1 } from "./ResultView";
import SaveSheetModal from "./SaveSheetModal.vue";
import SharePopover from "./SharePopover.vue";
import SheetConnectionIcon from "./SheetConnectionIcon.vue";

export {
  ConnectionHolder,
  EditorAction,
  ExecutingHintModal,
  SaveSheetModal,
  SharePopover,
  ResultViewV1,
  DisconnectedIcon,
  SheetConnectionIcon,
  OpenAIButton,
};

export * from "./hover-state";
export * from "./utils";
```

Change to (two deletions — the import line and the export-block entry):

```ts
import DisconnectedIcon from "./DisconnectedIcon.vue";
import EditorAction from "./EditorAction.vue";
import ExecutingHintModal from "./ExecutingHintModal.vue";
import OpenAIButton from "./OpenAIButton/OpenAIButton.vue";
import { ResultViewV1 } from "./ResultView";
import SaveSheetModal from "./SaveSheetModal.vue";
import SharePopover from "./SharePopover.vue";
import SheetConnectionIcon from "./SheetConnectionIcon.vue";

export {
  EditorAction,
  ExecutingHintModal,
  SaveSheetModal,
  SharePopover,
  ResultViewV1,
  DisconnectedIcon,
  SheetConnectionIcon,
  OpenAIButton,
};

export * from "./hover-state";
export * from "./utils";
```

- [ ] **Step 3: Delete the Vue file**

Run:
```bash
rm /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue
```

- [ ] **Step 4: Verify**

Run:
```bash
ls /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue 2>&1
```

Expected: `No such file or directory`.

Then:
```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. If you see a `Cannot find module "./ConnectionHolder.vue"` error, an import somewhere wasn't updated — revisit Task 5 or Step 2 here.

- [ ] **Step 5: Run tests**

Run:
```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: 1152+ tests pass.

- [ ] **Step 6: Stop for user review**

Report: "Vue `ConnectionHolder.vue` deleted, barrel updated, type-check clean, tests pass. Ready for commit."

---

## Task 7: Final verification

**Goal:** Run the full frontend verification suite before the Stage 2 PR is ready. Capture the manual UX + bridge integration checks for the user to perform in the browser.

- [ ] **Step 1: Run auto-fix**

Run:
```bash
pnpm --dir frontend fix
```

Expected: no changes, or only trivial formatting adjustments to the new files. If it produces non-trivial changes, report the diff.

- [ ] **Step 2: Run check (CI-equivalent)**

Run:
```bash
pnpm --dir frontend check
```

Expected: pass. ESLint + Biome + React i18n consistency + i18n sort — all clean.

- [ ] **Step 3: Run type-check and confirm baseline-only errors**

Run:
```bash
pnpm --dir frontend type-check 2>&1 | tail -30
```

Expected: exactly the 6 pre-existing `SchemaEditorLite` errors (missing `react-arborist` / `react-resizable-panels` module declarations). Zero new errors.

If a new error appears, STOP — the error must be fixed before marking Stage 2 complete.

- [ ] **Step 4: Run the full test suite**

Run:
```bash
pnpm --dir frontend test --run
```

Expected: all tests pass — existing 1152 plus new: 6 store tests + 2 ConnectionHolder tests = 1160+. Zero failures.

- [ ] **Step 5: Summarize file changes**

Run:
```bash
cd /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase
git status
```

Expected changes:
- **New (untracked):** 
  - `frontend/src/store/modules/sqlEditor/uiState.ts`
  - `frontend/src/store/modules/sqlEditor/uiState.test.ts`
  - `frontend/src/react/components/sql-editor/ConnectionHolder.tsx`
  - `frontend/src/react/components/sql-editor/ConnectionHolder.test.tsx`
- **Modified:**
  - `frontend/src/store/modules/sqlEditor/index.ts`
  - `frontend/src/views/sql-editor/context.ts`
  - `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue`
  - `frontend/src/views/sql-editor/EditorCommon/index.ts`
- **Deleted:**
  - `frontend/src/views/sql-editor/EditorCommon/ConnectionHolder.vue`

Any other modifications indicate scope creep — flag them.

- [ ] **Step 6: Bridge integration — user runs these manually in the browser**

Report the following checks for the user to perform in a dev server (`pnpm --dir frontend dev`). These are the critical checks that the `context.ts` refactor didn't break reactive data flow for existing Vue consumers:

1. **Open the SQL Editor landing screen** (no connection) → Welcome renders (the Stage 1 React leaf still works).
2. **Click "Connect to database" on Welcome** → connection panel opens and the aside switches to the SCHEMA tab. This exercises `showConnectionPanel` + `asidePanelTab` bidirectionally through the store.
3. **Toggle the OpenAI panel** → the editor panel width animates. This exercises `showAIPanel` + `editorPanelSize` through the store.
4. **Resize the OpenAI split** → the new split size persists to LocalStorage (reload the page — the split stays where you left it). This exercises `handleEditorPanelResize` through the store.
5. **Click through the aside tabs** (Worksheet ↔ Schema ↔ History ↔ Access) → each tab activates. This exercises `asidePanelTab` reactivity through the Vue consumers that read it (`AsidePanel`, `GutterBar`, etc.).
6. **Open the admin-mode Terminal panel with no connection** → React `ConnectionHolder` renders (the new Stage 2 React leaf). Click it → connection panel opens.
7. **Workspace with custom logo / fallback SVG** → Welcome's BytebaseLogo still renders correctly (regression check for the Stage 1 work).

- [ ] **Step 7: Final report**

Report the Stage 2 completion summary:

```
Stage 2 complete.

Summary:
- New Pinia store `useSQLEditorUIStore` with 7 fields (6 refs + 1 computed) + handleEditorPanelResize action
- `useSQLEditorContext` refactored to delegate via storeToRefs; API shape preserved
- New React ConnectionHolder.tsx replacing the Vue version
- TerminalPanel.vue swapped to ReactPageMount with flex wrapper
- Vue ConnectionHolder.vue deleted

Verification:
- pnpm fix: clean
- pnpm check: pass
- pnpm type-check: baseline only (6 pre-existing SchemaEditorLite errors)
- pnpm test: 1160+ tests pass, zero regressions

Deferred to user:
- Manual UX-parity screenshot for React ConnectionHolder vs Vue original
- Bridge integration smoke tests (listed above)

Ready for PR.
```

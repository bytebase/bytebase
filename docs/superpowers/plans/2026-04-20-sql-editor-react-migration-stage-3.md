# SQL Editor React Migration — Stage 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the `AsidePanel/GutterBar/` Vue subsystem (136 lines across 3 files) to a single React component pair (`GutterBar.tsx` + `TabItem.tsx`). First consumer that reactively reads `useSQLEditorUIStore().asidePanelTab` from React via `useVueState`.

**Architecture:** One React mount at `AsidePanel.vue:8`. React `GutterBar` renders 4 inline `TabItem` components. `TabItem` subscribes to `asidePanelTab` via `useVueState` for active-state styling; clicking writes `asidePanelTab` directly through the Pinia store. No new bridge infrastructure — reuses the Stage 2 `useSQLEditorUIStore`.

**Tech Stack:** React 18, `@base-ui/react`, shadcn `Button`, `Tooltip` primitive, `lucide-react`, `react-i18next`, Pinia via `useVueState`, `class-variance-authority` / `cn()`, vitest.

**Reference spec:** `docs/superpowers/specs/2026-04-20-sql-editor-react-migration-stage-3-design.md`

**Workflow note:** **Do not auto-commit** — user commits manually. Each task ends with "Stop for user review."

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/react/locales/en-US.json` | Modify | Add 3 missing keys: `worksheet.self`, `common.history`, `sql-editor.jit`. |
| `frontend/src/react/locales/zh-CN.json` | Modify | Same 3 keys with zh-CN translations. |
| `frontend/src/react/locales/es-ES.json` | Modify | Same 3 keys with es-ES translations. |
| `frontend/src/react/locales/ja-JP.json` | Modify | Same 3 keys with ja-JP translations. |
| `frontend/src/react/locales/vi-VN.json` | Modify | Same 3 keys with vi-VN translations. |
| `frontend/src/react/components/sql-editor/TabItem.tsx` | Create | Single icon+tooltip button. Props: `{ tab: AsidePanelTab; onClick: () => void }`. Reads `asidePanelTab` via `useVueState`. Applies active/inactive classes to shadcn `Button variant="ghost"`. |
| `frontend/src/react/components/sql-editor/TabItem.test.tsx` | Create | Tests: icon + label render, active class applied when matching, onClick fires. |
| `frontend/src/react/components/sql-editor/GutterBar.tsx` | Create | Container: logo link + divider + 4 `TabItem`s (ACCESS conditional on `project.allowJustInTimeAccess`). Zero props. Writes `asidePanelTab` via `uiStore.asidePanelTab = target`. |
| `frontend/src/react/components/sql-editor/GutterBar.test.tsx` | Create | Tests: 3 tabs when no JIT, 4 tabs when JIT, click writes store, logo `href` computed correctly. |
| `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue` | Modify | Line 8: `<GutterBar size="medium" />` → `<ReactPageMount page="GutterBar" />`. Line 81: `import GutterBar from "./GutterBar";` → `import ReactPageMount from "@/react/ReactPageMount.vue";`. |
| `frontend/src/views/sql-editor/AsidePanel/GutterBar/` (entire dir) | Delete | `GutterBar.vue`, `TabItem.vue`, `common.ts`, `index.ts` — all orphaned after the swap. |

---

## Task 1: Add missing i18n keys

**Goal:** Add `worksheet.self`, `common.history`, `sql-editor.jit` to all 5 React locale files with byte-exact values copied from Vue locales. `check-react-i18n.mjs` enforces 1:1 parity.

**Files:**
- Modify: `frontend/src/react/locales/{en-US,zh-CN,es-ES,ja-JP,vi-VN}.json`

### Exact values (copy from Vue locales)

| Locale | `worksheet.self` | `common.history` | `sql-editor.jit` |
|---|---|---|---|
| en-US | `Worksheet` | `History` | `Just-In-Time Access` |
| zh-CN | `工作表` | `历史记录` | `即时访问` |
| es-ES | `Hoja de trabajo` | `Historial` | `Just-In-Time Access` |
| ja-JP | `ワークシート` | `履歴` | `Just-In-Time Access` |
| vi-VN | `Bảng tính` | `Lịch sử` | `Just-In-Time Access` |

### Steps

- [ ] **Step 1: Cross-check values against Vue locales**

Run:
```bash
for loc in en-US zh-CN es-ES ja-JP vi-VN; do
  echo "=== $loc ==="
  for key in "worksheet.self" "common.history" "sql-editor.jit"; do
    parent="${key%.*}"
    leaf="${key##*.}"
    python3 -c "import json; d=json.load(open('/Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/locales/$loc.json')); print(f'$key:', d.get('$parent', {}).get('$leaf', 'MISSING'))"
  done
done
```

Confirm the output matches the table above. If any value differs, use the actual source value (not the table).

- [ ] **Step 2: Add each key in alphabetical position**

In each React locale file, add the keys alphabetically inside their respective parent blocks:

- `worksheet.self` → inside `"worksheet"` block, first alphabetically (before sibling keys).
- `common.history` → inside `"common"` block, alphabetically (between `hash` and `id` or wherever the natural position is — use the sort script to fix).
- `sql-editor.jit` → inside `"sql-editor"` block, alphabetically (between `invite` and `key`, roughly).

The exact positions don't need to be manually optimal — the next step re-sorts.

- [ ] **Step 3: Run the sort script**

Run:
```bash
node /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/scripts/sort_i18n_keys.mjs
```

Expected: re-sorts in place. Inspect the diff — only the 3 new keys should be repositioned.

- [ ] **Step 4: Run the consistency check**

Run:
```bash
node /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/scripts/check-react-i18n.mjs
```

Expected outcome at this point: the 3 new keys are flagged as "unused" warnings (consumer lands in Task 2 and 3). NO missing-key or cross-locale consistency errors. If the script hard-fails on unused-only errors, that is acceptable for now — report it and move on.

- [ ] **Step 5: Stop for user review**

Report: "3 i18n keys added to 5 React locale files. Sort script ran. Consistency check shows only the expected 'unused' warnings. Ready for commit."

---

## Task 2: Create `TabItem.tsx` (TDD)

**Goal:** Single icon+tooltip button. Accepts `tab` and `onClick` props. Reads `asidePanelTab` via `useVueState` to compute active state; applies active/inactive classNames to shadcn `Button variant="ghost"`.

**Files:**
- Create: `frontend/src/react/components/sql-editor/TabItem.tsx`
- Create: `frontend/src/react/components/sql-editor/TabItem.test.tsx`

### Step 1: Write the failing test

Create `frontend/src/react/components/sql-editor/TabItem.test.tsx`:

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
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorUIStore: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useSQLEditorUIStore: mocks.useSQLEditorUIStore,
}));

// Stub out Tooltip to render children directly — tooltip positioning is a
// primitive concern, not this component's.
vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let TabItem: typeof import("./TabItem").TabItem;

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
  // Default: WORKSHEET is active.
  mocks.useVueState.mockReturnValue(false);
  mocks.useSQLEditorUIStore.mockReturnValue({ asidePanelTab: "WORKSHEET" });
  ({ TabItem } = await import("./TabItem"));
});

describe("TabItem", () => {
  test("renders label for WORKSHEET tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="WORKSHEET" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("worksheet.self");
    expect(container.querySelector("button")).not.toBeNull();
    unmount();
  });

  test("renders label for SCHEMA tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("common.schema");
    unmount();
  });

  test("renders label for HISTORY tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="HISTORY" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("common.history");
    unmount();
  });

  test("renders label for ACCESS tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="ACCESS" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.jit");
    unmount();
  });

  test("applies active class when asidePanelTab matches", () => {
    mocks.useVueState.mockReturnValue(true);
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).toContain("bg-accent/10");
    expect(button?.className).toContain("text-accent");
    unmount();
  });

  test("does NOT apply active class when asidePanelTab differs", () => {
    mocks.useVueState.mockReturnValue(false);
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).not.toContain("bg-accent/10");
    unmount();
  });

  test("calls onClick when button is clicked", () => {
    const handler = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={handler} />
    );
    render();
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(handler).toHaveBeenCalledTimes(1);
    unmount();
  });
});
```

### Step 2: Run the test to verify it fails

```bash
pnpm --dir frontend test -- TabItem.test --run
```

Expected: FAIL with `Cannot find module './TabItem'`.

### Step 3: Write the implementation

Create `frontend/src/react/components/sql-editor/TabItem.tsx`:

```tsx
import { Database, FileCode, History, ShieldCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import type { AsidePanelTab } from "@/store";
import { useSQLEditorUIStore } from "@/store";

type TabItemProps = {
  readonly tab: AsidePanelTab;
  readonly onClick: () => void;
};

const iconByTab = {
  WORKSHEET: FileCode,
  SCHEMA: Database,
  HISTORY: History,
  ACCESS: ShieldCheck,
} as const;

const i18nKeyByTab = {
  WORKSHEET: "worksheet.self",
  SCHEMA: "common.schema",
  HISTORY: "common.history",
  ACCESS: "sql-editor.jit",
} as const;

/**
 * Single tab button in the SQL Editor aside panel's left gutter.
 * Replaces `frontend/src/views/sql-editor/AsidePanel/GutterBar/TabItem.vue`.
 * Active state reflects `useSQLEditorUIStore().asidePanelTab`; click handler
 * is supplied by the GutterBar parent (which writes the store).
 */
export function TabItem({ tab, onClick }: TabItemProps) {
  const { t } = useTranslation();
  const uiStore = useSQLEditorUIStore();
  const isActive = useVueState(() => uiStore.asidePanelTab === tab);

  const Icon = iconByTab[tab];
  const label = t(i18nKeyByTab[tab]);

  return (
    <Tooltip content={label} side="right" delayDuration={300}>
      <Button
        variant="ghost"
        className={cn(
          "size-10 p-0",
          isActive &&
            "bg-accent/10 text-accent hover:bg-accent/10 hover:text-accent"
        )}
        onClick={onClick}
        aria-label={label}
      >
        <Icon className="size-5" />
      </Button>
    </Tooltip>
  );
}
```

### Step 4: Run the test to verify it passes

```bash
pnpm --dir frontend test -- TabItem.test --run
```

Expected: PASS (7 tests).

### Step 5: Stop for user review

Report: "`TabItem.tsx` + 7 tests created and passing. Ready for commit."

---

## Task 3: Create `GutterBar.tsx` (TDD)

**Goal:** Container that renders the logo link + divider + 4 conditional `<TabItem>`s. Reads project from Pinia to gate ACCESS. Writes `asidePanelTab` on click. Zero props (size dropped — only caller passed `"medium"`).

**Files:**
- Create: `frontend/src/react/components/sql-editor/GutterBar.tsx`
- Create: `frontend/src/react/components/sql-editor/GutterBar.test.tsx`

### Step 1: Write the failing test

Create `frontend/src/react/components/sql-editor/GutterBar.test.tsx`:

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
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorStore: vi.fn(),
  useProjectV1Store: vi.fn(),
  useSQLEditorUIStore: vi.fn(),
  routerResolve: vi.fn(() => ({ href: "/project/test-project" })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useSQLEditorStore: mocks.useSQLEditorStore,
  useProjectV1Store: mocks.useProjectV1Store,
  useSQLEditorUIStore: mocks.useSQLEditorUIStore,
}));

vi.mock("@/router", () => ({
  router: {
    resolve: mocks.routerResolve,
    currentRoute: { value: { params: {} } },
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DETAIL: "project.detail",
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
}));

vi.mock("@/assets/logo-icon.svg", () => ({
  default: "/assets/logo-icon.svg",
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let GutterBar: typeof import("./GutterBar").GutterBar;

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
  // useVueState is called 3 times in GutterBar: project, routeProjectParam,
  // and once per TabItem's isActive check (4 TabItems rendered when JIT on).
  // The test uses mockImplementation with a counter for project-related calls.
  mocks.useSQLEditorStore.mockReturnValue({ project: "projects/test" });
  mocks.useProjectV1Store.mockReturnValue({
    getProjectByName: () => ({ allowJustInTimeAccess: false }),
  });
  mocks.useSQLEditorUIStore.mockReturnValue({ asidePanelTab: "WORKSHEET" });
  mocks.routerResolve.mockReturnValue({ href: "/workspace/home" });
  ({ GutterBar } = await import("./GutterBar"));
});

describe("GutterBar", () => {
  test("renders 3 tabs when project does not allow JIT access", () => {
    // All useVueState calls: first returns project (no JIT), second returns
    // route project param, rest are tab active-state booleans (all false).
    mocks.useVueState.mockImplementation((getter) => getter());
    mocks.useProjectV1Store.mockReturnValue({
      getProjectByName: () => ({ allowJustInTimeAccess: false }),
    });
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    expect(buttons).toHaveLength(3);
    unmount();
  });

  test("renders 4 tabs when project allows JIT access", () => {
    mocks.useVueState.mockImplementation((getter) => getter());
    mocks.useProjectV1Store.mockReturnValue({
      getProjectByName: () => ({ allowJustInTimeAccess: true }),
    });
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    expect(buttons).toHaveLength(4);
    unmount();
  });

  test("click writes asidePanelTab on the store", () => {
    mocks.useVueState.mockImplementation((getter) => getter());
    mocks.useProjectV1Store.mockReturnValue({
      getProjectByName: () => ({ allowJustInTimeAccess: false }),
    });
    const store = { asidePanelTab: "WORKSHEET" as const };
    mocks.useSQLEditorUIStore.mockReturnValue(store);
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    // Click the second tab (SCHEMA).
    act(() => {
      (buttons[1] as HTMLButtonElement).click();
    });
    expect(store.asidePanelTab).toBe("SCHEMA");
    unmount();
  });

  test("logo link uses project route when route has project param", () => {
    mocks.useVueState.mockImplementation((getter) => {
      // Return "test-proj" for the route-param getter call.
      const result = getter();
      if (result === undefined) return "test-proj";
      return result;
    });
    mocks.routerResolve.mockReturnValue({ href: "/projects/test-proj" });
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toBe("/projects/test-proj");
    expect(link?.getAttribute("target")).toBe("_blank");
    unmount();
  });
});
```

### Step 2: Run the test to verify it fails

```bash
pnpm --dir frontend test -- GutterBar.test --run
```

Expected: FAIL with `Cannot find module './GutterBar'`.

### Step 3: Write the implementation

Create `frontend/src/react/components/sql-editor/GutterBar.tsx`:

```tsx
import logoIcon from "@/assets/logo-icon.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import {
  type AsidePanelTab,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorUIStore,
} from "@/store";
import { TabItem } from "./TabItem";

/**
 * Left gutter of the SQL Editor aside panel. Shows the Bytebase logo at
 * the top and 4 tab buttons (WORKSHEET, SCHEMA, HISTORY, and optionally
 * ACCESS when the current project allows JIT).
 *
 * Replaces `frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue`.
 */
export function GutterBar() {
  const editorStore = useSQLEditorStore();
  const projectStore = useProjectV1Store();
  const uiStore = useSQLEditorUIStore();

  const project = useVueState(() => {
    const name = editorStore.project;
    return name ? projectStore.getProjectByName(name) : undefined;
  });

  const routeProjectParam = useVueState(
    () => router.currentRoute.value.params.project as string | undefined
  );

  const logoHref = routeProjectParam
    ? router.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId: routeProjectParam },
      }).href
    : router.resolve({ name: WORKSPACE_ROUTE_LANDING }).href;

  const handleClickTab = (target: AsidePanelTab) => {
    uiStore.asidePanelTab = target;
  };

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1">
      <div className="flex flex-col gap-y-1">
        <div className="flex flex-col justify-center items-center pb-1">
          <a href={logoHref} target="_blank" rel="noopener noreferrer">
            <img className="w-9 h-auto" src={logoIcon} alt="Bytebase" />
          </a>
        </div>
        <div className="w-full h-0 border-t" />
        <TabItem tab="WORKSHEET" onClick={() => handleClickTab("WORKSHEET")} />
        <TabItem tab="SCHEMA" onClick={() => handleClickTab("SCHEMA")} />
        <TabItem tab="HISTORY" onClick={() => handleClickTab("HISTORY")} />
        {project?.allowJustInTimeAccess && (
          <TabItem tab="ACCESS" onClick={() => handleClickTab("ACCESS")} />
        )}
      </div>
      <div className="flex flex-col justify-end items-center"></div>
    </div>
  );
}
```

### Step 4: Run the test to verify it passes

```bash
pnpm --dir frontend test -- GutterBar.test --run
```

Expected: PASS (4 tests).

### Step 5: Run the React i18n consistency check

```bash
node /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/scripts/check-react-i18n.mjs
```

Expected: PASS with no missing / unused / consistency errors. The 3 keys added in Task 1 now have a consumer via `TabItem`.

### Step 6: Stop for user review

Report: "`GutterBar.tsx` + 4 tests created and passing. React i18n check clean (0 unused, 0 missing). Ready for commit."

---

## Task 4: Swap `AsidePanel.vue` to use `<ReactPageMount>`

**Goal:** Replace the Vue `<GutterBar>` mount at line 8 of `AsidePanel.vue` with `<ReactPageMount page="GutterBar" />`. Update the import at line 81.

**File:**
- Modify: `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`

### Step 1: Read the current state

Open `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`. Confirm:

- Line 8: `<GutterBar size="medium" />`
- Line 81: `import GutterBar from "./GutterBar";`

If line numbers drift slightly from whitespace changes, adjust the edit but preserve semantics.

### Step 2: Replace the template

Change line 8 from:
```vue
<GutterBar size="medium" />
```
To:
```vue
<ReactPageMount page="GutterBar" />
```

### Step 3: Replace the import

Change line 81 from:
```ts
import GutterBar from "./GutterBar";
```
To:
```ts
import ReactPageMount from "@/react/ReactPageMount.vue";
```

Place the new import in the `@/...` absolute-path group (Vue file import convention: absolute paths before relative paths). In this file, that means it should go near the other `@/...` imports such as `PROJECT_V1_ROUTE_DASHBOARD`, `useActuatorV1Store`, etc.

### Step 4: Type-check

```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. Zero new errors.

### Step 5: Run the test suite

```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: all tests pass.

### Step 6: Stop for user review

Report: "`AsidePanel.vue` swapped to `<ReactPageMount page=\"GutterBar\" />`. Type-check clean, tests pass. Ready for commit."

---

## Task 5: Delete the Vue `AsidePanel/GutterBar/` directory

**Goal:** Verify zero remaining callers, then delete the entire orphaned Vue subsystem.

**Files to delete:**
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/TabItem.vue`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/common.ts`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/index.ts`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/` (directory itself, after files gone)

### Step 1: Search for remaining callers

From `/Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/`, run these 4 patterns (one per file/directory):

```bash
grep -rn "from.*AsidePanel/GutterBar" frontend/src/
grep -rn "import.*GutterBar.*from.*AsidePanel" frontend/src/
grep -rn "<GutterBar" frontend/src/views/
grep -rn "from.*GutterBar/common" frontend/src/
```

**Expected:** each pattern returns zero matches OUTSIDE the files being deleted.

The React `GutterBar.tsx` at `frontend/src/react/components/sql-editor/GutterBar.tsx` is the REPLACEMENT — not a caller. Its imports do not match these patterns.

If any pattern matches outside the files to delete, STOP and report BLOCKED with the caller details.

### Step 2: Delete the directory

Run:
```bash
rm -rf /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/views/sql-editor/AsidePanel/GutterBar
```

### Step 3: Verify

Run:
```bash
ls /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase/frontend/src/views/sql-editor/AsidePanel/GutterBar 2>&1
```
Expected: "No such file or directory".

### Step 4: Type-check

```bash
pnpm --dir frontend type-check 2>&1 | tail -15
```

Expected: only the 6 pre-existing `SchemaEditorLite` errors. If a new error like `Cannot find module "./GutterBar"` appears, Task 4's import swap was missed — revisit.

### Step 5: Test suite

```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: all tests pass.

### Step 6: Stop for user review

Report: "Vue `AsidePanel/GutterBar/` directory deleted (4 files). Type-check clean, tests pass. Ready for commit."

---

## Task 6: Final verification

**Goal:** Full frontend verification suite + user-facing manual verification checklist.

### Step 1: Auto-fix

```bash
pnpm --dir frontend fix
```

Expected: no changes or trivial format-only changes to the new files.

### Step 2: Check

```bash
pnpm --dir frontend check
```

Expected: pass (ESLint + Biome + React i18n + i18n sort).

### Step 3: Type-check with baseline confirmation

```bash
pnpm --dir frontend type-check 2>&1 | tail -20
```

Expected: exactly the 6 pre-existing `SchemaEditorLite` errors. Zero new.

### Step 4: Full test suite

```bash
pnpm --dir frontend test --run 2>&1 | tail -8
```

Expected: existing 1160 tests + new Stage 3 tests (7 from `TabItem.test.tsx`, 4 from `GutterBar.test.tsx` = 11 new) → 1171+ pass. Zero failures.

### Step 5: File change summary

Run:
```bash
cd /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase
git status
```

Expected Stage 3 changes (additional to the Stage 1 + 2 files already in the tree):

**New untracked:**
- `frontend/src/react/components/sql-editor/TabItem.tsx`
- `frontend/src/react/components/sql-editor/TabItem.test.tsx`
- `frontend/src/react/components/sql-editor/GutterBar.tsx`
- `frontend/src/react/components/sql-editor/GutterBar.test.tsx`

**Modified:**
- `frontend/src/react/locales/en-US.json`
- `frontend/src/react/locales/zh-CN.json`
- `frontend/src/react/locales/es-ES.json`
- `frontend/src/react/locales/ja-JP.json`
- `frontend/src/react/locales/vi-VN.json`
- `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`

**Deleted:**
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/TabItem.vue`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/common.ts`
- `frontend/src/views/sql-editor/AsidePanel/GutterBar/index.ts`

Any other modifications indicate scope creep — flag them.

### Step 6: Manual UX verification — user runs in dev server

Output these checks for the user to run through with `pnpm --dir frontend dev`:

**Visual parity:**
1. Open the SQL Editor. Left gutter shows: Bytebase logo at top, thin divider, then 3 tab buttons (WORKSHEET/SCHEMA/HISTORY) in medium size (40×40 px). Tab icons match Vue: `FileCode` for Worksheet, `Database` for Schema, `History` for History.
2. Switch to a project with `allowJustInTimeAccess=true` → 4th tab appears (ACCESS with `ShieldCheck` icon, labeled "Just-In-Time Access" in tooltip).
3. Active tab has accent-tinted background (`bg-accent/10`) and accent-colored icon. Inactive tabs use default control coloring with hover tint.
4. Hover each tab → tooltip appears to the right after ~300ms with the correct localized label.
5. Click the logo at the top → opens the project detail page (or workspace landing when no project context) in a new tab.
6. Locale switch (English ↔ Chinese) → tooltips update.

**Bridge reactivity (critical):**
7. From the Welcome screen's "Connect to database" button (Stage 1): click it. Expected: connection panel opens AND the SCHEMA tab in the gutter immediately shows active styling. This confirms `useVueState` subscribes to external writes of `useSQLEditorUIStore().asidePanelTab`.
8. Click any tab in the gutter → the aside panel's right side content switches to that pane (`WorksheetPane`, `SchemaPane`, etc.). This confirms the store write propagates to Vue consumers that read `asidePanelTab`.

### Step 7: Final report

Report template:

```
Stage 3 complete.

Summary:
- 3 i18n keys added to 5 React locale files
- New TabItem.tsx + 7 tests
- New GutterBar.tsx + 4 tests
- AsidePanel.vue swapped to <ReactPageMount page="GutterBar" />
- Vue AsidePanel/GutterBar/ directory (4 files) deleted

Verification:
- pnpm fix: clean
- pnpm check: pass
- pnpm type-check: baseline only (6 pre-existing SchemaEditorLite errors)
- pnpm test: 1171+ tests pass, zero regressions

Deferred to user:
- Manual UX parity (8 items listed above)
- Bridge reactivity check (first real React-subscribed store read)

Ready for PR.
```

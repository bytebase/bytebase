# Unified SelectionActionBar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace three near-identical batch-operation strips (`ProjectsPage`, `DatabaseBatchOperationsBar`, `InstancesPage`) with one shared `SelectionActionBar` — a floating bottom-center pill mounted into the overlay layer.

**Architecture:** New generic component at `frontend/src/react/components/SelectionActionBar.tsx` rendered via `createPortal` into `getLayerRoot("overlay")`, with declarative `actions[]` props plus a `children` slot for custom action nodes (e.g. `SyncDropdown`). The existing `DatabaseBatchOperationsBar` becomes a thin wrapper preserving its external API across its three callers (`DatabasesPage`, `InstanceDetailPage`, `ProjectDatabasesPage`).

**Tech Stack:** React 18 + Base UI + Tailwind CSS v4 + lucide-react + vitest + react-i18next.

> **Spec correction (worth noting):** The approved spec at `docs/superpowers/specs/2026-05-08-selection-action-bar-design.md` mentions `bg-control` for the pill surface. `--color-control` is actually a dark text color (`#52525b`), not a background. Existing primitives (`dialog.tsx`, `popover.tsx`) use `bg-background` for surfaces. This plan uses `bg-background` instead — same visual intent (white/dark surface), correct semantic token. Update the spec inline if you want consistency.

---

## File Structure

**Created files:**

- `frontend/src/react/components/SelectionActionBar.tsx` — the new shared component (the only place the floating-pill UI lives)
- `frontend/src/react/components/SelectionActionBar.test.tsx` — vitest unit tests

**Modified files:**

- `frontend/src/locales/en-US.json` — add two ARIA-label keys (`common.batch-actions`, `common.clear-selection`)
- `frontend/src/react/pages/settings/ProjectsPage.tsx` — delete local `BatchOperationsBar`, replace with `SelectionActionBar` usage
- `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` — rewritten as a thin wrapper around `SelectionActionBar` (external API unchanged)
- `frontend/src/react/pages/settings/InstancesPage.tsx` — delete local `BatchOperationsBar`, replace with `SelectionActionBar` usage. `SyncDropdown` (lines 539–589) stays and is passed as `children`.

**Untouched (compile unchanged through the wrapper):**

- `frontend/src/react/pages/settings/DatabasesPage.tsx`
- `frontend/src/react/pages/settings/InstanceDetailPage.tsx`
- `frontend/src/react/pages/project/ProjectDatabasesPage.tsx`
- `frontend/src/react/components/database/index.tsx` (re-exports `DatabaseBatchOperationsBar`; signature unchanged)

---

## Task 1: Add i18n keys for ARIA labels

**Files:**

- Modify: `frontend/src/locales/en-US.json`
- Verify: every other locale file under `frontend/src/locales/*.json` (zh-CN.json, ja-JP.json, etc.) — `vue-i18n` falls back to en-US for missing keys, but adding placeholders avoids future drift; for this task, **only add to en-US.json**. Other locales are picked up by Crowdin sync downstream.

- [ ] **Step 1: Open `frontend/src/locales/en-US.json` and locate the `"common"` block**

Find the existing `common` object containing keys like `"archive"`, `"cancel"`, `"clear"`, etc. (around line 209+ in the current file).

- [ ] **Step 2: Add two new keys inside the `common` block, alphabetized**

Add `"batch-actions"` (just after `"batch": ...` if present, or near the `b*` cluster) and `"clear-selection"` (just after `"clear"`):

```json
"batch-actions": "Batch actions",
"clear-selection": "Clear selection",
```

Example diff:

```diff
   "cancel": "Cancel",
+  "batch-actions": "Batch actions",
   "clear": "Clear",
+  "clear-selection": "Clear selection",
   "create": "Create",
```

(Exact alphabetic position depends on existing keys — keep the file alphabetized as the existing entries are.)

- [ ] **Step 3: Verify JSON is valid**

Run: `node -e "JSON.parse(require('fs').readFileSync('frontend/src/locales/en-US.json','utf8'))"`

Expected: no output (success). If a `SyntaxError` is thrown, fix the trailing comma / quotation issue before continuing.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/locales/en-US.json
git commit -m "i18n: add common.batch-actions and common.clear-selection keys"
```

---

## Task 2: Write the failing test for `SelectionActionBar`

**Files:**

- Create: `frontend/src/react/components/SelectionActionBar.test.tsx`

> Why this task exists alone: TDD. We write the test before any component code so we see it fail, then make it pass in Task 3.

- [ ] **Step 1: Create the test file with full content**

Path: `frontend/src/react/components/SelectionActionBar.test.tsx`

```tsx
import { Archive, Trash2 } from "lucide-react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SelectionActionBar } from "./SelectionActionBar";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SelectionActionBar", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    document.body.innerHTML = "";
  });

  test("renders nothing when count is 0", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={0}
          label="0 selected"
          onClear={() => {}}
          actions={[]}
        />
      );
    });
    // Bar is not in the overlay layer.
    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent ?? ""
    ).not.toContain("0 selected");
  });

  test("renders label, clear button, and visible actions when count > 0", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={2}
          label="2 selected"
          onClear={() => {}}
          actions={[
            {
              key: "archive",
              label: "Archive",
              icon: Archive,
              onClick: () => {},
            },
            {
              key: "delete",
              label: "Delete",
              icon: Trash2,
              onClick: () => {},
              tone: "destructive",
            },
          ]}
        />
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("2 selected");
    expect(overlay?.querySelector('[aria-label="Clear selection"]')).not.toBeNull();
    expect(overlay?.textContent).toContain("Archive");
    expect(overlay?.textContent).toContain("Delete");
  });

  test("omits hidden actions and disables disabled actions", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={() => {}}
          actions={[
            { key: "a", label: "AlphaAction", onClick: () => {} },
            { key: "b", label: "BetaAction", onClick: () => {}, hidden: true },
            { key: "c", label: "GammaAction", onClick: () => {}, disabled: true },
          ]}
        />
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay")!;
    expect(overlay.textContent).toContain("AlphaAction");
    expect(overlay.textContent).not.toContain("BetaAction");
    const gamma = Array.from(overlay.querySelectorAll("button")).find(
      (b) => b.textContent === "GammaAction"
    );
    expect(gamma).toBeDefined();
    expect(gamma?.hasAttribute("disabled")).toBe(true);
  });

  test("clicking an action invokes its onClick", async () => {
    const onClick = vi.fn();
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={() => {}}
          actions={[{ key: "a", label: "ActionLabel", onClick }]}
        />
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay")!;
    const btn = Array.from(overlay.querySelectorAll("button")).find(
      (b) => b.textContent === "ActionLabel"
    )!;
    await act(async () => {
      btn.click();
    });
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test("clicking the clear button invokes onClear", async () => {
    const onClear = vi.fn();
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={onClear}
          actions={[]}
        />
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay")!;
    const clearBtn = overlay.querySelector<HTMLButtonElement>(
      '[aria-label="Clear selection"]'
    )!;
    await act(async () => {
      clearBtn.click();
    });
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  test("Escape clears when count > 0 and not when count === 0", async () => {
    const onClear = vi.fn();

    // count > 0 → Escape clears.
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={onClear}
          actions={[]}
        />
      );
    });
    await act(async () => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    });
    expect(onClear).toHaveBeenCalledTimes(1);

    // count === 0 → Escape no-ops.
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={0}
          label="0 selected"
          onClear={onClear}
          actions={[]}
        />
      );
    });
    await act(async () => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    });
    expect(onClear).toHaveBeenCalledTimes(1); // unchanged
  });

  test("destructive tone applies the red-text override class", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={() => {}}
          actions={[
            {
              key: "del",
              label: "DestructiveAction",
              onClick: () => {},
              tone: "destructive",
            },
          ]}
        />
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay")!;
    const btn = Array.from(overlay.querySelectorAll("button")).find(
      (b) => b.textContent === "DestructiveAction"
    )!;
    expect(btn.className).toContain("text-error");
  });

  test("children render after declarative actions", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          onClear={() => {}}
          actions={[{ key: "a", label: "InlineAction", onClick: () => {} }]}
        >
          <span data-testid="custom-slot">CustomSlot</span>
        </SelectionActionBar>
      );
    });
    const overlay = document.getElementById("bb-react-layer-overlay")!;
    expect(overlay.querySelector('[data-testid="custom-slot"]')).not.toBeNull();
    expect(overlay.textContent).toContain("CustomSlot");
  });
});
```

> **Note on i18n in tests:** the test asserts hardcoded `"Clear selection"` for the clear button's aria-label. The test uses raw labels (no `t()`), so the component must accept the aria-label key from the consumer or render an English default in test environments. We solve this by having the component **call `t("common.clear-selection")` directly**; the project's vitest setup wires `react-i18next` to fall back to the literal key value or to `en-US` text. Verify by checking `frontend/src/react/test-utils/` for the test-side i18n setup; if the resolved string is the key itself (e.g. `"common.clear-selection"`), update this test's assertion accordingly. **Action when running:** if the test fails on aria-label string mismatch, change the assertion to match what `t()` returns in the test env (do not change the component).

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm --dir frontend test SelectionActionBar`

Expected: FAIL with module-not-found / import error for `./SelectionActionBar` (the component does not exist yet).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/SelectionActionBar.test.tsx
git commit -m "test: failing test for SelectionActionBar (component to follow)"
```

---

## Task 3: Implement `SelectionActionBar`

**Files:**

- Create: `frontend/src/react/components/SelectionActionBar.tsx`

- [ ] **Step 1: Create the component file with full content**

Path: `frontend/src/react/components/SelectionActionBar.tsx`

```tsx
import type { LucideIcon } from "lucide-react";
import { X } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { getLayerRoot } from "@/react/components/ui/layer";
import { Separator } from "@/react/components/ui/separator";
import { cn } from "@/react/lib/utils";

export interface SelectionAction {
  /** Stable key for React reconciliation. */
  key: string;
  label: string;
  icon?: LucideIcon;
  onClick: () => void;
  disabled?: boolean;
  /** When true, the action is omitted from the bar entirely. */
  hidden?: boolean;
  /**
   * Visual tone. "destructive" applies a red-text override on top of
   * the ghost variant. Default: "neutral".
   */
  tone?: "neutral" | "destructive";
}

export interface SelectionActionBarProps {
  /** Number of selected items. The bar renders only when count > 0. */
  count: number;
  /**
   * Pre-formatted label (e.g. "2 selected", "3 databases selected").
   * The call site owns i18n + pluralization.
   */
  label: string;
  /** Clears the selection. Also invoked on Escape. */
  onClear: () => void;
  /** Declarative actions. Rendered in order. Hidden actions are omitted. */
  actions?: SelectionAction[];
  /**
   * Custom action nodes rendered after `actions`. Used for actions that
   * require richer UI (e.g. InstancesPage's split-dropdown Sync).
   */
  children?: ReactNode;
}

const DESTRUCTIVE_TONE_CLASS =
  "text-error hover:bg-error/10 hover:text-error focus-visible:ring-error";

export function SelectionActionBar({
  count,
  label,
  onClear,
  actions,
  children,
}: SelectionActionBarProps) {
  const { t } = useTranslation();

  useEffect(() => {
    if (count <= 0) return;
    const handler = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      if (event.defaultPrevented) return;
      // Avoid hijacking dialog dismissal: bail if focus is inside a Base UI
      // dialog/popover surface.
      const active = document.activeElement;
      if (
        active instanceof HTMLElement &&
        active.closest("[data-base-ui-popup], [role='dialog']")
      ) {
        return;
      }
      onClear();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [count, onClear]);

  if (count <= 0) return null;

  const visibleActions = (actions ?? []).filter((a) => !a.hidden);

  return createPortal(
    <div
      role="toolbar"
      aria-label={t("common.batch-actions")}
      className={cn(
        "fixed bottom-6 left-1/2 -translate-x-1/2",
        "max-w-[90vw]",
        "flex items-center gap-x-1",
        "rounded-full bg-background border border-control-border shadow-lg",
        "px-2 py-1.5"
      )}
    >
      <Button
        variant="ghost"
        size="sm"
        aria-label={t("common.clear-selection")}
        onClick={onClear}
      >
        <X className="size-4" aria-hidden />
      </Button>

      <output
        aria-live="polite"
        className="text-sm font-medium text-control whitespace-nowrap px-1"
      >
        {label}
      </output>

      <Separator orientation="vertical" className="h-5 mx-1" />

      <div className="flex flex-wrap items-center gap-x-1 gap-y-1">
        {visibleActions.map((action) => {
          const Icon = action.icon;
          return (
            <Button
              key={action.key}
              variant="ghost"
              size="sm"
              disabled={action.disabled}
              onClick={action.onClick}
              className={cn(
                action.tone === "destructive" && DESTRUCTIVE_TONE_CLASS
              )}
            >
              {Icon && <Icon className="size-4" aria-hidden />}
              {action.label}
            </Button>
          );
        })}
        {children}
      </div>
    </div>,
    getLayerRoot("overlay")
  );
}
```

- [ ] **Step 2: Run the test to verify it passes**

Run: `pnpm --dir frontend test SelectionActionBar`

Expected: all tests in `SelectionActionBar.test.tsx` PASS.

If the aria-label assertion fails because `t()` returned the literal key (`"common.clear-selection"`) instead of the English string, update the assertion in `SelectionActionBar.test.tsx` to match what `t()` actually returns in the test env — do not change the component.

- [ ] **Step 3: Run the layering scanner**

Run: `node frontend/scripts/check-react-layering.mjs`

Expected: PASS. The component uses `getLayerRoot("overlay")` (no body portal) and has no raw `z-index` in its `className`.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`

Expected: no new errors. (If pre-existing errors are reported in unrelated files, ignore those — but ensure no errors point at `SelectionActionBar.tsx` or `SelectionActionBar.test.tsx`.)

- [ ] **Step 5: Lint and format**

Run: `pnpm --dir frontend fix`

Expected: no errors; the formatter may rewrite whitespace/import order in the new files.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/SelectionActionBar.tsx
git add -u frontend/src/react/components/SelectionActionBar.test.tsx
git commit -m "feat(react): add SelectionActionBar (floating selection toolbar)"
```

---

## Task 4: Migrate `ProjectsPage` to `SelectionActionBar`

**Files:**

- Modify: `frontend/src/react/pages/settings/ProjectsPage.tsx` (delete local `BatchOperationsBar` at lines 116–319; replace usage at line 729; add new actions array)

- [ ] **Step 1: Delete the local `BatchOperationsBar` function**

Remove the entire block from line 116 (the `// ============================================================` divider that precedes `BatchOperationsBar`) through line 319 (closing `}` of `BatchOperationsBar`). **Keep** the `ConfirmDialog` and `ProjectListPreview` functions — they're still used.

After deletion, reorganize the remaining helpers so the file structure is:

1. `projectIssuesRoute` (top)
2. `ConfirmDialog`
3. `ProjectListPreview`
4. `ProjectActionDropdown`
5. `ProjectsPage` (main)

Move `ConfirmDialog` and `ProjectListPreview` (and their dividers) to sit before `ProjectActionDropdown` if they're not already. The three confirm-dialog usages now live inside `ProjectsPage` itself (next steps).

- [ ] **Step 2: Add `SelectionActionBar` import and lift confirm-dialog state into `ProjectsPage`**

In `ProjectsPage.tsx`, add to the imports:

```tsx
import {
  SelectionActionBar,
  type SelectionAction,
} from "@/react/components/SelectionActionBar";
```

Inside `ProjectsPage`, just below the existing `selectedNames` / `selectedProjectList` / `handleBatchOperation` block (around line 665+), add the three confirm-dialog state hooks **and** the three batch handlers (move them from the deleted `BatchOperationsBar`):

```tsx
const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);
const [showRestoreConfirm, setShowRestoreConfirm] = useState(false);
const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

const hasActiveProjects = selectedProjectList.some(
  (p) => p.state === State.ACTIVE
);
const hasArchivedProjects = selectedProjectList.some(
  (p) => p.state === State.DELETED
);

const handleBatchArchive = useCallback(async () => {
  try {
    const activeProjects = selectedProjectList.filter(
      (p) => p.state === State.ACTIVE
    );
    await projectStore.batchDeleteProjects(activeProjects.map((p) => p.name));
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.batch.archive.success", {
        count: activeProjects.length,
      }),
    });
    setShowArchiveConfirm(false);
    handleBatchOperation();
  } catch (error: unknown) {
    const err = error as { message?: string };
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("project.batch.archive.error"),
      description: err.message,
    });
  }
}, [selectedProjectList, projectStore, t, handleBatchOperation]);

const handleBatchRestore = useCallback(async () => {
  try {
    const archivedProjects = selectedProjectList.filter(
      (p) => p.state === State.DELETED
    );
    await Promise.all(
      archivedProjects.map((project) => projectStore.restoreProject(project))
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.restored"),
    });
    setShowRestoreConfirm(false);
    handleBatchOperation();
  } catch (error: unknown) {
    const err = error as { message?: string };
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.restore"),
      description: err.message,
    });
  }
}, [selectedProjectList, projectStore, t, handleBatchOperation]);

const handleBatchDelete = useCallback(async () => {
  try {
    await projectStore.batchPurgeProjects(
      selectedProjectList.map((p) => p.name)
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.batch.delete.success", {
        count: selectedProjectList.length,
      }),
    });
    setShowDeleteConfirm(false);
    handleBatchOperation();
  } catch (error: unknown) {
    const err = error as { message?: string };
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("project.batch.delete.error"),
      description: err.message,
    });
  }
}, [selectedProjectList, projectStore, t, handleBatchOperation]);

const batchActions: SelectionAction[] = [
  {
    key: "archive",
    label: t("common.archive"),
    icon: Archive,
    onClick: () => setShowArchiveConfirm(true),
    hidden: !hasActiveProjects,
  },
  {
    key: "restore",
    label: t("common.restore"),
    onClick: () => setShowRestoreConfirm(true),
    hidden: !hasArchivedProjects,
  },
  {
    key: "delete",
    label: t("common.delete"),
    icon: Trash2,
    onClick: () => setShowDeleteConfirm(true),
    tone: "destructive",
  },
];
```

- [ ] **Step 3: Replace the old bar render and add the three confirm dialogs**

In the JSX, replace this block (currently around line 727–733):

```tsx
{canDelete && (
  <BatchOperationsBar
    selectedProjects={selectedProjectList}
    onUpdate={handleBatchOperation}
  />
)}
```

with:

```tsx
{canDelete && (
  <SelectionActionBar
    count={selectedProjectList.length}
    label={t("project.batch.selected", { count: selectedProjectList.length })}
    onClear={() => setSelectedNames(new Set())}
    actions={batchActions}
  />
)}

<ConfirmDialog
  open={showArchiveConfirm}
  variant="warning"
  title={t("project.batch.archive.title", {
    count: selectedProjectList.length,
  })}
  description={t("project.batch.archive.description")}
  okText={t("common.archive")}
  onOk={handleBatchArchive}
  onCancel={() => setShowArchiveConfirm(false)}
>
  <ProjectListPreview
    projects={selectedProjectList}
    iconColor="text-success"
  />
</ConfirmDialog>

<ConfirmDialog
  open={showRestoreConfirm}
  variant="warning"
  title={t("common.restore")}
  description={t("common.restore")}
  okText={t("common.restore")}
  onOk={handleBatchRestore}
  onCancel={() => setShowRestoreConfirm(false)}
>
  <ProjectListPreview
    projects={selectedProjectList}
    iconColor="text-success"
  />
</ConfirmDialog>

<ConfirmDialog
  open={showDeleteConfirm}
  variant="error"
  title={t("project.batch.delete.title", {
    count: selectedProjectList.length,
  })}
  description={t("project.batch.delete.description")}
  okText={t("common.delete")}
  onOk={handleBatchDelete}
  onCancel={() => setShowDeleteConfirm(false)}
>
  <div className="flex flex-col gap-y-3">
    <ProjectListPreview
      projects={selectedProjectList}
      iconColor="text-error"
    />
    <div className="rounded-sm border border-error bg-error/5 p-3">
      <p className="text-sm font-medium text-error">
        {t("common.cannot-undo-this-action")}
      </p>
      <p className="text-sm text-error/80 mt-1">
        {t("project.batch.delete.warning")}
      </p>
    </div>
  </div>
</ConfirmDialog>
```

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`

Expected: no new errors in `ProjectsPage.tsx`. If unused-import errors point at the deleted `BatchOperationsBar` artifacts (e.g. icons no longer referenced inside the deleted block), remove them. The page still uses `Archive`, `Trash2`, `EllipsisVertical`, `Plus`, `Check` — so verify each import is still needed.

- [ ] **Step 5: Lint, format, run tests**

Run: `pnpm --dir frontend fix && pnpm --dir frontend test ProjectsPage`

Expected: no lint errors. If a `ProjectsPage` test exists, it passes; if not, no test failure introduced.

- [ ] **Step 6: Manual smoke (optional but recommended)**

Run the dev server (`pnpm --dir frontend dev`) and on the Projects page, select 1+ projects → confirm:
- Floating pill appears at bottom-center, not as a top strip.
- Pill shows count + Archive (if any active selected) + Delete; Delete text is red.
- Click ✕ → selection clears.
- Press Escape → selection clears.
- Click Archive → existing `ConfirmDialog` opens with the project list preview.

- [ ] **Step 7: Commit**

```bash
git add -u frontend/src/react/pages/settings/ProjectsPage.tsx
git commit -m "refactor(react): migrate ProjectsPage batch bar to SelectionActionBar"
```

---

## Task 5: Convert `DatabaseBatchOperationsBar` to a thin wrapper

**Files:**

- Modify: `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` (full rewrite — preserve external API, body becomes a `SelectionActionBar` invocation)
- Untouched (compile unchanged): `DatabasesPage.tsx`, `InstanceDetailPage.tsx`, `ProjectDatabasesPage.tsx`

- [ ] **Step 1: Replace the file body with the thin wrapper**

Path: `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx`

Full new content:

```tsx
import {
  ArrowRightLeft,
  Download,
  Pencil,
  RefreshCw,
  SquareStack,
  Tag,
  Unlink,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  SelectionActionBar,
  type SelectionAction,
} from "@/react/components/SelectionActionBar";
import type { Permission } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE,
  PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE,
} from "@/utils";

export interface DatabaseBatchOperationsBarProps {
  databases: Database[];
  /** When provided, permission checks use project-level IAM. */
  project?: Project;
  onSyncSchema: () => void;
  onEditLabels: () => void;
  onEditEnvironment: () => void;
  // Global context: transfer to another project
  onTransferProject?: () => void;
  // Project context: unassign from current project
  onUnassign?: () => void;
  // Project context: change database schema
  onChangeDatabase?: () => void;
  // Project context: export data
  onExportData?: () => void;
  /** Optional: clears the upstream selection. If omitted, the bar still
   *  renders but the ✕/Escape clear is a no-op for the consumer. */
  onClearSelection?: () => void;
}

export function DatabaseBatchOperationsBar({
  databases,
  project,
  onSyncSchema,
  onEditLabels,
  onEditEnvironment,
  onTransferProject,
  onUnassign,
  onChangeDatabase,
  onExportData,
  onClearSelection,
}: DatabaseBatchOperationsBarProps) {
  const { t } = useTranslation();

  const hasPermission = (permission: Permission) =>
    project
      ? hasProjectPermissionV2(project, permission)
      : hasWorkspacePermissionV2(permission);

  const canSync = hasPermission("bb.databases.sync");
  const canUpdate = hasPermission("bb.databases.update");
  const canGetEnvironment = hasPermission("bb.settings.getEnvironment");
  const canChangeDatabase =
    PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE.every(hasPermission);
  const canExportData =
    PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE.every(hasPermission);

  const actions: SelectionAction[] = [
    {
      key: "change-database",
      label: t("database.change-database"),
      icon: Pencil,
      onClick: () => onChangeDatabase?.(),
      disabled: !canChangeDatabase,
      hidden: !onChangeDatabase,
    },
    {
      key: "export-data",
      label: t("custom-approval.risk-rule.risk.namespace.data_export"),
      icon: Download,
      onClick: () => onExportData?.(),
      disabled: !canExportData,
      hidden: !onExportData,
    },
    {
      key: "sync-schema",
      label: t("database.sync-schema-button"),
      icon: RefreshCw,
      onClick: onSyncSchema,
      disabled: !canSync,
    },
    {
      key: "edit-labels",
      label: t("database.edit-labels"),
      icon: Tag,
      onClick: onEditLabels,
      disabled: !canUpdate,
    },
    {
      key: "edit-environment",
      label: t("database.edit-environment"),
      icon: SquareStack,
      onClick: onEditEnvironment,
      disabled: !canUpdate || !canGetEnvironment,
    },
    {
      key: "transfer-project",
      label: t("database.transfer-project"),
      icon: ArrowRightLeft,
      onClick: () => onTransferProject?.(),
      disabled: !canUpdate,
      hidden: !onTransferProject,
    },
    {
      key: "unassign",
      label: t("database.unassign"),
      icon: Unlink,
      onClick: () => onUnassign?.(),
      disabled: !canUpdate,
      hidden: !onUnassign,
    },
  ];

  return (
    <SelectionActionBar
      count={databases.length}
      label={t("database.selected-n-databases", { n: databases.length })}
      onClear={() => onClearSelection?.()}
      actions={actions}
    />
  );
}
```

> Note: `onClearSelection` is added as an **optional** new prop. The three existing call sites (`DatabasesPage`, `InstanceDetailPage`, `ProjectDatabasesPage`) compile unchanged because they don't pass it. We thread it through in Task 7 so ✕/Escape actually clear the upstream selection. Until then the clear button is a visible no-op.

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`

Expected: no new errors. Both removed-import warnings (e.g. `Button`) and any external-call-site signatures pass cleanly.

- [ ] **Step 3: Lint and format**

Run: `pnpm --dir frontend fix`

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add -u frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx
git commit -m "refactor(react): rebuild DatabaseBatchOperationsBar on SelectionActionBar"
```

---

## Task 6: Migrate `InstancesPage` batch bar

**Files:**

- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx` (delete local `BatchOperationsBar` at lines 591–644; replace usage at line 1273; keep `SyncDropdown` lines 539–589)

- [ ] **Step 1: Delete the local `BatchOperationsBar` function**

Remove the function body for `BatchOperationsBar` (lines 591–644). **Keep** `SyncDropdown` (lines 539–589) — it's reused.

- [ ] **Step 2: Add the import**

In the imports block, add:

```tsx
import {
  SelectionActionBar,
  type SelectionAction,
} from "@/react/components/SelectionActionBar";
```

- [ ] **Step 3: Replace the bar usage**

Find the JSX usage of `BatchOperationsBar` (around line 1273). It looks like:

```tsx
<BatchOperationsBar
  selectedInstances={selectedInstanceList}
  syncing={syncing}
  onSync={handleBatchSync}
  onEditEnvironment={() => setShowEditEnvSheet(true)}
  onAssignLicense={() => setShowAssignLicenseSheet(true)}
  showAssignLicense={...}
/>
```

(Confirm exact prop names from the file — they may differ slightly.)

Replace with the inline `SelectionActionBar` call:

```tsx
{(() => {
  const canSync = hasWorkspacePermissionV2("bb.instances.sync");
  const canUpdate = hasWorkspacePermissionV2("bb.instances.update");
  const actions: SelectionAction[] = [
    {
      key: "edit-env",
      label: t("database.edit-environment"),
      icon: SquareStack,
      onClick: () => setShowEditEnvSheet(true),
      disabled: !canUpdate,
    },
    {
      key: "assign-license",
      label: t("subscription.instance-assignment.assign-license"),
      icon: GraduationCap,
      onClick: () => setShowAssignLicenseSheet(true),
      disabled: !canUpdate,
      hidden: !showAssignLicenseAction, // rename from old `showAssignLicense` prop
    },
  ];
  return (
    <SelectionActionBar
      count={selectedInstanceList.length}
      label={t("instance.selected-n-instances", {
        n: selectedInstanceList.length,
      })}
      onClear={() => setSelectedInstanceNames(new Set())}
      actions={actions}
    >
      <SyncDropdown
        disabled={!canSync || syncing}
        onSync={handleBatchSync}
      />
    </SelectionActionBar>
  );
})()}
```

> The exact local variable names (`selectedInstanceList`, `setSelectedInstanceNames`, `syncing`, `handleBatchSync`, `showAssignLicenseAction`, `setShowEditEnvSheet`, `setShowAssignLicenseSheet`) **must be confirmed by reading the file**. The old `BatchOperationsBar` already references these — copy whatever names it uses. Do not introduce new state variables.

- [ ] **Step 4: Sanity-check the i18n variable name still matches**

Run: `grep -n "selected-n-instances\|selected-n-databases" frontend/src/locales/en-US.json`

Expected (per current code): both keys declare the variable as `n` (e.g. `"{n} instance selected | {n} instances selected"`). The plan's calls in Step 3 (above) and in Task 5 already use `{ n: ... }`. If a future change has renamed the variable, mirror it in both call sites; mismatched names render the literal `{n}` in the UI.

- [ ] **Step 5: Type-check**

Run: `pnpm --dir frontend type-check`

Expected: no new errors. Remove now-unused imports (e.g. the file may have `useClickOutside` only used by the deleted bar — check and prune).

- [ ] **Step 6: Lint, format, run tests**

Run: `pnpm --dir frontend fix && pnpm --dir frontend test InstancesPage`

Expected: clean.

- [ ] **Step 7: Manual smoke**

`pnpm --dir frontend dev` → Instances page → select instances → confirm:
- Floating pill at bottom; SyncDropdown is the leftmost action node, opens its menu correctly.
- Edit Environment / Assign License (when applicable) appear.
- ✕ and Escape clear the selection.

- [ ] **Step 8: Commit**

```bash
git add -u frontend/src/react/pages/settings/InstancesPage.tsx
git commit -m "refactor(react): migrate InstancesPage batch bar to SelectionActionBar"
```

---

## Task 7: Wire `onClearSelection` through the database batch-bar callers

**Files:**

- Modify: `frontend/src/react/pages/settings/DatabasesPage.tsx` (line 479 area)
- Modify: `frontend/src/react/pages/settings/InstanceDetailPage.tsx` (line 427 area)
- Modify: `frontend/src/react/pages/project/ProjectDatabasesPage.tsx` (line 412 area)

> Why: in Task 5 we added `onClearSelection?: () => void`. Without it, the ✕ / Escape on the Database bar clears nothing. We pass the existing selection-clearing setter through.

- [ ] **Step 1: For each of the three pages, find the existing selection state**

Each page already maintains a selection state (whatever Set/array it passes to `selectedDatabases` or filters into `databases`). Locate the setter that clears it.

- [ ] **Step 2: Pass `onClearSelection` to `DatabaseBatchOperationsBar`**

For each call site, add the prop:

```tsx
<DatabaseBatchOperationsBar
  databases={selectedDatabaseList}
  /* ...existing props... */
  onClearSelection={() => setSelectedDatabaseNames(new Set())}
/>
```

(Use whatever the existing setter is — `setSelectedDatabaseNames`, `setSelectedDatabases`, etc.)

- [ ] **Step 3: Type-check, lint, manual smoke**

```
pnpm --dir frontend type-check
pnpm --dir frontend fix
```

Smoke: Databases / Instance detail / Project databases pages — select rows, click ✕ on the floating pill, confirm rows deselect.

- [ ] **Step 4: Commit**

```bash
git add -u frontend/src/react/pages/settings/DatabasesPage.tsx
git add -u frontend/src/react/pages/settings/InstanceDetailPage.tsx
git add -u frontend/src/react/pages/project/ProjectDatabasesPage.tsx
git commit -m "refactor(react): wire onClearSelection in DatabaseBatchOperationsBar callers"
```

---

## Task 8: Cleanup verification & full gates

- [ ] **Step 1: Confirm no `bg-blue-100` strip pattern remains in batch bars**

Run:

```bash
grep -rn "bg-blue-100" frontend/src/react/pages frontend/src/react/components/database
```

Expected: zero matches. If any remain in unrelated files (e.g. an info banner unrelated to selection), leave them.

- [ ] **Step 2: Confirm no leftover local `BatchOperationsBar` definitions**

Run:

```bash
grep -rn "function BatchOperationsBar" frontend/src/react
```

Expected: zero matches. (The shared component is named `SelectionActionBar`; the wrapper component is `DatabaseBatchOperationsBar`.)

- [ ] **Step 3: Confirm no `useClickOutside` orphans**

If the old InstancesPage `BatchOperationsBar` was the only consumer of `useClickOutside` in that file, the import should now be unused. Run:

```bash
grep -n "useClickOutside" frontend/src/react/pages/settings/InstancesPage.tsx
```

If it appears only in the import, delete the import line.

- [ ] **Step 4: Run the full frontend gate suite**

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
node frontend/scripts/check-react-layering.mjs
```

Expected:
- `fix`: no errors; may rewrite formatting.
- `check`: passes (CI-equivalent validation, includes the layering scanner).
- `type-check`: passes.
- `test`: passes; `SelectionActionBar.test.tsx` is in the run.
- Layering scanner: passes — no raw `z-index`, no `document.body` portals, no forbidden global overlay classes.

- [ ] **Step 5: Final commit (only if any tooling-driven fixes were applied)**

If `fix` produced changes:

```bash
git add -u
git commit -m "chore(react): apply formatter / lint fixups after SelectionActionBar migration"
```

If no changes, skip this step.

- [ ] **Step 6: Push & open PR (optional — at user's discretion)**

This plan does not auto-push. The branch is ready for `gh pr create`. Per `docs/pre-pr-checklist.md`, walk through the breaking-change review and SonarCloud properties before creating the PR.

---

## Summary of commits produced

1. `i18n: add common.batch-actions and common.clear-selection keys`
2. `test: failing test for SelectionActionBar (component to follow)`
3. `feat(react): add SelectionActionBar (floating selection toolbar)`
4. `refactor(react): migrate ProjectsPage batch bar to SelectionActionBar`
5. `refactor(react): rebuild DatabaseBatchOperationsBar on SelectionActionBar`
6. `refactor(react): migrate InstancesPage batch bar to SelectionActionBar`
7. `refactor(react): wire onClearSelection in DatabaseBatchOperationsBar callers`
8. (optional) `chore(react): apply formatter / lint fixups after SelectionActionBar migration`

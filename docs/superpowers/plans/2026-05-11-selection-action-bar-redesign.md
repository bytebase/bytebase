# SelectionActionBar Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop the SelectionActionBar bleeding into the sidebar (BYT-9445), make it usable on mobile (BYT-9446) — by changing it from viewport-fixed to sticky-in-main, adding a responsive "More" overflow dropdown, and shortening the label to a generic "N selected".

**Architecture:** Three coordinated changes inside one component plus six call-site label swaps and one new i18n key. The bar gains a `useSelectionMaxVisible` hook (1 / 3 / 5 inline buttons by Tailwind breakpoint) and splits its `actions` into inline + overflow; overflow goes into a single `MoreHorizontal` `DropdownMenu`. Positioning moves from `fixed bottom-6 left-1/2 -translate-x-1/2` to `sticky bottom-6 mx-auto w-fit` inside the page's main scroll container.

**Tech Stack:** React, `@base-ui/react` (DropdownMenu primitive), Tailwind CSS v4, `react-i18next`, vitest. Linear: [BYT-9445](https://linear.app/bytebase/issue/BYT-9445), [BYT-9446](https://linear.app/bytebase/issue/BYT-9446). Design doc: `docs/superpowers/specs/2026-05-11-selection-action-bar-redesign-design.md`.

**Testing note:** The overflow logic is unit-testable; positioning and responsive breakpoints are CSS/media-query behaviors best verified by browser QA. Task 4 covers the unit tests; Task 6 covers the cross-surface manual QA.

---

## File Structure

Files created or modified, by responsibility:

- `frontend/src/react/components/SelectionActionBar.tsx` — bar component. Add `useSelectionMaxVisible` hook, `maxVisibleActions` prop, overflow split + More dropdown, sticky positioning.
- `frontend/src/react/components/SelectionActionBar.test.tsx` — extend with overflow + destructive-in-menu coverage.
- `frontend/src/locales/en-US.json`, `es-ES.json`, `ja-JP.json`, `vi-VN.json`, `zh-CN.json` — new `common.n-selected` key.
- `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` — label swap.
- `frontend/src/react/components/database/TransferProjectSheet.tsx` — label swap (read-only display).
- `frontend/src/react/components/IssueTable.tsx` — label swap.
- `frontend/src/react/pages/settings/InstancesPage.tsx` — label swap.
- `frontend/src/react/pages/settings/ProjectsPage.tsx` — label swap.
- `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx` — label swap (read-only).

Not modified: `dropdown-menu.tsx` (no variant addition needed — destructive styling is applied via `className` on the consumer).

---

### Task 1: Add the `common.n-selected` i18n key in all locales

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/vi-VN.json`
- Modify: `frontend/src/locales/zh-CN.json`

- [ ] **Step 1: Add the key to each locale**

The key must live alphabetically inside the existing `"common": { ... }` object so the locale sorter doesn't have to reorder. Insert near `"n-items-selected"`.

For each file, find the `"common"` object and add this line in sorted position (after `"n-items-selected"` if present):

`en-US.json`:
```json
    "n-selected": "{n} selected",
```

`zh-CN.json`:
```json
    "n-selected": "已选择 {n} 项",
```

`ja-JP.json`:
```json
    "n-selected": "{n} 件選択中",
```

`es-ES.json`:
```json
    "n-selected": "{n} seleccionados",
```

`vi-VN.json`:
```json
    "n-selected": "Đã chọn {n}",
```

If `n-items-selected` is not present in a locale, insert `n-selected` in alphabetical order — the locale sorter will normalize.

- [ ] **Step 2: Run the locale sorter and check**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend run sort:i18n
```

Expected: "Locale sorter: no changes (30 file(s) checked)." If it reports changes, re-stage the formatted files.

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend check
```

Expected: passes (including the React-i18n cross-locale consistency check, which verifies the key exists in every locale).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/locales/en-US.json frontend/src/locales/es-ES.json frontend/src/locales/ja-JP.json frontend/src/locales/vi-VN.json frontend/src/locales/zh-CN.json
git commit -m "$(cat <<'EOF'
i18n: add common.n-selected for generic selection label

Used by SelectionActionBar after BYT-9446 — drops entity-specific
wording in favor of a uniform "N selected".

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Add `useSelectionMaxVisible` hook and overflow split in SelectionActionBar

**Files:**
- Modify: `frontend/src/react/components/SelectionActionBar.tsx`

- [ ] **Step 1: Add imports**

At the top of `frontend/src/react/components/SelectionActionBar.tsx`, replace the existing imports with the expanded list:

```tsx
import { MoreHorizontal, type LucideIcon } from "lucide-react";
import { useSyncExternalStore, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { Separator } from "@/react/components/ui/separator";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
```

- [ ] **Step 2: Add the `useSelectionMaxVisible` hook**

Add this hook above the existing `DESTRUCTIVE_TONE_CLASS` constant (i.e. just before `const DESTRUCTIVE_TONE_CLASS = ...`):

```tsx
const MQ_SM = "(min-width: 640px)";
const MQ_LG = "(min-width: 1024px)";

function subscribeMatchMedia(query: string) {
  return (cb: () => void) => {
    if (typeof window === "undefined") return () => {};
    const mql = window.matchMedia(query);
    mql.addEventListener("change", cb);
    return () => mql.removeEventListener("change", cb);
  };
}

function matchesMediaQuery(query: string): boolean {
  if (typeof window === "undefined") return true; // SSR: assume widest tier
  return window.matchMedia(query).matches;
}

/**
 * Returns the number of inline action buttons the bar should show before
 * collapsing the rest into a "More" dropdown:
 *   <  sm (<640px):   1
 *   sm — md (640–1024px): 3
 *   ≥ lg (≥1024px):   5
 *
 * SSR-safe: returns the largest tier on the first render, then resolves
 * to the actual viewport after hydration.
 */
function useSelectionMaxVisible(): number {
  const isLg = useSyncExternalStore(
    subscribeMatchMedia(MQ_LG),
    () => matchesMediaQuery(MQ_LG),
    () => true
  );
  const isSm = useSyncExternalStore(
    subscribeMatchMedia(MQ_SM),
    () => matchesMediaQuery(MQ_SM),
    () => true
  );
  if (isLg) return 5;
  if (isSm) return 3;
  return 1;
}
```

`useSyncExternalStore` is from `react` (already imported). The two `getServerSnapshot` callbacks return `true` so SSR renders at the widest tier (5 inline) — avoids a layout flash on hydration where actions briefly collapse into a More menu.

- [ ] **Step 3: Add `maxVisibleActions` prop**

In the `SelectionActionBarProps` interface, add the new prop right before `children`:

```tsx
export interface SelectionActionBarProps {
  count: number;
  label: string;
  allSelected: boolean;
  onToggleSelectAll: () => void;
  actions?: SelectionAction[];
  /**
   * Override the default responsive cap (1 / 3 / 5 by viewport).
   * Useful when a call site has only 1–2 actions and never wants a
   * More menu — set this to a high number like 99.
   */
  maxVisibleActions?: number;
  children?: ReactNode;
}
```

- [ ] **Step 4: Destructure the new prop and `useTranslation` in the function body**

Replace the `SelectionActionBar` function signature and the leading variable declarations with:

```tsx
export function SelectionActionBar({
  count,
  label,
  allSelected,
  onToggleSelectAll,
  actions,
  maxVisibleActions,
  children,
}: SelectionActionBarProps) {
  const { t } = useTranslation();
  const defaultMaxVisible = useSelectionMaxVisible();
  const maxVisible = maxVisibleActions ?? defaultMaxVisible;

  if (count <= 0) return null;

  const visibleActions = (actions ?? []).filter((a) => !a.hidden);
  const inlineActions = visibleActions.slice(0, maxVisible);
  const overflowActions = visibleActions.slice(maxVisible);
  // ...rest of the function (return ...) follows
}
```

The order matters: `useTranslation` and `useSelectionMaxVisible` must be called before the `count <= 0` early-return — React hooks rules. The split uses `slice()` so we don't mutate the input array.

- [ ] **Step 5: Render the inline + overflow split**

Replace the existing actions render block (the `<div className="flex items-center gap-x-3 min-w-0 overflow-x-auto">` and its children, plus the surrounding separator condition) with this. Keep the leading checkbox + label + separator above it unchanged.

```tsx
      {(inlineActions.length > 0 || overflowActions.length > 0 || children) && (
        <Separator orientation="vertical" className="h-5 shrink-0" />
      )}
      {/* Actions cluster — inline buttons + optional More dropdown.
          `min-w-0` lets flex shrink the container so any leftover overflow
          (e.g. very long custom children) can still scroll. */}
      <div className="flex items-center gap-x-3 min-w-0 overflow-x-auto">
        {inlineActions.map((action) => {
          const Icon = action.icon;
          const button = (
            <Button
              variant="outline"
              size="sm"
              disabled={action.disabled}
              onClick={action.onClick}
              className={cn(
                "rounded-full",
                action.tone === "destructive" && DESTRUCTIVE_TONE_CLASS
              )}
            >
              {Icon && <Icon className="size-4" aria-hidden />}
              {action.label}
            </Button>
          );
          if (action.disabled && action.disabledReason) {
            return (
              <Tooltip key={action.key} content={action.disabledReason}>
                {button}
              </Tooltip>
            );
          }
          return <div key={action.key}>{button}</div>;
        })}
        {overflowActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full"
                  aria-label={t("common.more")}
                >
                  <MoreHorizontal className="size-4" aria-hidden />
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              {overflowActions.map((action) => {
                const Icon = action.icon;
                const item = (
                  <DropdownMenuItem
                    key={action.key}
                    disabled={action.disabled}
                    onClick={action.onClick}
                    className={cn(
                      action.tone === "destructive" &&
                        "text-error data-highlighted:bg-error/10 data-highlighted:text-error"
                    )}
                  >
                    {Icon && <Icon className="size-4" aria-hidden />}
                    {action.label}
                  </DropdownMenuItem>
                );
                if (action.disabled && action.disabledReason) {
                  return (
                    <Tooltip key={action.key} content={action.disabledReason}>
                      <div>{item}</div>
                    </Tooltip>
                  );
                }
                return item;
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        {children}
      </div>
```

Notes on the rewrite:
- The Separator now shows whenever there's any cluster to the right (inline actions, More menu, or custom `children`) — previously it only checked `visibleActions.length > 0`.
- The More trigger uses `render={<Button ... />}` pattern, which is the Base UI / shadcn-style way to compose `DropdownMenuTrigger` with our styled Button. Confirm the existing `DropdownMenuTrigger` from `@/react/components/ui/dropdown-menu` supports `render` — if it instead expects `asChild` or wraps children directly, adapt. Check by reading the trigger function's signature at `frontend/src/react/components/ui/dropdown-menu.tsx`.
- Destructive in the menu uses `text-error` + a faint-red highlight on focus/hover (`data-highlighted` is the Base UI attribute the existing item already styles). The override matches the inline `DESTRUCTIVE_TONE_CLASS` semantically without re-using its exact classes (those target outline-button styling).
- Tooltip wrapping a `DropdownMenuItem` requires a wrapper `<div>` so the tooltip has a host element — `DropdownMenuItem` itself uses Base UI's `Menu.Item` which doesn't forward refs predictably for the Tooltip's positioning.

- [ ] **Step 6: Verify `common.more` exists in locales (or add it)**

```bash
grep -n "\"more\":" /Users/steven/Projects/bytebase/bb/frontend/src/locales/en-US.json
```

If `common.more` doesn't already exist, add it in the same way as `n-selected` (Task 1, Step 1). Suggested values:
- en-US: `"more": "More"`
- zh-CN: `"more": "更多"`
- ja-JP: `"more": "その他"`
- es-ES: `"more": "Más"`
- vi-VN: `"more": "Thêm"`

If you add `common.more`, include those locale files in the Task 2 commit.

- [ ] **Step 7: Run fix + type-check**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend fix
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend type-check
```

Expected: both pass.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/react/components/SelectionActionBar.tsx
# Include any locale files touched in Step 6:
# git add frontend/src/locales/*.json
git commit -m "$(cat <<'EOF'
feat(react-selection-bar): collapse overflow actions into More dropdown

Adds a responsive maxVisible ladder (1/3/5 at sm/md/lg) and a trailing
MoreHorizontal dropdown for any actions beyond the cap. Disabled-with-
reason and destructive tone are preserved inside the menu.

Part of BYT-9446 (mobile experience).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Switch positioning from fixed to sticky-inside-main

**Files:**
- Modify: `frontend/src/react/components/SelectionActionBar.tsx`

- [ ] **Step 1: Replace the bar's positioning classes**

In the outer `<div>` of `SelectionActionBar`, change:

```tsx
    <div
      className={cn(
        "fixed bottom-6 left-1/2 -translate-x-1/2 max-w-[90vw]",
        "flex items-center gap-x-3 rounded-full bg-background border border-control-border shadow-lg",
        "px-4 py-2",
        LAYER_SURFACE_CLASS
      )}
    >
```

to:

```tsx
    <div
      className={cn(
        "sticky bottom-6 mx-auto w-fit max-w-full",
        "flex items-center gap-x-3 rounded-full bg-background border border-control-border shadow-lg",
        "px-4 py-2",
        LAYER_SURFACE_CLASS
      )}
    >
```

Key changes:
- `fixed → sticky`: the bar joins the document flow; it sticks to `bottom-6` of the nearest scroll container.
- `left-1/2 -translate-x-1/2 → mx-auto w-fit`: horizontal centering now lives within the parent (the page's main column), not the viewport.
- `max-w-[90vw] → max-w-full`: parent (main column) drives the width cap; on extreme cases the bar still won't blow out its parent.

The `LAYER_SURFACE_CLASS` (z-index) stays — sticky elements still need to stack above table content.

- [ ] **Step 2: Verify sticky works against the real layout**

The bar's scroll container should be `#bb-layout-main` (see `frontend/src/react/components/DashboardBodyShell.tsx:141-150`: `flex-1 overflow-y-auto`). Open `frontend/src/react/pages/settings/DatabasesPage.tsx` and trace the JSX from `<DatabaseBatchOperationsBar>` upward: confirm no ancestor between the bar and `#bb-layout-main` has `overflow: hidden` (or any of `overflow-x-clip`, `overflow-y-clip`, `overflow-hidden`).

Read the page outer wrapper (line ~430+ in DatabasesPage.tsx — wherever the page-level `<div>` opens). If any ancestor has `overflow: hidden` on the vertical axis, sticky positioning will break and the bar will scroll away with the content.

- [ ] **Step 3: Run fix + type-check**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend fix
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend type-check
```

Expected: both pass. (CSS-only change, type-check unaffected, fix only formats.)

- [ ] **Step 4: Manual smoke test**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend dev
```

Open `/setting/database`. Select a database. The bar should appear pinned ~24px above the bottom of the visible content area, **horizontally centered in the area to the right of the sidebar** (not in the viewport midpoint).

Then open `/project/<any-project>/databases`. The bar should center between the project sidebar's right edge and the viewport right edge.

If the bar instead scrolls off-screen with the table, the sticky-ancestor check (Step 2) missed an `overflow: hidden`. Apply the fallback in Step 5 below; do not commit Step 1 alone.

- [ ] **Step 5 (fallback, only if Step 4 fails): CSS variable approach**

If sticky positioning doesn't work because of an `overflow: hidden` ancestor that can't be removed without breaking other layout, fall back to:

(a) Add `--main-content-left: 0px;` to `:root` in `frontend/src/assets/css/tailwind.css` (search for the existing `:root { ... }` block; insert the variable inside it).

(b) In `frontend/src/react/components/DashboardBodyShell.tsx`, find the outer container and set the variable on it via inline style:

```tsx
<div
  className="flex h-full flex-col overflow-hidden"
  style={{ "--main-content-left": isDesktop ? "208px" : "0px" } as React.CSSProperties}
>
```

(`208px` matches the existing `w-52` sidebar.)

(c) Revert Task 3 Step 1 to keep `fixed bottom-6 ... max-w-[90vw]`, but replace `left-1/2 -translate-x-1/2` with:

```tsx
"fixed bottom-6 max-w-[90vw]",
"-translate-x-1/2",
```

and set the `left` via inline style on the outer `<div>`:

```tsx
style={{ left: "calc(var(--main-content-left, 0px) + (100vw - var(--main-content-left, 0px)) / 2)" }}
```

Project pages that add their own sidebar (e.g. `ProjectSidebar`) would need to override the variable similarly — search `frontend/src/react/components/ProjectSidebar.tsx` for the parent layout and add an inline `style={{ "--main-content-left": "..." }}` if BYT-9445 still reproduces on project pages.

This fallback path is taken **only** if sticky truly can't work after Step 4 verification — don't pre-emptively adopt it.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/SelectionActionBar.tsx
git commit -m "$(cat <<'EOF'
fix(react-selection-bar): position sticky inside main content

Changes the bar from viewport-fixed centering to sticky-in-main-column
centering so it no longer bleeds into the dashboard or project sidebar.

BYT-9445

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If you took the fallback path in Step 5, expand the commit to include the additional files (`tailwind.css`, `DashboardBodyShell.tsx`, possibly `ProjectSidebar.tsx`).

---

### Task 4: Extend SelectionActionBar tests for overflow + destructive-in-menu

**Files:**
- Modify: `frontend/src/react/components/SelectionActionBar.test.tsx`

The existing tests cover render-on-count and basic action rendering. Add cases for the new overflow logic. The overflow split depends on the responsive hook, which reads `window.matchMedia`. In the test, mock `matchMedia` to force a known tier.

- [ ] **Step 1: Add a `matchMedia` mock helper at the top of the test file**

After the existing IS_REACT_ACT_ENVIRONMENT assignment (around line 9), add:

```tsx
function mockMatchMedia(matches: (query: string) => boolean) {
  window.matchMedia = ((query: string) => ({
    matches: matches(query),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  })) as unknown as typeof window.matchMedia;
}
```

- [ ] **Step 2: Add a new describe block with three tests**

After the existing tests inside `describe("SelectionActionBar", ...)`, before the closing `});`, add:

```tsx
  test("at lg breakpoint, shows first 5 actions inline and rest in More menu", async () => {
    mockMatchMedia(() => true); // both sm and lg match
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            { key: "a1", label: "Action 1", icon: Archive, onClick: () => {} },
            { key: "a2", label: "Action 2", icon: Archive, onClick: () => {} },
            { key: "a3", label: "Action 3", icon: Archive, onClick: () => {} },
            { key: "a4", label: "Action 4", icon: Archive, onClick: () => {} },
            { key: "a5", label: "Action 5", icon: Archive, onClick: () => {} },
            { key: "a6", label: "Action 6", icon: Archive, onClick: () => {} },
            { key: "a7", label: "Action 7", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    const inlineLabels = ["Action 1", "Action 2", "Action 3", "Action 4", "Action 5"];
    for (const label of inlineLabels) {
      expect(container.textContent ?? "").toContain(label);
    }
    // Actions 6 and 7 should not be in the rendered DOM (they're in a closed dropdown).
    expect(container.textContent ?? "").not.toContain("Action 6");
    expect(container.textContent ?? "").not.toContain("Action 7");
    // A More trigger button should exist.
    const moreButton = container.querySelector("button[aria-label]");
    expect(moreButton).not.toBeNull();
  });

  test("at < sm breakpoint, shows only 1 action inline and rest in More menu", async () => {
    mockMatchMedia(() => false); // neither sm nor lg matches
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={2}
          label="2 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            { key: "a1", label: "Inline Only", icon: Archive, onClick: () => {} },
            { key: "a2", label: "Overflow 1", icon: Archive, onClick: () => {} },
            { key: "a3", label: "Overflow 2", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    expect(container.textContent ?? "").toContain("Inline Only");
    expect(container.textContent ?? "").not.toContain("Overflow 1");
    expect(container.textContent ?? "").not.toContain("Overflow 2");
  });

  test("hidden actions don't count toward maxVisible", async () => {
    mockMatchMedia(() => false); // < sm — maxVisible = 1
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            { key: "h", label: "Hidden", icon: Archive, onClick: () => {}, hidden: true },
            { key: "v", label: "Visible", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    // Hidden action is filtered out, so Visible is the only "first" action and shows inline.
    expect(container.textContent ?? "").toContain("Visible");
    expect(container.textContent ?? "").not.toContain("Hidden");
    // No More menu because nothing overflows (only 1 visible action).
    const moreButton = container.querySelector("button[aria-label]");
    // The leading checkbox isn't aria-labeled in the bar, so any aria-label here = More trigger.
    expect(moreButton).toBeNull();
  });

  test("maxVisibleActions override skips the More menu entirely", async () => {
    mockMatchMedia(() => false); // < sm — would normally collapse to 1
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          maxVisibleActions={99}
          actions={[
            { key: "a1", label: "Action 1", icon: Archive, onClick: () => {} },
            { key: "a2", label: "Action 2", icon: Archive, onClick: () => {} },
            { key: "a3", label: "Action 3", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    expect(container.textContent ?? "").toContain("Action 1");
    expect(container.textContent ?? "").toContain("Action 2");
    expect(container.textContent ?? "").toContain("Action 3");
    const moreButton = container.querySelector("button[aria-label]");
    expect(moreButton).toBeNull();
  });
```

If the existing test file already imports `Archive` from lucide-react (it does in `SelectionActionBar.test.tsx`), the import is reusable; otherwise add it to the imports at the top.

- [ ] **Step 3: Verify the existing `useTranslation` mock still works**

The test file likely already mocks `react-i18next`. Confirm by grepping:

```bash
grep -n "react-i18next\|useTranslation" /Users/steven/Projects/bytebase/bb/frontend/src/react/components/SelectionActionBar.test.tsx
```

If the mock is missing, add at the top of the test file (after `import` block, before `describe`):

```tsx
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));
```

With this mock, `t("common.more")` returns the literal `"common.more"`, which is what the `aria-label` check above expects (any non-empty `aria-label` qualifies for the `querySelector("button[aria-label]")` assertion).

- [ ] **Step 4: Run the tests**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend test -- SelectionActionBar
```

Expected: all SelectionActionBar tests pass (existing + 4 new).

If a test fails because `DropdownMenuContent` is portaled (Base UI portals can mount outside `container` in tests), update the relevant assertions to query against `document.body` instead of `container` — the test for "at lg breakpoint" checks for `Action 6/7` NOT present, which works regardless of portal. The "More button exists" check uses `container.querySelector(...)`; if the trigger ends up portaled, switch to `document.body.querySelector(...)`. In practice Base UI's DropdownMenu trigger stays in-place; only the content is portaled.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/SelectionActionBar.test.tsx
git commit -m "$(cat <<'EOF'
test(react-selection-bar): cover responsive overflow + maxVisible override

Adds unit cases for the 1/3/5 ladder, hidden-action filtering, and the
maxVisibleActions=99 escape hatch.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Swap entity-specific labels to `common.n-selected` in all call sites

**Files:**
- Modify: `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx`
- Modify: `frontend/src/react/components/database/TransferProjectSheet.tsx`
- Modify: `frontend/src/react/components/IssueTable.tsx`
- Modify: `frontend/src/react/pages/settings/InstancesPage.tsx`
- Modify: `frontend/src/react/pages/settings/ProjectsPage.tsx`
- Modify: `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`

The translation function signature is `t(key, params)` and the existing keys use `{n}` or `{count}`. The new key uses `{n}` — adjust the param object accordingly.

- [ ] **Step 1: DatabaseBatchOperationsBar**

In `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx`, find:

```tsx
      label={t("database.selected-n-databases", { n: databases.length })}
```

Replace with:

```tsx
      label={t("common.n-selected", { n: databases.length })}
```

- [ ] **Step 2: TransferProjectSheet**

In `frontend/src/react/components/database/TransferProjectSheet.tsx`, find:

```tsx
            {t("database.selected-n-databases", { n: databases.length })}
```

Replace with:

```tsx
            {t("common.n-selected", { n: databases.length })}
```

- [ ] **Step 3: IssueTable**

In `frontend/src/react/components/IssueTable.tsx`, find:

```tsx
      label={`${issues.length} ${t("common.selected")}`}
```

Replace with:

```tsx
      label={t("common.n-selected", { n: issues.length })}
```

(`common.selected` was a non-existent key and was falling back to the literal — this is a quiet bug fix.)

- [ ] **Step 4: InstancesPage**

In `frontend/src/react/pages/settings/InstancesPage.tsx`, find:

```tsx
            label={t("instance.selected-n-instances", {
```

The wrapped call spans 2–3 lines; replace the whole call (typically `t("instance.selected-n-instances", { n: selectedNames.size })`) with:

```tsx
            label={t("common.n-selected", { n: selectedNames.size })}
```

- [ ] **Step 5: ProjectsPage**

In `frontend/src/react/pages/settings/ProjectsPage.tsx`, find:

```tsx
          label={t("project.batch.selected", {
            count: selectedProjectList.length,
          })}
```

Replace with:

```tsx
          label={t("common.n-selected", { n: selectedProjectList.length })}
```

(Note the param key change: `count` → `n`.)

- [ ] **Step 6: ProjectSyncSchemaPage**

In `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx`, find:

```tsx
            {t("database.selected-n-databases", { n: selected.size })}
```

Replace with:

```tsx
            {t("common.n-selected", { n: selected.size })}
```

- [ ] **Step 7: Run fix + type-check + tests**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend fix
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend type-check
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend test
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx \
        frontend/src/react/components/database/TransferProjectSheet.tsx \
        frontend/src/react/components/IssueTable.tsx \
        frontend/src/react/pages/settings/InstancesPage.tsx \
        frontend/src/react/pages/settings/ProjectsPage.tsx \
        frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx
git commit -m "$(cat <<'EOF'
refactor(react): unify selection labels as "N selected" via common.n-selected

Six call sites (databases bar, instances page, projects page, issues table,
transfer-project sheet, sync-schema page) drop their entity-specific
selection labels in favor of the generic key. Frees ~60px of horizontal
space on mobile — see BYT-9446.

Also incidentally fixes a missing key reference in IssueTable
(t("common.selected") never resolved).

BYT-9446

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Final validation + cross-surface manual QA

- [ ] **Step 1: Run the full frontend gate**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend check
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend type-check
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend test
```

Expected: all pass — including the cross-locale i18n check (verifies `common.n-selected` and any `common.more` you added exist in every locale).

- [ ] **Step 2: Dev server + cross-surface QA**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend dev
```

Sign in, then visit each surface and select 1+ rows to surface the bar. Verify:

| Surface | URL | Expected |
|---|---|---|
| Workspace Databases | `/setting/database` | Bar centers in main column (right of sidebar). |
| Workspace Instances | `/setting/instance` | Bar centers in main column. |
| Workspace Projects | `/setting/project` | Bar centers in main column. |
| Project Databases | `/project/<id>/databases` | Bar centers between project sidebar + dashboard sidebar and viewport right. |
| My Issues | `/my-issues` | Bar centers in main column; label reads "N selected". |
| Project Issues | `/project/<id>/issues` | Bar centers between project sidebar and viewport right; label "N selected". |

For each surface:
- Bar label reads "N selected" (English) or "已选择 N 项" (Chinese — verify with `/setting/general` language switch if needed).
- Selecting more than `maxVisible` actions: extra actions appear in the More dropdown trigger. Click the trigger to confirm the menu opens, items click through, disabled items show their tooltip, destructive items render red.
- Resize browser to ≤640px: only 1 inline action; rest in More menu.
- Resize browser to ~800px: 3 inline actions; rest in More menu.
- Resize browser to ≥1024px: up to 5 inline actions.

- [ ] **Step 3: Verify no leftover entity-specific selection keys are referenced**

```bash
grep -rn "selected-n-databases\|selected-n-instances\|project.batch.selected" /Users/steven/Projects/bytebase/bb/frontend/src --include="*.ts" --include="*.tsx"
```

Expected: no results. If any remain, they were missed in Task 5 — add a follow-up commit or amend.

The locale JSON entries (`database.selected-n-databases` etc.) are intentionally *not* removed in this PR — they may have non-bar callers and the i18n checker will flag them separately if they're truly unused. Don't grep the locale files; only source files.

- [ ] **Step 4: Verify the cross-locale i18n check is clean**

```bash
pnpm --dir /Users/steven/Projects/bytebase/bb/frontend check 2>&1 | grep -i "i18n\|locale"
```

Expected: `React i18n: all checks passed (missing keys, unused keys, cross-locale consistency).` and `Locale sorter: all 30 file(s) are normalized.`

- [ ] **Step 5: No commit unless QA found a polish issue**

If Steps 1–4 are clean and the manual QA pass surfaced no issues, no commit is needed — the work is on the branch ready for PR. Otherwise, commit polish fixes separately:

```bash
git status
git add <fixed-files>
git commit -m "fix(react-selection-bar): post-QA polish

BYT-9445 BYT-9446"
```

---

## Self-review

**Spec coverage:**

- Spec §1 "Position — sticky inside main content" → Task 3 (including the sticky-fallback CSS-variable path as Task 3 Step 5). ✓
- Spec §2 "Responsive `maxVisible` + More dropdown" → Task 2 (hook + render split). ✓
- Spec §3 "Generic 'N selected' label" → Task 1 (new key) + Task 5 (call sites). ✓
- Spec §4 "Component API summary" — `maxVisibleActions` prop → Task 2 Step 3. ✓
- Spec "Testing" section → Task 4 (unit) + Task 6 (manual QA). ✓
- Spec "File-level change summary" — all 9 files mapped to tasks. ✓

**Placeholder scan:** No "TBD" / "TODO" / "implement later" anywhere. Every code change has the complete code block. Task 3 Step 5 is conditional but spelled out fully, not a placeholder.

**Type consistency:** `maxVisibleActions: number` used identically in Task 2 (definition) and Task 4 (test invocation). `useSelectionMaxVisible(): number` matches. `SelectionAction` keys (`key`, `label`, `icon`, `onClick`, `disabled`, `disabledReason`, `tone`, `hidden`) match the existing interface — nothing renamed.

**Known unknowns to verify on-the-fly:**

- Task 2 Step 5 uses `DropdownMenuTrigger render={...}` — if the local primitive uses `asChild` instead, adapt. The plan tells the implementer to check `dropdown-menu.tsx`. The other parts of the code block don't depend on which prop is used.
- Task 2 Step 6 checks `common.more` existence — adds it if missing, includes the file in the commit. No guesswork.
- Task 3 verifies sticky against the real layout; Step 5 has the full fallback if sticky fails.
- Task 4 Step 4 notes the portaling edge case for `DropdownMenuContent` and how to adapt the assertion.

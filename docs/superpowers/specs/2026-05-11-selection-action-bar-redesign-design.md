# SelectionActionBar redesign: in-main positioning + responsive action overflow

- **Status:** Approved
- **Date:** 2026-05-11
- **Linear:** [BYT-9445](https://linear.app/bytebase/issue/BYT-9445/selection-bar-should-be-centered-in-main-page-not-extending-to-side), [BYT-9446](https://linear.app/bytebase/issue/BYT-9446/selection-bar-mobile-experience-needs-update)

## Problem

The `SelectionActionBar` (the floating pill that appears when one or more rows are selected) has two visual bugs:

1. **BYT-9445 — sidebar bleed.** The bar is `position: fixed bottom-6 left-1/2 -translate-x-1/2`, so it centers in the *viewport*. On pages with a sidebar (workspace pages have the 208px dashboard sidebar; project pages add a project sidebar on top), the visible main-content midpoint is right of viewport-midpoint, so the bar visibly extends into / under the sidebar at wider widths.
2. **BYT-9446 — mobile crush.** The leading cluster (checkbox + label like "3 databases selected") is `shrink-0` and consumes ~140px on its own. On narrow viewports, the first action button gets clipped or truncated, and `overflow-x-auto` on the actions cluster means hidden actions require horizontal scrolling — undiscoverable.

Both issues share a root cause: the bar is too wide for the space it has, in a way that the current layout can't accommodate.

## Affected files

- `frontend/src/react/components/SelectionActionBar.tsx` — the bar.
- Call sites that pass the `label`:
  - `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` — `database.selected-n-databases`
  - `frontend/src/react/components/database/TransferProjectSheet.tsx` — `database.selected-n-databases` (read-only display, also worth updating for consistency)
  - `frontend/src/react/pages/settings/InstancesPage.tsx` — `instance.selected-n-instances`
  - `frontend/src/react/pages/settings/ProjectsPage.tsx` — `project.batch.selected`
  - `frontend/src/react/components/IssueTable.tsx` — uses `${n} ${t("common.selected")}` (and the `common.selected` key currently doesn't exist — falls back to the literal key)
  - `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx` — `database.selected-n-databases` (read-only)
- New i18n key in `frontend/src/locales/*.json`.

## Design

Three coordinated changes — one component, several call sites, one new i18n key.

### 1. Position — `sticky` inside the main content scroll container

Change the bar's outer `<div>` classes from:

```
fixed bottom-6 left-1/2 -translate-x-1/2 max-w-[90vw]
```

to:

```
sticky bottom-6 mx-auto w-fit max-w-full
```

- The bar's existing JSX-parent is the page (e.g. `DatabasesPage.tsx`), which mounts inside `#bb-layout-main` (`flex-1 overflow-y-auto`, the page's scroll container — see `DashboardBodyShell.tsx:141-150`). `position: sticky bottom-6` makes the bar stick to the bottom of that scroll viewport, naturally constrained to the main content column's width.
- `mx-auto w-fit` centers the bar horizontally within its parent (the main content column), so it never extends into the sidebar.
- `max-w-full` keeps the safety belt — on extremely narrow viewports the bar can't blow out its parent.
- Keep `LAYER_SURFACE_CLASS` for the existing z-index policy.

**Edge case — short pages:** Sticky requires content tall enough for the bar's natural position to be below the scroll viewport. On a page with a near-empty table, the bar appears at the natural end of the content rather than pinned. This is acceptable: empty / near-empty selection bars are uncommon (the bar only renders when items are selected, which typically means a populated table) and the inline placement is visually reasonable.

**Implementation verification step:** If sticky breaks because of an `overflow: hidden` ancestor between the bar and `#bb-layout-main`, fall back to a CSS-variable approach:
- Introduce `--main-content-left: 0px` on `:root`.
- `DashboardBodyShell.tsx` updates it to `208px` on desktop (matching `w-52`). Project-sidebar pages add their sidebar width.
- The bar uses `position: fixed; left: calc(var(--main-content-left) + (100vw - var(--main-content-left)) / 2); transform: translate-x(-50%); bottom: 24px`.

This fallback is only adopted if the primary `sticky` approach fails verification — it is not the default.

### 2. Responsive `maxVisible` + "More" dropdown

Inside `SelectionActionBar`, compute a `maxVisible` count based on viewport breakpoints, then split `visibleActions` into "inline" (first `maxVisible`) and "overflow" (remainder). Render inline as today; render overflow inside a single trailing "More" `DropdownMenu`.

Breakpoint ladder (Tailwind defaults):

| Viewport | `maxVisible` |
|---|---|
| `< sm` (≤640px) | **1** |
| `sm` ≤ width < `lg` (640–1024px) | **3** |
| `≥ lg` (≥1024px) | **5** |

- Implementation: a small hook `useSelectionMaxVisible()` using `window.matchMedia` with `useSyncExternalStore`. No ResizeObserver. SSR-safe initial value is the largest tier (5) — initial paint shows everything inline, then the hook resolves on hydrate.
- Optional caller override:
  ```ts
  interface SelectionActionBarProps {
    // ...existing...
    /** Override the default responsive cap. Useful when a call site has
     *  only 1–2 actions and never wants a More menu. */
    maxVisibleActions?: number;
  }
  ```
- Counting: `hidden: true` actions are filtered out *before* the cap is applied — they don't count toward `maxVisible`.
- The "More" dropdown is rendered only when `overflow.length > 0`. The trigger is a `Button variant="outline" size="sm" className="rounded-full"` containing a `MoreHorizontal` icon (lucide), no label (icon-only, accessible label via `aria-label={t("common.more")}`).
- The dropdown menu reuses `DropdownMenu` from `@/react/components/ui/dropdown-menu`. Each overflow `SelectionAction` becomes a `DropdownMenuItem`:
  - Icon (if present) at left.
  - Label.
  - `disabled` applied; if `disabledReason` set, wrap item in a tooltip — same UX as the inline disabled button.
  - `tone: "destructive"` applies the same red text/border, restyled to fit the menu item — see "Destructive in dropdown" below.
- Custom `children` rendered by call sites (e.g. InstancesPage's split-dropdown Sync) stay at the end of the inline cluster and are never collapsed into the More menu — they're outside the `actions` array and the bar can't introspect them. Call sites that want their custom child to participate in the overflow rule should pass it through the `actions` array as a regular SelectionAction.

**Destructive in dropdown:** `DropdownMenuItem` exposes a `variant` prop in shadcn/Base UI's default config (`default | destructive`). If our local `dropdown-menu.tsx` already supports a destructive variant, use it. Otherwise, apply a className override mirroring the existing `DESTRUCTIVE_TONE_CLASS` translated to menu-item context (red text, red focus background). Verify during implementation; if neither path works cleanly, the planning step adds a small variant to our `dropdown-menu.tsx`.

### 3. Generic "N selected" label

Drop the entity name from the bar label everywhere. Add a single i18n key:

- **New key:** `common.n-selected` → `"{n} selected"` (English) / `"已选择 {n} 项"` (Chinese) / equivalents for other locales.
- Each call site replaces its current key:
  - `database.selected-n-databases` → `common.n-selected` (DatabaseBatchOperationsBar.tsx, TransferProjectSheet.tsx, ProjectSyncSchemaPage.tsx)
  - `instance.selected-n-instances` → `common.n-selected` (InstancesPage.tsx)
  - `project.batch.selected` → `common.n-selected` (ProjectsPage.tsx)
  - The `${n} ${t("common.selected")}` pattern in IssueTable.tsx → `t("common.n-selected", { n })`
- Old per-entity keys are *not* removed in this PR — they may have non-bar usages (e.g. read-only summaries in sheets / wizards). The i18n unused-key checker will flag truly unused keys; we'll prune them separately if it does.

The leading cluster on the bar drops from ~140px → ~80px on mobile, freeing enough space for one full-label action button beside the More dropdown.

## Component API summary

`SelectionActionBar` props after this change:

```ts
interface SelectionActionBarProps {
  count: number;
  label: string;
  allSelected: boolean;
  onToggleSelectAll: () => void;
  actions?: SelectionAction[];
  /** Optional cap override. Defaults to responsive 1 / 3 / 5 ladder. */
  maxVisibleActions?: number;
  children?: ReactNode;
}
```

`SelectionAction` is unchanged.

## Testing

**Unit (`SelectionActionBar.test.tsx`):**

- With 7 visible actions and viewport `lg`, only 5 render inline; the 6th and 7th appear in the "More" dropdown menu.
- With 7 actions and viewport `< sm`, only 1 renders inline; remaining 6 appear in the menu.
- `hidden: true` actions are not counted toward `maxVisible` and never appear in the menu.
- `disabled` + `disabledReason` on an overflow action renders the menu item disabled with a tooltip.
- `tone: "destructive"` on an overflow action renders the menu item with destructive styling.
- `maxVisibleActions={99}` override: no More menu, all actions inline.
- `count={0}`: bar renders nothing (unchanged behavior).

**Manual QA:**

- Workspace pages — Databases, Instances, Projects, My Issues. Bar centers in main content; doesn't bleed into the dashboard sidebar.
- Project pages — Project Databases, Project Issues. Bar centers between the project sidebar + dashboard sidebar and the right edge.
- Resize from desktop → tablet → mobile:
  - Desktop (`≥lg`): up to 5 inline + More if more.
  - Tablet (`sm`–`md`): 3 inline + More if more.
  - Mobile (`<sm`): 1 inline + More if more.
- Label reads "X selected" everywhere (English + Chinese sanity check).
- More menu: disabled items show tooltip on hover; destructive items render red; clicking a menu item fires the same `onClick` as the inline counterpart.
- Sticky behavior: on a page with many rows, bar stays pinned ~24px above bottom while scrolling. On a near-empty selection, bar renders in document flow (acceptable per Edge case note above).

## Out of scope

- Icon-only mode for inline buttons (the More menu obviates this; if button labels still overflow on tablet at `sm` with 3 actions, we'll address in a follow-up).
- Per-action "pinned" / "always-overflow" flags. Call-site action order is the priority — earlier = stays inline longer.
- Removing the deprecated `database.selected-n-databases` / `instance.selected-n-instances` / `project.batch.selected` keys. They may have non-bar callers; a separate cleanup PR after the i18n unused-key checker flags them.
- Visual restyling of the bar (color, shadow, border).
- Centralizing sidebar widths as CSS variables. Only adopted as a fallback if sticky positioning doesn't work cleanly.

## File-level change summary

- `frontend/src/react/components/SelectionActionBar.tsx` — sticky positioning, responsive `maxVisible`, More dropdown, `maxVisibleActions` prop.
- `frontend/src/react/components/SelectionActionBar.test.tsx` — overflow + destructive + disabled coverage.
- `frontend/src/react/components/database/DatabaseBatchOperationsBar.tsx` — label key swap.
- `frontend/src/react/components/database/TransferProjectSheet.tsx` — label key swap (read-only display).
- `frontend/src/react/components/IssueTable.tsx` — label key swap.
- `frontend/src/react/pages/settings/InstancesPage.tsx` — label key swap.
- `frontend/src/react/pages/settings/ProjectsPage.tsx` — label key swap.
- `frontend/src/react/pages/project/ProjectSyncSchemaPage.tsx` — label key swap (read-only).
- `frontend/src/locales/*.json` — add `common.n-selected` in every locale file.

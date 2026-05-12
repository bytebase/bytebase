# BYT-9191 Truncated-Name Tooltip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show the full resource name in a hover tooltip when the name is truncated in the React Instances, Projects, Databases, and My Issues list tables. Unify the behavior through a single shared `EllipsisText` component that also complies with the React overlay layering policy.

**Architecture:** Extend `EllipsisText` (`frontend/src/react/components/ui/ellipsis-text.tsx`) with an optional `children` prop so it can wrap `HighlightLabelText`. Replace its hand-rolled `document.body` portal with Base UI tooltip primitives portaled into `getLayerRoot("overlay")`, restyled with semantic tokens, and gated by a `ResizeObserver`-driven `isTruncated` flag. Update three call sites (Projects, Databases, My Issues) to use it.

**Tech Stack:** React, TypeScript, Base UI tooltip (`@base-ui/react/tooltip`), Tailwind CSS v4, semantic color tokens, the project's `cn()` helper.

**Spec:** `docs/superpowers/specs/2026-05-11-byt-9191-truncated-name-tooltip-design.md`

---

## File Structure

- **Modify:** `frontend/src/react/components/ui/ellipsis-text.tsx` — rewrite to use Base UI tooltip, `ResizeObserver` overflow detection, and accept optional `children`.
- **Modify:** `frontend/src/react/components/ProjectTable.tsx` — wrap project name `HighlightLabelText` with `EllipsisText`.
- **Modify:** `frontend/src/react/components/database/DatabaseTableView.tsx` — replace raw `truncate` span on the database name with `EllipsisText`.
- **Modify:** `frontend/src/react/components/IssueTable.tsx` — wrap issue title `HighlightLabelText` with `EllipsisText`.
- **No change:** `frontend/src/react/pages/settings/InstancesPage.tsx` — already uses `EllipsisText` with the backward-compatible string form; the rewrite must keep that call working.

No new files. No test files (per spec — `EllipsisText` has no unit tests today, and none are required for this change).

---

## Task 1: Rewrite `EllipsisText` to use Base UI tooltip with `children` support

**Files:**
- Modify: `frontend/src/react/components/ui/ellipsis-text.tsx` (full rewrite)

- [ ] **Step 1: Confirm baseline file contents**

Read `frontend/src/react/components/ui/ellipsis-text.tsx`. Expected: 68 lines, exports `EllipsisText({ text, className })`, uses `createPortal(..., document.body)` with `z-50`. This is what we're replacing.

- [ ] **Step 2: Replace the file with the new implementation**

Overwrite `frontend/src/react/components/ui/ellipsis-text.tsx` with exactly:

```tsx
import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

interface EllipsisTextProps {
  readonly text: string;
  readonly className?: string;
  readonly children?: ReactNode;
}

/**
 * Renders text (or `children`) with CSS truncation. Shows a tooltip with the
 * full `text` only when the rendered content is actually overflowing.
 *
 * `text` is the source-of-truth string used as both the default rendered
 * content and the tooltip body. Pass `children` to render a richer node
 * (e.g. <HighlightLabelText>) while still tooltip-ing the plain text.
 */
export function EllipsisText({ text, className, children }: EllipsisTextProps) {
  const ref = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const check = () => {
      setIsTruncated(el.scrollWidth > el.clientWidth);
    };
    check();
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => ro.disconnect();
  }, [text, children]);

  return (
    <BaseTooltip.Provider delay={300}>
      <BaseTooltip.Root open={isTruncated && open} onOpenChange={setOpen}>
        <BaseTooltip.Trigger
          render={
            <span ref={ref} className={cn("block truncate", className)} />
          }
        >
          {children ?? text}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal container={getLayerRoot("overlay")}>
          <BaseTooltip.Positioner
            side="top"
            sideOffset={4}
            className={LAYER_SURFACE_CLASS}
          >
            <BaseTooltip.Popup className="max-w-80 rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md whitespace-normal">
              {text}
              <BaseTooltip.Arrow className="fill-main" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}
```

Notes for the implementer:
- `BaseTooltip.Trigger` is given a `<span ref={ref} className="block truncate ...">` via the `render` prop, so the truncation container *is* the hover trigger. This matters: wrapping our span in another `inline-flex` span (the shape used by the shared `Tooltip` primitive at `frontend/src/react/components/ui/tooltip.tsx`) breaks truncation inside table cells.
- The tooltip is "armed" only when `isTruncated && open`. `open` is controlled internally so Base UI's hover state still works; the `&& isTruncated` guard means we never actually show the popup when the text fits.
- `ResizeObserver` re-checks on first paint and on every container resize (e.g. column drag, viewport resize). The `[text, children]` dependency re-checks when the content itself changes (e.g. table re-render with a different row).
- `whitespace-normal` on the popup lets long names wrap inside the 320px (`max-w-80`) limit. The shared `Tooltip` uses `max-w-56`; we bump to `max-w-80` because names are the *primary* content and we want a usable amount of horizontal room.
- The popup uses semantic tokens (`bg-main`, `text-main-text`, `fill-main`) so it themes correctly in dark mode.

- [ ] **Step 3: Type-check**

Run from the repo root: `pnpm --dir frontend type-check`
Expected: PASS. If it fails, the most likely cause is a missing/incorrect import or a Base UI tooltip prop name typo. Re-check imports against `frontend/src/react/components/ui/tooltip.tsx` for the correct Base UI module path.

- [ ] **Step 4: Layering policy check**

Run from the repo root: `node frontend/scripts/check-react-layering.mjs`
Expected: PASS, including for `ui/ellipsis-text.tsx`. The previous version of the file failed this check (raw `z-50` + `document.body` portal). If it still fails, the most likely cause is a leftover raw z-index or a `createPortal(document.body)`; both should be gone in the new code.

- [ ] **Step 5: Lint/format**

Run from the repo root: `pnpm --dir frontend fix`
Expected: completes without errors. Any auto-fixes should be limited to `ellipsis-text.tsx`.

- [ ] **Step 6: Manual verification — Instances page**

`InstancesPage.tsx:1063` already uses `<EllipsisText text={instance.title} />` (the backward-compatible string form). Start the dev server (`pnpm --dir frontend dev`), open Instances, find or create an instance with a long title, narrow the Name column until it truncates. Confirm:
- Hover → after ~300ms, a tooltip appears with the full name. ✓
- Resize the column wider so the name no longer truncates → hover → no tooltip. ✓
- Tooltip is not clipped by the table cell (i.e. it's portaling correctly). ✓

If the dev server cannot be started in this environment, say so explicitly and skip — type-check + layering check + the existing Instances call site continuing to compile is the strongest static guarantee.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/ui/ellipsis-text.tsx
git commit -m "$(cat <<'EOF'
refactor(react): rewrite EllipsisText with Base UI tooltip and children prop

Adds optional `children` prop so EllipsisText can wrap rich content
(e.g. HighlightLabelText) while still tooltip-ing a source-of-truth
plain string. Replaces the hand-rolled body portal with Base UI
tooltip primitives portaled into the overlay layer, removing the
direct document.body portal and raw z-50 that violated the React
overlay layering policy. Tooltip arms only when ResizeObserver
detects actual truncation.

Part of BYT-9191.
EOF
)"
```

---

## Task 2: Adopt `EllipsisText` on the Projects table

**Files:**
- Modify: `frontend/src/react/components/ProjectTable.tsx:232-235` (project name cell)

- [ ] **Step 1: Confirm baseline**

Read `frontend/src/react/components/ProjectTable.tsx` around lines 230-242. Expected current content for the name cell:

```tsx
<TableCell>
  <div className="flex items-center gap-x-2">
    <HighlightLabelText
      text={project.title || resourceId}
      keyword={keyword}
    />
    {project.state === State.DELETED ? (
      <Badge variant="warning" className="text-xs">
        {t("common.archived")}
      </Badge>
    ) : null}
  </div>
</TableCell>
```

- [ ] **Step 2: Add the `EllipsisText` import**

In `frontend/src/react/components/ProjectTable.tsx`, find the import block that already imports `HighlightLabelText` and add an import for `EllipsisText` next to it:

```tsx
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
```

Place it in the existing import group sorted alphabetically by path (same group as other `@/react/components/ui/...` imports if present, otherwise next to the `HighlightLabelText` import).

- [ ] **Step 3: Wrap the name cell with `EllipsisText`**

Replace the name cell block (lines 230-242) with:

```tsx
<TableCell>
  <div className="flex items-center gap-x-2 min-w-0">
    <EllipsisText text={project.title || resourceId} className="min-w-0 flex-1">
      <HighlightLabelText
        text={project.title || resourceId}
        keyword={keyword}
      />
    </EllipsisText>
    {project.state === State.DELETED ? (
      <Badge variant="warning" className="text-xs shrink-0">
        {t("common.archived")}
      </Badge>
    ) : null}
  </div>
</TableCell>
```

What changed:
- Wrapped `HighlightLabelText` inside `<EllipsisText>` with `text` set to the same string (it's the tooltip source). The `HighlightLabelText` is passed as `children` for rendering with search-match highlighting preserved.
- Added `min-w-0` to the flex row so its child (`EllipsisText`) can actually shrink below its content width (default flex children won't shrink past `min-content`).
- Added `min-w-0 flex-1` to `EllipsisText`'s `className` so it consumes the available width and allows truncation.
- Added `shrink-0` to the Archived `Badge` so it doesn't get squeezed when the name is long.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Lint/format**

Run: `pnpm --dir frontend fix`
Expected: completes without errors.

- [ ] **Step 6: Manual verification — Projects page**

Open Projects list. Find or temporarily create a project with a long title. Narrow the Name column. Confirm:
- Hover on truncated name → after ~300ms, full name appears in a tooltip. ✓
- Wide column (no truncation) → no tooltip on hover. ✓
- Type a keyword in the project search box that matches part of a long name → highlight still renders inside the truncated cell, tooltip still shows the full plain text. ✓
- Archived badge (if any archived project exists) still renders correctly and is not squeezed. ✓

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/ProjectTable.tsx
git commit -m "$(cat <<'EOF'
feat(react): show full project name on hover when truncated

Wraps the project-name HighlightLabelText in EllipsisText so a long
title that is clipped by the column width reveals on hover. Search
highlighting and the archived badge are preserved.

Part of BYT-9191.
EOF
)"
```

---

## Task 3: Adopt `EllipsisText` on the Databases table

**Files:**
- Modify: `frontend/src/react/components/database/DatabaseTableView.tsx` (database name cell, around line 164)

- [ ] **Step 1: Confirm baseline**

Read `frontend/src/react/components/database/DatabaseTableView.tsx` around lines 155-170. Expected current content for the name cell:

```tsx
render: (db) => {
  const instanceResource = getInstanceResource(db);
  return (
    <div className="flex items-center gap-x-2">
      <EngineIcon engine={instanceResource.engine} className="h-5 w-5" />
      <span className="truncate">
        {extractDatabaseResourceName(db.name).databaseName}
      </span>
    </div>
  );
},
```

- [ ] **Step 2: Add the `EllipsisText` import**

In `frontend/src/react/components/database/DatabaseTableView.tsx`, add:

```tsx
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
```

Place it in the same import group as other `@/react/components/ui/...` imports.

- [ ] **Step 3: Replace the raw `truncate` span**

Replace the name cell render with:

```tsx
render: (db) => {
  const instanceResource = getInstanceResource(db);
  const databaseName = extractDatabaseResourceName(db.name).databaseName;
  return (
    <div className="flex items-center gap-x-2 min-w-0">
      <EngineIcon engine={instanceResource.engine} className="h-5 w-5" />
      <EllipsisText text={databaseName} className="min-w-0 flex-1" />
    </div>
  );
},
```

What changed:
- Pulled `databaseName` to a local so we pass the same string to `EllipsisText` once.
- Replaced `<span className="truncate">{databaseName}</span>` with `<EllipsisText text={databaseName} ... />`.
- Added `min-w-0` to the flex row and `min-w-0 flex-1` to `EllipsisText` for the same flex-truncation reason as in Task 2.

**Out of scope** (do not touch in this task): the `project`, `instance`, and `address` columns in this file also use raw `truncate`. The spec explicitly excludes them. Leave them as-is.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Lint/format**

Run: `pnpm --dir frontend fix`
Expected: completes without errors.

- [ ] **Step 6: Manual verification — Databases page**

Open a project's Databases list (or the global Databases view). Find a database with a long name. Narrow the Name column. Confirm:
- Hover on truncated name → after ~300ms, full name appears. ✓
- Wide column → no tooltip. ✓
- EngineIcon still renders correctly to the left and is not squeezed by long names. ✓

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/database/DatabaseTableView.tsx
git commit -m "$(cat <<'EOF'
feat(react): show full database name on hover when truncated

Replaces the raw truncate span on the database-name column with
EllipsisText so a long database name reveals on hover. Other columns
that also use raw truncate (project, instance, address) are out of
scope for BYT-9191 and remain unchanged.

Part of BYT-9191.
EOF
)"
```

---

## Task 4: Adopt `EllipsisText` on the My Issues table

**Files:**
- Modify: `frontend/src/react/components/IssueTable.tsx` (issue title cell, around lines 640-665)

- [ ] **Step 1: Confirm baseline**

Read `frontend/src/react/components/IssueTable.tsx` around lines 640-665. Expected current content for the title link:

```tsx
<div className="flex items-center gap-x-1.5">
  <div className="h-6 flex justify-center items-center">
    <IssueStatusIcon status={issue.status} />
  </div>
  {issue.title ? (
    <a
      href={issueUrl}
      className="font-medium text-main text-base truncate hover:underline"
      onClick={(e) => e.stopPropagation()}
    >
      <HighlightLabelText
        text={issue.title}
        keyword={highlightWords}
      />
    </a>
  ) : (
    <a
      href={issueUrl}
      className="font-medium text-base truncate hover:underline italic text-control-placeholder"
      onClick={(e) => e.stopPropagation()}
    >
      {t("common.untitled")}
    </a>
  )}
  ...
</div>
```

- [ ] **Step 2: Add the `EllipsisText` import**

In `frontend/src/react/components/IssueTable.tsx`, add:

```tsx
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
```

In the same import group as other `@/react/components/ui/...` imports.

- [ ] **Step 3: Wrap the issue-title link with `EllipsisText`**

Replace the `issue.title ?` branch with:

```tsx
{issue.title ? (
  <a
    href={issueUrl}
    className="font-medium text-main text-base hover:underline min-w-0 flex-1 block"
    onClick={(e) => e.stopPropagation()}
  >
    <EllipsisText text={issue.title}>
      <HighlightLabelText
        text={issue.title}
        keyword={highlightWords}
      />
    </EllipsisText>
  </a>
) : (
  ...
)}
```

What changed in the title link:
- Removed `truncate` from the `<a>` className (the inner `EllipsisText` now owns truncation).
- Added `min-w-0 flex-1 block` so the `<a>` fills the available row width in its flex parent and allows children to shrink — required for `EllipsisText`'s inner `block truncate` span to actually clip.
- Wrapped `HighlightLabelText` inside `<EllipsisText text={issue.title}>`.

Leave the untitled `<a>` branch (the `else` arm) unchanged — `t("common.untitled")` is short and never truncates in practice; wrapping it is unnecessary churn.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Lint/format**

Run: `pnpm --dir frontend fix`
Expected: completes without errors.

- [ ] **Step 6: Manual verification — My Issues page**

Open the My Issues page. Find or create an issue with a long title. Confirm:
- Hover on truncated title → after ~300ms, full title appears. ✓
- Wide row (no truncation) → no tooltip. ✓
- Search/filter by a keyword that appears in the title → highlight renders inside the truncated cell, tooltip still shows the full plain text. ✓
- Clicking the title still navigates to the issue (i.e. wrapping with `EllipsisText` didn't break the link). ✓
- The `RiskLevelIcon` and labels to the right of the title still render in line and are not squeezed off-screen. ✓

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/IssueTable.tsx
git commit -m "$(cat <<'EOF'
feat(react): show full issue title on hover when truncated

Wraps the issue-title HighlightLabelText (inside the title link) in
EllipsisText so a long title that is clipped by the row width reveals
on hover. Link navigation and search highlighting are preserved.

Part of BYT-9191.
EOF
)"
```

---

## Task 5: Final cross-table verification

**Files:** none modified — verification only.

- [ ] **Step 1: Run the full frontend gate**

Run from the repo root, in order:

```bash
pnpm --dir frontend type-check
pnpm --dir frontend check
node frontend/scripts/check-react-layering.mjs
pnpm --dir frontend test
```

Expected: all four PASS. If `pnpm --dir frontend test` reports a pre-existing failure unrelated to these files, document it but do not let it block the PR.

- [ ] **Step 2: Visual cross-page sanity sweep**

Walk all four tables in the running dev server back-to-back and confirm the behavior is consistent:
- Instances list → long name, hover, tooltip after ~300ms.
- Projects list → same.
- Databases list → same.
- My Issues list → same.

The hover delay and tooltip styling should look identical across all four — that is the user-visible deliverable of "unified behavior" in BYT-9191.

- [ ] **Step 3: Confirm scope guardrails held**

`git diff main` should touch only these files:
- `frontend/src/react/components/ui/ellipsis-text.tsx`
- `frontend/src/react/components/ProjectTable.tsx`
- `frontend/src/react/components/database/DatabaseTableView.tsx`
- `frontend/src/react/components/IssueTable.tsx`
- (Plus the already-committed spec under `docs/superpowers/specs/`.)

If anything else changed, revert it — it's out of scope for this issue.

- [ ] **Step 4: Open the PR (separate step from this plan)**

Hand off to the user to drive PR creation per `docs/pre-pr-checklist.md`. Do not auto-create the PR.

---

## Notes on the React UI rules

- All wraps use the shared `EllipsisText` from `ui/`, not native controls or raw `truncate` spans. This conforms to the "use existing UI components first" rule in `frontend/AGENTS.md`.
- No new `dark:` overrides; the popup uses semantic tokens (`bg-main`, `text-main-text`, `fill-main`).
- No `space-x-*` / `space-y-*` added; existing `gap-x-*` patterns are preserved.
- No raw global `z-index` introduced; the popup uses `LAYER_SURFACE_CLASS` and Base UI's positioner.
- No new portal to `document.body`; the popup portals into `getLayerRoot("overlay")`.

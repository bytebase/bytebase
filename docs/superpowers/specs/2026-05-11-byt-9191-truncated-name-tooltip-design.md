# BYT-9191 — Unified hover-tooltip for truncated names in React list tables

Linear: https://linear.app/bytebase/issue/BYT-9191

## Problem

When a resource name/title is too long in a React list table, it is truncated with CSS ellipsis and the full text is unrecoverable — there is no hover affordance to reveal it. The Vue counterparts already show the full name on hover. The behavior is inconsistent across React list pages:

| Page | Name column | Tooltip on truncation? |
|---|---|---|
| Instances (`InstancesPage.tsx:1063`) | `<EllipsisText text={instance.title} />` | Yes |
| Projects (`ProjectTable.tsx:232-235`) | `<HighlightLabelText text={project.title \|\| resourceId} keyword={...} />` | No (raw `truncate`) |
| Databases (`DatabaseTableView.tsx:164`) | `<span className="truncate">{databaseName}</span>` | No (raw `truncate`) |
| My Issues (`IssueTable.tsx:645-654`) | `<HighlightLabelText text={issue.title} ... />` inside `truncate` | No |

A working component, `EllipsisText` (`frontend/src/react/components/ui/ellipsis-text.tsx`), already exists and is correct in principle (detects `scrollWidth > clientWidth`, shows tooltip only when actually truncated). Two problems block reuse on the other three tables:

1. Its API only accepts a plain `text: string`, so it cannot wrap `HighlightLabelText` (used by Projects and My Issues to highlight search-match substrings).
2. Its implementation violates the React overlay layering policy in `frontend/AGENTS.md`: it portals directly to `document.body` and uses raw `z-50`.

## Goal

A single shared component owns "name cell that may truncate" behavior, used consistently across the four React list tables in scope. Hovering a truncated name reveals the full string in a tooltip after a short delay; when the cell is wide enough, no tooltip appears.

## Non-goals

- Non-name columns in the same tables (address, environment, project label on Databases page, instance label on Databases page, etc.). They have the same raw-`truncate` issue but are explicitly out of scope for this issue.
- Other React list pages (environments, users, roles, etc.).
- Vue list pages — they already work correctly per the GIFs in BYT-9191.
- Renaming `EllipsisText`.

## Design

### Component change: `EllipsisText`

File: `frontend/src/react/components/ui/ellipsis-text.tsx`

**New API:**

```ts
interface EllipsisTextProps {
  readonly text: string;           // tooltip content + default rendered text
  readonly className?: string;
  readonly children?: ReactNode;   // optional: custom rendered content (e.g. <HighlightLabelText>)
}
```

**Rendering:**

- Renders a truncation container: `<span className={cn("block truncate", className)} ref={...}>`.
- If `children` is provided, render `children` inside the container; otherwise render `text`.
- The tooltip popup always uses `text` (the source-of-truth string).

**Overflow detection:**

- A `ResizeObserver` on the container tracks `scrollWidth > clientWidth` and sets local `isTruncated` state.
- Re-runs on first paint and on any container resize (e.g. column drag, viewport resize).

**Tooltip:**

- Use Base UI tooltip primitives directly (`@base-ui/react/tooltip`): `Provider`, `Root`, `Trigger`, `Portal`, `Positioner`, `Popup`.
- The shared `Tooltip` wrapper in `ui/tooltip.tsx` is not reused: its trigger is `<span className="inline-flex" />`, which breaks `block truncate` inside flex/table cells. `EllipsisText` needs the truncation container itself to be the trigger, rendered via `BaseTooltip.Trigger`'s `render` prop.
- Portal target: `getLayerRoot("overlay")` (resolves the overlay layering-policy violation in the current implementation).
- Positioner uses `LAYER_SURFACE_CLASS`.
- Popup uses semantic tokens (`bg-main`, `text-main-text`, etc.) — same style as the shared `Tooltip` for visual consistency.
- The tooltip is suppressed when `!isTruncated` by setting `open={false}` on `BaseTooltip.Root` in that case, leaving it uncontrolled otherwise.
- Hover delay: `delay={300}` on the `Provider`. Standard hover-affordance delay; matches typical "title-attribute"-style behavior.

**What goes away:**

- `createPortal(..., document.body)`.
- Raw `z-50`.
- Manual `mouseenter`/`mouseleave` handlers and `pos` state — Base UI's positioner handles placement.

### Call-site changes

| File | Change |
|---|---|
| `frontend/src/react/pages/settings/InstancesPage.tsx:1063` | No change — already uses `EllipsisText`. Confirms the API is backward-compatible. |
| `frontend/src/react/components/project/ProjectTable.tsx:232-235` | Wrap `HighlightLabelText` with `EllipsisText`. Pass `text={displayTitle}` to `EllipsisText`; pass the same string + keyword to `HighlightLabelText` as `children`. |
| `frontend/src/react/components/database/DatabaseTableView.tsx:164` | Replace the raw `<span className="truncate">` with `<EllipsisText text={databaseName} />`. |
| `frontend/src/react/components/issue/IssueTable.tsx:645-654` | Same pattern as `ProjectTable` — wrap `HighlightLabelText` with `EllipsisText`. Drop the outer `truncate` (`EllipsisText` already applies it). |

Call-site pattern for highlighted name cells:

```tsx
const displayTitle = project.title || resourceId;
return (
  <EllipsisText text={displayTitle}>
    <HighlightLabelText text={displayTitle} keyword={keyword} />
  </EllipsisText>
);
```

### Why not the alternatives

- **Add a `truncate` prop to `HighlightLabelText`.** Splits truncation behavior across two components — `EllipsisText` for plain-string cells, `HighlightLabelText` for highlighted ones. Two code paths means two opportunities for drift.
- **Build a new `TruncatedCell` primitive.** No real gain over option above; replaces a working component with a new one and adds churn at the existing Instances call site.

### Risk

Low. One shared component change with a backward-compatible API extension (new optional `children` prop) plus three small call-site changes. The `EllipsisText` rewrite to use the shared overlay layer is a behavioral improvement that's already mandated by the layering policy — fixing it now closes an existing violation.

## Testing

Manual on each of the 4 tables (Instances, Projects, Databases, My Issues):

1. Find a row with a long name. Tooltip appears on hover (~300ms delay) showing the full name.
2. Resize the column wider until the name no longer truncates. Tooltip no longer appears on hover.
3. For Projects and My Issues: type a keyword that matches part of a long name. The highlight still renders inside the truncated cell, and the tooltip still shows the full plain text.

Automated:

- `pnpm --dir frontend type-check`.
- `pnpm --dir frontend fix` and `pnpm --dir frontend check`.
- `node frontend/scripts/check-react-layering.mjs` — must pass (verifies the body-portal and raw `z-50` removal).
- `pnpm --dir frontend test` — `EllipsisText` does not currently have a unit test. No new test required for this change; existing tests must continue to pass.

## Open questions

None.

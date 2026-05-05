# Unify React list-page tables on the shared `Table` component

**Date:** 2026-05-06
**Status:** Approved

## Goal

Replace the remaining raw `<table>` callsites in React list pages with the
unified `Table` component at
`frontend/src/react/components/ui/table.tsx`, normalizing visual appearance to
the unified component's defaults.

The unified `Table` already provides striping, sortable headers (with
`ArrowUp/Down/UpDown` indicators), and column resizing via the
`ColumnResizeHandle`. Most React list surfaces already use it. A handful still
ship raw HTML — those are the targets.

## Scope

### In scope (4 callsites)

1. `frontend/src/react/components/ProjectTable.tsx` — used by `ProjectsPage`.
2. `frontend/src/react/components/AuditLogTable.tsx`.
3. `frontend/src/react/pages/settings/InstancesPage.tsx`.
4. The inline `ReleaseTable` in
   `frontend/src/react/pages/project/ProjectReleaseDashboardPage.tsx`
   (currently around lines 186–280).

### Out of scope

- Pages that already use the unified `Table` (changelog, revisions, groups, SQL
  review rules, instance roles, release files, approval steps,
  `DatabaseTableView`, etc.). They are fine as-is.
- Vue list pages. They will pick up the unified `Table` when they are migrated
  to React on their own schedules.
- Building a `DataTable` wrapper or adopting `@tanstack/react-table`. Premature
  given there are only four callsites left.
- Adding new APIs to the unified `Table` itself (no expansion API, no sticky
  header API). All four migrations should be possible with the current API.
  If implementation hits an unavoidable gap, stop and flag it rather than
  silently extending the component.

## Per-page changes

### 1. `ReleaseTable` (in `ProjectReleaseDashboardPage.tsx`)

Smallest case: no sort, no resize, no selection.

- Swap raw `<table>` / `<thead>` / `<tbody>` / `<tr>` / `<th>` / `<td>` for
  `Table` / `TableHeader` / `TableHead` / `TableBody` / `TableRow` / `TableCell`.
- Drop the custom `py-2 px-4` and accept the unified `px-4 py-3`.
- Row click handler stays on `<TableRow onClick=...>`.
- Default striping is fine.

### 2. `ProjectTable.tsx`

- Same markup swap.
- Drop the `bg-control-bg` header row and the rotating `ChevronDown`. The
  unified `ArrowUp/Down/UpDown` sort indicators take over via
  `sortable` / `sortActive` / `sortDir` / `onSort` props on `<TableHead>`.
- Current-project highlight (check icon) stays as a name-cell concern. No
  Table API change.
- Optional checkbox column stays as a leading `<TableHead>` / `<TableCell>`
  slot.

### 3. `AuditLogTable.tsx`

- Replace raw HTML.
- Column resize wires the existing width state through
  `<TableHead resizable onResizeStart=...>` (already supported by the unified
  `Table`).
- Header dropdown filters keep working — `<TableHead>` accepts arbitrary
  children.
- If a sticky header is currently in use, keep it as a wrapper-level CSS class
  on the surrounding scroll container. No Table API change.

### 4. `InstancesPage.tsx`

Largest case.

- Same markup swap with `sortable` / `resizable` props wired to existing state.
- Expandable data sources stay as in-cell rendering: the address cell shows
  either the single `hostPortOfInstanceV1(instance)` line or a vertical list
  of `hostPortOfDataSource(ds)` lines based on `expandedDataSources` state.
  Page-level logic only — no Table API change.
- The page builds column widths via `useColumnWidths` + `<colgroup>` and sets
  `style={{ width: totalWidth }}` + `table-fixed` on the table element.
  Pass these through to the unified `Table` (it spreads `...props` and
  accepts arbitrary children including `<colgroup>`).
- Striping default-on is fine; in-cell expansion does not interact with
  alternating row bands.

## Validation

For each migrated page:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test` for any tests touching the surface.
- Manual visual check via `pnpm --dir frontend dev`: load the page, verify
  sort, resize, expand, and row-click behavior; eyeball the visual diff against
  the current build.
- For `InstancesPage` specifically: expand a row's data sources, sort while
  expanded, resize a column, collapse and re-expand.

## PR strategy

One PR per page, smallest and safest first:

1. `ReleaseTable` — trivial swap, low blast radius.
2. `ProjectTable` — visual normalization, used by `ProjectsPage`.
3. `AuditLogTable` — column-resize wiring.
4. `InstancesPage` — column-resize plus expandable rows; most risk.

This keeps each diff small and reviewable, and a regression in one does not
block the others.

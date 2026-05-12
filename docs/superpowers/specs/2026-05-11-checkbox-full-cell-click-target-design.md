# Checkbox column: full-cell click target

- **Status:** Approved
- **Date:** 2026-05-11
- **Linear:** [BYT-9447](https://linear.app/bytebase/issue/BYT-9447/checkbox-area-is-sensitive-very-easy-to-click-on-the-database-instead) — "Checkbox area is sensitive, very easy to click on the database instead of checking the database"

## Problem

Selection columns across React tables render a 16px `Checkbox` inside a 48px-wide cell on a row whose `onClick` navigates away. Clicks on cell padding (between the checkbox and the cell edge) miss the checkbox's small `<span>` wrapper, bubble to the row, and navigate to the resource — instead of toggling selection. The Vue version exposed the entire checkbox column as the selection target; the React port regressed.

The bug is filed against the databases table, but the same pattern exists across the React selection tables.

## Affected surfaces

| # | File | Table shape | Current behavior |
|---|------|-------------|------------------|
| 1 | `frontend/src/react/components/database/DatabaseTableView.tsx` | Column-driven `<Table>` | Row navigates; cell-padding navigates |
| 2 | `frontend/src/react/pages/settings/InstancesPage.tsx` (inline `InstanceColumn[]`) | Column-driven `<Table>` | Same as #1 |
| 3 | `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx` | Inline JSX `<Table>` | Row navigates; cell-padding navigates |
| 4 | `frontend/src/react/components/IssueTable.tsx` | Flex `<div>` rows (no `<Table>`) | Row navigates; padding around checkbox navigates |
| 5 | `frontend/src/react/components/ProjectTable.tsx` | Inline JSX `<Table>` | Cell already has `onClick={e => e.stopPropagation()}` — safe, but clicking padding does nothing (not toggle) |

Out of scope (verified, no bug): `ProjectPlanDashboardPage` (no select column on the plan list), `BatchQuerySelect`, and the database picker inside the plan dashboard (row-click toggles selection — no navigation conflict).

## Design

Three local fixes, one per table shape. Same conceptual change ("the entire select cell is the click target"), but the surfaces are different enough that a shared helper would obscure more than unify. Each fix is 2–4 lines per call site.

### Shape A — column-driven (`DatabaseTableView`, `InstancesPage`)

Add two optional fields to the column type:

```ts
interface DatabaseColumn {
  // ...existing fields...
  onCellClick?: (db: Database, e: React.MouseEvent) => void;
  onHeaderClick?: (e: React.MouseEvent) => void;
}
```

In the render loop, wire them on `<TableCell>` and `<TableHead>`. Cells with `onCellClick` get `cursor-pointer`:

```tsx
<TableCell
  key={col.key}
  className={cn("overflow-hidden", col.cellClassName, col.onCellClick && "cursor-pointer")}
  onClick={col.onCellClick ? (e) => col.onCellClick!(db, e) : undefined}
>
```

```tsx
<TableHead
  // ...existing sortable / resizable wiring...
  className={cn(col.onHeaderClick && "cursor-pointer")}
  onClick={col.onHeaderClick}
>
```

In the select column definition:

```ts
{
  key: "select",
  title: (
    <Checkbox
      checked={someSelected ? "indeterminate" : allSelected}
      onCheckedChange={toggleSelectAll}
      onClick={(e) => e.stopPropagation()} // NEW — required once onHeaderClick is wired
    />
  ),
  defaultWidth: 48,
  onCellClick: (db, e) => {
    e.stopPropagation();
    toggleSelection(db.name);
  },
  onHeaderClick: (e) => {
    e.stopPropagation();
    toggleSelectAll();
  },
  render: (db) => (
    <Checkbox
      checked={selectedNames?.has(db.name) ?? false}
      onCheckedChange={() => toggleSelection(db.name)} // preserved — keyboard/space activation
      onClick={(e) => e.stopPropagation()}             // preserved — prevents cell from re-toggling
    />
  ),
}
```

`TableHead` already forwards consumer `onClick` (verified in `table.tsx:94-97`), running it before `onSort`. The select column isn't sortable, so only `onHeaderClick` fires. Both header and body checkboxes need `onClick={(e) => e.stopPropagation()}` to consume their own click before it bubbles to the parent `<TableHead>` / `<TableCell>` — without this, the checkbox's `onCheckedChange` and the parent's `onClick` both run on a direct checkbox click and toggle twice.

Reuse note: `onCellClick` / `onHeaderClick` are generic enough that future selection-style columns (e.g. quick-action toggles) can use the same fields without special-casing.

### Shape B — inline `<Table>` JSX (`DatabaseRevisionTable`, `ProjectTable`)

No column type to extend. Add the handlers directly on the select `<TableHead>` / `<TableCell>`:

```tsx
<TableHead
  className="w-12 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    toggleSelectAll();
  }}
>
  <Checkbox
    checked={someSelected ? "indeterminate" : allSelected}
    onCheckedChange={toggleSelectAll}
    onClick={(e) => e.stopPropagation()} // prevents double-toggle on direct checkbox click
  />
</TableHead>
```

```tsx
<TableCell
  className="w-12 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    toggleSelection(revision.name);
  }}
>
  <Checkbox
    checked={selectedNames.has(revision.name)}
    onCheckedChange={() => toggleSelection(revision.name)}
    onClick={(e) => e.stopPropagation()}
  />
</TableCell>
```

`ProjectTable.tsx` already has `onClick={(e) => e.stopPropagation()}` on its select cell — upgrade that handler to also toggle, and add `cursor-pointer`. The `disabled` (default project) case must still be respected: skip the toggle when the project is the default.

### Shape C — flex-div rows (`IssueTable`)

No `<TableCell>` to attach to. Wrap the `Checkbox` in a click-target `<div>` that extends past the 16px box:

```tsx
<div
  className="shrink-0 -my-3 py-3 pr-2 cursor-pointer"
  onClick={(e) => {
    e.stopPropagation();
    onToggleSelection();
  }}
>
  <Checkbox
    className="mt-1"
    checked={selected}
    onClick={(e) => e.stopPropagation()}
  />
</div>
```

`-my-3 py-3` matches the row's vertical padding (`py-3` on the parent), so the click target spans the full row height with no layout shift. `pr-2` widens it horizontally to the row's existing inter-column `gap-x-2`.

## Why this avoids double-toggle

The `Checkbox` component already wraps `Checkbox.Root` in a `<span>` when an `onClick` is passed (`frontend/src/react/components/ui/checkbox.tsx:67-76`). The span's `onClick={(e) => e.stopPropagation()}` consumes the click from inside the checkbox button before it bubbles further.

- **Click on the checkbox itself** → `Checkbox.Root` fires `onCheckedChange` → click bubbles to the wrapper span → span's `stopPropagation` consumes it → cell `onClick` never fires → **single toggle**.
- **Click on cell padding** → no checkbox involvement → cell `onClick` fires → toggle + `stopPropagation` blocks row navigation → **single toggle**.

The existing per-checkbox `onClick={(e) => e.stopPropagation()}` is load-bearing. It must remain on the inner `Checkbox` in every fix above.

## Testing (manual, per surface)

For each of the 5 surfaces:

1. Click directly on the checkbox → toggles selection, no navigation.
2. Click in the cell padding (≥10px from the checkbox edge, still inside the cell) → toggles selection, no navigation.
3. Click on row content (name cell, etc.) → navigates as before.
4. Click the header checkbox cell padding → toggles select-all.
5. Indeterminate state still resolves correctly (partial selection → click clears all; full selection → click clears; empty → click selects all).
6. `ProjectTable` only: click the select cell for the default project → no toggle (disabled), no navigation.

## Out of scope

- Extracting a shared `<SelectCell>` / `SelectionTable` primitive. The three shapes are too different; the fix is too small.
- Widening the Checkbox primitive's own hit area (would affect every checkbox in the app, including non-table uses — a larger UX decision).
- Refactoring `InstancesPage`'s inline `InstanceColumn[]` to share types with `DatabaseColumn`.

## File-level change summary

- `frontend/src/react/components/database/DatabaseTableView.tsx` — add `onCellClick` / `onHeaderClick` to `DatabaseColumn`, wire in render loop, set on select column.
- `frontend/src/react/pages/settings/InstancesPage.tsx` — same as above on `InstanceColumn`.
- `frontend/src/react/pages/project/database-detail/revision/DatabaseRevisionTable.tsx` — add `onClick` + `cursor-pointer` on select `<TableHead>` and `<TableCell>`.
- `frontend/src/react/components/ProjectTable.tsx` — upgrade existing cell `onClick` to also toggle (respect `disabled`), add `cursor-pointer`, add header `onClick`.
- `frontend/src/react/components/IssueTable.tsx` — wrap `Checkbox` in a click-target `<div>`. No header to mirror — `IssueTable` has no in-table select-all; the select-all is on the parent selection toolbar.
- `frontend/src/react/components/ui/checkbox.tsx` — **no change**.

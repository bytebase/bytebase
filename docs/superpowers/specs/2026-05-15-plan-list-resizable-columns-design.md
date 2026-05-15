# Plan list — resizable columns

## Background

The project plan list (`/projects/<id>/plans`) renders columns at fixed widths declared with Tailwind classes (`min-w-80`, `w-50`, `w-35`, `w-65`, `w-38`). The database list on the same project (`/projects/<id>/databases`) already supports drag-to-resize column headers via the shared `Table` primitive and `useColumnWidths` hook. We want the plan list to behave the same way.

## Goal

Bring the plan list to feature parity with the database list for column resizing:

- The user can drag the right edge of any plan-list column header to resize that column.
- Behavior, styling, and code path match `DatabaseTableView` exactly.

## Non-goals

- Column reorder (drag-and-drop column headers to change order).
- Row drag-to-reorder.
- Persistence of widths across remounts. The database list does not persist either; we match that.
- Any change to the shared infrastructure (`useColumnWidths`, `Table` primitive, `ColumnResizeHandle`).
- Any change to filter bar, paged footer, `AddSpecDrawer`, or other parts of `ProjectPlanDashboardPage`.

## Reference implementation

Already present in the codebase:

- `frontend/src/react/hooks/useColumnWidths.ts` — positional `widths[i]` state, `onResizeStart(colIndex, e)` mouse handler, document-level listener cleanup on unmount-mid-drag.
- `frontend/src/react/components/ui/table.tsx` — `TableHead` accepts `resizable` + `onResizeStart`, renders the `ColumnResizeHandle` on the right edge when both are set.
- `frontend/src/react/components/database/DatabaseTableView.tsx` — column-config pattern: `DatabaseColumn[]` with `defaultWidth`/`minWidth`/`resizable`/`render`, `<colgroup>` of `<col style={{ width }}>`, `Table` rendered with `className="table-fixed"` and `style={{ minWidth: totalWidth }}`.

This spec applies that pattern, unchanged, to the plan list.

## Scope of change

Only `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx` is touched. Inside the file, only `PlanTable` and `PlanRow` change. The page-level state, search, paging, drawer, helpers (`StatusTag`, `TaskStatusIcon`, `AddSpecDrawer`, selectors) are unchanged.

## Design

### Column config

Define `PlanColumn` and a column array in `PlanTable`, mirroring `DatabaseColumn`:

```ts
interface PlanColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  cellClassName?: string;
  render: (plan: Plan, ctx: PlanRowContext) => React.ReactNode;
}
```

The `ctx` argument carries per-row derived values that `PlanRow` already computes today (`creator`, `updateTimeTs`, `approvalTag`, `checkSummary`, `hasAnyCheck`, `isDeleted`, `showDraftTag`, `environmentStore`). `DatabaseTableView` doesn't need a context arg because its render functions only use the row object and a couple of utility imports; plan rows have heavier per-row state, so passing a context object is cleaner than duplicating the memos inside each `render`.

Default and minimum widths (Tailwind v3 default scale: `w-N` = `N * 4px`):

| Column   | Current Tailwind | `defaultWidth` (px) | `minWidth` (px) | `resizable` |
|----------|------------------|---------------------|-----------------|-------------|
| Name     | `min-w-80`       | 320                 | 200             | true        |
| Checks   | `w-50`           | 200                 | 120             | true        |
| Review   | `w-35`           | 140                 | 100             | true        |
| Stages   | `w-65`           | 260                 | 140             | true        |
| Updated  | `w-38`           | 152                 | 100             | true        |
| Creator  | `w-38`           | 152                 | 100             | true        |

Total default width = 1224px. The current `min-w-[1000px]` on the table is replaced with `style={{ minWidth: totalWidth }}`. The outer `overflow-x-auto` wrapper stays, so narrow viewports scroll horizontally just as today.

### `PlanTable`

- Build the `columns` array inside `PlanTable` (memoized on `t`).
- Call `useColumnWidths(columns)` to get `{ widths, totalWidth, onResizeStart }`.
- Render `<Table className="table-fixed" style={{ minWidth: totalWidth + "px" }}>`.
- Render `<colgroup>` with one `<col>` per column, `style={{ width: widths[i] + "px" }}`.
- Each `<TableHead>` gets `resizable={col.resizable}` and `onResizeStart={col.resizable ? (e) => onResizeStart(colIdx, e) : undefined}`.
- Pass the same `columns` array down to each `PlanRow` so headers and cells share a single column order/list.

### `PlanRow`

- Keep its current structure: the wrapping `<TableRow>` with `cn("cursor-pointer", isDeleted && "opacity-60")` and `onClick={onRowClick}`.
- Keep all existing memos (`approvalTag`, `checkSummary`, `planUrl`, `onRowClick`, `getRolloutStageStatus`, `creator`, `updateTimeTs`).
- Build a `ctx: PlanRowContext` object from those values.
- Replace the hand-written `<TableCell>` list with `columns.map(col => <TableCell key={col.key} className={cn("overflow-hidden", col.cellClassName)}>{col.render(plan, ctx)}</TableCell>)`.

Each `<TableCell>` carries `overflow-hidden` so a row's content can never push the column wider than its drag-set width. This matches `DatabaseTableView`.

### Cell content notes

- The Name cell uses `flex items-center gap-x-2 overflow-hidden` with `truncate` on the title span. The added cell-level `overflow-hidden` is redundant but harmless and keeps the pattern uniform.
- The Checks cell uses `flex items-center gap-3 flex-wrap`. `flex-wrap` is vertical wrap under `table-fixed` — works as today inside a fixed-width cell.
- The Stages cell uses `flex items-center gap-1 flex-wrap` with each stage as a small icon + label group. Same wrap semantics; no change.
- Updated, Review, Creator cells already have `whitespace-nowrap` on their text spans where appropriate; no change needed.

## Behavior

- Drag right edge of any plan-list column header → cursor becomes `col-resize`, body `user-select: none`, the column resizes live.
- Release → handler tears down document listeners, restores cursor and `user-select`.
- Unmount mid-drag (route change, etc.) → existing `useColumnWidths` cleanup tears down listeners (no change required, just by adoption).
- Reload, navigate away and back → widths reset to `defaultWidth`. Matches database list.
- Resize beyond `minWidth` is clamped by `useColumnWidths`.
- Horizontal page overflow → `overflow-x-auto` wrapper scrolls; same as today.

## Testing

Manual:

- On `/projects/<id>/plans`, drag the right edge of each resizable column header. Confirm the column resizes, cursor changes to `col-resize`, neighboring columns do not shift (only that column changes; total table width changes).
- Try to drag a column smaller than its `minWidth`. Confirm it stops at the minimum.
- Resize a column wide enough to push the table beyond the viewport. Confirm the table scrolls horizontally inside the page.
- Resize, then click a row. Confirm row navigation still works.
- Resize, then navigate away and come back. Confirm widths reset to defaults.
- Verify on a row mid-drag that closing the tab or navigating doesn't leave the page with `col-resize` cursor or `user-select: none` on `<body>` (the existing hook handles this; this is a regression check after adoption).

No automated tests are added. The resize hook is already exercised by the database list; we are reusing it unchanged.

## Risks

- **`table-fixed` changes cell sizing semantics.** Under `table-fixed`, cells no longer expand to fit content — they obey the `<col>` width. We mitigate by setting realistic `defaultWidth`s based on today's content, plus `truncate` / `flex-wrap` inside cells that already handle constrained widths, plus cell-level `overflow-hidden`.
- **`Stages` column wrap may look different.** Today the stages chunk uses `flex-wrap` inside a column sized to fit two stages. Under `table-fixed` with `defaultWidth: 260`, two stages fit on one line; more wrap to the next. Acceptable, and users can widen the column. No change unless the user reports it.
- **No persistence may frustrate users who resize often.** This is an existing limitation of the database list. If user feedback after rollout asks for persistence, that's a follow-up that should land on both lists at once, not on the plan list alone.

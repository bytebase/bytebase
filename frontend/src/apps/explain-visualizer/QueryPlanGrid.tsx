import { memo, useCallback, useMemo } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import {
  PLAN_INDENT_STEP,
  planGuideLevels,
  planIndentPixels,
} from "./plan-layout";
import {
  formatPlanCost,
  formatPlanCount,
  type PlanRow,
  type PlanTree,
  planRows,
  planSelfCostShare,
} from "./plan-model";
import { PlanCostShareBar, SECONDARY_COLUMN_CLASS } from "./plan-shared";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
}

const HEAD_CLASS = "sticky top-0 z-10 bg-control-bg text-xs leading-4";
const NUMERIC_CELL_CLASS =
  "py-2 text-right text-xs leading-4 text-control tabular-nums";

const GUIDE_LINE_CLASS = "absolute bg-control-border";

/**
 * Connector lines from a row up to its parent, drawn over the indent the row
 * already reserves so neither the row height nor the column widths move.
 *
 * Pure decoration: `aria-level` on the row is what states the nesting, and a
 * reader would gain nothing from a column of announced line segments.
 *
 * `left-4` is `TableCell`'s own horizontal padding, which is where the row's
 * indent starts; the two have to move together.
 */
function PlanTreeGuides({
  branchContinues,
}: {
  branchContinues: readonly boolean[];
}) {
  const levels = planGuideLevels(branchContinues);
  if (levels.length === 0) return null;
  return (
    <span
      aria-hidden="true"
      data-testid="plan-grid-guides"
      className="pointer-events-none absolute inset-y-0 left-4 flex"
    >
      {levels.map((continues, index) => {
        const last = index === levels.length - 1;
        // Every level but the last is a pass-through: a line only where an
        // ancestor still has rows to come. The last level turns into the row,
        // so it always arrives from above and carries on only when a sibling
        // follows.
        const vertical = last || continues;
        const toBottom = last ? continues : true;
        return (
          <span
            // A level has no identity apart from its position in the indent.
            key={index}
            className="relative shrink-0"
            style={{ width: PLAN_INDENT_STEP }}
          >
            {vertical ? (
              <span
                className={cn(
                  GUIDE_LINE_CLASS,
                  "left-1/2 w-px",
                  toBottom ? "inset-y-0" : "top-0 h-1/2"
                )}
              />
            ) : null}
            {last ? (
              <span
                className={cn(
                  GUIDE_LINE_CLASS,
                  "top-1/2 right-0 left-1/2 h-px"
                )}
              />
            ) : null}
          </span>
        );
      })}
    </span>
  );
}

/**
 * One plan node as a row.
 *
 * Memoized because a selection change re-renders the grid, and every other row
 * of a large plan is drawing exactly what it drew before.
 */
const PlanGridRow = memo(function PlanGridRow({
  row: { node, depth, branchContinues },
  index,
  tree,
  selected,
  tabStop,
  onSelect,
  onKeyDown,
}: {
  row: PlanRow;
  index: number;
  tree: PlanTree;
  selected: boolean;
  tabStop: boolean;
  onSelect: (id: string) => void;
  onKeyDown: (
    event: React.KeyboardEvent<HTMLTableRowElement>,
    id: string
  ) => void;
}) {
  return (
    <TableRow
      data-testid="plan-grid-row"
      data-plan-node-id={node.id}
      aria-selected={selected}
      aria-level={depth + 1}
      tabIndex={tabStop ? 0 : -1}
      onClick={() => onSelect(node.id)}
      onKeyDown={(event) => onKeyDown(event, node.id)}
      className={cn(
        "cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent",
        selected && "bg-accent/10 hover:bg-accent/15"
      )}
    >
      <TableCell className="px-3 py-2 text-right text-xs leading-4 text-control-light tabular-nums">
        {index + 1}
      </TableCell>

      <TableCell className="relative py-2">
        <PlanTreeGuides branchContinues={branchContinues} />
        <div
          data-testid="plan-grid-node"
          className="relative flex min-w-0 items-baseline gap-2"
          style={{ paddingLeft: planIndentPixels(depth) }}
        >
          <span
            data-testid="plan-grid-node-type"
            className={cn(
              "truncate text-sm leading-5",
              selected ? "font-medium text-accent" : "text-main"
            )}
          >
            {node.nodeType}
          </span>
          {node.subject ? (
            <span
              data-testid="plan-grid-node-subject"
              title={node.subject}
              className="min-w-0 flex-1 truncate text-xs leading-4 text-control-light"
            >
              {node.subject}
            </span>
          ) : null}
        </div>
      </TableCell>

      {tree.estimates.cost ? (
        <>
          <TableCell className="py-2 pr-4 pl-2">
            <div className="flex items-center justify-end gap-2">
              <PlanCostShareBar
                share={planSelfCostShare(node, tree)}
                className="w-10 shrink-0 sm:w-16"
              />
              <span className="text-xs leading-4 text-main tabular-nums">
                {formatPlanCost(node.selfCost)}
              </span>
            </div>
          </TableCell>
          <TableCell className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}>
            {formatPlanCost(node.totalCost)}
          </TableCell>
        </>
      ) : null}
      {tree.estimates.rows ? (
        <TableCell className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}>
          {formatPlanCount(node.rows)}
        </TableCell>
      ) : null}
      {tree.estimates.width ? (
        <TableCell className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}>
          {formatPlanCount(node.width)}
        </TableCell>
      ) : null}
    </TableRow>
  );
});

export function QueryPlanGrid({ tree, selectedId, onSelect }: Props) {
  const rows = useMemo(() => planRows(tree.root), [tree]);

  // Roving tab stop: Tab reaches the grid once and the arrow keys move within
  // it, rather than making every row of a 30-node plan its own tab stop.
  const tabStopId = rows.some((row) => row.node.id === selectedId)
    ? selectedId
    : rows[0]?.node.id;

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTableRowElement>, id: string) => {
      // Each row carries its own node id, so arrow navigation can walk
      // siblings instead of holding a ref per row.
      const moveTo = (element: Element | null | undefined) => {
        if (!(element instanceof HTMLTableRowElement)) return;
        const next = element.dataset.planNodeId;
        if (next === undefined) return;
        element.focus();
        onSelect(next);
      };

      const row = event.currentTarget;
      switch (event.key) {
        case "ArrowDown":
          moveTo(row.nextElementSibling);
          break;
        case "ArrowUp":
          moveTo(row.previousElementSibling);
          break;
        case "Home":
          moveTo(row.parentElement?.firstElementChild);
          break;
        case "End":
          moveTo(row.parentElement?.lastElementChild);
          break;
        case "Enter":
        case " ":
          onSelect(id);
          break;
        default:
          return;
      }
      event.preventDefault();
    },
    [onSelect]
  );

  return (
    <div
      data-testid="plan-grid-scroll"
      className="min-h-0 w-full flex-1 overflow-auto"
    >
      <Table className="table-fixed">
        <caption className="sr-only">
          Plan nodes in the order the planner nests them, parents before
          children.
        </caption>
        <TableHeader>
          <TableRow>
            <TableHead className={cn(HEAD_CLASS, "w-12 px-3 text-right")}>
              #
            </TableHead>
            <TableHead className={HEAD_CLASS}>Node</TableHead>
            {tree.estimates.cost ? (
              <>
                <TableHead
                  className={cn(HEAD_CLASS, "w-32 pl-2 text-right sm:w-48")}
                  title="Cost this node adds on top of its children; some engines also count an input's repeats here, as under a nested loop. The bar is that cost as a share of the plan."
                >
                  Added cost
                </TableHead>
                <TableHead
                  className={cn(
                    HEAD_CLASS,
                    SECONDARY_COLUMN_CLASS,
                    "text-right sm:w-28"
                  )}
                >
                  Total cost
                </TableHead>
              </>
            ) : null}
            {tree.estimates.rows ? (
              <TableHead
                className={cn(
                  HEAD_CLASS,
                  SECONDARY_COLUMN_CLASS,
                  "text-right sm:w-24"
                )}
                title="Estimated rows"
              >
                Rows
              </TableHead>
            ) : null}
            {tree.estimates.width ? (
              <TableHead
                className={cn(
                  HEAD_CLASS,
                  SECONDARY_COLUMN_CLASS,
                  "text-right sm:w-24"
                )}
                title="Estimated row width in bytes"
              >
                Width
              </TableHead>
            ) : null}
          </TableRow>
        </TableHeader>

        <TableBody striped={false}>
          {rows.map((row, index) => (
            <PlanGridRow
              key={row.node.id}
              row={row}
              index={index}
              tree={tree}
              selected={row.node.id === selectedId}
              tabStop={row.node.id === tabStopId}
              onSelect={onSelect}
              onKeyDown={handleKeyDown}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

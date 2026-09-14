import { useMemo } from "react";
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
  type PlanTree,
  planRows,
  planSelfCostShare,
} from "./plan-model";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
}

const HEAD_CLASS = "sticky top-0 z-10 bg-control-bg text-xs leading-4";
/** Estimates a phone-width reader can give up to keep the node column usable. */
const SECONDARY_COLUMN_CLASS = "hidden sm:table-cell";
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

export function QueryPlanGrid({ tree, selectedId, onSelect }: Props) {
  const rows = useMemo(() => planRows(tree.root), [tree]);

  // Roving tab stop: Tab reaches the grid once and the arrow keys move within
  // it, rather than making every row of a 30-node plan its own tab stop.
  const tabStopId = rows.some((row) => row.node.id === selectedId)
    ? selectedId
    : rows[0]?.node.id;

  // Each row carries its own node id, so arrow navigation can walk siblings
  // instead of holding a ref per row.
  const moveTo = (element: Element | null | undefined) => {
    if (!(element instanceof HTMLTableRowElement)) return;
    const id = element.dataset.planNodeId;
    if (id === undefined) return;
    element.focus();
    onSelect(id);
  };

  const handleKeyDown = (
    event: React.KeyboardEvent<HTMLTableRowElement>,
    id: string
  ) => {
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
  };

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
            <TableHead
              className={cn(HEAD_CLASS, "w-32 pl-2 text-right sm:w-48")}
              title="Cost the node adds on top of its children; the bar is that cost as a share of the plan"
            >
              Self cost
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
          </TableRow>
        </TableHeader>

        <TableBody striped={false}>
          {rows.map(({ node, depth, branchContinues }, index) => {
            const selected = node.id === selectedId;
            return (
              <TableRow
                key={node.id}
                data-testid="plan-grid-row"
                data-plan-node-id={node.id}
                aria-selected={selected}
                aria-level={depth + 1}
                tabIndex={node.id === tabStopId ? 0 : -1}
                onClick={() => onSelect(node.id)}
                onKeyDown={(event) => handleKeyDown(event, node.id)}
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

                <TableCell className="py-2 pr-4 pl-2">
                  <div className="flex items-center justify-end gap-2">
                    <span
                      aria-hidden="true"
                      className="h-1.5 w-10 shrink-0 overflow-hidden rounded-full bg-control-bg sm:w-16"
                    >
                      <span
                        data-testid="plan-grid-cost-bar"
                        className="block h-full rounded-full bg-warning"
                        style={{
                          width: `${planSelfCostShare(node, tree) * 100}%`,
                        }}
                      />
                    </span>
                    <span className="text-xs leading-4 text-main tabular-nums">
                      {formatPlanCost(node.selfCost)}
                    </span>
                  </div>
                </TableCell>

                <TableCell
                  className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}
                >
                  {formatPlanCost(node.totalCost)}
                </TableCell>
                <TableCell
                  className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}
                >
                  {formatPlanCount(node.rows)}
                </TableCell>
                <TableCell
                  className={cn(NUMERIC_CELL_CLASS, SECONDARY_COLUMN_CLASS)}
                >
                  {formatPlanCount(node.width)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

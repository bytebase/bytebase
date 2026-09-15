import { Info } from "lucide-react";
import type { ReactNode } from "react";
import { memo, useMemo } from "react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { planIndentPixels } from "./plan-layout";
import {
  formatPlanCost,
  formatPlanCount,
  formatPlanShare,
  type PlanTimelineRow,
  type PlanTree,
  planCostByOperation,
  planCostliestNodes,
  planHighlightIntensity,
  planSelfCostShare,
  planTimeline,
} from "./plan-model";
import {
  PlanCostShareBar,
  PlanMetric,
  SECONDARY_COLUMN_CLASS,
} from "./plan-shared";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
}

/**
 * Faintest a timeline span is drawn. A node that adds no cost of its own still
 * occupies a span of the plan, and hiding it would misreport the shape.
 */
const MIN_SPAN_OPACITY = 0.3;

/**
 * Narrowest a timeline span is drawn, as a fraction of the axis. A node whose
 * startup and total costs are the same spans nothing, and a bar of no width
 * would read as an absent row rather than an instant one.
 */
const MIN_SPAN_FRACTION = 0.005;

/** A span's placement on the axis, kept whole and inside the track. */
function spanGeometry(start: number, end: number) {
  const width = Math.min(1, Math.max(end - start, MIN_SPAN_FRACTION));
  return {
    left: `${Math.min(start, 1 - width) * 100}%`,
    width: `${width * 100}%`,
  };
}

/** Width of the label column once the row has room to put it beside the track. */
const LABEL_COLUMN_CLASS = "sm:w-56 sm:shrink-0";
/** Width of the trailing share column in the lists that carry one. */
const SHARE_COLUMN_CLASS = "w-12 shrink-0 text-right";

const ROW_BUTTON_CLASS =
  "h-auto w-full justify-start rounded-xs px-2 py-1.5 text-left font-normal";

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h2 className="text-base leading-6 font-semibold text-main">{title}</h2>
        {description ? (
          <p className="text-xs leading-4 text-control-light">{description}</p>
        ) : null}
      </div>
      {children}
    </section>
  );
}

function PlanTotals({ tree }: { tree: PlanTree }) {
  return (
    <div className="flex flex-col gap-3">
      <dl className="flex flex-wrap gap-4">
        <PlanMetric label="Nodes" value={formatPlanCount(tree.nodes.length)} />
        <PlanMetric
          label="Total estimated cost"
          value={formatPlanCost(tree.root.totalCost)}
        />
        {tree.root.rows === undefined ? null : (
          <PlanMetric
            label="Estimated rows returned"
            value={formatPlanCount(tree.root.rows)}
          />
        )}
      </dl>
      <p className="flex items-start gap-2 text-xs leading-4 text-control-light">
        <Info aria-hidden="true" className="mt-px size-3.5 shrink-0" />
        <span>
          The statement was planned but not run, so every figure here is an
          optimizer estimate rather than a measurement. Cost units are arbitrary
          and comparable only within this plan.
        </span>
      </p>
    </div>
  );
}

/**
 * One node as a span on the plan's cost axis.
 *
 * Memoized because a selection change re-renders the timeline, and every other
 * row of a large plan is drawing exactly what it drew before.
 */
const TimelineRow = memo(function TimelineRow({
  row,
  tree,
  selected,
  onSelect,
}: {
  row: PlanTimelineRow;
  tree: PlanTree;
  selected: boolean;
  onSelect: (id: string) => void;
}) {
  const { node, depth, start, end, share } = row;
  return (
    <li>
      <Button
        appearance="secondary"
        data-testid="plan-timeline-row"
        aria-pressed={selected}
        aria-label={`${[node.nodeType, node.subject].filter(Boolean).join(", ")}, cost ${formatPlanCost(node.startupCost)} to ${formatPlanCost(node.totalCost)}, ${formatPlanShare(share)} of plan cost`}
        onClick={() => onSelect(node.id)}
        className={cn(
          ROW_BUTTON_CLASS,
          "flex-col items-stretch gap-1 sm:flex-row sm:items-center sm:gap-3",
          selected && "bg-accent/10 hover:bg-accent/15"
        )}
      >
        <span
          className={cn(
            "flex min-w-0 items-baseline gap-2",
            LABEL_COLUMN_CLASS
          )}
          style={{ paddingLeft: planIndentPixels(depth) }}
        >
          <span
            data-testid="plan-timeline-node-type"
            className={cn(
              "truncate text-xs leading-4",
              selected ? "font-medium text-accent" : "text-main"
            )}
          >
            {node.nodeType}
          </span>
          <span className="min-w-0 flex-1 truncate text-xs leading-4 text-control-light">
            {node.subject ?? ""}
          </span>
          <span className="shrink-0 text-xs leading-4 text-control tabular-nums">
            {formatPlanShare(share)}
          </span>
        </span>
        <span
          aria-hidden="true"
          // `flex-1` only once the row is horizontal: in the stacked layout it
          // is the column's main axis and would flatten the track to nothing.
          className="relative h-2.5 w-full min-w-0 overflow-hidden rounded-xs bg-control-bg sm:flex-1"
        >
          <span
            data-testid="plan-timeline-span"
            className="absolute inset-y-0 rounded-xs bg-warning"
            style={{
              ...spanGeometry(start, end),
              opacity:
                MIN_SPAN_OPACITY +
                (1 - MIN_SPAN_OPACITY) *
                  planHighlightIntensity(node, tree, "cost"),
            }}
          />
        </span>
      </Button>
    </li>
  );
});

function CostTimeline({
  tree,
  selectedId,
  onSelect,
}: {
  tree: PlanTree;
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}) {
  const rows = useMemo(() => planTimeline(tree), [tree]);
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-3">
        <span
          aria-hidden="true"
          className={cn("hidden sm:block", LABEL_COLUMN_CLASS)}
        />
        <span
          data-testid="plan-timeline-axis"
          className="flex min-w-0 flex-1 items-baseline justify-between px-2 text-xs leading-4 text-control-light tabular-nums sm:px-0"
        >
          <span>0</span>
          <span>{formatPlanCost(tree.costBasis)}</span>
        </span>
      </div>
      <ol data-testid="plan-timeline" className="flex flex-col">
        {rows.map((row) => (
          <TimelineRow
            key={row.node.id}
            row={row}
            tree={tree}
            selected={row.node.id === selectedId}
            onSelect={onSelect}
          />
        ))}
      </ol>
    </div>
  );
}

function CostliestOperators({
  tree,
  selectedId,
  onSelect,
}: {
  tree: PlanTree;
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}) {
  const nodes = useMemo(() => planCostliestNodes(tree), [tree]);

  if (nodes.length === 0) {
    return (
      <p className="text-sm leading-5 text-control-light">
        No node in this plan adds an estimated cost of its own.
      </p>
    );
  }

  return (
    <ol data-testid="plan-costliest" className="flex flex-col">
      {nodes.map((node, index) => {
        const selected = node.id === selectedId;
        return (
          <li key={node.id}>
            <Button
              appearance="secondary"
              data-testid="plan-costliest-row"
              aria-pressed={selected}
              onClick={() => onSelect(node.id)}
              className={cn(
                ROW_BUTTON_CLASS,
                "items-baseline gap-3",
                selected && "bg-accent/10 hover:bg-accent/15"
              )}
            >
              <span className="w-4 shrink-0 text-xs leading-4 text-control-light tabular-nums">
                {index + 1}
              </span>
              <span className="flex min-w-0 flex-1 items-baseline gap-2">
                <span
                  className={cn(
                    "truncate text-sm leading-5",
                    selected ? "font-medium text-accent" : "text-main"
                  )}
                >
                  {node.nodeType}
                </span>
                <span className="min-w-0 flex-1 truncate text-xs leading-4 text-control-light">
                  {node.subject ?? ""}
                </span>
              </span>
              <span className="hidden shrink-0 text-xs leading-4 text-control tabular-nums sm:inline">
                {formatPlanCost(node.selfCost)}
              </span>
              <span
                className={cn(
                  SHARE_COLUMN_CLASS,
                  "text-xs leading-4 text-main tabular-nums"
                )}
              >
                {formatPlanShare(planSelfCostShare(node, tree))}
              </span>
            </Button>
          </li>
        );
      })}
    </ol>
  );
}

function CostByOperation({ tree }: { tree: PlanTree }) {
  const operations = useMemo(() => planCostByOperation(tree), [tree]);
  return (
    <Table className="table-fixed">
      <TableHeader>
        <TableRow>
          <TableHead className="text-xs leading-4">Operation</TableHead>
          <TableHead className="w-16 text-right text-xs leading-4">
            Count
          </TableHead>
          <TableHead
            className={cn(
              SECONDARY_COLUMN_CLASS,
              "text-right text-xs leading-4 sm:w-28"
            )}
          >
            Added cost
          </TableHead>
          <TableHead className="w-28 text-right text-xs leading-4 sm:w-40">
            Share
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody striped={false}>
        {operations.map((operation) => (
          <TableRow key={operation.nodeType} data-testid="plan-operation-row">
            <TableCell className="py-2 text-sm leading-5 text-main">
              <span className="block truncate">{operation.nodeType}</span>
            </TableCell>
            <TableCell className="py-2 text-right text-xs leading-4 text-control tabular-nums">
              {formatPlanCount(operation.count)}
            </TableCell>
            <TableCell
              className={cn(
                SECONDARY_COLUMN_CLASS,
                "py-2 text-right text-xs leading-4 text-control tabular-nums"
              )}
            >
              {formatPlanCost(operation.selfCost)}
            </TableCell>
            <TableCell className="py-2 pr-4">
              <span className="flex items-center justify-end gap-2">
                <PlanCostShareBar
                  share={operation.share}
                  className="w-10 shrink-0 sm:w-20"
                />
                <span className="w-12 shrink-0 text-right text-xs leading-4 text-main tabular-nums">
                  {formatPlanShare(operation.share)}
                </span>
              </span>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

/**
 * Where the plan's estimated cost goes, read three ways: how much of it each
 * node spans, which nodes own the most of it, and which kinds of operation do.
 */
export function QueryPlanSummary({ tree, selectedId, onSelect }: Props) {
  return (
    <div
      data-testid="plan-summary"
      className="min-h-0 w-full flex-1 overflow-auto"
    >
      <div className="flex flex-col gap-6 p-4">
        <PlanTotals tree={tree} />

        {/* A span starts at a startup cost, which not every engine reports. */}
        {tree.estimates.startupCost ? (
          <Section
            title="Cost ranges"
            description="Each node spans its startup cost — the cost before its first row — to its total cost, both of which include everything below it. Darker spans carry more of the plan's cost. Costs are not a schedule: two inputs of a join overlap here but run one after the other."
          >
            <CostTimeline
              tree={tree}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          </Section>
        ) : null}

        <Section
          title="Costliest operators"
          description="Ranked by the cost a node adds on top of its children. Where an input runs repeatedly, as under a nested loop, some engines count the repeats in the node that reruns it rather than in the input itself."
        >
          <CostliestOperators
            tree={tree}
            selectedId={selectedId}
            onSelect={onSelect}
          />
        </Section>

        <Section
          title="Cost by operation"
          description="Every node type in the plan and what it costs in total."
        >
          <CostByOperation tree={tree} />
        </Section>
      </div>
    </div>
  );
}

import { TriangleAlert } from "lucide-react";
import { useMemo } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useQueryPlanTranslation } from "./plan-i18n";
import {
  formatPlanCount,
  formatPlanShare,
  type PlanTree,
  type PlanWarningGroup,
  planCostliestNodes,
  planRankMetric,
  planSelfShare,
  planWarningGroups,
} from "./plan-model";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
}

const HEADING_CLASS = "text-xs leading-4 text-control-light";
const ROW_CLASS =
  "h-auto w-full items-baseline justify-start gap-2 rounded-xs px-2 py-1 text-left font-normal";
const SELECTED_ROW_CLASS = "bg-accent/10 hover:bg-accent/15";

function titleClass(selected: boolean): string {
  return cn(
    "truncate text-sm leading-5",
    selected ? "font-medium text-accent" : "text-main"
  );
}

/** Where a warning fired: the operator itself, or how many reported it. */
function warningSubject(
  group: PlanWarningGroup,
  translate: ReturnType<typeof useQueryPlanTranslation>
): string {
  if (group.nodes.length > 1) {
    return translate("summary.warning-operators", {
      count: formatPlanCount(group.nodes.length),
    });
  }
  const [node] = group.nodes;
  return [node.nodeType, node.subject].filter(Boolean).join(" · ");
}

/**
 * The operators that own the most of the plan, ranked, and the warnings worth
 * a reader's time, so the step to look at is named outright with no graph to
 * read first. Both pick a node, which the diagram and the table follow.
 *
 * Measured plans rank by measured time and say so: planner costs are arbitrary
 * units and are wrong in exactly the cases worth investigating.
 */
export function QueryPlanSummary({ tree, selectedId, onSelect }: Props) {
  const translate = useQueryPlanTranslation();
  const nodes = useMemo(() => planCostliestNodes(tree), [tree]);
  const warnings = useMemo(() => planWarningGroups(tree), [tree]);
  const measured = planRankMetric(tree) === "time";
  const hasMetric = measured || tree.estimates.cost;

  return (
    <section data-testid="plan-summary" className="flex flex-col gap-4">
      {/* An engine that reports neither a cost nor a time says so, rather than
          leaving the panel to imply there was nothing worth ranking. */}
      {!hasMetric ? (
        <p className="text-sm leading-5 text-control-light">
          {translate("summary.no-ranking")}
        </p>
      ) : (
        <div className="flex flex-col gap-1">
          <h2 className={HEADING_CLASS}>
            {translate(
              measured ? "summary.slowest-measured" : "summary.costliest"
            )}
          </h2>
          {nodes.length === 0 ? (
            <p className="text-sm leading-5 text-control-light">
              {translate(
                measured ? "summary.no-added-time" : "summary.no-added-cost"
              )}
            </p>
          ) : (
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
                      className={cn(ROW_CLASS, selected && SELECTED_ROW_CLASS)}
                    >
                      <span className="w-3 shrink-0 text-xs leading-4 text-control-light tabular-nums">
                        {index + 1}
                      </span>
                      <span className="flex min-w-0 flex-1 items-baseline gap-2">
                        <span className={titleClass(selected)}>
                          {node.nodeType}
                        </span>
                        <span className="min-w-0 flex-1 truncate text-xs leading-4 text-control-light">
                          {node.subject ?? ""}
                        </span>
                      </span>
                      <span className="shrink-0 text-xs leading-4 text-main tabular-nums">
                        {formatPlanShare(planSelfShare(node, tree))}
                      </span>
                    </Button>
                  </li>
                );
              })}
            </ol>
          )}
        </div>
      )}

      {warnings.length > 0 ? (
        <div className="flex flex-col gap-1">
          <h2 className={HEADING_CLASS}>{translate("summary.warnings")}</h2>
          <ul data-testid="plan-summary-warnings" className="flex flex-col">
            {warnings.map((group) => {
              const selected = group.nodes.some(
                (node) => node.id === selectedId
              );
              return (
                <li key={group.warning.title}>
                  <Button
                    appearance="secondary"
                    data-testid="plan-summary-warning"
                    aria-pressed={selected}
                    // The worst reporting node, so a grouped warning opens on
                    // the one most worth reading.
                    onClick={() => onSelect(group.nodes[0].id)}
                    className={cn(ROW_CLASS, selected && SELECTED_ROW_CLASS)}
                  >
                    <TriangleAlert
                      aria-hidden="true"
                      className="shrink-0 self-center text-warning"
                    />
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className={titleClass(selected)}>
                        {group.warning.title}
                      </span>
                      <span className="truncate text-xs leading-4 text-control-light">
                        {warningSubject(group, translate)}
                      </span>
                    </span>
                  </Button>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </section>
  );
}

import { useMemo } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useQueryPlanTranslation } from "./plan-i18n";
import {
  formatPlanShare,
  type PlanTree,
  planCostliestNodes,
  planSelfCostShare,
} from "./plan-model";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
}

/**
 * The operators that add the most estimated cost of their own, ranked, so the
 * expensive step is named outright with no graph to read first.
 */
export function QueryPlanSummary({ tree, selectedId, onSelect }: Props) {
  const translate = useQueryPlanTranslation();
  const nodes = useMemo(() => planCostliestNodes(tree), [tree]);
  return (
    <section data-testid="plan-summary" className="flex flex-col gap-1">
      <h2 className="text-xs leading-4 text-control-light">
        {translate("summary.costliest")}
      </h2>
      {nodes.length === 0 ? (
        <p className="text-sm leading-5 text-control-light">
          {translate("summary.no-added-cost")}
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
                  className={cn(
                    "h-auto w-full items-baseline justify-start gap-2 rounded-xs px-2 py-1 text-left font-normal",
                    selected && "bg-accent/10 hover:bg-accent/15"
                  )}
                >
                  <span className="w-3 shrink-0 text-xs leading-4 text-control-light tabular-nums">
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
                  <span className="shrink-0 text-xs leading-4 text-main tabular-nums">
                    {formatPlanShare(planSelfCostShare(node, tree))}
                  </span>
                </Button>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}

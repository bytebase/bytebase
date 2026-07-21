import { useMemo } from "react";
import { SpannerPlanNode } from "./SpannerPlanNode";
import type { SpannerPlanNodeData } from "./spanner-types";

interface Props {
  planSource: string;
  planQuery?: string;
}

export function SpannerQueryPlan({ planSource, planQuery }: Props) {
  const planNodes = useMemo<SpannerPlanNodeData[]>(() => {
    try {
      const parsed = JSON.parse(planSource);
      return parsed.planNodes || [];
    } catch {
      return [];
    }
  }, [planSource]);

  // The first node (index 0) is the root node in Spanner query plans.
  const rootNode = useMemo<SpannerPlanNodeData | undefined>(() => {
    if (planNodes.length === 0) return undefined;
    return planNodes.find((node) => node.index === 0);
  }, [planNodes]);

  return (
    <div className="ev-spanner">
      <div className="ev-spanner-header">
        <h3>Query Plan</h3>
        {planQuery ? <div className="ev-spanner-query">{planQuery}</div> : null}
      </div>
      <div className="ev-spanner-tree">
        {rootNode ? (
          <SpannerPlanNode node={rootNode} allNodes={planNodes} depth={0} />
        ) : (
          <div className="ev-spanner-empty">No query plan available</div>
        )}
      </div>
    </div>
  );
}

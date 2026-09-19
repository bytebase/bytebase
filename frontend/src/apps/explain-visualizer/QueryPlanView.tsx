import { useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { VisualizerEngine } from "@/utils/explainToken";
import { parseMssqlPlan } from "./mssql-plan";
import type { PlanParseResult } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanViewer } from "./QueryPlanViewer";
import { parseSpannerPlan } from "./spanner-plan";

const PARSERS: Record<VisualizerEngine, (source: string) => PlanParseResult> = {
  [Engine.POSTGRES]: parsePostgresPlan,
  [Engine.MSSQL]: parseMssqlPlan,
  [Engine.SPANNER]: parseSpannerPlan,
};

const STATUS_CLASS =
  "flex h-full min-h-0 w-full flex-col gap-4 overflow-auto bg-background p-4";

export function QueryPlanView({
  engine,
  planSource,
  textPlanSource,
  textTabLabel,
  planQuery,
}: {
  engine: VisualizerEngine;
  planSource: string;
  textPlanSource?: string;
  textTabLabel?: string;
  planQuery?: string;
}) {
  const result = useMemo(
    () => PARSERS[engine](planSource),
    [engine, planSource]
  );

  if (!result.ok) {
    return (
      <div className={STATUS_CLASS}>
        <Alert
          variant="error"
          title="This query plan could not be read"
          description={result.message}
        />
        {planSource.trim() ? (
          <pre className="font-mono text-xs leading-4 break-words whitespace-pre-wrap text-control">
            {planSource}
          </pre>
        ) : null}
      </div>
    );
  }

  return (
    <QueryPlanViewer
      tree={result.tree}
      rawPlan={planSource}
      textPlan={textPlanSource}
      textTabLabel={textTabLabel}
      query={planQuery}
    />
  );
}

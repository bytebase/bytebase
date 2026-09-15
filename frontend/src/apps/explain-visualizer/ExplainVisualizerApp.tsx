import { parse } from "qs";
import { useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  isVisualizerEngine,
  readExplainFromToken,
  type VisualizerEngine,
} from "@/utils/explainToken";
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

function PlanView({
  parsePlan,
  planSource,
  planQuery,
}: {
  parsePlan: (source: string) => PlanParseResult;
  planSource: string;
  planQuery?: string;
}) {
  const result = useMemo(() => parsePlan(planSource), [parsePlan, planSource]);

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
      query={planQuery}
    />
  );
}

export function ExplainVisualizerApp() {
  const storedQuery = useMemo(() => {
    const query = location.search.replace(/^\?/, "");
    const token = (parse(query).token as string) || "";
    return readExplainFromToken(token);
  }, []);

  if (!storedQuery) {
    return (
      <div className={STATUS_CLASS}>
        <Alert
          variant="warning"
          title="Session expired"
          description="Run the statement again to open a fresh query plan."
        />
      </div>
    );
  }

  if (!isVisualizerEngine(storedQuery.engine)) {
    return (
      <div className={STATUS_CLASS}>
        <Alert
          variant="warning"
          title="Unsupported database engine"
          description="Query plan visualization is not available for this database engine."
        />
      </div>
    );
  }

  return (
    <PlanView
      parsePlan={PARSERS[storedQuery.engine]}
      planSource={storedQuery.explain}
      planQuery={storedQuery.statement}
    />
  );
}

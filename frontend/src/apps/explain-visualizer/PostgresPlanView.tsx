import { useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanViewer } from "./QueryPlanViewer";

interface Props {
  /** `EXPLAIN (FORMAT JSON)` output as PostgreSQL returned it. */
  planSource: string;
  planQuery?: string;
}

export function PostgresPlanView({ planSource, planQuery }: Props) {
  const result = useMemo(() => parsePostgresPlan(planSource), [planSource]);

  if (!result.ok) {
    return (
      <div className="flex h-full min-h-0 w-full flex-col gap-4 overflow-auto bg-background p-4">
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

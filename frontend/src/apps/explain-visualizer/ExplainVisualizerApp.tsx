import { parse } from "qs";
import { useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { isVisualizerEngine, readExplainFromToken } from "@/utils/explainToken";
import { QueryPlanView } from "./QueryPlanView";

const STATUS_CLASS =
  "flex h-full min-h-0 w-full flex-col gap-4 overflow-auto bg-background p-4";

// Legacy standalone entry; the SQL Editor renders QueryPlanView inline.
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
    <QueryPlanView
      engine={storedQuery.engine}
      planSource={storedQuery.explain}
      planQuery={storedQuery.statement}
    />
  );
}

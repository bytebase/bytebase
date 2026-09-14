import { parse } from "qs";
import { type ReactNode, useMemo } from "react";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { readExplainFromToken } from "@/utils/explainToken";
import { MSSQLPlanView } from "./MSSQLPlanView";
import { PostgresPlanView } from "./PostgresPlanView";
import { SpannerQueryPlan } from "./SpannerQueryPlan";

export function ExplainVisualizerApp() {
  const storedQuery = useMemo(() => {
    const query = location.search.replace(/^\?/, "");
    const token = (parse(query).token as string) || "";
    return readExplainFromToken(token);
  }, []);

  if (!storedQuery) {
    return (
      <div className="ev-app">
        <div className="ev-unsupported">
          <h2>Session expired</h2>
          <p>Run the statement again to open a fresh query plan.</p>
        </div>
      </div>
    );
  }

  let body: ReactNode;
  switch (storedQuery.engine) {
    case Engine.POSTGRES:
      body = (
        <PostgresPlanView
          planSource={storedQuery.explain}
          planQuery={storedQuery.statement}
        />
      );
      break;
    case Engine.MSSQL:
      body = <MSSQLPlanView planXml={storedQuery.explain} />;
      break;
    case Engine.SPANNER:
      body = (
        <SpannerQueryPlan
          planSource={storedQuery.explain}
          planQuery={storedQuery.statement}
        />
      );
      break;
    default:
      body = (
        <div className="ev-unsupported">
          <h2>Unsupported Database Engine</h2>
          <p>
            Query plan visualization is not available for this database engine.
          </p>
        </div>
      );
  }

  return <div className="ev-app">{body}</div>;
}

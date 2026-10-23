import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";

export type PlanAvailability = "offered" | "needs-permission" | "absent";

/**
 * Whether a result's view switch offers Plan. It is absent when there is no
 * plan to ask for: the result already is one, the engine cannot explain, the
 * run recorded no executed statement, or an earlier statement of the run
 * returned no rows and so may have set session state the plan would miss.
 */
export function planAvailability({
  result,
  results,
  resultIndex,
  engineExplains,
  canExplain,
}: {
  result: QueryResult;
  results: QueryResult[];
  resultIndex: number;
  engineExplains: boolean;
  canExplain: boolean;
}): PlanAvailability {
  if (result.queryPlan || !engineExplains || !result.statement) {
    return "absent";
  }
  if (!results.slice(0, resultIndex).every(returnedRows)) {
    return "absent";
  }
  return canExplain ? "offered" : "needs-permission";
}

function returnedRows(result: QueryResult): boolean {
  const names = result.columnNames;
  return (
    names.length > 0 && !(names.length === 1 && names[0] === "Affected Rows")
  );
}

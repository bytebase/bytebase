import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";

export type PlanAvailability = "offered" | "needs-permission" | "absent";

/**
 * Whether the statement behind results[index] can be explained again on its
 * own. A plan request runs it alone, so an earlier statement of the run that
 * executed may have set session state the plan would miss, and a result cannot
 * show that: `SELECT set_config('search_path', ...)` returns a row like any
 * read. Only a plan that did not execute is known to leave the session as it
 * was.
 */
export function canReplayStatement(
  results: QueryResult[],
  index: number
): boolean {
  return results
    .slice(0, index)
    .every((earlier) => !!earlier.queryPlan && !earlier.queryPlan.executed);
}

/**
 * Whether a result's view switch offers Plan. It is absent when there is no
 * plan to ask for: the result already is one, the engine cannot explain, the
 * run recorded no executed statement, or the statement cannot be replayed on
 * its own.
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
  if (!canReplayStatement(results, resultIndex)) {
    return "absent";
  }
  return canExplain ? "offered" : "needs-permission";
}

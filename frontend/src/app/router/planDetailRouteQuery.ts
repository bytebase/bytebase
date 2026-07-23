import { isPlanDetailPhase } from "./handles";

export const PLAN_DETAIL_SELECTION_QUERY_KEYS: ReadonlySet<string> = new Set([
  "phase",
  "specId",
  "stageId",
  "taskId",
]);

export const stripPlanDetailSelectionQuery = (
  query: Record<string, unknown>
): Record<string, unknown> =>
  Object.fromEntries(
    Object.entries(query).filter(
      ([key]) => !PLAN_DETAIL_SELECTION_QUERY_KEYS.has(key)
    )
  );

// `phase` is a read-only compatibility input. Preserve a valid value already
// present on an old URL, but expose no phase argument that could create a new
// phase query.
export const buildPlanDetailLegacySearch = (requestUrl: string): string => {
  const source = new URL(requestUrl);
  const query = new URLSearchParams();
  const phase = source.searchParams.get("phase");
  if (isPlanDetailPhase(phase)) {
    query.set("phase", phase);
  }
  for (const [key, value] of source.searchParams) {
    if (!PLAN_DETAIL_SELECTION_QUERY_KEYS.has(key)) {
      query.append(key, value);
    }
  }
  return query.toString();
};

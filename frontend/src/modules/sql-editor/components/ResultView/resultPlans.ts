import { parsePlan } from "@/apps/explain-visualizer/plan-parsers";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { isVisualizerEngine } from "@/utils/queryPlanEngines";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import type { InlineQueryPlan } from "./QueryPlanResultView";

/** The plan a result carries in its rows, or undefined when it has none. */
export function inlineQueryPlan(
  result: QueryResult
): InlineQueryPlan | undefined {
  const lines = result.rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValuePlain(value)))
  );
  // Multi-column plans keep every column, tab-separated under a header.
  const source =
    result.columnNames.length > 1
      ? [result.columnNames, ...lines].map((line) => line.join("\t")).join("\n")
      : lines.map((line) => line[0]).join("\n");
  if (!source) return undefined;
  return { source };
}

/**
 * Each result's statement as a share of the whole script's estimated cost,
 * which says which of several plans is the one to read — the signal SSMS puts
 * on a plan tab.
 *
 * Undefined unless every result is a plan this engine's parser reads and every
 * one of them states a cost: a share of an incomplete total is worse than no
 * share. In practice that means a run of the Explain action, which plans every
 * statement at once; plans fetched one tab at a time never complete the total.
 */
export function planCostShares(
  results: readonly QueryResult[],
  engine: Engine
): number[] | undefined {
  if (!isVisualizerEngine(engine) || results.length === 0) return undefined;
  const costs: number[] = [];
  for (const result of results) {
    if (!result.queryPlan) return undefined;
    const source = inlineQueryPlan(result)?.source;
    if (!source) return undefined;
    const parsed = parsePlan(engine, source);
    if (!parsed.ok) return undefined;
    const cost = parsed.tree.root.totalCost;
    if (cost === undefined) return undefined;
    costs.push(cost);
  }
  const total = costs.reduce((sum, cost) => sum + cost, 0);
  if (total <= 0) return undefined;
  return costs.map((cost) => cost / total);
}

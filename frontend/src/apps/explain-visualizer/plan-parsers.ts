import { Engine } from "@/types/proto-es/v1/common_pb";
import type { VisualizerEngine } from "@/utils/queryPlanEngines";
import { parseMssqlPlan } from "./mssql-plan";
import type { PlanParseResult } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { parseSpannerPlan } from "./spanner-plan";

/**
 * Each engine's explain output on the engine-neutral model, keyed on the same
 * table as the format the SQL editor asks that engine for, so the two cannot
 * drift. Adding an engine means adding a parser and its format.
 */
const PARSERS: Record<VisualizerEngine, (source: string) => PlanParseResult> = {
  [Engine.POSTGRES]: parsePostgresPlan,
  [Engine.MSSQL]: parseMssqlPlan,
  [Engine.SPANNER]: parseSpannerPlan,
};

export function parsePlan(
  engine: VisualizerEngine,
  source: string
): PlanParseResult {
  return PARSERS[engine](source);
}

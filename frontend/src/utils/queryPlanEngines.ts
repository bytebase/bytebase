import { Engine } from "@/types/proto-es/v1/common_pb";

/**
 * The engines the query-plan visualizer draws, each with the explain output it
 * reads, named as `QueryOption.ExplainFormat` names it. The SQL editor offers
 * Visualize for exactly these engines and hands over the plan in this format;
 * the visualizer keys its parsers on the same table, so the two cannot drift.
 */
export const VISUALIZER_EXPLAIN_FORMATS = {
  [Engine.POSTGRES]: "JSON",
  [Engine.MSSQL]: "XML",
  [Engine.SPANNER]: "JSON",
} as const;

export type VisualizerEngine = keyof typeof VISUALIZER_EXPLAIN_FORMATS;

export const isVisualizerEngine = (
  engine: Engine
): engine is VisualizerEngine => engine in VISUALIZER_EXPLAIN_FORMATS;

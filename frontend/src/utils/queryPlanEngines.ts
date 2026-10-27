import { create } from "@bufbuild/protobuf";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  QueryOption_ExplainFormat,
  QueryOptionSchema,
} from "@/types/proto-es/v1/sql_service_pb";

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
  [Engine.MYSQL]: "JSON",
  [Engine.MARIADB]: "JSON",
  [Engine.TIDB]: "JSON",
  [Engine.ORACLE]: "JSON",
  [Engine.CLICKHOUSE]: "JSON",
  [Engine.TRINO]: "JSON",
  [Engine.OCEANBASE]: "JSON",
  [Engine.MONGODB]: "JSON",
  [Engine.COCKROACHDB]: "JSON",
  [Engine.SNOWFLAKE]: "JSON",
  [Engine.HIVE]: "JSON",
  // BigQuery states no plan for a statement it has not run. What it reports is
  // the stages of a job that did, which the result carries beside its rows.
  [Engine.BIGQUERY]: "JSON",
  // These three publish no plan as a document; their text plan is what there
  // is.
  [Engine.REDSHIFT]: "TEXT",
  [Engine.STARROCKS]: "TEXT",
  [Engine.DORIS]: "TEXT",
} as const;

export type VisualizerEngine = keyof typeof VISUALIZER_EXPLAIN_FORMATS;

export const isVisualizerEngine = (
  engine: Engine
): engine is VisualizerEngine => engine in VISUALIZER_EXPLAIN_FORMATS;

/**
 * The options of an explain request: the format the visualizer draws where the
 * engine has one, and the engine's default otherwise.
 *
 * BigQuery is the exception on both counts. Its explain request is a dry run,
 * which validates a statement rather than planning it, and the plan the viewer
 * draws arrives with an ordinary run instead — so asking for a format here
 * would only refuse the dry run.
 */
export const explainQueryOption = (engine: Engine) =>
  create(QueryOptionSchema, {
    explainFormat:
      isVisualizerEngine(engine) && engine !== Engine.BIGQUERY
        ? QueryOption_ExplainFormat[VISUALIZER_EXPLAIN_FORMATS[engine]]
        : QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED,
  });

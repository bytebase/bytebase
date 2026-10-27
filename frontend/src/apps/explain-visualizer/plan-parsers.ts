import { Engine } from "@/types/proto-es/v1/common_pb";
import type { VisualizerEngine } from "@/utils/queryPlanEngines";
import { parseBigqueryPlan } from "./bigquery-plan";
import { parseClickhousePlan } from "./clickhouse-plan";
import { parseCockroachdbPlan } from "./cockroachdb-plan";
import { parseHivePlan } from "./hive-plan";
import { parseMongodbPlan } from "./mongodb-plan";
import { parseMssqlPlan } from "./mssql-plan";
import { parseMysqlPlan } from "./mysql-plan";
import { parseOceanbasePlan } from "./oceanbase-plan";
import { parseOraclePlan } from "./oracle-plan";
import type { PlanParseResult } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { parseRedshiftPlan } from "./redshift-plan";
import { parseSnowflakePlan } from "./snowflake-plan";
import { parseSpannerPlan } from "./spanner-plan";
import { parseStarrocksPlan } from "./starrocks-plan";
import { parseTidbPlan } from "./tidb-plan";
import { parseTrinoPlan } from "./trino-plan";

/**
 * Each engine's explain output on the engine-neutral model, keyed on the same
 * table as the format the SQL editor asks that engine for, so the two cannot
 * drift. Adding an engine means adding a parser and its format.
 */
const PARSERS: Record<VisualizerEngine, (source: string) => PlanParseResult> = {
  [Engine.POSTGRES]: parsePostgresPlan,
  [Engine.MSSQL]: parseMssqlPlan,
  [Engine.SPANNER]: parseSpannerPlan,
  // MySQL and MariaDB differ in which keys they put in FORMAT=JSON, not in its
  // shape, so one parser reads both.
  [Engine.MYSQL]: parseMysqlPlan,
  [Engine.MARIADB]: parseMysqlPlan,
  [Engine.TIDB]: parseTidbPlan,
  // Oracle states its plan as plan-table rows, which the server renders as
  // JSON because no EXPLAIN option of its own does.
  [Engine.ORACLE]: parseOraclePlan,
  [Engine.CLICKHOUSE]: parseClickhousePlan,
  [Engine.TRINO]: parseTrinoPlan,
  // OceanBase spells its plan FORMAT=JSON as MySQL does, but returns an
  // operator tree rather than MySQL's nested query block.
  [Engine.OCEANBASE]: parseOceanbasePlan,
  // MongoDB asks for a plan with a cursor method rather than a keyword, and
  // states it as one document.
  [Engine.MONGODB]: parseMongodbPlan,
  // CockroachDB publishes no plan as a document; the server decodes the one it
  // encodes in the diagram link EXPLAIN (DISTSQL) prints.
  [Engine.COCKROACHDB]: parseCockroachdbPlan,
  // Snowflake states each operation's parents rather than its children, so its
  // parser turns those edges around.
  [Engine.SNOWFLAKE]: parseSnowflakePlan,
  // Hive answers with Calcite's own plan, as a JSON string within the JSON.
  [Engine.HIVE]: parseHivePlan,
  // BigQuery's plan is the stages of the job that returned the rows, so it is
  // measured rather than estimated.
  [Engine.BIGQUERY]: parseBigqueryPlan,
  // The text plans drawn here, because none of these engines publishes
  // anything else. StarRocks forked from Doris and their plans still agree on
  // the fragments and the tree, so one parser reads both.
  [Engine.REDSHIFT]: parseRedshiftPlan,
  [Engine.STARROCKS]: parseStarrocksPlan,
  [Engine.DORIS]: parseStarrocksPlan,
};

export function parsePlan(
  engine: VisualizerEngine,
  source: string
): PlanParseResult {
  return PARSERS[engine](source);
}

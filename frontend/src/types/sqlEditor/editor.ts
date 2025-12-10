import type * as monaco from "monaco-editor";
import { Engine } from "../proto-es/v1/common_pb";
import { type QueryOption } from "../proto-es/v1/sql_service_pb";

export type Language = "sql" | "javascript" | "redis" | "json";

export type SQLDialect =
  | "MYSQL"
  | "CLICKHOUSE"
  | "POSTGRES"
  | "SNOWFLAKE"
  | "TIDB"
  | "SPANNER"
  | "OCEANBASE";
const EngineToSQLDialectMap = new Map<Engine, SQLDialect>([
  [Engine.MYSQL, "MYSQL"],
  [Engine.CLICKHOUSE, "CLICKHOUSE"],
  [Engine.POSTGRES, "POSTGRES"],
  [Engine.SNOWFLAKE, "SNOWFLAKE"],
  [Engine.TIDB, "TIDB"],
  [Engine.SPANNER, "SPANNER"],
  [Engine.OCEANBASE, "OCEANBASE"],
]);

export const languageOfEngineV1 = (engine?: Engine): Language => {
  if (engine === Engine.MONGODB) {
    return "javascript";
  }
  if (engine === Engine.REDIS) {
    return "redis";
  }

  return "sql";
};

export const dialectOfEngineV1 = (
  engine: Engine = Engine.ENGINE_UNSPECIFIED
): SQLDialect => {
  return EngineToSQLDialectMap.get(engine) ?? "MYSQL";
};

export interface SQLEditorConnection {
  instance: string; // instance resource name, empty if not connected
  database: string; // database resource name, empty if not connected to a database
  dataSourceId?: string;
  schema?: string;
  table?: string;
}

export type SQLEditorQueryParams = {
  connection: SQLEditorConnection; // the connection snapshot of the query
  statement: string; // the statement snapshot of the query
  engine: Engine;
  explain: boolean;
  // Use to calculate the advice position.
  selection: monaco.Selection | null;
  queryOption?: QueryOption;
};

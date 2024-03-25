import type * as monaco from "monaco-editor";
import type { EngineType } from "@/types";
import { Engine } from "../proto/v1/common";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type Language = "sql" | "javascript" | "redis";

export const EngineTypesUsingSQL = [
  "MYSQL",
  "CLICKHOUSE",
  "POSTGRES",
  "SNOWFLAKE",
  "TIDB",
  "SPANNER",
  "OCEANBASE",
] as const;
export type SQLDialect = (typeof EngineTypesUsingSQL)[number];
export const EngineToSQLDialectMap = new Map<Engine, SQLDialect>([
  [Engine.MYSQL, "MYSQL"],
  [Engine.CLICKHOUSE, "CLICKHOUSE"],
  [Engine.POSTGRES, "POSTGRES"],
  [Engine.SNOWFLAKE, "SNOWFLAKE"],
  [Engine.TIDB, "TIDB"],
  [Engine.SPANNER, "SPANNER"],
  [Engine.OCEANBASE, "OCEANBASE"],
]);

export const languageOfEngine = (engine?: EngineType | "unknown"): Language => {
  if (engine === "MONGODB") {
    return "javascript";
  }
  if (engine === "REDIS") {
    return "redis";
  }

  return "sql";
};

export const languageOfEngineV1 = (engine?: Engine): Language => {
  if (engine === Engine.MONGODB) {
    return "javascript";
  }
  if (engine === Engine.REDIS) {
    return "redis";
  }

  return "sql";
};

export const dialectOfEngine = (engine = "unknown"): SQLDialect => {
  if (EngineTypesUsingSQL.includes(engine as any)) {
    return engine as SQLDialect;
  }
  // Fallback to MYSQL otherwise
  return "MYSQL";
};

export const dialectOfEngineV1 = (
  engine: Engine = Engine.UNRECOGNIZED
): SQLDialect => {
  return EngineToSQLDialectMap.get(engine) ?? "MYSQL";
};

export enum SortText {
  DATABASE = "0",
  TABLE = "1",
  COLUMN = "2",
  KEYWORD = "3",
}

export type SQLEditorQueryHistory = {
  name: string; // Format: logs/{uid}

  // Standard fields
  // creator in users/{email} format.
  creator: string;
  createTime: Date;

  // Domain fields
  statement: string;
  durationNs: number;
  instance: string;
  database: string;
  error: string;
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
};

import type * as monaco from "monaco-editor";
import { InstanceId, DatabaseId, ActivityId, EngineType } from "../types";
import { Principal } from "./principal";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type Language = "sql" | "javascript";

export const EngineTypesUsingSQL = [
  "MYSQL",
  "CLICKHOUSE",
  "POSTGRES",
  "SNOWFLAKE",
  "TIDB",
  "SPANNER",
] as const;
export type SQLDialect = typeof EngineTypesUsingSQL[number];

export const languageOfEngine = (engine?: EngineType | "unknown"): Language => {
  if (engine === "MONGODB") {
    return "javascript";
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

export enum SortText {
  DATABASE = "0",
  TABLE = "1",
  COLUMN = "2",
  KEYWORD = "3",
}

export interface QueryHistory {
  id: ActivityId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updatedTs: number;

  // Domain fields
  statement: string;
  durationNs: number;
  instanceName: string;
  databaseName: string;
  instanceId: InstanceId;
  databaseId: DatabaseId;
  error: string;

  // Customized fields
  createdAt: string;
}

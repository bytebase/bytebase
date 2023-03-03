import type * as monaco from "monaco-editor";
import { InstanceId, DatabaseId, ActivityId, EngineType } from "../types";
import { Principal } from "./principal";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type Language = "sql" | "javascript";
export type SQLDialect =
  | "mysql"
  | "postgresql"
  | "snowflake"
  | "clickhouse"
  | "tidb"
  | "spanner";

export const languageOfEngine = (engine?: EngineType | "unknown"): Language => {
  if (engine === "MONGODB") {
    return "javascript";
  }

  return "sql";
};

export const dialectOfEngine = (
  engine?: EngineType | "unknown"
): SQLDialect => {
  if (engine === "POSTGRES") {
    return "postgresql";
  }
  if (engine === "SNOWFLAKE") {
    return "snowflake";
  }
  // fallback to mysql dialect anyway
  return "mysql";
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

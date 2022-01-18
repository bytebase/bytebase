import * as monaco from "monaco-editor";

import { InstanceId, DatabaseId, TableId, ViewId, ActivityId } from "../types";
import { Principal } from "./principal";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type ConnectionAtomType = "instance" | "database" | "table" | "view";
export type SqlDialect = "mysql" | "postgresql";

export interface ConnectionAtom {
  parentId: InstanceId | DatabaseId | TableId | ViewId;
  id: InstanceId | DatabaseId | TableId | ViewId;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
}

export enum SortText {
  TABLE = "0",
  COLUMN = "1",
  KEYWORD = "2",
  DATABASE = "3",
  INSTASNCE = "4",
}

export type ConnectionContext = {
  hasSlug: boolean;
  instanceId: InstanceId;
  instanceName: string;
  databaseId?: DatabaseId;
  databaseName?: string;
  tableId?: TableId;
  tableName?: string;
  isLoadingTree: boolean;
  selectedDatabaseId: number;
  selectedTableName: string;
};

export interface QueryHistory {
  id: ActivityId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updatedTs: number;
  instanceId: number;

  // Domain fields
  statement: string;
  durationNs: number;
  instanceName: string;
  databaseName: string;
  error: string;

  // Customerize fields
  createdAt: string;
}

import type * as monaco from "monaco-editor";
import { InstanceId, DatabaseId, TableId, ViewId, ActivityId } from "../types";
import { Principal } from "./principal";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type ConnectionAtomType = "instance" | "database" | "table" | "view";
export type SQLDialect = "mysql" | "postgresql";

export interface ConnectionAtom {
  parentId: InstanceId | DatabaseId | TableId | ViewId;
  id: InstanceId | DatabaseId | TableId | ViewId;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
  isLeaf?: boolean;
}

export enum SortText {
  DATABASE = "0",
  TABLE = "1",
  COLUMN = "2",
  KEYWORD = "3",
}

// TODO(Jim): refactor <TableSchema> to get rid of this structure totally.
export type ConnectionContext = {
  option: ConnectionAtom;
};

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
  error: string;

  // Customerize fields
  createdAt: string;
}

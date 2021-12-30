import * as monaco from "monaco-editor";

import { InstanceId, DatabaseId, TableId, ViewId } from "../types";

export type EditorModel = monaco.editor.ITextModel;
export type EditorPosition = monaco.Position;
export type CompletionItems = monaco.languages.CompletionItem[];

export type ConnectionAtomType = "instance" | "database" | "table" | "view";

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

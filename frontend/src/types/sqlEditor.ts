import * as monaco from "monaco-editor";
import { DropdownOption } from "naive-ui";

import {
  ProjectId,
  InstanceId,
  DatabaseId,
  TableId,
  ViewId,
  ActivityId,
} from "../types";
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
  DATABASE = "0",
  TABLE = "1",
  COLUMN = "2",
  KEYWORD = "3",
}

export type ConnectionContext = {
  hasSlug: boolean;
  projectId: ProjectId;
  projectName: string;
  instanceId: InstanceId;
  instanceName: string;
  databaseId: DatabaseId;
  databaseName: string;
  databaseType: string;
  tableId?: TableId;
  tableName?: string;
  isLoadingTree: boolean;
  option: DropdownOption;
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

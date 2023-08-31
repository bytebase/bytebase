import { SQLResultSet } from "../types";
import { SQLResultSetV1 } from "./v1/sql";

export type ExecuteConfig = {
  databaseType: string;
};

export type ExecuteOption = {
  explain: boolean;
};

export type Connection = {
  instanceId: string;
  databaseId: string;
};

export enum TabMode {
  ReadOnly = 1,
  Admin = 2,
}

export type TabSheetType =
  | "TEMP" // Unsaved local sheet
  | "CLEAN" // Saved and untouched sheet
  | "DIRTY"; // Saved remotely, touched and unsaved locally

export type EditMode = "SQL-EDITOR" | "CHAT-TO-SQL";

export interface TabInfo {
  id: string;
  name: string;
  connection: Connection;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
  mode: TabMode;
  editMode: EditMode;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  isExecutingSQL: boolean;
  queryResult?: SQLResultSet;
  sheetName?: string;
  sqlResultSet?: SQLResultSetV1;
  isFreshNew?: boolean;
}

export type CoreTabInfo = Pick<TabInfo, "connection" | "sheetName" | "mode">;
export type AnyTabInfo = Partial<TabInfo>;

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

export interface ExecuteParams {
  query: string;
  config: ExecuteConfig;
  option?: Partial<ExecuteOption>;
}

export interface BatchQueryContext {
  selectedLabels: string[];
}

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
  executeParams?: ExecuteParams;
  isExecutingSQL: boolean;
  sheetName?: string;
  isFreshNew?: boolean;
  // batchQueryContext saves the context of batch query, including the selected labels.
  batchQueryContext?: BatchQueryContext;
  // databaseQueryResultMap is used to store the query result of each database.
  // It's used for the case that the user selects multiple databases to query.
  // The key is the databaseName. Format: instances/{instance}/databases/{database}
  databaseQueryResultMap?: Map<string, SQLResultSetV1>;
}

export type CoreTabInfo = Pick<TabInfo, "connection" | "sheetName" | "mode">;
export type AnyTabInfo = Partial<TabInfo>;

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
  dataSourceId?: string;
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

export interface BatchQueryContext {
  // selectedDatabaseNames is used to store the selected database names.
  // Format: instances/{instance}/databases/{database}
  selectedDatabaseNames: string[];
}

export type QueryContext = {
  beginTimestampMS: number;
  abortController: AbortController;
};

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
  sheetName?: string;
  isFreshNew?: boolean;
  // batchQueryContext saves the context of batch query, including the selected labels.
  batchQueryContext?: BatchQueryContext;
  // queryContext saves the context of a query, including beginTimestampMS and abortController
  queryContext?: QueryContext;
  // databaseQueryResultMap is used to store the query result of each database.
  // It's used for the case that the user selects multiple databases to query.
  // The key is the databaseName. Format: instances/{instance}/databases/{database}
  databaseQueryResultMap?: Map<string, SQLResultSetV1>;
}

export type CoreTabInfo = Pick<TabInfo, "connection" | "sheetName" | "mode">;
export type AnyTabInfo = Partial<TabInfo>;

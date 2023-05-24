import { Advice, DatabaseId, InstanceId, SQLResultSet } from "../types";

export type ExecuteConfig = {
  databaseType: string;
};

export type ExecuteOption = {
  explain: boolean;
};

export type Connection = {
  instanceId: InstanceId;
  databaseId: DatabaseId;
};

export enum TabMode {
  ReadOnly = 1,
  Admin = 2,
}

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
  adviceList?: Advice[];
}

export type CoreTabInfo = Pick<TabInfo, "connection" | "sheetName" | "mode">;
export type AnyTabInfo = Partial<TabInfo>;

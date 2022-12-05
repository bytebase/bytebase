import { Advice, DatabaseId, InstanceId, SheetId } from "../types";

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

export interface TabInfo {
  id: string;
  name: string;
  connection: Connection;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
  mode: TabMode;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  isExecutingSQL: boolean;
  // [columnNames: string[], types: string[], data: any[][]]
  queryResult?: [string[], string[], any[][]];
  sheetId?: SheetId;
  adviceList?: Advice[];
}

export type AnyTabInfo = Partial<TabInfo>;

import {
  Advice,
  DatabaseId,
  InstanceId,
  ProjectId,
  SheetId,
  TableId,
} from "../types";

export type ExecuteConfig = {
  databaseType: string;
};

export type ExecuteOption = {
  explain: boolean;
};

export type Connection = {
  projectId: ProjectId;
  instanceId: InstanceId;
  databaseId: DatabaseId;
  tableId: TableId;
};

export interface TabInfo {
  id: string;
  name: string;
  connection: Connection;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
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

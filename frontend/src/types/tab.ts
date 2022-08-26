import { Advice, SheetId } from "../types";

export type ExecuteConfig = {
  databaseType: string;
};

export type ExecuteOption = {
  explain: boolean;
};

export interface TabInfo {
  id: string;
  name: string;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  // [columnNames: string[], types: string[], data: any[][]]
  queryResult?: [string[], string[], any[][]];
  sheetId?: SheetId;
  adviceList?: Advice[];
}

export type AnyTabInfo = Partial<TabInfo>;

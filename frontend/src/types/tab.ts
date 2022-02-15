export interface TabInfo {
  id: string;
  name: string;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
  queryResult?: Record<string, any>[];
  sheetId?: number;
}

export type AnyTabInfo = Partial<TabInfo>;

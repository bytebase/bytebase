export interface TabInfo {
  id: string;
  label: string;
  isSaved: boolean;
  savedAt: string;
  queryStatement: string;
  selectedStatement: string;
  queryResult?: Record<string, any>[];
  sheetId?: number;
}

export type AnyTabInfo = Partial<TabInfo>;

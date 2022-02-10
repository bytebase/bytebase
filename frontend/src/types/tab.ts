export interface TabInfo {
  id: string;
  label: string;
  isSaved: boolean;
  savedAt: string;
  queryStatement: string;
  selectedStatement: string;
  sheetId?: number;
  queryResult?: Record<string, any>[];
}

export type AnyTabInfo = Partial<TabInfo>;

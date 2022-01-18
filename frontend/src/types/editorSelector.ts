export interface TabInfo {
  id: string;
  label: string;
  isSaved: boolean;
  savedAt: string;
  queryStatement: string;
  selectedStatement: string;
  queryResult?: Record<string, any>[];
  currentQueryId?: string;
}

export type AnyTabInfo = Partial<TabInfo>;

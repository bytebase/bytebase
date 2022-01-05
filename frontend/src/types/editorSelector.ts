export interface TabInfo {
  id: string;
  idx: number;
  label: string;
  isSaved: boolean;
  savedAt: string;
  queries: string;
}

export type AnyTabInfo = Partial<TabInfo>;

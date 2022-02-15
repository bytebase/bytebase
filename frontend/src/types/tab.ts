import { SheetId } from "../types";

export interface TabInfo {
  id: string;
  name: string;
  isSaved: boolean;
  savedAt: string;
  statement: string;
  selectedStatement: string;
  queryResult?: Record<string, any>[];
  sheetId?: SheetId;
}

export type AnyTabInfo = Partial<TabInfo>;

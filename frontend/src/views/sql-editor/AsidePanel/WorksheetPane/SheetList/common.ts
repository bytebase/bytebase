import { escape } from "lodash-es";
import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { extractWorksheetUID, getHighlightHTMLByRegExp } from "@/utils";

export type TabItem = {
  type: "TAB";
  target: SQLEditorTab;
};
export type SheetItem = {
  type: "SHEET";
  target: Worksheet;
};
export type MergedItem = TabItem | SheetItem;
export const isTabItem = (item: MergedItem): item is TabItem => {
  return item.type === "TAB";
};
export const isSheetItem = (item: MergedItem): item is SheetItem => {
  return item.type === "SHEET";
};

export const keyOfItem = (item: MergedItem) => {
  return isTabItem(item) ? item.target.id : item.target.name;
};

export const domIDForItem = (item: MergedItem) => {
  const key = keyOfItem(item);
  if (isTabItem(item)) {
    // tab
    return `bb-sheet-list-tab-${key}`;
  }
  // sheet
  return `bb-sheet-list-sheet-${extractWorksheetUID(key)}`;
};

export const titleHTML = (item: MergedItem, keyword: string) => {
  const kw = keyword.toLowerCase().trim();

  const title = item.target.title;

  if (!kw) {
    return escape(title);
  }

  return getHighlightHTMLByRegExp(
    escape(title),
    escape(kw),
    false /* !caseSensitive */
  );
};

export type DropdownState = {
  item: MergedItem;
  x: number;
  y: number;
};

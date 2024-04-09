import { escape } from "lodash-es";
import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { extractWorksheetUID, getHighlightHTMLByRegExp } from "@/utils";

export type ListItemType = "WORKSHEET" | "DRAFT";
export type ListItem<T extends ListItemType = ListItemType> = {
  type: T;
  target: T extends "WORKSHEET"
    ? Worksheet
    : T extends "DRAFT"
      ? SQLEditorTab
      : never;
};

export const isTypedListItem = <T extends ListItemType>(
  type: T,
  item: ListItem
): item is ListItem<T> => {
  return item.type === type;
};

export const keyForListItem = (item: ListItem) => {
  return isTypedListItem("WORKSHEET", item)
    ? item.target.name
    : isTypedListItem("DRAFT", item)
      ? item.target.id
      : "";
};

export const domIDForListItem = (item: ListItem) => {
  const key = keyForListItem(item);
  if (isTypedListItem("DRAFT", item)) {
    // tab
    return `bb-sheet-list-item-${key}`;
  }
  if (isTypedListItem("WORKSHEET", item)) {
    // sheet
    return `bb-sheet-list-worksheet-${extractWorksheetUID(key)}`;
  }
  return "bb-sheet-list-worksheet-unknown";
};

export const titleHTML = (item: ListItem, keyword: string) => {
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

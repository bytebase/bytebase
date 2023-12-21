import { VueClass } from "@/utils";
import { ColumnWidth } from "./BBGrid";

export type BBButtonType =
  | "NORMAL"
  | "PRIMARY"
  | "SECONDARY"
  | "DANGER"
  | "SUCCESS";

export type BBButtonConfirmStyle =
  | "NORMAL"
  | "DELETE"
  | "ARCHIVE"
  | "RESTORE"
  | "DISABLE"
  | "EDIT"
  | "CLONE";

export type BBTableColumn = {
  title: string;
  center?: boolean;
  nowrap?: boolean;
};

export type BBGridColumn = {
  title?: string;
  width: ColumnWidth;
  class?: VueClass;
};

export type BBGridRow<T = any> = {
  item: T;
  row: number;
};

export type BBTableSectionDataSource<T> = {
  title: string;
  link?: string;
  list: T[];
};

export type BBTabItem<T = any> = {
  title: string;
  // Used as the anchor
  id: string;
  data?: T;
};

export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";
export type BBNotificationPlacement =
  | "TOP_LEFT"
  | "TOP_RIGHT"
  | "BOTTOM_LEFT"
  | "BOTTOM_RIGHT";
export type BBNotificationItem = {
  style: BBNotificationStyle;
  title: string;
  description: string;
  link: string;
  linkTitle: string;
  onClose: () => void;
};

export type BBAlertStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";

export type BBAvatarSizeType = "TINY" | "SMALL" | "NORMAL" | "LARGE" | "HUGE";

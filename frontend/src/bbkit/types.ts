import type { VueClass } from "@/utils";
import type { ColumnWidth } from "./BBGrid";

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

export type BBAvatarSizeType =
  | "MINI"
  | "TINY"
  | "SMALL"
  | "NORMAL"
  | "LARGE"
  | "HUGE";

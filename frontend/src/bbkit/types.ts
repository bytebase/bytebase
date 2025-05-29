import type { VueClass } from "@/utils";
import type { ColumnWidth } from "./BBGrid";

export type BBButtonType =
  | "NORMAL"
  | "PRIMARY"
  | "SECONDARY"
  | "DANGER"
  | "SUCCESS";

export type BBButtonConfirmType = "NORMAL" | "DELETE" | "ARCHIVE" | "RESTORE";

export type BBGridColumn = {
  title?: string;
  width: ColumnWidth;
  class?: VueClass;
};

export type BBGridRow<T = any> = {
  item: T;
  row: number;
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

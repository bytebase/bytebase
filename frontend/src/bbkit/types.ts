import { VNode } from "vue";
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

export type BBTabFilterItem = {
  title: string;
  alert: boolean;
};

export type BBStepStatus =
  | "PENDING"
  | "PENDING_ACTIVE"
  | "PENDING_APPROVAL"
  | "PENDING_APPROVAL_ACTIVE"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED"
  | "SKIPPED";

export type BBStep = {
  status: BBStepStatus;
  payload: any;
};

export type BBStepTabItem = {
  title: string;
  hideNext?: boolean;
};

export type BBOutlineItem = {
  id: string;
  name: string;
  link?: string;
  childList?: BBOutlineItem[];
  // Only applicable if childList is specified.
  childCollapse?: boolean;
  prefix?: VNode;
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
};

export type BBAlertStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";

export type BBAttentionStyle = "INFO" | "WARN" | "CRITICAL";

export type BBAttentionSide = "BETWEEN" | "CENTER";

export type BBAvatarSizeType = "TINY" | "SMALL" | "NORMAL" | "LARGE" | "HUGE";

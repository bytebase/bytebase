import type { VNodeChild } from "vue";

export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";

export type NotificationCreate = {
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string | (() => VNodeChild);
  link?: string;
  linkTitle?: string;
  manualHide?: boolean;
};

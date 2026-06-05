export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";

export type NotificationCreate = {
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string | (() => unknown);
  link?: string;
  linkTitle?: string;
  manualHide?: boolean;
};

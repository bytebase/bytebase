import { VNodeChild } from "vue";

export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";

export type Notification = {
  id: string;
  createdTs: number;
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string | (() => VNodeChild);
  link?: string;
  linkTitle?: string;
  manualHide?: boolean;
};

// "id" and "createdTs" is auto generated upon the notification store
// receives.
export type NotificationCreate = Omit<Notification, "id" | "createdTs">;

export type NotificationFilter = {
  module: string;
  id?: string;
};

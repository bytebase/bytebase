import { BBNotificationStyle } from "../bbkit/types";

export type Notification = {
  id: string;
  createdTs: number;
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string;
  link?: string;
  linkTitle?: string;
  manualHide?: boolean;
};

// "id" and "createdTs" is auto generated upon the notification store
// receives.
export type NewNotification = Omit<Notification, "id" | "createdTs">;

export type NotificationFilter = {
  module: string;
  id?: string;
};

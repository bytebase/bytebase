import { Activity } from "./activity";
import { InboxID, PrincipalID } from "./id";

export type InboxStatus = "UNREAD" | "READ";

export type Inbox = {
  id: InboxID;

  // Domain specific fields
  receiver_id: PrincipalID;
  activity: Activity;
  status: InboxStatus;
};

export type InboxPatch = {
  status: InboxStatus;
};

export type InboxSummary = {
  hasUnread: boolean;
  hasUnreadError: boolean;
};

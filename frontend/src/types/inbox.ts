import { Activity } from "./activity";
import { InboxId, PrincipalId } from "./id";

export type InboxStatus = "UNREAD" | "READ";

export type Inbox = {
  id: InboxId;

  // Domain specific fields
  receiver_id: PrincipalId;
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

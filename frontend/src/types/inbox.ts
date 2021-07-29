import { Activity } from "./activity";
import { InboxId, PrincipalId } from "./id";

export type InboxStatus = "UNREAD" | "READ";

export type InboxLevel = "INFO" | "WARNING" | "ERROR";

export type Inbox = {
  id: InboxId;

  // Domain specific fields
  receiver_id: PrincipalId;
  activity: Activity;
  status: InboxStatus;
  level: InboxLevel;
};

export type InboxPatch = {
  status: InboxStatus;
};

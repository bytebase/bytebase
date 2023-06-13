import { IdType, InboxId } from "./id";
import { LogEntity } from "@/types/proto/v1/logging_service";

export type InboxStatus = "UNREAD" | "READ";

export type Inbox = {
  id: InboxId;

  // Domain specific fields
  activityId: IdType;
  activity?: LogEntity;
  status: InboxStatus;
};

export type InboxPatch = {
  status: InboxStatus;
};

export type InboxSummary = {
  hasUnread: boolean;
  hasUnreadError: boolean;
};

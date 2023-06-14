import { LogEntity } from "@/types/proto/v1/logging_service";
import { InboxMessage } from "@/types/proto/v1/inbox_service";

export interface ComposedInbox extends InboxMessage {
  activity: LogEntity;
}

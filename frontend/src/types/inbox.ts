import { LogEntity } from "@/types/proto/v1/logging_service";
import { InboxMessage } from "@/types/proto/v1/inbox_service";

export interface InboxV1 extends InboxMessage {
  activity: LogEntity;
}

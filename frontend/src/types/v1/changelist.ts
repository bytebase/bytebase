import { SYSTEM_BOT_USER_NAME } from "@/types";
import { Changelist } from "@/types/proto/v1/changelist_service";

export const UNKNOWN_CHANGELIST_NAME = "projects/-1/changelists/-1";

export const unknownChangelist = () => {
  return Changelist.fromPartial({
    name: UNKNOWN_CHANGELIST_NAME,
    description: "<<Unknown Changelist>>",
    creator: SYSTEM_BOT_USER_NAME,
    updater: SYSTEM_BOT_USER_NAME,
  });
};

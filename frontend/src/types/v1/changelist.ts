import { create } from "@bufbuild/protobuf";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { ChangelistSchema } from "@/types/proto-es/v1/changelist_service_pb";

export const UNKNOWN_CHANGELIST_NAME = "projects/-1/changelists/-1";

export const unknownChangelist = () => {
  return create(ChangelistSchema, {
    name: UNKNOWN_CHANGELIST_NAME,
    description: "<<Unknown Changelist>>",
    creator: SYSTEM_BOT_USER_NAME,
  });
};

export const Changelist_Change_Source_List = ["CHANGELOG", "RAW_SQL"] as const;

export type Changelist_Change_Source =
  (typeof Changelist_Change_Source_List)[number];

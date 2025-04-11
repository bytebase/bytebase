import { groupNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "../const";
import { Group } from "../proto/v1/group_service";

export const UNKNOWN_GROUP_NAME = `${groupNamePrefix}${UNKNOWN_ID}`;

export const unknownGroup = () => {
  return Group.fromPartial({
    name: UNKNOWN_GROUP_NAME,
    title: "<<Unknown group>>",
  });
};

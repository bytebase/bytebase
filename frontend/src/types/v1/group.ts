import { groupNamePrefix } from "@/store/modules/v1/common";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Group } from "../proto/v1/group_service";

export const UNKNOWN_GROUP_NAME = `${groupNamePrefix}${UNKNOWN_ID}`;

export const emptyGroup = () => {
  return Group.fromPartial({
    name: `${groupNamePrefix}${EMPTY_ID}`,
    title: "",
    description: "",
    members: [],
  });
};

export const unknownGroup = () => {
  return {
    ...emptyGroup(),
    name: UNKNOWN_GROUP_NAME,
    title: "<<Unknown group>>",
  };
};

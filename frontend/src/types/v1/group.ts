import { userGroupNamePrefix } from "@/store/modules/v1/common";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { UserGroup } from "../proto/v1/user_group";

export const UNKNOWN_GROUP_NAME = `${userGroupNamePrefix}${UNKNOWN_ID}`;

export const emptyGroup = () => {
  return UserGroup.fromPartial({
    name: `${userGroupNamePrefix}${EMPTY_ID}`,
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

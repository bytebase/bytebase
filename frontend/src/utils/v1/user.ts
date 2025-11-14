import { extractUserId, userNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL } from "@/types";

export const ensureUserFullName = (identifier: string) => {
  const id = extractUserId(identifier);
  return `${userNamePrefix}${id}`;
};

export const isUserIncludedInList = (
  identifier: string,
  userList: string[]
) => {
  const validId = ensureUserFullName(identifier);
  for (const name of userList) {
    if (
      name === ALL_USERS_USER_EMAIL ||
      name === `${userNamePrefix}${ALL_USERS_USER_EMAIL}` ||
      name === validId
    ) {
      return true;
    }
  }
  return false;
};

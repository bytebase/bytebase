import { extractUserEmail, userNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL } from "@/types";

export const ensureUserFullName = (identifier: string) => {
  const id = extractUserEmail(identifier);
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

export const extractEmailPrefix = (email: string, suffix: string): string => {
  if (email.endsWith(suffix)) {
    return email.slice(0, -suffix.length);
  }
  return email.split("@")[0];
};

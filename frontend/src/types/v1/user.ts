import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import { extractUserId } from "@/store/modules/v1/common";
import { SYSTEM_BOT_EMAIL } from "../common";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { User } from "../proto-es/v1/user_service_pb";
import { UserSchema, UserType } from "../proto-es/v1/user_service_pb";

export const UNKNOWN_USER_NAME = `users/${UNKNOWN_ID}`;
export const SYSTEM_BOT_USER_NAME = `users/${SYSTEM_BOT_EMAIL}`;

export const emptyUser = (): User => {
  return create(UserSchema, {
    name: `users/${EMPTY_ID}`,
    state: State.ACTIVE,
    email: "",
    title: "",
    userType: UserType.USER,
  });
};

export const unknownUser = (name: string = ""): User => {
  const empty = emptyUser();
  if (name) {
    empty.name = name;
    const email = extractUserId(name);
    empty.email = email;
    empty.title = email;
  } else {
    empty.name = UNKNOWN_USER_NAME;
    empty.title = "<<Unknown user>>";
  }
  return empty;
};

export const ALL_USERS_USER_EMAIL = "allUsers";
// Pseudo allUsers account.
export const allUsersUser = (): User => {
  return {
    ...emptyUser(),
    name: `users/${ALL_USERS_USER_EMAIL}`,
    title: t("settings.members.all-users"),
    email: ALL_USERS_USER_EMAIL,
    userType: UserType.SYSTEM_BOT,
  };
};

export const userBindingPrefix = "user:";

export const getUserEmailInBinding = (email: string) => {
  if (email === ALL_USERS_USER_EMAIL) {
    return email;
  }
  return `${userBindingPrefix}${email}`;
};

export const groupBindingPrefix = "group:";

export const getGroupEmailInBinding = (email: string) => {
  return `${groupBindingPrefix}${email}`;
};

export const isValidUserName = (name: string) => {
  return (
    !!name &&
    /^users\/(.+)$/.test(name) &&
    name !== emptyUser().name &&
    name !== unknownUser().name &&
    name !== allUsersUser().name
  );
};

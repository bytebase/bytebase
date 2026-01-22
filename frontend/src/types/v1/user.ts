import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import { extractUserId } from "@/store/modules/v1/common";
import { SYSTEM_BOT_EMAIL } from "../common";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { User } from "../proto-es/v1/user_service_pb";
import { UserSchema, UserType } from "../proto-es/v1/user_service_pb";

export const UNKNOWN_USER_NAME = `users/${UNKNOWN_ID}`;
export const SYSTEM_BOT_USER_NAME = `users/${SYSTEM_BOT_EMAIL}`;

export const unknownUser = (name: string = ""): User => {
  const user = create(UserSchema, {
    name: UNKNOWN_USER_NAME,
    state: State.ACTIVE,
    userType: UserType.USER,
  });
  if (name) {
    user.name = name;
    const email = extractUserId(name);
    user.email = email;
    user.title = email;
  }
  return user;
};

export const ALL_USERS_USER_EMAIL = "allUsers";
// Pseudo allUsers account.
export const allUsersUser = (): User => {
  return create(UserSchema, {
    name: `users/${ALL_USERS_USER_EMAIL}`,
    state: State.ACTIVE,
    title: t("settings.members.all-users"),
    email: ALL_USERS_USER_EMAIL,
    userType: UserType.SYSTEM_BOT,
  });
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

export const serviceAccountBindingPrefix = "serviceAccount:";

export const getServiceAccountEmailInBinding = (email: string) => {
  return `${serviceAccountBindingPrefix}${email}`;
};

export const workloadIdentityBindingPrefix = "workloadIdentity:";

export const getWorkloadIdentityNameInBinding = (name: string) => {
  return `${workloadIdentityBindingPrefix}${name}`;
};

export const isValidUserName = (name: string) => {
  return (
    !!name &&
    /^users\/(.+)$/.test(name) &&
    name !== unknownUser().name &&
    name !== allUsersUser().name
  );
};

import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import {
  extractUserEmail,
  serviceAccountNamePrefix,
  workloadIdentityNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { User } from "../proto-es/v1/user_service_pb";
import { UserSchema } from "../proto-es/v1/user_service_pb";

// Local AccountType enum for UI routing/display. AccountType was removed from the
// User proto message because it is now implied by the resource table, but the
// frontend still needs to distinguish account types for UI purposes.
export enum AccountType {
  USER = 1,
  WORKLOAD_IDENTITY = 2,
  SERVICE_ACCOUNT = 3,
}

export const UNKNOWN_USER_NAME = `users/${UNKNOWN_ID}`;

export const unknownUser = (name: string = ""): User => {
  const user = create(UserSchema, {
    name: UNKNOWN_USER_NAME,
    state: State.ACTIVE,
    title: t("common.unknown"),
  });
  if (name) {
    user.name = name;
    const email = extractUserEmail(name);
    user.email = email;
    user.title = email.split("@")[0];
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

export const serviceAccountSuffix = "service.bytebase.com";
export const serviceAccountBindingPrefix = "serviceAccount:";

export const getServiceAccountNameInBinding = (email: string) => {
  return `${serviceAccountBindingPrefix}${email}`;
};

export const workloadIdentitySuffix = "workload.bytebase.com";
export const workloadIdentityBindingPrefix = "workloadIdentity:";

export const getServiceAccountSuffix = (projectId?: string) => {
  if (projectId) {
    return `${projectId}.service.bytebase.com`;
  }
  return serviceAccountSuffix;
};

export const getWorkloadIdentitySuffix = (projectId?: string) => {
  if (projectId) {
    return `${projectId}.workload.bytebase.com`;
  }
  return workloadIdentitySuffix;
};

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

export const getAccountTypeByFullname = (fullname: string): AccountType => {
  if (fullname.startsWith(serviceAccountNamePrefix)) {
    return AccountType.SERVICE_ACCOUNT;
  }
  if (fullname.startsWith(workloadIdentityNamePrefix)) {
    return AccountType.WORKLOAD_IDENTITY;
  }
  return AccountType.USER;
};

export const getAccountTypeByEmail = (email: string): AccountType => {
  if (email.endsWith(serviceAccountSuffix)) {
    return AccountType.SERVICE_ACCOUNT;
  }
  if (email.endsWith(workloadIdentitySuffix)) {
    return AccountType.WORKLOAD_IDENTITY;
  }
  return AccountType.USER;
};

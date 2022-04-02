// Auth

import { VCSId } from "./id";

// For now, a single user's auth provider should either belong to  GITLAB_SELF_HOST or BYTEBASE
export type AuthProviderType = "GITLAB_SELF_HOST" | "BYTEBASE";

export type LoginInfo = {
  authProvider: AuthProviderType;
  payload: VCSLoginInfo | BytebaseLoginInfo;
};

export type SignupInfo = {
  email: string;
  password: string;
  name: string;
};

export type ActivateInfo = {
  email: string;
  password: string;
  name: string;
  token: string;
};

export type BytebaseLoginInfo = {
  email: string;
  password: string;
};

export type AuthProvider = {
  id: VCSId;
  type: AuthProviderType;
  name: string;
  instanceUrl: string;
  applicationId: string;
  secret: string;
};

export const EmptyAuthProvider: AuthProvider = {
  id: 0,
  type: "BYTEBASE",
  name: "",
  instanceUrl: "",
  applicationId: "",
  secret: "",
};

export type VCSLoginInfo = {
  vcsId: VCSId;
  name: string;
  code: string;
};

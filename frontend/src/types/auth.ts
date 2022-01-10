// Auth
export type LoginInfo = {
  email: string;
  password: string;
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

// For now, a single user's auth provider should either belong to BYTEBASE or GITLAB_SELF_HOST
export type AuthProviderType = "GITLAB_SELF_HOST" | "BYTEBASE" | unknown;

export type AuthProvider = {
  type: AuthProviderType;
  instanceUrl: string;
  applicationId: string;
  secret: string;
};

export const EmptyAuthProvider: AuthProvider = {
  type: "",
  instanceUrl: "",
  applicationId: "unknown",
  secret: "unknown",
};

export type GitlabLoginInfo = {
  applicationId: string;
  secret: string;
  instanceUrl: string;
  accessToken: string;
};

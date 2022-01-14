import { randomString } from "../utils";

export type OAuthConfig = {
  endpoint: string;
  applicationId: string;
  secret: string;
  redirectUrl: string;
};

export type OAuthToken = {
  accessToken: string;
  expiresTs: number;
  refreshToken: string;
};

export const OAuthStateSessionKey = "oauthstate";

type OAuthWindowEvent =
  | "bb.oauth.event.login"
  | "bb.oauth.event.register_vcs"
  | "unknown";

export const getOAuthEventName = (type: OAuthType): OAuthWindowEvent => {
  switch (type) {
    case "login":
      return "bb.oauth.event.login";
    case "register_vcs":
      return "bb.oauth.event.register_vcs";
    default:
      return "unknown";
  }
};

export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export function redirectUrl(): string {
  return `${window.location.origin}/oauth/callback`;
}

// login: users try to login via oauth
// register: users try to bind a vcs to her workspace
export type OAuthType = "login" | "register_vcs";

export function openWindowForOAuth(
  endpoint: string,
  applicationId: string,
  type: OAuthType
): Window | null {
  // we use type to determine oauth type when receiving the callback
  const stateQueryParameter = `${type}-${randomString(20)}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  return window.open(
    `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
      redirectUrl()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

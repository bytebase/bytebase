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

export const OAuthWindowEvent = "oauthevent";
export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export function redirectUrl(): string {
  return `${window.location.origin}/oauth/callback`;
}

type OAuthType = "login" | "register";

export function openWindowForOAuth(
  endpoint: string,
  applicationId: string,
  type: OAuthType
): Window | null {
  const stateQueryParameter = `${randomString(20)}-${type}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  return window.open(
    `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
      redirectUrl()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

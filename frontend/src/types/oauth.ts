import { randomString } from "../utils";

export type OAuthConfig = {
  endpoint: string;
  applicationID: string;
  secret: string;
  redirectURL: string;
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

export function redirectURL(): string {
  return `${window.location.origin}/oauth/callback`;
}

export function openWindowForOAuth(
  endpoint: string,
  applicationID: string
): Window | null {
  const stateQueryParameter = randomString(40);
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  return window.open(
    `${endpoint}?client_id=${applicationID}&redirect_uri=${encodeURIComponent(
      redirectURL()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

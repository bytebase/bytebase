import { useWindowScroll } from "@vueuse/core";
import { VCSType } from ".";
import { randomString, vcsSlug } from "../utils";

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

export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export function redirectUrl(): string {
  return `${window.location.origin}/oauth/callback`;
}

// signin: users try to login via oauth
// register-vcs: users try to bind a vcs to her workspace
// link-vcs-repository: users try to bind a vcs repo to her project
export type OAuthType =
  | "bb.oauth.signin"
  | "bb.oauth.register-vcs"
  | "bb.oauth.link-vcs-repository"
  | "bb.oauth.unknown";

export function openWindowForOAuth(
  endpoint: string,
  applicationId: string,
  type: OAuthType,
  vcsType: VCSType
): Window | null {
  // we use type to determine oauth type when receiving the callback
  const stateQueryParameter = `${type}-${randomString(20)}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  if (vcsType == "GITHUB_COM") {
    return window.open(
      `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
        redirectUrl()
      )}&state=${stateQueryParameter}&response_type=code&scope=api,repo`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  }
  // GITLAB_SELF_HOST
  return window.open(
    `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
      redirectUrl()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

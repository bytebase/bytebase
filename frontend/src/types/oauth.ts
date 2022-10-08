import { VCSType } from ".";
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
    // GitHub OAuth App scopes: https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps
    // We need the workflow scope to update GitHub action files.
    return window.open(
      `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
        redirectUrl()
      )}&state=${stateQueryParameter}&response_type=code&scope=api,repo,workflow`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  }
  // GITLAB_SELF_HOST
  // GitLab OAuth App scopes: https://docs.gitlab.com/ee/integration/oauth_provider.html#authorized-applications
  return window.open(
    `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
      redirectUrl()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

import {
  ExternalVersionControl_Type,
  externalVersionControl_TypeToJSON,
} from "@/types/proto/v1/externalvs_service";

export const OAuthStateSessionKey = "oauth-state";

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
  vcsType: ExternalVersionControl_Type
): Window | null {
  // we use type to determine oauth type when receiving the callback
  const stateQueryParameter = `${type}.${externalVersionControl_TypeToJSON(
    vcsType
  )}-${applicationId}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  if (vcsType == ExternalVersionControl_Type.GITHUB) {
    // GitHub OAuth App scopes: https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps
    // We need the workflow scope to update GitHub action files.
    return window.open(
      `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
        redirectUrl()
      )}&state=${stateQueryParameter}&response_type=code&scope=api,repo,workflow`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else if (vcsType == ExternalVersionControl_Type.BITBUCKET) {
    // Bitbucket OAuth App scopes: https://developer.atlassian.com/cloud/bitbucket/rest/intro/#authentication
    return window.open(
      `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
        redirectUrl()
      )}&state=${stateQueryParameter}&response_type=code&scope=account%20repository:write%20webhook`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=720,scrollbars=yes,status=yes"
    );
  }
  // GITLAB
  // GitLab OAuth App scopes: https://docs.gitlab.com/ee/integration/oauth_provider.html#authorized-applications
  return window.open(
    `${endpoint}?client_id=${applicationId}&redirect_uri=${encodeURIComponent(
      redirectUrl()
    )}&state=${stateQueryParameter}&response_type=code&scope=api`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

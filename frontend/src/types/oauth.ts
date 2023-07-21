import { stringify } from "qs";
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

export type OAuthState = {
  event: string;
  popup?: boolean;
  redirect?: string;
};

export function openWindowForOAuth(
  endpoint: string,
  applicationId: string,
  type: OAuthType,
  vcsType: ExternalVersionControl_Type
): Window | null {
  const state: OAuthState = {
    // we use type to determine oauth type when receiving the callback
    event: `${type}.${externalVersionControl_TypeToJSON(
      vcsType
    )}-${applicationId}`,
    popup: true,
  };

  const endpointQueryParams: Record<string, string> = {
    client_id: applicationId,
    state: stringify(state),
    response_type: "code",
    redirect_uri: redirectUrl(),
  };

  // Set proper popup window size for different VCS types
  let windowOpenOptions = "";

  if (vcsType == ExternalVersionControl_Type.GITHUB) {
    // GitHub OAuth App scopes: https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps
    // We need the workflow scope to update GitHub action files.

    endpointQueryParams["scope"] = "api,repo,workflow";
    windowOpenOptions =
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes";
  } else if (vcsType == ExternalVersionControl_Type.BITBUCKET) {
    // Bitbucket OAuth App scopes: https://developer.atlassian.com/cloud/bitbucket/rest/intro/#authentication
    // Do not call `encodeURIComponent` here, will be encoded later.
    endpointQueryParams["scope"] = "account repository:write webhook";
    windowOpenOptions =
      "location=yes,left=200,top=200,height=640,width=720,scrollbars=yes,status=yes";
  } else if (vcsType == ExternalVersionControl_Type.GITLAB) {
    // GITLAB
    // GitLab OAuth App scopes: https://docs.gitlab.com/ee/integration/oauth_provider.html#authorized-applications
    endpointQueryParams["scope"] = "api";
    windowOpenOptions =
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes";
  } else if (vcsType == ExternalVersionControl_Type.AZURE_DEVOPS) {
    // Scopes for Azure: https://learn.microsoft.com/zh-cn/azure/devops/integrate/get-started/authentication/oauth?view=azure-devops#scopes
    // We need full scopes in the application: https://stackoverflow.com/questions/56143321/azure-devops-oauth-enpoint-always-returns-error-invalidscope
    // TODO: decide necessary scopes
    endpointQueryParams["scope"] = "vso.code_full vso.identity";
    endpointQueryParams["response_type"] = "Assertion";
    windowOpenOptions =
      "location=yes,left=200,top=200,height=640,width=720,scrollbars=yes,status=yes";
  }

  const fullEndpointURL = `${endpoint}?${stringify(endpointQueryParams)}`;

  return window.open(fullEndpointURL, "oauth", windowOpenOptions);
}

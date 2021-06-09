import { randomString } from "../utils";
import { VCS } from "./vcs";

export const OAuthStateSessionKey = "oauthstate";

export const OAuthWindowEvent = "oauthevent";
export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export function redirectURL(): string {
  return `${window.location.origin}/oauth/callback`;
}

export function openWindowForVCSOAuth(vcs: VCS): Window | null {
  const stateQueryParameter = randomString(40);
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  return window.open(
    `${vcs.instanceURL}/oauth/authorize?client_id=${
      vcs.applicationId
    }&redirect_uri=${encodeURIComponent(
      redirectURL()
    )}&response_type=code&state=${stateQueryParameter}`,
    "oauth",
    "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
  );
}

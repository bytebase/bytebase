import { VCSId } from "./id";

export type OAuthResourceType = "VCS";
export type OAuthResourceId = VCSId;

export type OAuthState = {
  resourceType: OAuthResourceType;
  resourceId: OAuthResourceId;
};

export function oauthStateKey(state: string): string {
  return "oauthstate." + state;
}

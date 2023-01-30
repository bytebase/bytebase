import { OAuthStateSessionKey } from "@/types";
import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";

export function openWindowForSSO(
  identityProvider: IdentityProvider
): Window | null {
  // we use type to determine oauth type when receiving the callback
  const stateQueryParameter = `bb.oauth.signin.${identityProvider.name}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);

  if (identityProvider.type === IdentityProviderType.OAUTH2) {
    const oauth2Config = identityProvider.config?.oauth2Config;
    if (!oauth2Config) {
      return null;
    }

    const redirectUrl = encodeURIComponent(
      `${window.location.origin}/oauth/callback`
    );
    return window.open(
      `${oauth2Config.authUrl}?client_id=${
        oauth2Config.clientId
      }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
        oauth2Config.scopes.join(" ")
      )}`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else {
    throw new Error(
      `identity provider type ${identityProvider.type.toString()} is not supported`
    );
  }

  return null;
}

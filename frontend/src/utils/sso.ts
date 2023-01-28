import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";

export function openWindowForSSO(
  identityProvider: IdentityProvider
): Window | null {
  // we use type to determine oauth type when receiving the callback
  const stateQueryParameter = `bb.oauth.signin-${identityProvider.name}`;
  sessionStorage.setItem("sso-state", stateQueryParameter);

  if (identityProvider.type === IdentityProviderType.OAUTH2) {
    const oauthConfig = identityProvider.config?.oauth2Config;
    if (!oauthConfig) {
      return null;
    }

    const redirectUrl = encodeURIComponent(
      `${window.location.origin}/oauth/callback`
    );

    console.log(
      "link",
      `${oauthConfig.authUrl}?client_id=${
        oauthConfig.clientId
      }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
        oauthConfig.scopes.join(",")
      )}`
    );
    return window.open(
      `${oauthConfig.authUrl}?client_id=${
        oauthConfig.clientId
      }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
        oauthConfig.scopes.join(",")
      )}`,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else if (identityProvider.type === IdentityProviderType.OIDC) {
    // TODO
  }

  return null;
}

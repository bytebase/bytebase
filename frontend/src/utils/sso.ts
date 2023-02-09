import { OAuthStateSessionKey } from "@/types";
import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";
import axios from "axios";
import { trimEnd } from "lodash-es";

export async function openWindowForSSO(
  identityProvider: IdentityProvider
): Promise<Window | null> {
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
  } else if (identityProvider.type === IdentityProviderType.OIDC) {
    const oidcConfig = identityProvider.config?.oidcConfig;
    if (!oidcConfig) {
      return null;
    }

    const openidConfig = (
      await axios.get(
        `${trimEnd(oidcConfig.issuer, "/")}/.well-known/openid-configuration`
      )
    ).data;

    // Some IdPs like authning.cn doesn't expose "username" as part of standard claims,
    // so we need to request the claim explictly when possible.
    if (openidConfig.scopes_supported.includes("username")) {
      oidcConfig.scopes.push("username");
    }

    const redirectUrl = encodeURIComponent(
      `${window.location.origin}/oidc/callback`
    );
    return window.open(
      `${openidConfig.authorization_endpoint}?client_id=${
        oidcConfig.clientId
      }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
        oidcConfig.scopes.join(" ")
      )}`,
      "oidc",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else {
    throw new Error(
      `identity provider type ${identityProvider.type.toString()} is not supported`
    );
  }
}

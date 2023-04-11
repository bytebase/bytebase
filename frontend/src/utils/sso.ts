import axios from "axios";
import { trimEnd, uniq } from "lodash-es";
import { OAuthStateSessionKey } from "@/types";
import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";

export const SSOConfigSessionKey = "sso-config";

// defaultOIDCScopes is a list of scopes that are part of OIDC standard claims. Same as backend.
const defaultOIDCScopes = ["openid", "profile", "email"];

export async function openWindowForSSO(
  identityProvider: IdentityProvider,
  openAsPopup = true,
  redirect = ""
) {
  const stateQueryParameter = `bb.oauth.signin.${identityProvider.name}`;
  sessionStorage.setItem(OAuthStateSessionKey, stateQueryParameter);
  // Set SSO config in session storage so that we can use it in the callback page.
  sessionStorage.setItem(
    SSOConfigSessionKey,
    JSON.stringify({
      identityProviderName: identityProvider.name,
      openAsPopup,
      redirect,
    })
  );

  let authUrl = "";
  if (identityProvider.type === IdentityProviderType.OAUTH2) {
    const oauth2Config = identityProvider.config?.oauth2Config;
    if (!oauth2Config) {
      return null;
    }

    const redirectUrl = encodeURIComponent(
      `${window.location.origin}/oauth/callback`
    );
    authUrl = `${oauth2Config.authUrl}?client_id=${
      oauth2Config.clientId
    }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
      oauth2Config.scopes.join(" ")
    )}`;
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
    // so we need to request the claim explicitly when possible.
    if (openidConfig.scopes_supported.includes("username")) {
      oidcConfig.scopes.push("username");
    }
    oidcConfig.scopes = uniq([...oidcConfig.scopes, ...defaultOIDCScopes]);

    const redirectUrl = encodeURIComponent(
      `${window.location.origin}/oidc/callback`
    );
    authUrl = `${openidConfig.authorization_endpoint}?client_id=${
      oidcConfig.clientId
    }&redirect_uri=${redirectUrl}&state=${stateQueryParameter}&response_type=code&scope=${encodeURIComponent(
      oidcConfig.scopes.join(" ")
    )}`;
  } else {
    throw new Error(
      `identity provider type ${identityProvider.type.toString()} is not supported`
    );
  }

  if (!authUrl) {
    throw new Error("Invalid authentication URL");
  }
  if (openAsPopup) {
    window.open(
      authUrl,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else {
    // Redirect to the auth URL.
    window.location.href = authUrl;
  }
}

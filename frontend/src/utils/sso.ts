import axios from "axios";
import { trimEnd, uniq } from "lodash-es";
import { stringify } from "qs";
import { OAuthState } from "@/types";
import {
  IdentityProvider,
  IdentityProviderType,
} from "@/types/proto/v1/idp_service";

export const SSOConfigSessionKey = "sso-config";

// defaultOIDCScopes is a list of scopes that are part of OIDC standard claims. Same as backend.
const defaultOIDCScopes = ["openid", "profile", "email"];

export async function openWindowForSSO(
  identityProvider: IdentityProvider,
  popup = true,
  redirect = ""
) {
  const state: OAuthState = {
    // we use type to determine oauth type when receiving the callback
    event: `bb.oauth.signin.${identityProvider.name}`,
    popup,
    redirect,
  };

  const uri = {
    basePath: "",
    query: {
      state: stringify(state),
      response_type: "code",
    } as Record<string, string>,
  };

  if (identityProvider.type === IdentityProviderType.OAUTH2) {
    const oauth2Config = identityProvider.config?.oauth2Config;
    if (!oauth2Config) {
      return null;
    }
    uri.basePath = oauth2Config.authUrl;
    Object.assign(uri.query, {
      client_id: oauth2Config.clientId,
      scope: oauth2Config.scopes.join(" "),
      redirect_uri: `${window.location.origin}/oauth/callback`,
    });
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

    uri.basePath = openidConfig.authorization_endpoint;
    Object.assign(uri.query, {
      client_id: oidcConfig.clientId,
      scope: oidcConfig.scopes.join(" "),
      redirect_uri: `${window.location.origin}/oidc/callback`,
    });
  } else {
    throw new Error(
      `identity provider type ${identityProvider.type.toString()} is not supported`
    );
  }

  const authUrl = `${uri.basePath}?${stringify(uri.query)}`;

  if (!authUrl) {
    throw new Error("Invalid authentication URL");
  }

  if (popup) {
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

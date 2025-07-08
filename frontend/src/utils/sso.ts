import { stringify } from "qs";
import type { OAuthState } from "@/types";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

export const SSOConfigSessionKey = "sso-config";

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
    if (identityProvider.config?.config?.case !== "oauth2Config") {
      return null;
    }
    const oauth2Config = identityProvider.config.config.value;
    uri.basePath = oauth2Config.authUrl;
    Object.assign(uri.query, {
      client_id: oauth2Config.clientId,
      scope: oauth2Config.scopes.join(" "),
      redirect_uri: `${window.location.origin}/oauth/callback`,
    });
  } else if (identityProvider.type === IdentityProviderType.OIDC) {
    if (identityProvider.config?.config?.case !== "oidcConfig") {
      return null;
    }
    const oidcConfig = identityProvider.config.config.value;
    if (oidcConfig.authEndpoint === "") {
      throw new Error(
        `Invalid authentication URL from issuer ${oidcConfig.issuer}, please check your configuration`
      );
    }
    uri.basePath = oidcConfig.authEndpoint;
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

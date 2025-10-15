import { stringify } from "qs";
import type { OAuthState } from "@/types";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

export const SSOConfigSessionKey = "sso-config";

/**
 * Validates that a URL is a safe HTTP/HTTPS URL to prevent XSS attacks.
 * Rejects javascript:, data:, and other dangerous protocols.
 */
function isValidHttpUrl(url: string): boolean {
  try {
    const urlObj = new URL(url);
    return urlObj.protocol === "http:" || urlObj.protocol === "https:";
  } catch {
    return false;
  }
}

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

    // Validate auth URL to prevent XSS via javascript: URIs
    if (!isValidHttpUrl(oauth2Config.authUrl)) {
      throw new Error(
        "Invalid authentication URL: must be a valid HTTP or HTTPS URL"
      );
    }

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

    // Validate auth endpoint to prevent XSS via javascript: URIs
    if (!isValidHttpUrl(oidcConfig.authEndpoint)) {
      throw new Error(
        "Invalid authentication endpoint: must be a valid HTTP or HTTPS URL"
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

  // Use URL API for safe URL construction with query parameters
  const authUrl = new URL(uri.basePath);
  Object.entries(uri.query).forEach(([key, value]) => {
    authUrl.searchParams.set(key, value);
  });

  const authUrlString = authUrl.toString();
  if (!authUrlString) {
    throw new Error("Invalid authentication URL");
  }

  if (popup) {
    window.open(
      authUrlString,
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else {
    // Redirect to the auth URL.
    window.location.href = authUrlString;
  }
}

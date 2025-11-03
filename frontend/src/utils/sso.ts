import type { OAuthState } from "@/types";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

export const SSOConfigSessionKey = "sso-config";
const OAUTH_STATE_PREFIX = "oauth_state_";
const OAUTH_STATE_TTL = 10 * 60 * 1000; // 10 minutes in milliseconds

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

/**
 * Generates a cryptographically secure random token for OAuth state parameter.
 * Uses Web Crypto API to generate 32 bytes of random data.
 * Returns base64url-encoded string for URL safety.
 */
function generateSecureToken(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  // Convert to base64url encoding (URL-safe, no padding)
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Stores OAuth state in localStorage with a prefixed key.
 * The state includes security token, routing info, and timestamp.
 */
function storeOAuthState(state: OAuthState): void {
  const key = `${OAUTH_STATE_PREFIX}${state.token}`;
  try {
    localStorage.setItem(key, JSON.stringify(state));
  } catch (error) {
    console.error("Failed to store OAuth state:", error);
    throw new Error("Failed to store authentication state");
  }
}

/**
 * Retrieves and validates OAuth state from localStorage.
 * Returns the state if valid, or null if missing/invalid/expired.
 */
export function retrieveOAuthState(token: string): OAuthState | null {
  const key = `${OAUTH_STATE_PREFIX}${token}`;
  try {
    const stored = localStorage.getItem(key);
    if (!stored) {
      return null;
    }
    const state = JSON.parse(stored) as OAuthState;

    // Validate timestamp (must be within TTL)
    const now = Date.now();
    if (now - state.timestamp > OAUTH_STATE_TTL) {
      localStorage.removeItem(key);
      return null;
    }

    return state;
  } catch (error) {
    console.error("Failed to retrieve OAuth state:", error);
    return null;
  }
}

/**
 * Clears OAuth state from localStorage after use.
 * This ensures single-use tokens for security.
 */
export function clearOAuthState(token: string): void {
  const key = `${OAUTH_STATE_PREFIX}${token}`;
  try {
    localStorage.removeItem(key);
  } catch (error) {
    console.error("Failed to clear OAuth state:", error);
  }
}

export async function openWindowForSSO(
  identityProvider: IdentityProvider,
  popup = true,
  redirect?: string
) {
  // Generate cryptographically secure random token for CSRF protection
  const token = generateSecureToken();

  const state: OAuthState = {
    token,
    // we use type to determine oauth type when receiving the callback
    event: `bb.oauth.signin.${identityProvider.name}`,
    popup,
    redirect,
    timestamp: Date.now(),
    // Store IdP type to determine correct context type in callback
    idpType: identityProvider.type,
  };

  // Store state in localStorage before redirecting
  storeOAuthState(state);

  const uri = {
    basePath: "",
    query: {
      // Only send the opaque token as per RFC 6749 and Auth0 best practices
      // All other state data is stored server-side for security
      state: token,
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

import { create } from "@bufbuild/protobuf";
import type { OAuthState } from "@/types";
import type {
  AuthorizationRequest,
  LoginIdentityProvider,
} from "@/types/proto-es/v1/auth_service_pb";
import {
  AuthorizationRequestSchema,
  LoginIdentityProviderSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

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
  identityProvider: LoginIdentityProvider,
  popup = true,
  redirect?: string
) {
  // The callback route differs by protocol; anything else has no redirect flow.
  let callbackPath: string;
  if (identityProvider.type === IdentityProviderType.OAUTH2) {
    callbackPath = "/oauth/callback";
  } else if (identityProvider.type === IdentityProviderType.OIDC) {
    callbackPath = "/oidc/callback";
  } else {
    throw new Error(
      `identity provider type ${identityProvider.type.toString()} is not supported`
    );
  }

  const request = identityProvider.authorizationRequest;
  if (!request || request.endpoint === "") {
    throw new Error(
      "The identity provider published no authorization endpoint; check its configuration"
    );
  }
  // Validate the endpoint to prevent XSS via javascript: URIs.
  if (!isValidHttpUrl(request.endpoint)) {
    throw new Error(
      "Invalid authentication URL: must be a valid HTTP or HTTPS URL"
    );
  }

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

  const authUrl = new URL(request.endpoint);
  // Only send the opaque token as per RFC 6749 and Auth0 best practices;
  // all other state data is stored client-side under that token.
  authUrl.searchParams.set("state", token);
  authUrl.searchParams.set("response_type", "code");
  authUrl.searchParams.set("client_id", request.clientId);
  authUrl.searchParams.set("scope", request.scopes.join(" "));
  authUrl.searchParams.set(
    "redirect_uri",
    `${window.location.origin}${callbackPath}`
  );

  if (popup) {
    window.open(
      authUrl.toString(),
      "oauth",
      "location=yes,left=200,top=200,height=640,width=480,scrollbars=yes,status=yes"
    );
  } else {
    // Redirect to the auth URL.
    window.location.href = authUrl.toString();
  }
}

/**
 * Projects a full IdentityProvider onto the shape the redirect needs, so the
 * SSO admin test drives the same path the login page does.
 */
export function toLoginIdentityProvider(
  identityProvider: IdentityProvider
): LoginIdentityProvider {
  let authorizationRequest: AuthorizationRequest | undefined;
  const config = identityProvider.config?.config;
  if (config?.case === "oauth2Config") {
    authorizationRequest = create(AuthorizationRequestSchema, {
      endpoint: config.value.authUrl,
      clientId: config.value.clientId,
      scopes: config.value.scopes,
    });
  } else if (config?.case === "oidcConfig") {
    authorizationRequest = create(AuthorizationRequestSchema, {
      endpoint: config.value.authEndpoint,
      clientId: config.value.clientId,
      scopes: config.value.scopes,
    });
  }
  return create(LoginIdentityProviderSchema, {
    name: identityProvider.name,
    type: identityProvider.type,
    title: identityProvider.title,
    authorizationRequest,
  });
}

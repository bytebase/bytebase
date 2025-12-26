import { authServiceClientConnect } from "@/grpcweb";

let refreshPromise: Promise<void> | null = null;

/**
 * Refresh the access token using the refresh token cookie.
 * Uses singleton pattern to prevent concurrent refresh calls within the same tab.
 * Cross-tab races are handled by server-side grace period (30 seconds).
 */
export async function refreshTokens(): Promise<void> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = authServiceClientConnect
    .refresh({})
    .then(() => {})
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
}

import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import { t } from "@/plugins/i18n";
import { router } from "@/router";
import { pushNotification, useAuthStore } from "@/store";
import { ignoredCodesContextKey, silentContextKey } from "../context-key";
import { refreshTokens } from "../refreshToken";

export type IgnoreErrorsOptions = {
  /**
   * If set to true, will NOT show redirect to other pages(e.g., 403, sign in page).
   */
  silent?: boolean;

  /**
   * If set, will NOT handle specified status codes is this array.
   */
  ignoredCodes?: Code[];
};

export const authInterceptor: Interceptor = (next) => async (req) => {
  try {
    const resp = await next(req);
    return resp;
  } catch (error) {
    const authStore = useAuthStore();
    const silent = req.contextValues.get(silentContextKey);
    const ignoredCodes = req.contextValues.get(ignoredCodesContextKey);

    if (!silent && error instanceof ConnectError) {
      const { code } = error;
      if (ignoredCodes?.includes(code)) {
        // omit specified errors
      } else {
        if (code === Code.Unauthenticated && req.method.name !== "Login") {
          // Don't retry refresh endpoint failures
          if (req.method.name === "Refresh") {
            authStore.unauthenticatedOccurred = true;
            if (authStore.isLoggedIn) {
              pushNotification({
                module: "bytebase",
                style: "WARN",
                title: t("auth.token-expired-title"),
                description: t("auth.token-expired-description"),
              });
            }
            throw error;
          }

          // Try to refresh the token
          try {
            await refreshTokens();
            // Retry the original request
            return await next(req);
          } catch {
            // Refresh failed, show login notification
            authStore.unauthenticatedOccurred = true;
            if (authStore.isLoggedIn) {
              pushNotification({
                module: "bytebase",
                style: "WARN",
                title: t("auth.token-expired-title"),
                description: t("auth.token-expired-description"),
              });
            }
          }
        } else if (code === Code.PermissionDenied) {
          router.push({ name: "error.403" });
        }
      }
    }
    throw error;
  }
};

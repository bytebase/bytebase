import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import { t } from "@/plugins/i18n";
import { router } from "@/router";
import { useAuthStore, pushNotification } from "@/store";
import { silentContextKey, ignoredCodesContextKey } from "../context-key";

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
    // If silent is set to true, will NOT show redirect to other pages(e.g., 403, sign in page).
    const silent = req.contextValues.get(silentContextKey);
    const ignoredCodes = req.contextValues.get(ignoredCodesContextKey);

    if (!silent && error instanceof ConnectError) {
      const { code } = error;
      if (ignoredCodes?.includes(code)) {
        // omit specified errors
      } else {
        if (code === Code.Unauthenticated && req.method.name !== "Login") {
          // Skip show login modal when the request is to get current user.
          // When receiving 401 and is returned by our server, it means the current
          // login user's token becomes invalid. Thus we force the user to login again.
          authStore.unauthenticatedOccurred = true;
          if (authStore.isLoggedIn) {
            pushNotification({
              module: "bytebase",
              style: "WARN",
              title: t("auth.token-expired-title"),
              description: t("auth.token-expired-description"),
            });
          }
        } else if (code === Code.PermissionDenied) {
          // Jump to 403 page
          router.push({ name: "error.403" });
        }
      }
    }
    throw error;
  }
};

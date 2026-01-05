import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import { t } from "@/plugins/i18n";
import { router } from "@/router";
import { WORKSPACE_ROUTE_403 } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useAuthStore } from "@/store";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import { ignoredCodesContextKey, silentContextKey } from "../context-key";
import { refreshTokens } from "../refreshToken";

const extractPermissionDeniedDetail = (error: unknown) => {
  if (error instanceof ConnectError) {
    const details = error.findDetails(PermissionDeniedDetailSchema);
    if (details.length > 0) {
      return details[0];
    }
  }
  return undefined;
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
          // Don't retry refresh endpoint failures - just propagate error
          // The caller (refreshTokens catch block) will handle the notification
          if (req.method.name === "Refresh") {
            throw error;
          }

          // Try to refresh - catch ONLY refresh failures
          try {
            await refreshTokens();
          } catch {
            // Refresh itself failed - auth is broken
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

          // Refresh succeeded - retry the original request
          try {
            return await next(req);
          } catch (retryError) {
            // Retry failed - check if it's also an auth error
            if (
              retryError instanceof ConnectError &&
              retryError.code === Code.Unauthenticated
            ) {
              // New token also invalid (edge case) - auth is broken
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
            // Throw retry error (not original) - let other handlers deal with it
            throw retryError;
          }
        } else if (code === Code.PermissionDenied) {
          const errorDetail = extractPermissionDeniedDetail(error);
          router.push({
            name: WORKSPACE_ROUTE_403,
            query: errorDetail
              ? {
                  method: errorDetail.method,
                  permissions: errorDetail.requiredPermissions.join(","),
                  resources: errorDetail.resources.join(","),
                }
              : undefined,
          });
        }
      }
    }
    throw error;
  }
};

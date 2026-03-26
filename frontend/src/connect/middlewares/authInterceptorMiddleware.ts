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

const handleUnauthenticatedFailure = ({
  silent,
  isLoggedIn,
}: {
  silent: boolean;
  isLoggedIn: boolean;
}) => {
  const authStore = useAuthStore();
  authStore.unauthenticatedOccurred = true;
  if (!silent && isLoggedIn) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("auth.token-expired-title"),
      description: t("auth.token-expired-description"),
    });
  }
};

export const authInterceptor: Interceptor = (next) => async (req) => {
  try {
    return await next(req);
  } catch (error) {
    const authStore = useAuthStore();
    const silent = req.contextValues.get(silentContextKey);
    const ignoredCodes = req.contextValues.get(ignoredCodesContextKey);

    if (error instanceof ConnectError) {
      const { code } = error;
      if (ignoredCodes?.includes(code)) {
        throw error;
      }

      if (
        code === Code.Unauthenticated &&
        req.method.name !== "Login" &&
        req.method.name !== "Signup"
      ) {
        // Don't retry refresh endpoint failures - just propagate error.
        if (req.method.name === "Refresh") {
          throw error;
        }

        try {
          await refreshTokens();
        } catch (e) {
          console.error(e);
          handleUnauthenticatedFailure({
            silent,
            isLoggedIn: authStore.isLoggedIn,
          });
          throw error;
        }

        try {
          return await next(req);
        } catch (retryError) {
          if (
            retryError instanceof ConnectError &&
            retryError.code === Code.Unauthenticated
          ) {
            handleUnauthenticatedFailure({
              silent,
              isLoggedIn: authStore.isLoggedIn,
            });
          }
          throw retryError;
        }
      }

      if (
        !silent &&
        code === Code.PermissionDenied &&
        router.currentRoute.value.name !== WORKSPACE_ROUTE_403
      ) {
        const errorDetail = extractPermissionDeniedDetail(error);
        router.push({
          name: WORKSPACE_ROUTE_403,
          query: errorDetail
            ? {
                from: router.currentRoute.value.fullPath,
                api: errorDetail.method,
                permissions: errorDetail.requiredPermissions.join(","),
                resources: errorDetail.resources.join(","),
              }
            : undefined,
        });
      }
    }

    throw error;
  }
};

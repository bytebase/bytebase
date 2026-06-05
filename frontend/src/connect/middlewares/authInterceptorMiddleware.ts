import {
  Code,
  ConnectError,
  type Interceptor,
  type StreamRequest,
  type UnaryRequest,
} from "@connectrpc/connect";
import i18n from "@/react/i18n";
import { router } from "@/react/router";
import { WORKSPACE_ROUTE_403 } from "@/react/router/handles";
import { buildPermissionDeniedRouteQuery } from "@/react/router/permissionDenied";
import { pushNotification } from "@/store";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";
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

const buildPermissionDeniedQuery = ({
  errorDetail,
  req,
}: {
  errorDetail: ReturnType<typeof extractPermissionDeniedDetail>;
  req: StreamRequest | UnaryRequest;
}) => {
  const route = router.currentRoute.value;
  const permissions =
    errorDetail?.requiredPermissions ?? route.requiredPermissions;
  const resources = errorDetail?.resources ?? [];
  return buildPermissionDeniedRouteQuery({
    route,
    api: errorDetail?.method ?? `/${req.service.typeName}/${req.method.name}`,
    permissions,
    resources,
  });
};

const handleUnauthenticatedFailure = ({
  silent,
  isLoggedIn,
}: {
  silent: boolean;
  isLoggedIn: boolean;
}) => {
  // Flag the app store (the session source of truth) so React's
  // SessionExpiredSurface fires. Routed through the util bridge (a leaf module)
  // to avoid a load-time cycle — the app store imports the connect clients this
  // interceptor wraps.
  appStoreUtilBridge()?.setUnauthenticatedOccurred(true);
  if (!silent && isLoggedIn) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: i18n.t("auth.token-expired-title"),
      description: i18n.t("auth.token-expired-description"),
    });
  }
};

export const authInterceptor: Interceptor = (next) => async (req) => {
  try {
    return await next(req);
  } catch (error) {
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
            isLoggedIn: appStoreUtilBridge()?.isLoggedIn() ?? false,
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
              isLoggedIn: appStoreUtilBridge()?.isLoggedIn() ?? false,
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
          query: buildPermissionDeniedQuery({ errorDetail, req }),
        });
      }
    }

    throw error;
  }
};

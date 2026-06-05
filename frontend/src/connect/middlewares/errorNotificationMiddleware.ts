import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import i18n from "@/react/i18n";
import { pushNotification } from "@/store";
import { ignoredCodesContextKey, silentContextKey } from "../context-key";

export type SilentRequestOptions = {
  /**
   * If set to true, will NOT show push notifications when request error occurs.
   */
  silent?: boolean;

  /**
   * If set, will NOT handle specified status codes is this array.
   * Default to [NOT_FOUND], can be override.
   */
  ignoredCodes?: Code[];
};

export const errorNotificationInterceptor: Interceptor =
  (next) => async (req) => {
    try {
      const resp = await next(req);
      return resp;
    } catch (error) {
      const maybePushNotification = (title: string, description?: string) => {
        const silent = req.contextValues.get(silentContextKey);
        if (silent) return;
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title,
          description,
        });
      };
      if (error instanceof ConnectError) {
        const ignoredCodes = req.contextValues.get(ignoredCodesContextKey);
        if (
          (ignoredCodes.length === 0
            ? [Code.NotFound, Code.Unauthenticated]
            : ignoredCodes
          ).includes(error.code)
        ) {
          // ignored
        } else if (error.code === Code.PermissionDenied) {
          // The auth interceptor navigates permission failures to /403, where
          // the route-level guard displays the missing permission details.
        } else {
          const details = [error.message];
          maybePushNotification(
            `Code ${error.code}: ${Code[error.code]}`,
            details.join("\n")
          );
        }
      } else {
        // Other non-grpc errors.
        // E.g,. failed to encode protobuf for request data.
        // or other frontend exception.
        // Expect not to be here.
        maybePushNotification(
          `${i18n.t("common.error")}: ${req.service.name}/${req.method.name}`,
          String(error)
        );
      }
      throw error;
    }
  };

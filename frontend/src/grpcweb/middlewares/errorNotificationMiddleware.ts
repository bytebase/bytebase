import { ClientError, ServerError, Status } from "nice-grpc-common";
import { ClientMiddleware } from "nice-grpc-web";
import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";

export type SilentRequestOptions = {
  /**
   * if set to true, will NOT show push notifications when request error occurs.
   */
  silent?: boolean;

  /**
   * If set, will NOT handle specified status codes is this array.
   * Default to [NOT_FOUND], can be override.
   */
  ignoredCodes?: Status[];
};

/**
 * Way to define a grpc-web middleware
 * ClientMiddleware<CallOptionsExt = {}, RequiredCallOptionsExt = {}>
 * See
 *   - https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-client-middleware-deadline/src/index.ts
 *   - https://github.com/deeplay-io/nice-grpc/tree/master/packages/nice-grpc-web#middleware
 *   as an example.
 */
export const errorNotificationMiddleware: ClientMiddleware<SilentRequestOptions> =
  async function* (call, options) {
    const maybePushNotification = (title: string, description?: string) => {
      if (options.silent) return;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title,
        description,
      });
    };

    const handleError = async (error: unknown) => {
      if (error instanceof ClientError || error instanceof ServerError) {
        const { ignoredCodes = [Status.NOT_FOUND] } = options;
        if (ignoredCodes.includes(error.code)) {
          // ignored
        } else {
          maybePushNotification(
            `Code ${error.code}: ${error.message}`,
            error.details
          );
        }
      } else {
        // Other non-grpc errors.
        // E.g,. failed to encode protobuf for request data.
        // or other frontend exception.
        // Expect not to be here.
        maybePushNotification(
          `${t("common.error")}: ${call.method.path}`,
          String(error)
        );
      }
      throw error;
    };

    if (!call.responseStream) {
      try {
        const response = yield* call.next(call.request, options);
        return response;
      } catch (error) {
        await handleError(error);
      }
    } else {
      try {
        for await (const response of call.next(call.request, options)) {
          yield response;
        }
      } catch (error) {
        await handleError(error);
      }

      return;
    }
  };

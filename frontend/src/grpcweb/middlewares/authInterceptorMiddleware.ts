import { ClientError, ServerError, Status } from "nice-grpc-common";
import { ClientMiddleware } from "nice-grpc-web";
import { router } from "@/router";
import { useAuthStore } from "@/store";

export type IgnoreErrorsOptions = {
  /**
   * If set, will NOT handle specified status codes is this array.
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
export const authInterceptorMiddleware: ClientMiddleware<IgnoreErrorsOptions> =
  async function* (call, options) {
    const handleError = async (error: unknown) => {
      if (error instanceof ClientError || error instanceof ServerError) {
        const { code } = error;
        if (options.ignoredCodes?.includes(code)) {
          // omit specified errors
        } else {
          if (code === Status.UNAUTHENTICATED) {
            // "Kick out" sign in status if access token expires.
            try {
              await useAuthStore().logout();
            } finally {
              router.push({ name: "auth.signin" });
            }
          } else if (code === Status.PERMISSION_DENIED) {
            // Jump to 403 page
            router.push({ name: "error.403" });
          }
        }
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

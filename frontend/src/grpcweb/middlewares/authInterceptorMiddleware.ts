import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import { ClientError, ServerError, Status } from "nice-grpc-common";
import type { ClientMiddleware } from "nice-grpc-web";
import { router } from "@/router";
import { useAuthStore } from "@/store";
import { UserService } from "@/types/proto-es/v1/user_service_pb";
import { UserServiceDefinition } from "@/types/proto/v1/user_service";
import { silentContextKey, ignoredCodesContextKey } from "../context-key";

export type IgnoreErrorsOptions = {
  /**
   * If set to true, will NOT show redirect to other pages(e.g., 403, sign in page).
   */
  silent?: boolean;

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
      // If silent is set to true, will NOT show redirect to other pages(e.g., 403, sign in page).
      if (
        !options.silent &&
        (error instanceof ClientError || error instanceof ServerError)
      ) {
        const { code } = error;
        if (options.ignoredCodes?.includes(code)) {
          // omit specified errors
        } else {
          if (code === Status.UNAUTHENTICATED) {
            // Skip show login modal when the request is to get current user.
            if (
              call.method.path ===
              `/${UserServiceDefinition.fullName}/${UserServiceDefinition.methods.getCurrentUser.name}`
            ) {
              return;
            }
            // When receiving 401 and is returned by our server, it means the current
            // login user's token becomes invalid. Thus we force the user to login again.
            useAuthStore().unauthenticatedOccurred = true;
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

export const authInterceptor: Interceptor = (next) => async (req) => {
  try {
    const resp = await next(req);
    return resp;
  } catch (error) {
    // If silent is set to true, will NOT show redirect to other pages(e.g., 403, sign in page).
    const silent = req.contextValues.get(silentContextKey);
    const ignoredCodes = req.contextValues.get(ignoredCodesContextKey);

    if (!silent && error instanceof ConnectError) {
      const { code } = error;
      if (ignoredCodes?.includes(code)) {
        // omit specified errors
      } else {
        if (code === Code.Unauthenticated) {
          // Skip show login modal when the request is to get current user.
          if (
            req.method.parent.name === UserService.name &&
            req.method.name === UserService.method.getCurrentUser.name
          ) {
            // skip
          } else {
            // When receiving 401 and is returned by our server, it means the current
            // login user's token becomes invalid. Thus we force the user to login again.
            useAuthStore().unauthenticatedOccurred = true;
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

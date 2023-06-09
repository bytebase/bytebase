import { ClientError, ServerError, Status } from "nice-grpc-common";

import { useAuthStore } from "@/store";
import { router } from "@/router";
import { ClientMiddleware } from "nice-grpc-web";

/**
 * Way to define a grpc-web middleware
 * ClientMiddleware<CallOptionsExt = {}, RequiredCallOptionsExt = {}>
 * See https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-client-middleware-deadline/src/index.ts
 *   as an example.
 */
export const authInterceptorMiddleware: ClientMiddleware = async function* (
  call,
  options
) {
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

const handleError = async (error: unknown) => {
  if (error instanceof ClientError || error instanceof ServerError) {
    if (error.code === Status.UNAUTHENTICATED) {
      // "Kick out" sign in status if access token expires.
      try {
        await useAuthStore().logout();
      } finally {
        router.push({ name: "auth.signin" });
      }
    } else if (error.code === Status.PERMISSION_DENIED) {
      // Jump to 403 page
      router.push({ name: "error.403" });
    }
  }
  throw error;
};

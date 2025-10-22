import { type Interceptor } from "@connectrpc/connect";
import { useLastActivity } from "@/composables/useLastActivity";
import { useCurrentUserV1 } from "@/store";

// TODO(ed): some requests are called automatically by the script, for example, refresh the issue status.
export const activeInterceptor: Interceptor = (next) => async (req) => {
  const resp = await next(req);
  const me = useCurrentUserV1();
  // ignore the GetCurrentUser method, it's automatically called by the script.
  if (me.value && req.method.name !== "GetCurrentUser") {
    const { lastActivityTs } = useLastActivity();
    lastActivityTs.value = Date.now();
  }
  return resp;
};

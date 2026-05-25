import { type Interceptor } from "@connectrpc/connect";
import { useCurrentUserV1 } from "@/store";
import { storageKeyLastActivity } from "@/utils/storage-keys";

export const activeInterceptor: Interceptor = (next) => async (req) => {
  const resp = await next(req);
  const me = useCurrentUserV1();
  // ignore the GetCurrentUser method, it's automatically called by the script.
  if (me.value && req.method.name !== "GetCurrentUser") {
    try {
      localStorage.setItem(
        storageKeyLastActivity(me.value.email),
        String(Date.now())
      );
    } catch {
      // ignore quota / disabled-storage errors
    }
  }
  return resp;
};

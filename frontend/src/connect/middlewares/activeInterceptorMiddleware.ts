import { type Interceptor } from "@connectrpc/connect";
import { getCurrentUserV1 } from "@/store";
import { storageKeyLastActivity } from "@/utils/storage-keys";

export const activeInterceptor: Interceptor = (next) => async (req) => {
  const resp = await next(req);
  const me = getCurrentUserV1();
  // ignore the GetCurrentUser method, it's automatically called by the script.
  if (me.email && req.method.name !== "GetCurrentUser") {
    try {
      localStorage.setItem(
        storageKeyLastActivity(me.email),
        String(Date.now())
      );
    } catch {
      // ignore quota / disabled-storage errors
    }
  }
  return resp;
};

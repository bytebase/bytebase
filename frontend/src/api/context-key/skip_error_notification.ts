import { createContextKey } from "@connectrpc/connect";

// Unlike silentContextKey, permission-denied redirects and session-expiry
// handling still apply.
export const skipErrorNotificationContextKey = createContextKey<boolean>(
  false,
  {
    description:
      "the caller shows its own error notification, skip the global one",
  }
);

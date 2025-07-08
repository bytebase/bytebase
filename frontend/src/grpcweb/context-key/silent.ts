import { createContextKey } from "@connectrpc/connect";

export const silentContextKey = createContextKey<boolean>(false, {
  description: "silent mode, won't redirect nor pop notifications",
});

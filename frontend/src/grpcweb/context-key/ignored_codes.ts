import { Code, createContextKey } from "@connectrpc/connect";

export const ignoredCodesContextKey = createContextKey<Code[]>([], {
  description: "ignored status codes",
});

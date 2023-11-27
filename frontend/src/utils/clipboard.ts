import type { toClipboard as ToClipboard } from "@soerenmartius/vue3-clipboard";

export const toClipboard: typeof ToClipboard = async (text, action) => {
  const { toClipboard } = await import("@soerenmartius/vue3-clipboard");
  return toClipboard(text, action);
};

import { useLocalStorage } from "@vueuse/core";

export const DISMISS_PLACEHOLDER = useLocalStorage(
  "bb.plugin.open-ai.dismiss-placeholder",
  false
);

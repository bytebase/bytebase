import { useLocalStorage } from "@vueuse/core";
import { STORAGE_KEY_AI_DISMISS } from "@/utils";

export const DISMISS_PLACEHOLDER = useLocalStorage(
  STORAGE_KEY_AI_DISMISS,
  false
);

import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { STORAGE_KEY_BACK_PATH } from "@/utils";

export const useRouterStore = defineStore("router", () => {
  const backPath = useLocalStorage(STORAGE_KEY_BACK_PATH, "/");

  const setBackPath = (path: string) => {
    backPath.value = path;
    return path;
  };

  return {
    backPath,
    setBackPath,
  };
});

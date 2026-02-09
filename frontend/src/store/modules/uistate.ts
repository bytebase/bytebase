import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import {
  STORAGE_KEY_HELP_DRAWERS_DISMISSED,
  storageKeyCollapseState,
  storageKeyIntroState,
  useDynamicLocalStorage,
} from "@/utils";

export const useUIStateStore = defineStore("uistate", () => {
  const currentUser = useCurrentUserV1();

  const collapseState = useDynamicLocalStorage<Record<string, boolean>>(
    computed(() => storageKeyCollapseState(currentUser.value.email)),
    {}
  );

  const introState = useDynamicLocalStorage<Record<string, boolean>>(
    computed(() => storageKeyIntroState(currentUser.value.email)),
    {}
  );

  const allHelpDrawersDismissed = useLocalStorage(
    STORAGE_KEY_HELP_DRAWERS_DISMISSED,
    false
  );

  const getIntroStateByKey = (key: string): boolean => {
    return introState.value[key] ?? false;
  };

  const saveIntroStateByKey = async ({
    key,
    newState,
  }: {
    key: string;
    newState: boolean;
  }) => {
    introState.value = { ...introState.value, [key]: newState };
    return newState;
  };

  const restoreState = () => {
    // No-op: useDynamicLocalStorage handles restore automatically
  };

  return {
    allHelpDrawersDismissed,
    collapseState,
    introState,
    saveIntroStateByKey,
    restoreState,
    getIntroStateByKey,
  };
});

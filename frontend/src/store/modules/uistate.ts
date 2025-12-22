import { isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { useCurrentUserV1 } from "@/store";

export interface UIState {
  collapseStateByKey: Map<string, boolean>;
  introStateByKey: Map<string, boolean>;
}

export const useUIStateStore = defineStore("uistate", () => {
  const state = reactive<UIState>({
    collapseStateByKey: new Map(),
    introStateByKey: new Map(),
  });
  const currentUser = useCurrentUserV1();

  const COLLAPSE_MODULE_KEY = computed(
    () => `ui.list.collapse.${currentUser.value.name}`
  );
  const INTRO_MODULE_KEY = computed(() => `ui.intro.${currentUser.value.name}`);

  const getIntroStateByKey = (key: string): boolean => {
    return stateByKey(state.introStateByKey, INTRO_MODULE_KEY.value, key);
  };

  const setCollapseState = (fullState: Record<string, boolean>) => {
    const newMap = new Map();
    for (const key of Object.keys(fullState)) {
      newMap.set(key, fullState[key]);
    }
    state.collapseStateByKey = newMap;
  };

  const setIntroState = (fullState: Record<string, boolean>) => {
    const newMap = new Map();
    for (const key of Object.keys(fullState)) {
      newMap.set(key, fullState[key]);
    }
    state.introStateByKey = newMap;
  };

  const setIntroStateByKey = ({
    key,
    newState,
  }: {
    key: string;
    newState: boolean;
  }) => {
    state.introStateByKey.set(key, newState);
  };

  const restoreState = () => {
    const storedCollapseState = localStorage.getItem(COLLAPSE_MODULE_KEY.value);
    const collapseState = storedCollapseState
      ? JSON.parse(storedCollapseState)
      : {};
    setCollapseState(collapseState);

    const storedIntroState = localStorage.getItem(INTRO_MODULE_KEY.value);
    const introState = storedIntroState ? JSON.parse(storedIntroState) : {};
    setIntroState(introState);
  };

  const saveIntroStateByKey = async ({
    key,
    newState,
  }: {
    key: string;
    newState: boolean;
  }) => {
    const state = saveStateByKey(INTRO_MODULE_KEY.value, key, newState);
    setIntroStateByKey({ key, newState });
    return state;
  };

  return {
    saveIntroStateByKey,
    restoreState,
    getIntroStateByKey,
  };
});

const stateByKey = function (
  stateMap: Map<string, boolean>,
  module: string,
  key: string
): boolean {
  const memState = stateMap.get(key);
  if (!isUndefined(memState)) {
    return memState;
  }
  const localStorageState = localStorage.getItem(module);
  if (localStorageState) {
    return JSON.parse(localStorageState)[key] || false;
  }
  return false;
};

const saveStateByKey = function (
  module: string,
  key: string,
  state: boolean
): boolean {
  const item = localStorage.getItem(module);
  const fullState = item ? JSON.parse(item) : {};
  fullState[key] = state;
  localStorage.setItem(module, JSON.stringify(fullState));
  return fullState[key];
};

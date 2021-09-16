import isUndefined from "lodash-es/isUndefined";

const COLLAPSE_MODULE = "ui.list.collapse";
const INTRO_MODULE = "ui.intro";

export interface UIState {
  collapseStateByKey: Map<string, boolean>;
  introStateByKey: Map<string, boolean>;
}

const state: () => UIState = () => ({
  collapseStateByKey: new Map(),
  introStateByKey: new Map(),
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
  const fullState = item ? JSON.parse(item) || {} : {};
  fullState[key] = state;
  localStorage.setItem(module, JSON.stringify(fullState));
  return fullState[key];
};

const getters = {
  collapseStateByKey:
    (state: UIState) =>
    (key: string): boolean => {
      return stateByKey(state.collapseStateByKey, COLLAPSE_MODULE, key);
    },

  introStateByKey:
    (state: UIState) =>
    (key: string): boolean => {
      return stateByKey(state.introStateByKey, INTRO_MODULE, key);
    },
};

const actions = {
  async restoreState({ commit }: any) {
    const storedCollapseState = localStorage.getItem(COLLAPSE_MODULE);
    const collapseState = storedCollapseState
      ? JSON.parse(storedCollapseState) || {}
      : {};
    commit("setCollapseState", collapseState);

    const storedIntroState = localStorage.getItem(INTRO_MODULE);
    const introState = storedIntroState
      ? JSON.parse(storedIntroState) || {}
      : {};
    commit("setIntroState", introState);
  },

  async savecollapseStateByKey(
    { commit }: any,
    {
      key,
      collapse,
    }: {
      key: string;
      collapse: boolean;
    }
  ) {
    const state = saveStateByKey(COLLAPSE_MODULE, key, collapse);
    commit("setcollapseStateByKey", { key, collapse: state });
    return state;
  },

  async saveIntroStateByKey(
    { commit }: any,
    {
      key,
      newState,
    }: {
      key: string;
      newState: boolean;
    }
  ) {
    const state = saveStateByKey(INTRO_MODULE, key, newState);
    commit("setIntroStateByKey", { key, newState });
    return state;
  },
};

const mutations = {
  setCollapseState(state: UIState, fullState: any) {
    const newMap = new Map();
    for (const key in fullState) {
      newMap.set(key, fullState[key]);
    }
    state.collapseStateByKey = newMap;
  },

  setcollapseStateByKey(
    state: UIState,
    {
      key,
      collapse,
    }: {
      key: string;
      collapse: boolean;
    }
  ) {
    state.collapseStateByKey.set(key, collapse);
  },

  setIntroState(state: UIState, fullState: any) {
    const newMap = new Map();
    for (const key in fullState) {
      newMap.set(key, fullState[key]);
    }
    state.introStateByKey = newMap;
  },

  setIntroStateByKey(
    state: UIState,
    {
      key,
      newState,
    }: {
      key: string;
      newState: boolean;
    }
  ) {
    state.introStateByKey.set(key, newState);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};

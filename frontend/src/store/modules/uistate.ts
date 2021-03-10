import isUndefined from "lodash-es/isUndefined";

const EXPAND_MODULE = "ui.list.expand";
const INTRO_MODULE = "ui.intro";

export interface UIState {
  expandStateByKey: Map<string, boolean>;
  introStateByKey: Map<string, boolean>;
}

const state: () => UIState = () => ({
  expandStateByKey: new Map(),
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
    return JSON.parse(localStorageState)[key];
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

const getters = {
  expandStateByKey: (state: UIState) => (key: string) => {
    return stateByKey(state.expandStateByKey, EXPAND_MODULE, key);
  },

  introStateByKey: (state: UIState) => (key: string) => {
    return stateByKey(state.introStateByKey, INTRO_MODULE, key);
  },
};

const actions = {
  async restoreState({ commit }: any) {
    const storedExpandState = localStorage.getItem(EXPAND_MODULE);
    const expandState = storedExpandState ? JSON.parse(storedExpandState) : {};
    if (expandState) {
      commit("setExpandState", expandState);
    }

    const storedIntroState = localStorage.getItem(INTRO_MODULE);
    const introState = storedIntroState ? JSON.parse(storedIntroState) : {};
    if (introState) {
      commit("setIntroState", introState);
    }
  },

  async saveExpandStateByKey(
    { commit }: any,
    {
      key,
      expand,
    }: {
      key: string;
      expand: boolean;
    }
  ) {
    const state = saveStateByKey(EXPAND_MODULE, key, expand);
    commit("setExpandStateByKey", { key, expand: state });
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
  setExpandState(state: UIState, fullState: any) {
    const newMap = new Map();
    for (const key in fullState) {
      newMap.set(key, fullState[key]);
    }
    state.expandStateByKey = newMap;
  },

  setExpandStateByKey(
    state: UIState,
    {
      key,
      expand,
    }: {
      key: string;
      expand: boolean;
    }
  ) {
    state.expandStateByKey.set(key, expand);
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

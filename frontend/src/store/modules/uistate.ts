import isUndefined from "lodash-es/isUndefined";

export interface UIState {
  expandStateByKey: Map<string, boolean>;
}

const state: () => UIState = () => ({
  expandStateByKey: new Map(),
});

const expandStateByKey = function (state: UIState, key: string): boolean {
  const expandState = state.expandStateByKey.get(key);
  if (!isUndefined(expandState)) {
    return expandState;
  }
  const localStorageExpandState = localStorage.getItem("ui.list.expand");
  if (localStorageExpandState) {
    return JSON.parse(localStorageExpandState)[key];
  }
  return false;
};

const saveExpandStateByKey = function (
  commit: any,
  key: string,
  expand: boolean
): boolean {
  const item = localStorage.getItem("ui.list.expand");
  const fullState = item ? JSON.parse(item) : {};
  fullState[key] = expand;
  localStorage.setItem("ui.list.expand", JSON.stringify(fullState));
  commit("setExpandStateByKey", { key, expand });
  return fullState[key];
};

const getters = {
  expandStateByKey: (state: UIState) => (key: string) => {
    return expandStateByKey(state, key);
  },
};

const actions = {
  async restoreExpandState({ commit }: any) {
    const item = localStorage.getItem("ui.list.expand");
    const fullState = item ? JSON.parse(item) : {};
    if (fullState) {
      commit("setExpandState", fullState);
    }
    return fullState;
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
    return saveExpandStateByKey(commit, key, expand);
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};

import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { TabInfo, AnyTabInfo, EditorSelectorState } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

const state: () => EditorSelectorState = () => ({
  queryTabs: [],
  activeTab: {
    id: uuidv1(),
    idx: 0,
    label: "Untitled Queries",
    isSaved: true,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    queries: "",
  },
  activeTabIdx: 1,
});

const getters = {
  currentTab(state: EditorSelectorState) {
    const idx = state.queryTabs.findIndex(
      (tab: TabInfo) => tab.idx === state.activeTabIdx
    );
    return idx === -1 ? {} : state.queryTabs[idx];
  },
  hasTabs(state: EditorSelectorState) {
    return state.queryTabs.length > 0;
  },
};

const mutations = {
  [types.SET_EDITOR_SELECTOR_STATE](
    state: EditorSelectorState,
    payload: Partial<EditorSelectorState>
  ) {
    Object.assign(state, payload);
  },
  [types.ADD_TAB](state: EditorSelectorState, payload: TabInfo) {
    state.queryTabs.push(payload);
  },
  [types.REMOVE_TAB](state: EditorSelectorState, payload: TabInfo) {
    state.queryTabs.splice(state.queryTabs.indexOf(payload), 1);
  },
  [types.UPDATE_TAB](state: EditorSelectorState, payload: TabInfo) {
    const idx = state.queryTabs.findIndex(
      (tab: TabInfo) => tab.id === payload.id
    );
    Object.assign(state.queryTabs[idx], payload);
  },
};

type ActionsMap = {
  setEditorSelectorState: typeof mutations.SET_EDITOR_SELECTOR_STATE;
};

const actions = {
  ...makeActions<ActionsMap>({
    setEditorSelectorState: types.SET_EDITOR_SELECTOR_STATE,
  }),
  addTab({ commit, state }: any, payload: AnyTabInfo) {
    const id = uuidv1();
    const idx =
      state.queryTabs.length === 0
        ? 0
        : Math.max(...state.queryTabs.map((tab: TabInfo) => tab.idx)) + 1;
    const newTab = {
      id,
      idx,
      label: payload.label ? payload.label : `Untitled Queries`,
      isSaved: true,
      savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
      queries: payload.queries || "",
    };

    commit(types.SET_EDITOR_SELECTOR_STATE, {
      activeTab: newTab,
      activeTabIdx: idx,
    });
    commit(types.ADD_TAB, newTab);
  },
  removeTab({ commit, state, dispatch }: any, payload: TabInfo) {
    commit(types.REMOVE_TAB, payload);
    const tabsLength = state.queryTabs.length;

    if (tabsLength > 0) {
      dispatch("setActiveTab", state.queryTabs[tabsLength - 1]);
    }
  },
  updateTab({ commit, state }: any, payload: AnyTabInfo) {
    const { idx } = payload;
    const tab = state.queryTabs.find((tab: TabInfo) => tab.idx === idx);

    if (tab) {
      commit(types.SET_EDITOR_SELECTOR_STATE, {
        activeTab: tab,
        activeTabIdx: idx,
      });
      commit(types.UPDATE_TAB, payload);
    }
  },
  setActiveTab({ commit, state }: any, payload: TabInfo) {
    const { idx } = payload;
    const tab = state.queryTabs.find((tab: TabInfo) => tab.idx === idx);

    if (tab) {
      commit(types.SET_EDITOR_SELECTOR_STATE, {
        activeTab: tab,
        activeTabIdx: idx,
      });
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};

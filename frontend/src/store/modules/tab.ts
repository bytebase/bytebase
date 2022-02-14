import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { TabInfo, AnyTabInfo, TabState } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

export const getDefaultTab = () => {
  return {
    id: uuidv1(),
    label: "Untitled Sheet",
    isSaved: false,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    queryStatement: "",
    selectedStatement: "",
  };
};

// tab store is the sheet store for frontend
const state: () => TabState = () => {
  return {
    tabList: [],
    currentTabId: "",
  };
};

const getters = {
  currentTab(state: TabState) {
    const idx = state.tabList.findIndex(
      (tab: TabInfo) => tab.id === state.currentTabId
    );
    return (idx === -1 ? {} : state.tabList[idx]) as TabInfo;
  },
  hasTabs(state: TabState) {
    return state.tabList.length > 0;
  },
};

const mutations = {
  [types.SET_TAB_STATE](state: TabState, payload: Partial<TabState>) {
    Object.assign(state, payload);
  },
  [types.ADD_TAB](state: TabState, payload: TabInfo) {
    state.tabList.push(payload);
  },
  [types.REMOVE_TAB](state: TabState, payload: TabInfo) {
    state.tabList.splice(state.tabList.indexOf(payload), 1);
  },
  [types.UPDATE_TAB](state: TabState, payload: TabInfo) {
    const idx = state.tabList.findIndex(
      (tab: TabInfo) => tab.id === payload.id
    );
    Object.assign(state.tabList[idx], payload);
  },
  [types.UPDATE_CURRENT_TAB](state: TabState, payload: AnyTabInfo) {
    const idx = state.tabList.findIndex(
      (tab: TabInfo) => tab.id === state.currentTabId
    );
    Object.assign(state.tabList[idx], {
      ...state.tabList[idx],
      ...payload,
    });
  },
  [types.SET_CURRENT_TAB_ID](state: TabState, payload: string) {
    state.currentTabId = payload;
  },
};

type ActionsMap = {
  setTabState: typeof mutations.SET_TAB_STATE;
  removeTab: typeof mutations.REMOVE_TAB;
};

const actions = {
  ...makeActions<ActionsMap>({
    setTabState: types.SET_TAB_STATE,
    removeTab: types.REMOVE_TAB,
  }),
  addTab({ commit }: any, payload: AnyTabInfo) {
    const defaultTab = getDefaultTab();

    const newTab = {
      ...defaultTab,
      ...payload,
    };

    commit(types.SET_TAB_STATE, {
      currentTabId: newTab.id,
    });
    commit(types.ADD_TAB, newTab);
  },
  updateCurrentTab({ commit }: any, payload: AnyTabInfo) {
    commit(types.UPDATE_CURRENT_TAB, payload);
  },
  setCurrentTabId({ commit, state }: any, payload: string) {
    commit(types.SET_CURRENT_TAB_ID, payload);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};

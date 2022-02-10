import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { TabInfo, AnyTabInfo, TabState } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

const getDefaultTab = () => {
  return {
    id: uuidv1(),
    label: "Untitled Queries",
    isSaved: false,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    queryStatement: "",
    selectedStatement: "",
  };
};

// this store for frontend
const state: () => TabState = () => {
  return {
    tabList: [],
    activeTabId: "",
  };
};

const getters = {
  currentTab(state: TabState) {
    const idx = state.tabList.findIndex(
      (tab: TabInfo) => tab.id === state.activeTabId
    );
    return (idx === -1 ? {} : state.tabList[idx]) as TabInfo;
  },
  hasTabs(state: TabState) {
    return state.tabList.length > 0;
  },
};

const mutations = {
  [types.SET_EDITOR_SELECTOR_STATE](
    state: TabState,
    payload: Partial<TabState>
  ) {
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
  [types.UPDATE_ACTIVE_TAB](state: TabState, payload: AnyTabInfo) {
    const idx = state.tabList.findIndex(
      (tab: TabInfo) => tab.id === state.activeTabId
    );
    Object.assign(state.tabList[idx], {
      ...state.tabList[idx],
      ...payload,
    });
  },
  [types.SET_ACTIVE_TAB_ID](state: TabState, payload: string) {
    state.activeTabId = payload;
  },
};

type ActionsMap = {
  setTabState: typeof mutations.SET_EDITOR_SELECTOR_STATE;
};

const actions = {
  ...makeActions<ActionsMap>({
    setTabState: types.SET_EDITOR_SELECTOR_STATE,
  }),
  addTab({ commit }: any, payload: AnyTabInfo) {
    const defaultTab = getDefaultTab();

    const newTab = {
      ...defaultTab,
      ...payload,
    };

    commit(types.SET_EDITOR_SELECTOR_STATE, {
      activeTabId: newTab.id,
    });
    commit(types.ADD_TAB, newTab);
  },
  removeTab({ commit, state, dispatch }: any, payload: TabInfo) {
    commit(types.REMOVE_TAB, payload);
    const tabsLength = state.tabList.length;

    if (tabsLength > 0) {
      dispatch("setActiveTabId", state.tabList[tabsLength - 1].id);
    }
  },
  updateActiveTab({ commit }: any, payload: AnyTabInfo) {
    commit(types.UPDATE_ACTIVE_TAB, payload);
  },
  setActiveTabId({ commit, state }: any, payload: string) {
    commit(types.SET_ACTIVE_TAB_ID, payload);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};

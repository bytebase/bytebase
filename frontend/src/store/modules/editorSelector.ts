import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { TabInfo, AnyTabInfo, EditorSelectorState } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

const getDefaultTab = () => {
  return {
    id: uuidv1(),
    label: "Untitled Queries",
    isSaved: true,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    queryStatement: "",
    selectedStatement: "",
  };
};

const state: () => EditorSelectorState = () => {
  const defaultTab = getDefaultTab();

  return {
    queryTabList: [defaultTab],
    activeTabId: defaultTab.id,
  };
};

const getters = {
  currentTab(state: EditorSelectorState) {
    const idx = state.queryTabList.findIndex(
      (tab: TabInfo) => tab.id === state.activeTabId
    );
    return (idx === -1 ? {} : state.queryTabList[idx]) as TabInfo;
  },
  hasTabs(state: EditorSelectorState) {
    return state.queryTabList.length > 0;
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
    state.queryTabList.push(payload);
  },
  [types.REMOVE_TAB](state: EditorSelectorState, payload: TabInfo) {
    state.queryTabList.splice(state.queryTabList.indexOf(payload), 1);
  },
  [types.UPDATE_TAB](state: EditorSelectorState, payload: TabInfo) {
    const idx = state.queryTabList.findIndex(
      (tab: TabInfo) => tab.id === payload.id
    );
    Object.assign(state.queryTabList[idx], payload);
  },
  [types.UPDATE_ACTIVE_TAB](state: EditorSelectorState, payload: AnyTabInfo) {
    const idx = state.queryTabList.findIndex(
      (tab: TabInfo) => tab.id === state.activeTabId
    );
    Object.assign(state.queryTabList[idx], {
      ...state.queryTabList[idx],
      ...payload,
    });
  },
  [types.SET_ACTIVE_TAB_ID](state: EditorSelectorState, payload: string) {
    state.activeTabId = payload;
  },
};

type ActionsMap = {
  setEditorSelectorState: typeof mutations.SET_EDITOR_SELECTOR_STATE;
};

const actions = {
  ...makeActions<ActionsMap>({
    setEditorSelectorState: types.SET_EDITOR_SELECTOR_STATE,
  }),
  addTab({ commit }: any, payload: AnyTabInfo) {
    const newTab = {
      ...getDefaultTab(),
      label: payload.label ? payload.label : `Untitled Queries`,
      isSaved: true,
      savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
      queryStatement: payload.queryStatement || "",
      selectedStatement: payload.selectedStatement || "",
    };

    commit(types.SET_EDITOR_SELECTOR_STATE, {
      activeTabId: newTab.id,
    });
    commit(types.ADD_TAB, newTab);
  },
  removeTab({ commit, state, dispatch }: any, payload: TabInfo) {
    commit(types.REMOVE_TAB, payload);
    const tabsLength = state.queryTabList.length;

    if (tabsLength > 0) {
      dispatch("setActiveTabId", state.queryTabList[tabsLength - 1].id);
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

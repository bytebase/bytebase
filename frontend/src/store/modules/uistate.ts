import { useLocalStorage } from "@vueuse/core";
import { isUndefined } from "lodash-es";
import { defineStore } from "pinia";

const COLLAPSE_MODULE = "ui.list.collapse";
const INTRO_MODULE = "ui.intro";

// default to false
const issueFormatStatementOnSave = useLocalStorage(
  "ui.state.issue.format-statement-on-save",
  false
);

export interface UIState {
  collapseStateByKey: Map<string, boolean>;
  introStateByKey: Map<string, boolean>;
}

export const useUIStateStore = defineStore("uistate", {
  state: (): UIState => ({
    collapseStateByKey: new Map(),
    introStateByKey: new Map(),
  }),
  getters: {
    issueFormatStatementOnSave: () => {
      return issueFormatStatementOnSave.value;
    },
  },
  actions: {
    getCollapseStateByKey(key: string): boolean {
      return stateByKey(this.collapseStateByKey, COLLAPSE_MODULE, key);
    },
    getIntroStateByKey(key: string): boolean {
      return stateByKey(this.introStateByKey, INTRO_MODULE, key);
    },
    setCollapseState(fullState: any) {
      const newMap = new Map();
      for (const key in fullState) {
        newMap.set(key, fullState[key]);
      }
      this.collapseStateByKey = newMap;
    },
    setCollapseStateByKey({
      key,
      collapse,
    }: {
      key: string;
      collapse: boolean;
    }) {
      this.collapseStateByKey.set(key, collapse);
    },
    setIntroState(fullState: any) {
      const newMap = new Map();
      for (const key in fullState) {
        newMap.set(key, fullState[key]);
      }
      this.introStateByKey = newMap;
    },
    setIntroStateByKey({ key, newState }: { key: string; newState: boolean }) {
      this.introStateByKey.set(key, newState);
    },
    setIssueFormatStatementOnSave(value: boolean) {
      issueFormatStatementOnSave.value = value;
    },
    async restoreState() {
      const storedCollapseState = localStorage.getItem(COLLAPSE_MODULE);
      const collapseState = storedCollapseState
        ? JSON.parse(storedCollapseState)
        : {};
      this.setCollapseState(collapseState);

      const storedIntroState = localStorage.getItem(INTRO_MODULE);
      const introState = storedIntroState ? JSON.parse(storedIntroState) : {};
      this.setIntroState(introState);
    },
    async saveCollapseStateByKey({
      key,
      collapse,
    }: {
      key: string;
      collapse: boolean;
    }) {
      const state = saveStateByKey(COLLAPSE_MODULE, key, collapse);
      this.setCollapseStateByKey({ key, collapse: state });
      return state;
    },
    async saveIntroStateByKey({
      key,
      newState,
    }: {
      key: string;
      newState: boolean;
    }) {
      const state = saveStateByKey(INTRO_MODULE, key, newState);
      this.setIntroStateByKey({ key, newState });
      return state;
    },
  },
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

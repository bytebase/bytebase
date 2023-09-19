import { defineStore } from "pinia";
import { ref, watch, WatchCallback } from "vue";
import { IssueState } from "@/types";

export const useIssueStore = defineStore("issue", {
  state: (): IssueState => ({
    issueById: new Map(),
    isCreatingIssue: false,
  }),
  getters: {
    issueList: (state) => {
      return [...state.issueById.values()];
    },
  },
  actions: {},
});

// expose global list refresh features
const REFRESH_ISSUE_LIST = ref(Math.random());
export const refreshIssueList = () => {
  REFRESH_ISSUE_LIST.value = Math.random();
};
export const useRefreshIssueList = (callback: WatchCallback) => {
  watch(REFRESH_ISSUE_LIST, callback);
};

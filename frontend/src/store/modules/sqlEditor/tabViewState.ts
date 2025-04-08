import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { useCurrentUserV1, extractUserId } from "@/store";
import {
  defaultViewState,
  type EditorPanelViewState as ViewState,
} from "@/types";
import { useDynamicLocalStorage } from "@/utils";

export const useTabViewStateStore = defineStore("tabViewState", () => {
  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));

  const viewStateByTab = useDynamicLocalStorage<
    Map</* tab.id */ string, ViewState>
  >(
    computed(() => `bb.sql-editor-tab-state.${userUID.value}`),
    new Map()
  );

  const removeViewState = (tabId: string) => {
    viewStateByTab.value.delete(tabId);
  };

  const getViewState = (tabId: string) => {
    return viewStateByTab.value.get(tabId) ?? defaultViewState();
  };

  const updateViewState = (tabId: string, patch: Partial<ViewState>) => {
    const curr = getViewState(tabId);
    if (!curr) return;

    Object.assign(curr, patch);
    viewStateByTab.value.set(tabId, curr);
  };

  const cloneViewState = (from: string, to: string) => {
    const vs = viewStateByTab.value.get(from);
    if (!vs) return;
    const cloned = cloneDeep(vs);
    viewStateByTab.value.set(to, cloned);
    return cloned;
  };

  return {
    getViewState,
    removeViewState,
    updateViewState,
    cloneViewState,
  };
});

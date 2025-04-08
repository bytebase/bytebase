import type { MaybeRef } from "@vueuse/core";
import { watchThrottled } from "@vueuse/core";
import { head, omit, pick, uniqBy, cloneDeep } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import {
  computed,
  inject,
  provide,
  watch,
  type InjectionKey,
  type Ref,
} from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useCurrentUserV1,
  extractUserId,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import {
  defaultViewState,
  type EditorPanelViewState as ViewState,
} from "@/types";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
  useDynamicLocalStorage,
} from "@/utils";

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

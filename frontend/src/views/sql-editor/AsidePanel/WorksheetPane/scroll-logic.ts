import scrollIntoView from "scroll-into-view-if-needed";
import { nextTick, ref, watch } from "vue";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { keyForDraft, keyForWorksheet, type GroupType } from "./common";

export const useScrollLogic = () => {
  const tabStore = useSQLEditorTabStore();
  const readyQueue = ref<GroupType[]>([]);
  const ready = ref<boolean>(false);

  const setReady = (group: GroupType) => {
    if (readyQueue.value.includes(group)) return;
    readyQueue.value.push(group);
  };

  const pendingScrollCurrentItemIntoView = ref<SQLEditorTab>();

  const scrollCurrentItemIntoView = async (tab: SQLEditorTab | undefined) => {
    pendingScrollCurrentItemIntoView.value = tab;
  };

  watch(
    () => readyQueue.value.length,
    (length) => {
      if (length === 4) {
        ready.value = true;
        scrollCurrentItemIntoView(tabStore.currentTab);
      }
    },
    { immediate: true }
  );

  const cleanup = () => {
    pendingScrollCurrentItemIntoView.value = undefined;
  };

  const scrollItemIntoViewByKey = async (key: string) => {
    await nextTick();
    const dom = document.querySelector(`[data-item-key="${key}"]`);
    if (dom) {
      scrollIntoView(dom, {
        scrollMode: "if-needed",
      });
    }
    cleanup();
  };

  watch(
    [pendingScrollCurrentItemIntoView, ready],
    ([tab, ready]) => {
      if (!tab) return;
      if (!ready) return;
      if (tab.worksheet) {
        const worksheet = useWorkSheetStore().getWorksheetByName(tab.worksheet);
        if (!worksheet) return cleanup();
        const key = keyForWorksheet(worksheet);
        scrollItemIntoViewByKey(key);
        return;
      }
      const key = keyForDraft(tab);
      scrollItemIntoViewByKey(key);
    },
    {
      immediate: true,
    }
  );

  return { setReady, scrollCurrentItemIntoView };
};

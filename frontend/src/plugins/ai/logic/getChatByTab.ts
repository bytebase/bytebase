import { last } from "lodash-es";
import { computed, ref, watch } from "vue";
import type { SQLEditorTab } from "@/types";
import { useConversationStore } from "../store";
import type { AIChatInfo, Conversation } from "../types";

const chatsByTab = new Map<string, AIChatInfo>();

const emptyChat: AIChatInfo = {
  list: ref([]),
  ready: ref(false),
  selected: ref(undefined),
};

const initializeChat = (tab: SQLEditorTab): AIChatInfo => {
  const store = useConversationStore();
  const ready = ref(false);
  store.fetchConversationListByConnection(tab.connection).then(() => {
    ready.value = true;
  });
  const list = computed(() => {
    const { instance, database } = tab.connection;
    return store.conversationList.filter(
      (c) => c.instance === instance && c.database === database
    );
  });
  const selected = ref<Conversation>();
  watch(
    [list, ready, selected],
    ([list, ready]) => {
      if (ready) {
        if (!selected.value) {
          selected.value = last(list);
        }
      }
    },
    { immediate: true }
  );
  return { list, ready, selected };
};

/**
 * Per-tab AI chat state, keyed by the tab's `(instance, database)`
 * connection. The returned `AIChatInfo` still holds Vue refs because the
 * conversation list is backed by a Pinia store — React consumers bridge
 * those refs (e.g. via `useVueState`). The tab itself is supplied by the
 * caller (sourced from the Zustand tab store) so this module no longer
 * depends on any Vue-reactive view of the current tab.
 */
export const getChatByTab = (tab: SQLEditorTab | undefined): AIChatInfo => {
  if (!tab) return emptyChat;
  const key = JSON.stringify({
    instance: tab.connection.instance,
    database: tab.connection.database,
  });
  const existed = chatsByTab.get(key);
  if (existed) return existed;
  const chat = initializeChat(tab);
  chatsByTab.set(key, chat);
  return chat;
};

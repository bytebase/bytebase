import { last } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { useConversationStore } from "../store";
import type { AIChatInfo, Conversation } from "../types";
import { useAIContext } from "./context";

const chatsByTab = new Map<string, AIChatInfo>();

export const useChatByTab = () => {
  const store = useConversationStore();

  const initializeChat = (tab: SQLEditorTab): AIChatInfo => {
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

  const getChatByTab = (tab: SQLEditorTab) => {
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

  return computed(() => {
    const tab = useSQLEditorTabStore().currentTab!;
    return getChatByTab(tab);
  });
};

export const useCurrentChat = () => {
  const { chat } = useAIContext();
  const list = computed(() => chat.value.list.value);
  const ready = computed(() => chat.value.ready.value);
  const selected = computed({
    get() {
      return chat.value.selected.value;
    },
    set(val) {
      chat.value.selected.value = val;
    },
  });
  return { list, ready, selected };
};

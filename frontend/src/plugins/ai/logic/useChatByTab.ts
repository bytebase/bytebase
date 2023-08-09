import { last } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useCurrentTab } from "@/store";
import { TabInfo } from "@/types";
import { useConversationStore } from "../store";
import { AIChatInfo, Conversation } from "../types";
import { useAIContext } from "./context";

const chatsByTab = new Map<string, AIChatInfo>();

export const useChatByTab = () => {
  const tab = useCurrentTab();
  const store = useConversationStore();

  const initializeChat = (tab: TabInfo): AIChatInfo => {
    const ready = ref(false);
    store.fetchConversationListByConnection(tab.connection).then(() => {
      ready.value = true;
    });
    const list = computed(() => {
      const { instanceId, databaseId } = tab.connection;
      return store.conversationList.filter(
        (c) => c.instanceId === instanceId && c.databaseId === databaseId
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

  const getChatByTab = (tab: TabInfo) => {
    const key = JSON.stringify(tab.connection);
    const existed = chatsByTab.get(key);
    if (existed) return existed;
    const chat = initializeChat(tab);
    chatsByTab.set(key, chat);
    return chat;
  };

  return computed(() => {
    return getChatByTab(tab.value);
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

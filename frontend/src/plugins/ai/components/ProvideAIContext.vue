<template>
  <slot />
</template>

<script lang="ts" setup>
import Emittery from "emittery";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, toRef } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useConnectionOfCurrentSQLEditorTab,
  useMetadata,
  useSettingV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { wrapRefAsPromise } from "@/utils";
import { provideAIContext, useChatByTab, useCurrentChat } from "../logic";
import { useConversationStore } from "../store";
import type { AIContext, AIContextEvents } from "../types";

type LocalState = {
  showHistoryDialog: boolean;
};

const state = reactive<LocalState>({
  showHistoryDialog: false,
});

const settingV1Store = useSettingV1Store();
const aiSetting = settingV1Store.getSettingByName("bb.ai");
const openAIKey = computed(() => aiSetting?.value?.aiSetting?.apiKey ?? "");
const openAIEndpoint = computed(
  () => aiSetting?.value?.aiSetting?.endpoint ?? ""
);
const openAIModel = computed(() => aiSetting?.value?.aiSetting?.model ?? "");
const { instance, database } = useConnectionOfCurrentSQLEditorTab();

const databaseMetadata = useMetadata(
  computed(() => database.value.name),
  false /* !skipCache */
);

const events: AIContextEvents = new Emittery();
const chat = useChatByTab();
const showHistoryDialog = toRef(state, "showHistoryDialog");
const pendingSendChat = ref<{ content: string }>();
const pendingPreInput = ref<string>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const schema = computed(() => tab.value?.connection.schema);
const store = useConversationStore();

const context: AIContext = {
  openAIKey,
  openAIEndpoint,
  openAIModel,
  engine: computed(() => instance.value.engine),
  databaseMetadata,
  schema,
  showHistoryDialog,
  chat,
  pendingSendChat,
  pendingPreInput,
  events,
};
provideAIContext(context);

const { ready, selected: selectedConversation } = useCurrentChat(context);

useEmitteryEventListener(events, "new-conversation", async ({ input }) => {
  if (!tab.value) return;
  await wrapRefAsPromise(ready, /* expected */ true);
  showHistoryDialog.value = false;

  if (
    !selectedConversation.value ||
    selectedConversation.value.messageList.length !== 0
  ) {
    // reuse if current chat is empty
    // create new chat otherwise
    const c = await store.createConversation({
      name: "",
      ...tab.value.connection,
    });
    selectedConversation.value = c;
  }
  if (input) {
    requestAnimationFrame(() => {
      pendingPreInput.value = input;
    });
  }
});

useEmitteryEventListener(events, "send-chat", async ({ content, newChat }) => {
  if (!tab.value) return;
  await wrapRefAsPromise(ready, /* expected */ true);
  if (newChat) {
    showHistoryDialog.value = false;
    const c = await store.createConversation({
      name: "",
      ...tab.value.connection,
    });
    selectedConversation.value = c;
  }
  requestAnimationFrame(() => {
    pendingSendChat.value = { content };
  });
});
</script>

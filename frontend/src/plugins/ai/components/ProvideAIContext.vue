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
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
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
const openAIKeySetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.key"
);
const openAIEndpointSetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.endpoint"
);
const openAIKey = computed(() => openAIKeySetting?.value?.stringValue ?? "");
const openAIEndpoint = computed(
  () => openAIEndpointSetting?.value?.stringValue ?? ""
);
const { instance, database } = useConnectionOfCurrentSQLEditorTab();

const databaseMetadata = useMetadata(
  computed(() => database.value.name),
  false /* !skipCache */,
  DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
);

const events: AIContextEvents = new Emittery();
const chat = useChatByTab();
const showHistoryDialog = toRef(state, "showHistoryDialog");
const pendingSendChat = ref<{ content: string }>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const store = useConversationStore();

const context: AIContext = {
  openAIKey,
  openAIEndpoint,
  engine: computed(() => instance.value.engine),
  databaseMetadata,
  showHistoryDialog,
  chat,
  pendingSendChat,
  events,
};
provideAIContext(context);

const { ready, selected: selectedConversation } = useCurrentChat(context);

useEmitteryEventListener(events, "new-conversation", async () => {
  if (!tab.value) return;
  await wrapRefAsPromise(ready, /* expected */ true);
  showHistoryDialog.value = false;
  const c = await store.createConversation({
    name: "",
    ...tab.value.connection,
  });
  selectedConversation.value = c;
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

<template>
  <slot />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
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
import {
  AISettingSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
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
const aiSetting = computed(() => {
  const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
  if (setting?.value?.value?.case === "ai") {
    return setting.value.value.value;
  }
  return create(AISettingSchema, {});
});
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
  aiSetting: aiSetting,
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

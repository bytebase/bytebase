<template>
  <slot />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import Emittery from "emittery";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, toRef, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useConnectionOfCurrentSQLEditorTab,
  useMetadata,
  useSettingV1Store,
  useSQLEditorTabStore,
  useCurrentUserV1,
} from "@/store";
import {
  AISettingSchema,
  Setting_SettingName,
  type AIProvider,
} from "@/types/proto-es/v1/setting_service_pb";
import { wrapRefAsPromise, useDynamicLocalStorage } from "@/utils";
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
  if (setting?.value?.value?.case === "aiSetting") {
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
const currentUser = useCurrentUserV1();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const schema = computed(() => tab.value?.connection.schema);
const store = useConversationStore();
const selectedProvider = useDynamicLocalStorage<AIProvider>(
  computed(
    () => `bb.sql-editor.ai-suggestion.provider.${currentUser.value.name}`
  ),
  aiSetting.value.providers[0]
);

watch(
  () => aiSetting.value.providers[0],
  (provider) => {
    if (!selectedProvider.value && provider) {
      selectedProvider.value = provider;
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

const context: AIContext = {
  aiSetting: aiSetting,
  provider: selectedProvider,
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

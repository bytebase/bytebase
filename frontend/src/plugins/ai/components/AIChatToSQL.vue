<template>
  <ChatPanel v-if="openAIKey" />
  <MockInputPlaceholder v-else />
</template>

<script lang="ts" setup>
import { computed, reactive, toRef } from "vue";
import Emittery from "emittery";
import { useLocalStorage } from "@vueuse/core";

import type { AIContextEvents } from "../types";
import { useChatByTab, provideAIContext } from "../logic";
import {
  useCurrentTab,
  useInstanceById,
  useMetadataByDatabaseId,
} from "@/store";
import ChatPanel from "./ChatPanel.vue";
import MockInputPlaceholder from "./MockInputPlaceholder.vue";
import { Connection } from "@/types";
import { useSettingV1Store } from "@/store/modules/v1/setting";

type LocalState = {
  showHistoryDialog: boolean;
};

const emit = defineEmits<{
  (
    event: "apply-statement",
    statement: string,
    conn: Connection,
    run: boolean
  ): void;
}>();

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
const tab = useCurrentTab();

const instance = useInstanceById(
  computed(() => tab.value.connection.instanceId)
);
const databaseMetadata = useMetadataByDatabaseId(
  computed(() => tab.value.connection.databaseId),
  false /* !skipCache */
);

const events: AIContextEvents = new Emittery();

events.on("apply-statement", ({ statement, run }) => {
  emit("apply-statement", statement, tab.value.connection, run);
});

const autoRun = useLocalStorage("bb.plugin.ai.auto-run", false);
autoRun.value = false;

const chat = useChatByTab();

provideAIContext({
  openAIKey,
  openAIEndpoint,
  engineType: computed(() => instance.value.engine),
  databaseMetadata: databaseMetadata,
  autoRun,
  showHistoryDialog: toRef(state, "showHistoryDialog"),
  chat,
  events,
});
</script>

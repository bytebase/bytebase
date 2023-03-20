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
  useSettingByName,
} from "@/store";
import ChatPanel from "./ChatPanel.vue";
import MockInputPlaceholder from "./MockInputPlaceholder.vue";
import { Connection } from "@/types";

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

const openAIKeySetting = useSettingByName("bb.plugin.openai.key");
const openAIKey = computed(() => openAIKeySetting.value?.value ?? "");
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
  engineType: computed(() => instance.value.engine),
  databaseMetadata: databaseMetadata,
  autoRun,
  showHistoryDialog: toRef(state, "showHistoryDialog"),
  chat,
  events,
});
</script>

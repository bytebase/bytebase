<template>
  <ChatPanel v-if="openAIKey" />
  <MockInputPlaceholder v-else />
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import { computed, reactive, toRef } from "vue";
import {
  useCurrentTab,
  useInstanceV1ByUID,
  useDatabaseV1ByUID,
  useMetadata,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { Connection } from "@/types";
import { useChatByTab, provideAIContext } from "../logic";
import type { AIContextEvents } from "../types";
import ChatPanel from "./ChatPanel.vue";
import MockInputPlaceholder from "./MockInputPlaceholder.vue";

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

const { instance } = useInstanceV1ByUID(
  computed(() => tab.value.connection.instanceId)
);
const { database } = useDatabaseV1ByUID(
  computed(() => tab.value.connection.databaseId)
);
const databaseMetadata = useMetadata(
  database.value.name,
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
  engine: computed(() => instance.value.engine),
  databaseMetadata: databaseMetadata,
  autoRun,
  showHistoryDialog: toRef(state, "showHistoryDialog"),
  chat,
  events,
});
</script>

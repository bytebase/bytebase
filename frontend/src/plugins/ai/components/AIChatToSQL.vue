<template>
  <ChatPanel v-if="openAIKey" />
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import { computed, reactive, toRef } from "vue";
import { useMetadata, useConnectionOfCurrentSQLEditorTab } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { SQLEditorConnection } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import type { AIContextEvents } from "../types";

const [{ useChatByTab, provideAIContext }, { default: ChatPanel }] =
  await Promise.all([import("../logic"), import("./ChatPanel.vue")]);

type LocalState = {
  showHistoryDialog: boolean;
};

withDefaults(
  defineProps<{
    allowConfig?: boolean;
  }>(),
  {
    allowConfig: true,
  }
);

const emit = defineEmits<{
  (
    event: "apply-statement",
    statement: string,
    conn: SQLEditorConnection,
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
const { connection, instance, database } = useConnectionOfCurrentSQLEditorTab();

const databaseMetadata = useMetadata(
  database.value.name,
  false /* !skipCache */,
  DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
);

const events: AIContextEvents = new Emittery();

events.on("apply-statement", ({ statement, run }) => {
  emit("apply-statement", statement, connection.value, run);
});

const autoRun = useLocalStorage("bb.plugin.ai.auto-run", false);
autoRun.value = false;

const chat = useChatByTab();

provideAIContext({
  openAIKey,
  openAIEndpoint,
  engine: computed(() => instance.value.engine),
  databaseMetadata,
  autoRun,
  showHistoryDialog: toRef(state, "showHistoryDialog"),
  chat,
  events,
});
</script>

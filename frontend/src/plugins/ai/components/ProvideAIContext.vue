<template>
  <slot />
</template>

<script lang="ts" setup>
import Emittery from "emittery";
import { computed, reactive, toRef } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useMetadata,
  useSettingV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { provideAIContext, useChatByTab } from "../logic";
import type { AIContextEvents } from "../types";

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

provideAIContext({
  openAIKey,
  openAIEndpoint,
  engine: computed(() => instance.value.engine),
  databaseMetadata,
  showHistoryDialog: toRef(state, "showHistoryDialog"),
  chat,
  events,
});
</script>

<template>
  <button
    v-if="openAIKey"
    class="w-full flex items-center gap-x-1 rounded-[3px] border border-control-border bg-white hover:bg-gray-100 pl-2 pr-1 py-1.5 outline-none"
    @click="state.showDialog = true"
  >
    <heroicons-outline:sparkles class="w-4 h-4 text-accent" />
    <span class="text-control-placeholder flex-1 text-left text-sm">
      {{ $t("plugin.ai.text-to-sql-placeholder") }}
    </span>

    <AIDialog v-if="state.showDialog" @close="state.showDialog = false" />
  </button>
</template>

<script lang="ts" setup>
import { computed, reactive, toRef } from "vue";

import AIDialog from "./AIDialog.vue";
import type { AIContextEvents } from "../types";
import { DatabaseMetadata } from "@/types/proto/store/database";
import type { EngineType } from "@/types";
import Emittery from "emittery";
import { provideAIContext } from "../logic";
import { useLocalStorage } from "@vueuse/core";
import { useSettingByName } from "@/store";

type LocalState = {
  showDialog: boolean;
};

const props = withDefaults(
  defineProps<{
    engineType?: EngineType;
    databaseMetadata?: DatabaseMetadata;
  }>(),
  {
    engineType: undefined,
    databaseMetadata: undefined,
  }
);

const emit = defineEmits<{
  (event: "apply-statement", statement: string, run: boolean): void;
  (event: "error", error: string): void;
  (event: "update:loading", loading: boolean): void;
}>();

const state = reactive<LocalState>({
  showDialog: false,
});

const openAIKeySetting = useSettingByName("bb.plugin.openai-key");
const openAIKey = computed(() => openAIKeySetting.value?.value ?? "");

const events: AIContextEvents = new Emittery();

events.on("apply-statement", ({ statement, run }) => {
  emit("apply-statement", statement, run);
  state.showDialog = false;
});
events.on("error", (error) => {
  emit("error", error);
});

const autoRun = useLocalStorage("bb.plugin.ai.auto-run", true);

provideAIContext({
  showDialog: toRef(state, "showDialog"),
  openAIKey,
  engineType: toRef(props, "engineType"),
  databaseMetadata: toRef(props, "databaseMetadata"),
  autoRun,
  events,
});
</script>

<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <EditorAction @execute="handleExecute" @save-sheet="trySaveSheet" />

    <ConnectionPathBar />

    <template v-if="!tabStore.isDisconnected">
      <SQLEditor @execute="handleExecute" @save-sheet="trySaveSheet" />
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>

    <ExecutingHintModal />

    <SaveSheetModal ref="saveSheetModal" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import type { ExecuteConfig, ExecuteOption } from "@/types";
import { useTabStore } from "@/store";
import SQLEditor from "./SQLEditor.vue";
import {
  EditorAction,
  ConnectionPathBar,
  ConnectionHolder,
  ExecutingHintModal,
  SaveSheetModal,
} from "../EditorCommon";
import { useExecuteSQL } from "@/composables/useExecuteSQL";

const tabStore = useTabStore();
const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();

const { execute } = useExecuteSQL();

const handleExecute = (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  execute(query, config, option);
};

const trySaveSheet = (sheetName?: string) => {
  saveSheetModal.value?.trySaveSheet(sheetName);
};
</script>

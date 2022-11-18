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
import EditorAction from "../EditorCommon/EditorAction.vue";
import SQLEditor from "./SQLEditor.vue";
import ConnectionPathBar from "../EditorCommon/ConnectionPathBar.vue";
import ConnectionHolder from "../EditorCommon/ConnectionHolder.vue";
import ExecutingHintModal from "../EditorCommon/ExecutingHintModal.vue";
import SaveSheetModal from "../EditorCommon/SaveSheetModal.vue";
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

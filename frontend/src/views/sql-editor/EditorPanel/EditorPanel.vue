<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <EditorAction @save-sheet="trySaveSheet" />

    <ConnectionPathBar />

    <template v-if="!tabStore.isDisconnected">
      <SQLEditor @save-sheet="trySaveSheet" />
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

import { useTabStore } from "@/store";
import EditorAction from "../EditorCommon/EditorAction.vue";
import SQLEditor from "./SQLEditor.vue";
import ConnectionPathBar from "../EditorCommon/ConnectionPathBar.vue";
import ConnectionHolder from "../EditorCommon/ConnectionHolder.vue";
import ExecutingHintModal from "../EditorCommon/ExecutingHintModal.vue";
import SaveSheetModal from "../EditorCommon/SaveSheetModal.vue";

const tabStore = useTabStore();
const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();

const trySaveSheet = (sheetName?: string) => {
  saveSheetModal.value?.trySaveSheet(sheetName);
};
</script>

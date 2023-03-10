<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <template v-if="tab.editMode === 'SQL-EDITOR'">
      <EditorAction @execute="handleExecute" @save-sheet="trySaveSheet" />

      <ConnectionPathBar />

      <template v-if="!tabStore.isDisconnected">
        <SQLEditor @execute="handleExecute" @save-sheet="trySaveSheet" />
      </template>
      <template v-else>
        <ConnectionHolder />
      </template>
    </template>

    <AIChatToSQL
      v-if="!tabStore.isDisconnected"
      @apply-statement="handleApplyStatement"
    />

    <ExecutingHintModal />

    <SaveSheetModal ref="saveSheetModal" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import type { Connection, ExecuteConfig, ExecuteOption } from "@/types";
import { useCurrentTab, useInstanceStore, useTabStore } from "@/store";
import SQLEditor from "./SQLEditor.vue";
import {
  EditorAction,
  ConnectionPathBar,
  ConnectionHolder,
  ExecutingHintModal,
  SaveSheetModal,
} from "../EditorCommon";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";

const tabStore = useTabStore();
const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();
const tab = useCurrentTab();

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

const handleApplyStatement = async (
  statement: string,
  conn: Connection,
  run: boolean
) => {
  tab.value.statement = statement;
  if (run) {
    const instanceStore = useInstanceStore();
    const instance = await instanceStore.getOrFetchInstanceById(
      conn.instanceId
    );
    handleExecute(statement, {
      databaseType: instanceStore.formatEngine(instance),
    });
  }
};
</script>

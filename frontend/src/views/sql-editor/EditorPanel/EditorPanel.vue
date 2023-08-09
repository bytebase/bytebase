<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <template v-if="tab.editMode === 'SQL-EDITOR'">
      <EditorAction @execute="handleExecute" @save-sheet="trySaveSheet" />

      <ConnectionPathBar />

      <SheetForIssueTipsBar />

      <template
        v-if="
          !tabStore.isDisconnected || isSheetOversize || sheetBacktracePayload
        "
      >
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
import { computed, ref } from "vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useCurrentTab,
  useInstanceV1Store,
  useSheetV1Store,
  useTabStore,
} from "@/store";
import type { Connection, ExecuteConfig, ExecuteOption } from "@/types";
import { formatEngineV1, getSheetIssueBacktracePayloadV1 } from "@/utils";
import {
  EditorAction,
  ConnectionPathBar,
  ConnectionHolder,
  ExecutingHintModal,
  SaveSheetModal,
} from "../EditorCommon";
import SQLEditor from "./SQLEditor.vue";
import SheetForIssueTipsBar from "./SheetForIssueTipsBar.vue";

const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();
const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();
const tab = useCurrentTab();

const sheet = computed(() => {
  const sheetName = tabStore.currentTab.sheetName;
  if (!sheetName) return undefined;
  const sheet = sheetV1Store.getSheetByName(sheetName);
  return sheet;
});

const sheetBacktracePayload = computed(() => {
  if (!sheet.value) return undefined;
  return getSheetIssueBacktracePayloadV1(sheet.value);
});

const { executeReadonly } = useExecuteSQL();

const handleExecute = (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  executeReadonly(query, config, option);
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
    const instanceStore = useInstanceV1Store();
    const instance = await instanceStore.getOrFetchInstanceByUID(
      conn.instanceId
    );
    handleExecute(statement, {
      databaseType: formatEngineV1(instance),
    });
  }
};

const isSheetOversize = computed(() => {
  if (!sheet.value) {
    return false;
  }

  return (
    new TextDecoder().decode(sheet.value.content).length <
    sheet.value.contentSize
  );
});
</script>

<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <template v-if="tab.editMode === 'SQL-EDITOR'">
      <EditorAction @execute="handleExecute" />

      <ConnectionPathBar />

      <SheetForIssueTipsBar />

      <SQLEditor @execute="handleExecute" />
    </template>

    <AIChatToSQL
      v-if="!tabStore.isDisconnected && showAIChatBox"
      :allow-config="pageMode === 'BUNDLED'"
      @apply-statement="handleApplyStatement"
    />

    <ExecutingHintModal />

    <SaveSheetModal />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useCurrentTab,
  useInstanceV1Store,
  useTabStore,
  useActuatorV1Store,
} from "@/store";
import type { Connection, ExecuteConfig, ExecuteOption } from "@/types";
import { formatEngineV1 } from "@/utils";
import {
  EditorAction,
  ConnectionPathBar,
  ExecutingHintModal,
  SaveSheetModal,
} from "../EditorCommon";
import { useSQLEditorContext } from "../context";
import SQLEditor from "./SQLEditor.vue";
import SheetForIssueTipsBar from "./SheetForIssueTipsBar.vue";

const tabStore = useTabStore();
const tab = useCurrentTab();
const { showAIChatBox } = useSQLEditorContext();
const actuatorStore = useActuatorV1Store();
const { pageMode } = storeToRefs(actuatorStore);

const { executeReadonly } = useExecuteSQL();

const handleExecute = (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  executeReadonly(query, config, option);
};

// const trySaveSheet = (sheetName?: string) => {
//   saveSheetModal.value?.trySaveSheet(sheetName);
// };

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
</script>

<template>
  <Splitpanes
    v-if="!tab || tab.mode === 'READONLY' || tab.mode === 'STANDARD'"
    horizontal
    class="default-theme"
    :dbl-click-splitter="false"
  >
    <Pane class="flex flex-col overflow-hidden">
      <div
        v-if="isDisconnected || allowReadonlyMode"
        class="flex-1 h-full w-full flex flex-col justify-start items-stretch"
      >
        <template v-if="!tab || tab.editMode === 'SQL-EDITOR'">
          <EditorAction @execute="handleExecute" />
          <div v-if="tab" class="w-full flex-1 flex flex-row items-stretch overflow-hidden">
            <Suspense>
              <SQLEditor @execute="handleExecute" />
              <template #fallback>
                <div
                  class="w-full h-auto flex-grow flex flex-col items-center justify-center"
                >
                  <BBSpin />
                </div>
              </template>
            </Suspense>
          </div>
          <template v-else>
            <Welcome />
          </template>
        </template>

        <Suspense>
          <AIChatToSQL
            v-if="tab && !isDisconnected && showAIChatBox"
            @apply-statement="handleApplyStatement"
          />
        </Suspense>

        <ExecutingHintModal />

        <SaveSheetModal />
      </div>

      <ReadonlyModeNotSupported v-else />
    </Pane>
    <Pane
      v-if="!isDisconnected && allowReadonlyMode"
      class="relative"
      :size="40"
    >
      <ResultPanel />
    </Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { Pane, Splitpanes } from "splitpanes";
import { computed, defineAsyncComponent, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorConnection, SQLEditorQueryParams } from "@/types";
import { instanceV1HasReadonlyMode } from "@/utils";
import {
  EditorAction,
  ExecutingHintModal,
  SaveSheetModal,
} from "../../EditorCommon";
import { useSQLEditorContext } from "../../context";
import ReadonlyModeNotSupported from "../ReadonlyModeNotSupported.vue";
import ResultPanel from "../ResultPanel";
import Welcome from "../Welcome.vue";

const SQLEditor = defineAsyncComponent(() => import("./SQLEditor.vue"));

const tabStore = useSQLEditorTabStore();
const { currentTab: tab, isDisconnected } = storeToRefs(tabStore);
const { showAIChatBox, standardModeEnabled } = useSQLEditorContext();
const { instance } = useConnectionOfCurrentSQLEditorTab();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});

const { execute } = useExecuteSQL();

const handleExecute = (params: SQLEditorQueryParams) => {
  execute(params);
};

const handleApplyStatement = async (
  statement: string,
  connection: SQLEditorConnection,
  run: boolean
) => {
  if (!tab.value) {
    return;
  }
  tab.value.statement = statement;
  if (run) {
    const database = useDatabaseV1Store().getDatabaseByName(
      connection.database
    );
    handleExecute({
      connection,
      statement,
      engine: database.instanceResource.engine,
      explain: false,
    });
  }
};

watch(
  [() => tab.value?.id, standardModeEnabled],
  () => {
    if (!tab.value) return;
    // Fallback to READONLY mode if standard.value mode is not allowed.
    if (!standardModeEnabled.value && tab.value.mode === "STANDARD") {
      tab.value.mode = "READONLY";
    }
  },
  {
    immediate: true,
    deep: false,
  }
);
</script>

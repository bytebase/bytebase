<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <ConnectionPathBar class="border-b" />

    <template v-if="!tab || tab.editMode === 'SQL-EDITOR'">
      <EditorAction @execute="handleExecute" />
      <template v-if="tab">
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
      </template>
      <Welcome v-else />
    </template>

    <Suspense>
      <AIChatToSQL
        v-if="tab && !isDisconnected && showAIChatBox"
        :allow-config="pageMode === 'BUNDLED'"
        @apply-statement="handleApplyStatement"
      />
    </Suspense>

    <ExecutingHintModal />

    <SaveSheetModal />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { defineAsyncComponent, watch } from "vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import { useInstanceV1Store, usePageMode, useSQLEditorTabStore } from "@/store";
import type { SQLEditorConnection, SQLEditorQueryParams } from "@/types";
import {
  EditorAction,
  ConnectionPathBar,
  ExecutingHintModal,
  SaveSheetModal,
} from "../EditorCommon";
import { useSQLEditorContext } from "../context";
import Welcome from "./Welcome.vue";

const SQLEditor = defineAsyncComponent(() => import("./SQLEditor.vue"));

const tabStore = useSQLEditorTabStore();
const { currentTab: tab, isDisconnected } = storeToRefs(tabStore);
const { showAIChatBox, standardModeEnabled } = useSQLEditorContext();
const pageMode = usePageMode();

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
    const instance = useInstanceV1Store().getInstanceByName(
      connection.instance
    );
    handleExecute({
      connection,
      statement,
      engine: instance.engine,
      explain: false,
    });
  }
};

watch(
  [() => tab.value, standardModeEnabled],
  ([tab, standardModeEnabled]) => {
    if (!tab) return;
    // Fallback to READONLY mode if standard mode is not allowed.
    if (!standardModeEnabled && tab.mode === "STANDARD") {
      tab.mode = "READONLY";
    }
  },
  {
    immediate: true,
    deep: false,
  }
);
</script>

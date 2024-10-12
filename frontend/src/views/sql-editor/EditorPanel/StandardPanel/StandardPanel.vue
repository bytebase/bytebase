<template>
  <Splitpanes
    v-if="!tab || tab.mode === 'WORKSHEET'"
    horizontal
    class="default-theme overflow-hidden"
    :dbl-click-splitter="false"
  >
    <Pane class="flex flex-col overflow-hidden justify-start items-stretch">
      <template v-if="isDisconnected || allowReadonlyMode">
        <EditorAction @execute="handleExecute" />

        <Splitpanes
          v-if="tab"
          class="default-theme overflow-hidden"
          @resized="handleAIPanelResize($event, 1)"
        >
          <Pane>
            <Suspense>
              <SQLEditor @execute="handleExecute" />
              <template #fallback>
                <div
                  class="w-full h-full flex-grow flex flex-col items-center justify-center"
                >
                  <BBSpin />
                </div>
              </template>
            </Suspense>
          </Pane>
          <Pane
            v-if="showAIPanel"
            :size="AIPanelSize"
            class="overflow-hidden flex flex-col"
          >
            <Suspense>
              <AIChatToSQL key="ai-chat-to-sql" />
              <template #fallback>
                <div
                  class="w-full h-full flex-grow flex flex-col items-center justify-center"
                >
                  <BBSpin />
                </div>
              </template>
            </Suspense>
          </Pane>
        </Splitpanes>
        <template v-else>
          <Welcome />
        </template>

        <ExecutingHintModal />

        <SaveSheetModal />
      </template>

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
import { computed, defineAsyncComponent } from "vue";
import { BBSpin } from "@/bbkit";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import { instanceV1HasReadonlyMode } from "@/utils";
import {
  EditorAction,
  ExecutingHintModal,
  SaveSheetModal,
} from "../../EditorCommon";
import { useSQLEditorContext } from "../../context";
import ReadonlyModeNotSupported from "../ReadonlyModeNotSupported.vue";
import ResultPanel from "../ResultPanel";
import Welcome from "../Welcome";

const SQLEditor = defineAsyncComponent(() => import("./SQLEditor.vue"));

const tabStore = useSQLEditorTabStore();
const { currentTab: tab, isDisconnected } = storeToRefs(tabStore);
const { instance } = useConnectionOfCurrentSQLEditorTab();
const { showAIPanel, AIPanelSize, handleAIPanelResize } = useSQLEditorContext();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});

const { execute } = useExecuteSQL();

const handleExecute = (params: SQLEditorQueryParams) => {
  execute(params);
};
</script>

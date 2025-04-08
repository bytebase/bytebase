<template>
  <Splitpanes
    v-if="!tab || tab.mode === 'WORKSHEET'"
    horizontal
    class="default-theme overflow-hidden"
    :dbl-click-splitter="false"
  >
    <Pane class="flex flex-col overflow-hidden justify-start items-stretch">
      <template v-if="isDisconnected || allowReadonlyMode">
        <EditorAction @execute="handleExecuteFromActionBar" />

        <Splitpanes
          v-if="tab"
          class="default-theme overflow-hidden"
          @resized="handleAIPanelResize($event, 1)"
        >
          <Pane>
            <Suspense>
              <SQLEditor
                ref="sqlEditorRef"
                @execute="handleExecute"
                @execute-in-new-tab="handleExecuteInNewTab"
              />
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
import { cloneDeep } from "lodash-es";
import { storeToRefs } from "pinia";
import { Pane, Splitpanes } from "splitpanes";
import { computed, defineAsyncComponent, nextTick, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
  useTabViewStateStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import {
  defaultSQLEditorTab,
  instanceV1HasReadonlyMode,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
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
const { cloneViewState } = useTabViewStateStore();
const sqlEditorRef = ref<InstanceType<typeof SQLEditor>>();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});

const { execute } = useExecuteSQL();

const handleExecuteFromActionBar = (params: SQLEditorQueryParams) => {
  if (!tab.value || !sqlEditorRef.value) {
    return;
  }
  const statement = sqlEditorRef.value.getActiveStatement();
  handleExecute({ ...params, statement });
};

const handleExecute = (params: SQLEditorQueryParams) => {
  execute(params);
};
const handleExecuteInNewTab = (params: SQLEditorQueryParams) => {
  const fromTab = tabStore.currentTab;
  const clonedTab = defaultSQLEditorTab();
  if (fromTab) {
    clonedTab.connection = cloneDeep(fromTab.connection);
    clonedTab.treeState = cloneDeep(fromTab.treeState);
  }
  clonedTab.title = suggestedTabTitleForSQLEditorConnection(
    clonedTab.connection
  );
  const newTab = tabStore.addTab(clonedTab, /* beside */ true);
  if (fromTab) {
    const vs = cloneViewState(fromTab.id, newTab.id);
    if (vs) {
      vs.view = "CODE";
      vs.detail = {};
    }
  }
  newTab.statement = params.statement;

  nextTick(() => {
    execute(params);
  });
};
</script>

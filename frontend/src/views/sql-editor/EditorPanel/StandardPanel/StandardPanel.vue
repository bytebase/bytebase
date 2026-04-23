<template>
  <NSplit
    v-if="!tab || tab.mode === 'WORKSHEET'"
    direction="vertical"
    :max="0.8"
    :resize-trigger-size="3"
  >
    <template #1>
      <div class="h-full flex flex-col overflow-hidden justify-start items-stretch">
        <template v-if="isDisconnected || allowReadonlyMode">
          <EditorAction @execute="handleExecuteFromActionBar" />

          <NSplit
            v-if="tab"
            :disabled="!showAIPanel"
            :size="editorPanelSize.size"
            :min="editorPanelSize.min"
            :max="editorPanelSize.max"
            :resize-trigger-size="3"
            @update:size="handleEditorPanelResize"
          >
            <template #1>
              <Suspense>
                <SQLEditor
                  ref="sqlEditorRef"
                  @execute="handleExecute"
                />
                <template #fallback>
                  <div
                    class="w-full h-full grow flex flex-col items-center justify-center"
                  >
                    <BBSpin />
                  </div>
                </template>
              </Suspense>
            </template>
            <template #2>
              <div
                v-if="showAIPanel"
                class="h-full overflow-hidden flex flex-col"
              >
                <Suspense>
                  <AIChatToSQL key="ai-chat-to-sql" />
                  <template #fallback>
                    <div
                      class="w-full h-full grow flex flex-col items-center justify-center"
                    >
                      <BBSpin />
                    </div>
                  </template>
                </Suspense>
              </div>
            </template>
          </NSplit>
          <div v-else class="flex-1 flex flex-col min-h-0">
            <ReactPageMount
              page="Welcome"
              :onChangeConnection="changeConnection"
            />
          </div>

          <ExecutingHintModal />

          <ReactPageMount page="SaveSheetModal" />
        </template>

        <ReadonlyModeNotSupported v-else />
      </div>
    </template>

    <template #2>
      <div
        v-if="!isDisconnected && allowReadonlyMode"
        class="relative h-full"
      >
        <ResultPanel />
      </div>
    </template>
  </NSplit>
</template>

<script lang="ts" setup>
import { NSplit } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, defineAsyncComponent, nextTick, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import ReactPageMount from "@/react/ReactPageMount.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import { getInstanceResource, instanceV1HasReadonlyMode } from "@/utils";
import { useSQLEditorContext } from "../../context";
import { EditorAction, ExecutingHintModal } from "../../EditorCommon";
import { sqlEditorEvents } from "../../events";
import ReadonlyModeNotSupported from "../ReadonlyModeNotSupported.vue";
import ResultPanel from "../ResultPanel";

const SQLEditor = defineAsyncComponent(() => import("./SQLEditor.vue"));

const tabStore = useSQLEditorTabStore();
const { currentTab: tab, isDisconnected } = storeToRefs(tabStore);
const { instance } = useConnectionOfCurrentSQLEditorTab();
const {
  showAIPanel,
  editorPanelSize,
  handleEditorPanelResize,
  showConnectionPanel,
  asidePanelTab,
} = useSQLEditorContext();

const changeConnection = () => {
  asidePanelTab.value = "SCHEMA";
  showConnectionPanel.value = true;
};
const sqlEditorRef = ref<InstanceType<typeof SQLEditor>>();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});

const { execute } = useExecuteSQL();

useEmitteryEventListener(
  sqlEditorEvents,
  "execute-sql",
  async ({ connection, statement, batchQueryContext }) => {
    const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
      connection.database
    );
    const newTab = tabStore.addTab(
      { connection, statement, batchQueryContext },
      /* beside */ true
    );
    nextTick(() => {
      execute({
        connection: { ...newTab.connection },
        statement,
        engine: getInstanceResource(database).engine,
        explain: false,
        selection: null,
      });
    });
  }
);

const handleExecuteFromActionBar = (params: SQLEditorQueryParams) => {
  if (!tab.value || !sqlEditorRef.value) {
    return;
  }
  const statement = sqlEditorRef.value.getActiveStatement();
  handleExecute({
    params: { ...params, statement },
    newTab: false,
  });
};

const handleExecute = ({
  params,
  newTab,
}: {
  params: SQLEditorQueryParams;
  newTab: boolean;
}) => {
  if (newTab) {
    tabStore.cloneTab(tabStore.currentTabId, {
      statement: params.statement,
    });
  }

  nextTick(() => {
    execute(params);
  });
};
</script>

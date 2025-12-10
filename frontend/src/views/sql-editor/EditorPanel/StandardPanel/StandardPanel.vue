<template>
  <NSplit
    v-if="!tab || tab.mode === 'WORKSHEET'"
    direction="vertical"
    :max="0.8"
    :resize-trigger-size="1"
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
            :resize-trigger-size="1"
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
              <div class="h-full overflow-hidden flex flex-col">
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
          <Welcome v-else />

          <ExecutingHintModal />

          <SaveSheetModal />
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
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorQueryParams } from "@/types";
import { instanceV1HasReadonlyMode } from "@/utils";
import { useSQLEditorContext } from "../../context";
import {
  EditorAction,
  ExecutingHintModal,
  SaveSheetModal,
} from "../../EditorCommon";
import ReadonlyModeNotSupported from "../ReadonlyModeNotSupported.vue";
import ResultPanel from "../ResultPanel";
import Welcome from "../Welcome";

const SQLEditor = defineAsyncComponent(() => import("./SQLEditor.vue"));

const tabStore = useSQLEditorTabStore();
const { currentTab: tab, isDisconnected } = storeToRefs(tabStore);
const { instance } = useConnectionOfCurrentSQLEditorTab();
const { showAIPanel, editorPanelSize, handleEditorPanelResize } =
  useSQLEditorContext();
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

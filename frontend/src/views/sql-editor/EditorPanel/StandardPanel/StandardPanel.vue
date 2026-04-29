<template>
  <template v-if="!tab || tab.mode === 'WORKSHEET'">
    <NSplit
      v-if="showResultPanel"
      direction="vertical"
      :max="0.8"
      :resize-trigger-size="3"
    >
      <template #1>
        <NSplit
          :disabled="!showAIPanel || !tab"
          :size="editorPanelSize.size"
          :min="editorPanelSize.min"
          :max="editorPanelSize.max"
          :resize-trigger-size="3"
          @update:size="handleEditorPanelResize"
        >
          <template #1>
            <ReactPageMount
              page="EditorMain"
              container-class="h-full"
              :page-props="{ onChangeConnection: handleChangeConnection }"
            />
          </template>
          <template v-if="showAIPanel && tab" #2>
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
      </template>
      <template #2>
        <div class="relative h-full">
          <ResultPanel />
        </div>
      </template>
    </NSplit>
    <NSplit
      v-else
      class="h-full"
      :disabled="!showAIPanel || !tab"
      :size="editorPanelSize.size"
      :min="editorPanelSize.min"
      :max="editorPanelSize.max"
      :resize-trigger-size="3"
      @update:size="handleEditorPanelResize"
    >
      <template #1>
        <ReactPageMount
          page="EditorMain"
          container-class="h-full"
          :page-props="{ onChangeConnection: handleChangeConnection }"
        />
      </template>
      <template v-if="showAIPanel && tab" #2>
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
  </template>
</template>

<script lang="ts" setup>
import { NSplit } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { BBSpin } from "@/bbkit";
import { AIChatToSQL } from "@/plugins/ai";
import ReactPageMount from "@/react/ReactPageMount.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
} from "@/store";
import { instanceV1HasReadonlyMode } from "@/utils";
import { useSQLEditorContext } from "../../context";
import ResultPanel from "../ResultPanel";

const { currentTab: tab, isDisconnected } = storeToRefs(useSQLEditorTabStore());
const { instance } = useConnectionOfCurrentSQLEditorTab();

// ResultPanel only renders when connected to a read-only-capable instance;
// when it won't render we skip the outer NSplit so the editor body (incl.
// the Welcome screen) can occupy the full height instead of being squeezed
// into an arbitrary top pane.
const showResultPanel = computed(
  () => !isDisconnected.value && instanceV1HasReadonlyMode(instance.value)
);

// AI side pane is hosted here (Vue) instead of inside the React
// `EditorMain` because `AIChatToSQL` is Vue-only.
const uiStore = useSQLEditorUIStore();
const { showAIPanel, editorPanelSize } = storeToRefs(uiStore);
const handleEditorPanelResize = uiStore.handleEditorPanelResize;

const { showConnectionPanel, asidePanelTab } = useSQLEditorContext();
const handleChangeConnection = () => {
  asidePanelTab.value = "SCHEMA";
  showConnectionPanel.value = true;
};
</script>

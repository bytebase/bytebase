<template>
  <template v-if="!tab || tab.mode === 'WORKSHEET'">
    <NSplit
      v-if="showResultPanel"
      direction="vertical"
      :max="0.8"
      :resize-trigger-size="3"
    >
      <template #1>
        <EditorMain />
      </template>
      <template #2>
        <div class="relative h-full">
          <ResultPanel />
        </div>
      </template>
    </NSplit>
    <EditorMain v-else class="h-full" />
  </template>
</template>

<script lang="ts" setup>
import { NSplit } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import { instanceV1HasReadonlyMode } from "@/utils";
import ResultPanel from "../ResultPanel";
import EditorMain from "./EditorMain.vue";

const { currentTab: tab, isDisconnected } = storeToRefs(useSQLEditorTabStore());
const { instance } = useConnectionOfCurrentSQLEditorTab();

// ResultPanel only renders when connected to a read-only-capable instance;
// when it won't render we skip the outer NSplit so the editor body (incl.
// the Welcome screen) can occupy the full height instead of being squeezed
// into an arbitrary top pane.
const showResultPanel = computed(
  () => !isDisconnected.value && instanceV1HasReadonlyMode(instance.value)
);
</script>

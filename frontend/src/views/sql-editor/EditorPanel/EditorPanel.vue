<template>
  <div class="w-full flex-1 flex flex-row items-stretch overflow-hidden">
    <Splitpanes
      v-if="
        !currentTab ||
        currentTab.mode === 'READONLY' ||
        currentTab.mode === 'STANDARD'
      "
      horizontal
      class="default-theme"
      :dbl-click-splitter="false"
    >
      <Pane class="flex flex-row overflow-hidden">
        <StandardPanel v-if="isDisconnected || allowReadonlyMode" />
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

    <TerminalPanel v-else-if="currentTab.mode === 'ADMIN'" />

    <AccessDenied v-else />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { Pane, Splitpanes } from "splitpanes";
import { computed } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import { instanceV1HasReadonlyMode } from "@/utils";
import AccessDenied from "./AccessDenied.vue";
import ReadonlyModeNotSupported from "./ReadonlyModeNotSupported.vue";
import ResultPanel from "./ResultPanel";
import StandardPanel from "./StandardPanel";
import TerminalPanel from "./TerminalPanel";

const tabStore = useSQLEditorTabStore();
const { currentTab, isDisconnected } = storeToRefs(tabStore);
const { instance } = useConnectionOfCurrentSQLEditorTab();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});
</script>

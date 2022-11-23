<template>
  <div class="sqleditor--wrapper">
    <TabList />
    <Splitpanes class="default-theme flex flex-col flex-1 overflow-hidden">
      <Pane size="20">
        <AsidePanel />
      </Pane>
      <Pane size="80" class="relative">
        <Splitpanes
          v-if="tabStore.currentTab.mode === TabMode.ReadOnly"
          horizontal
          class="default-theme"
        >
          <Pane :size="isDisconnected ? 100 : 60">
            <EditorPanel />
          </Pane>
          <Pane :size="isDisconnected ? 0 : 40">
            <TablePanel />
          </Pane>
        </Splitpanes>

        <TerminalPanel v-if="tabStore.currentTab.mode === TabMode.Admin" />

        <div
          v-if="isFetchingSheet"
          class="flex items-center justify-center absolute inset-0 bg-white/50 z-20"
        >
          <BBSpin />
        </div>
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Splitpanes, Pane } from "splitpanes";

import { TabMode } from "@/types";
import { useSQLEditorStore, useTabStore } from "@/store";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import TerminalPanel from "./TerminalPanel/TerminalPanel.vue";
import TabList from "./TabList";
import TablePanel from "./TablePanel/TablePanel.vue";

const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const isDisconnected = computed(() => tabStore.isDisconnected);
const isFetchingSheet = computed(() => sqlEditorStore.isFetchingSheet);
</script>

<style>
@import "splitpanes/dist/splitpanes.css";

/* splitpanes pane style */
.splitpanes.default-theme .splitpanes__pane {
  @apply bg-transparent;
}

.splitpanes.default-theme .splitpanes__splitter {
  @apply bg-gray-100;
  min-height: 8px;
  min-width: 8px;
}

.splitpanes.default-theme .splitpanes__splitter:hover {
  @apply bg-indigo-400;
}

.splitpanes.default-theme .splitpanes__splitter::before,
.splitpanes.default-theme .splitpanes__splitter::after {
  @apply bg-gray-700 opacity-50 text-white;
}

.splitpanes.default-theme .splitpanes__splitter:hover::before,
.splitpanes.default-theme .splitpanes__splitter:hover::after {
  @apply bg-white opacity-100;
}
</style>

<style scoped>
.sqleditor--wrapper {
  color: var(--base);
  --base: #444;
  --font-code: "Source Code Pro", monospace;
  --color-branding: #4f46e5;
  --border-color: rgba(200, 200, 200, 0.2);

  @apply flex-1 overflow-hidden flex flex-col;
}
</style>

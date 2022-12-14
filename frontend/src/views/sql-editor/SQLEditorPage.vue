<template>
  <div class="sqleditor--wrapper">
    <TabList />
    <Splitpanes class="default-theme flex flex-col flex-1 overflow-hidden">
      <Pane size="20">
        <AsidePanel />
      </Pane>
      <Pane size="80" class="relative">
        <template v-if="allowQuery">
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
        </template>
        <div
          v-else
          class="w-full h-full flex flex-col items-center justify-center"
        >
          <img src="../../assets/illustration/403.webp" class="max-h-[40%]" />
          <div class="textinfolabel">
            {{ $t("database.access-denied") }}
          </div>
        </div>

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

import { TabMode, UNKNOWN_ID } from "@/types";
import {
  useCurrentUser,
  useDatabaseStore,
  useSQLEditorStore,
  useTabStore,
} from "@/store";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import TerminalPanel from "./TerminalPanel/TerminalPanel.vue";
import TabList from "./TabList";
import TablePanel from "./TablePanel/TablePanel.vue";
import { isDatabaseAccessible } from "@/utils";

const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const sqlEditorStore = useSQLEditorStore();
const currentUser = useCurrentUser();

const isDisconnected = computed(() => tabStore.isDisconnected);
const isFetchingSheet = computed(() => sqlEditorStore.isFetchingSheet);

const allowQuery = computed(() => {
  const { databaseId } = tabStore.currentTab.connection;
  const database = databaseStore.getDatabaseById(databaseId);
  if (database.id === UNKNOWN_ID) {
    // Allowed if connected to an instance
    return true;
  }
  const { accessControlPolicyList } = sqlEditorStore;
  return isDatabaseAccessible(
    database,
    accessControlPolicyList,
    currentUser.value
  );
});
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

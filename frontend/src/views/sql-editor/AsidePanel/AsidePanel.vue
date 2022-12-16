<template>
  <div class="aside-panel h-full">
    <n-tabs type="segment" default-value="databases" class="h-full">
      <n-tab-pane name="databases" :tab="$t('common.databases')">
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
        >
          <Pane :size="databasePaneSize"><DatabaseTree /></Pane>
          <Pane :size="FULL_HEIGHT - databasePaneSize">
            <TableSchema @close-pane="handleCloseTableSchemaPane" />
          </Pane>
        </Splitpanes>
      </n-tab-pane>
      <n-tab-pane name="history" :tab="$t('common.history')">
        <QueryHistoryContainer />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useConnectionTreeStore } from "@/store";
import DatabaseTree from "./DatabaseTree.vue";
import QueryHistoryContainer from "./QueryHistoryContainer.vue";
import TableSchema from "./TableSchema.vue";
import { Splitpanes, Pane } from "splitpanes";
import { unknown, UNKNOWN_ID } from "@/types";

const FULL_HEIGHT = 100;
const DATABASE_PANE_SIZE = 60;

const connectionTreeStore = useConnectionTreeStore();
const databasePaneSize = computed(() => {
  if (connectionTreeStore.selectedTable.id !== UNKNOWN_ID) {
    return DATABASE_PANE_SIZE;
  }
  return FULL_HEIGHT;
});

const handleCloseTableSchemaPane = () => {
  connectionTreeStore.selectedTable = unknown("TABLE");
};
</script>

<style scoped>
.aside-panel .n-tab-pane {
  height: calc(100% - 40px);
  @apply pt-0;
}
</style>

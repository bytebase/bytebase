<template>
  <div class="aside-panel h-full">
    <n-tabs type="segment" default-value="databases" class="h-full">
      <n-tab-pane name="databases" :tab="$t('common.databases')">
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
          @resized="handleResized"
        >
          <Pane :size="databasePaneSzie"><DatabaseTree /></Pane>
          <Pane
            :size="FULL_HEIGHT - databasePaneSzie"
            :max-size="TABLE_SCHEMA_PANE_SIZE"
          >
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
import { ref, watch } from "vue";
import { useSQLEditorStore } from "@/store";
import DatabaseTree from "./DatabaseTree.vue";
import QueryHistoryContainer from "./QueryHistoryContainer.vue";
import TableSchema from "./TableSchema.vue";
import { Splitpanes, Pane } from "splitpanes";

const FULL_HEIGHT = 100;
const DATABASE_PANE_SIZE = 60;
const TABLE_SCHEMA_PANE_SIZE = FULL_HEIGHT - DATABASE_PANE_SIZE;

const sqlEditorStore = useSQLEditorStore();
const databasePaneSzie = ref(FULL_HEIGHT);

const handleResized = (data: any) => {
  const [top] = data;
  databasePaneSzie.value = top.size;
};

const handleCloseTableSchemaPane = () => {
  databasePaneSzie.value = FULL_HEIGHT;
};

watch(
  () => sqlEditorStore.connectionContext.option,
  (option) => {
    if (option && option.type === "table") {
      databasePaneSzie.value = DATABASE_PANE_SIZE;
    }
  }
);
</script>

<style scoped>
.aside-panel .n-tab-pane {
  height: calc(100% - 40px);
  @apply pt-0;
}
</style>

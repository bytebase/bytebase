<template>
  <div class="aside-panel h-full">
    <n-tabs type="segment" default-value="databases" class="h-full">
      <n-tab-pane name="databases" tab="Databases">
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
          @resized="handleResized"
        >
          <Pane :size="databasesPaneSize"><DatabaseTree /></Pane>
          <Pane :size="100 - databasesPaneSize" max-size="40"
            ><TableSchema @close-table-schema-pane="handleCloseTableSchemaPane"
          /></Pane>
        </Splitpanes>
      </n-tab-pane>
      <n-tab-pane name="queries" tab="Queries"> Queries Page </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import { useNamespacedState } from "vuex-composition-helpers";

import type { SqlEditorState } from "../../../types";
import DatabaseTree from "./DatabaseTree.vue";
import TableSchema from "./TableSchema.vue";

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);

const databasesPaneSize = ref(100);

const handleResized = (data: any) => {
  const [top, bottom] = data;
  databasesPaneSize.value = top.size;
};

const handleCloseTableSchemaPane = () => {
  databasesPaneSize.value = 100;
};

watch(
  () => connectionContext.value.selectedTableName,
  (newVal, oldVal) => {
    databasesPaneSize.value = 60;
  }
);
</script>

<style scoped>
.aside-panel .n-tab-pane {
  height: calc(100% - 40px);
  @apply pt-0;
}
</style>

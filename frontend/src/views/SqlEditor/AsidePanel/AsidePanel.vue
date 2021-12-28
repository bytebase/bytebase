<template>
  <div class="aside-panel h-full">
    <n-tabs type="segment" default-value="databases" class="h-full">
      <n-tab-pane name="databases" tab="Databases">
        <Splitpanes horizontal class="default-theme">
          <Pane size="100"><DatabasesTree /></Pane>
          <Pane size="0" max-size="40"><TableSchema /></Pane>
        </Splitpanes>
      </n-tab-pane>
      <n-tab-pane name="queries" tab="Queries">
        Queries Page
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script lang="ts" setup>
import DatabasesTree from "./DatabasesTree.vue";
import TableSchema from "./TableSchema.vue";
</script>

<style scoped>
.aside-panel .n-tab-pane {
  height: calc(100% - 40px);
  @apply pt-0;
}
</style>

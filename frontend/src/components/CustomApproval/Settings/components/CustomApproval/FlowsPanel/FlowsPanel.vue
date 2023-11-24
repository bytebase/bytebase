<template>
  <div class="flex flex-col items-end space-y-4">
    <FlowsToolbar />
    <FlowTable />
  </div>
</template>

<script lang="ts" setup>
import FlowTable from "./FlowTable.vue";
import FlowsToolbar from "./Toolbar.vue";
</script>

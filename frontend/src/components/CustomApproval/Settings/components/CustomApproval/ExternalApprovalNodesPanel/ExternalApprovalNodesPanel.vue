<template>
  <div class="flex flex-col items-end space-y-4">
    <ExternalApprovalNodesToolbar />
    <NodeTable />
  </div>
</template>

<script setup lang="ts">
import NodeTable from "./NodeTable.vue";
import ExternalApprovalNodesToolbar from "./Toolbar.vue";
</script>

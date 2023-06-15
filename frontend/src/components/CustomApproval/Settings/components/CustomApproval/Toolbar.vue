<template>
  <FlowsToolbar v-if="tab === 'flows'" />
  <ExternalApprovalNodesToolbar v-if="tab === 'external-approval'" />
</template>

<script lang="ts" setup>
import FlowsToolbar from "./FlowsPanel/Toolbar.vue";
import ExternalApprovalNodesToolbar from "./ExternalApprovalNodesPanel/Toolbar.vue";
import { useCustomApprovalContext } from "./context";

const { tab } = useCustomApprovalContext();
</script>

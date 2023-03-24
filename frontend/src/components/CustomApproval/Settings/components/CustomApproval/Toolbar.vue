<template>
  <FlowsToolbar v-if="tab === 'flows'" />
</template>

<script lang="ts" setup>
import FlowsToolbar from "./FlowsPanel/Toolbar.vue";
import { useCustomApprovalContext } from "./context";

const { tab } = useCustomApprovalContext();
</script>

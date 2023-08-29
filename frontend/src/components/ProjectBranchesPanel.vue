<template>
  <BranchTablePanel :project-id="projectId" />
</template>

<script lang="ts" setup>
import BranchTablePanel from "@/components/BranchTablePanel.vue";

defineProps<{
  projectId?: string;
}>();
</script>

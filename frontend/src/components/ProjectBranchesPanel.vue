<template>
  <BranchTablePanel :project-id="projectId" :hide-project-column="true" />
</template>

<script lang="ts" setup>
import BranchTablePanel from "@/components/BranchTablePanel.vue";

defineProps<{
  projectId?: string;
}>();
</script>

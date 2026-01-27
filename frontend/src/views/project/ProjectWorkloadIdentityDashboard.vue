<template>
  <WorkloadIdentityPanel :project-id="projectId" />
</template>

<script setup lang="ts">
import WorkloadIdentityPanel from "@/components/User/Settings/WorkloadIdentityPanel.vue";

defineProps<{
  projectId: string;
}>();
</script>

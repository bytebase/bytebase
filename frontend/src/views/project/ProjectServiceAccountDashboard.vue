<template>
  <ServiceAccountPanel :project-id="projectId" />
</template>

<script setup lang="ts">
import ServiceAccountPanel from "@/components/User/Settings/ServiceAccountPanel.vue";

defineProps<{
  projectId: string;
}>();
</script>

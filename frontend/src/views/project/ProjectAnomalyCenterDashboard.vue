<template>
  <AnomalyCenterDashboard :project="project" :selected-tab="'database'" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import AnomalyCenterDashboard from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

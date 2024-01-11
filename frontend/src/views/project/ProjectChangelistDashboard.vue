<template>
  <ChangelistDashboard :project="project" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ChangelistDashboard from "@/components/Changelist/ChangelistDashboard";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
</script>

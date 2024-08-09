<template>
  <ProjectBranchesPanel :project="project" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectBranchesPanel from "@/components/ProjectBranchesPanel.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

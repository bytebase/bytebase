<template>
  <ProjectVersionControlPanel :project="project" :allow-edit="allowEdit" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectVersionControlPanel from "@/components/ProjectVersionControlPanel.vue";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
</script>

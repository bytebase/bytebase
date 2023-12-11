<template>
  <ProjectIssuesPanel :project="project" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps({
  projectId: {
    required: true,
    type: String,
  },
});

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
</script>

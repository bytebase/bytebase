<template>
  <ProjectIssuesPanel v-if="ready" :project="project" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectIssuesPanel from "@/components/Project/ProjectIssuesPanel.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project, ready } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

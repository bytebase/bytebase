<template>
  <ProjectDatabasesPanel class="py-4" :project="project" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

<template>
  <ChangelistDashboard :project="project" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ChangelistDashboard from "@/components/Changelist/ChangelistDashboard";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

<template>
  <SyncDatabaseSchema class="py-4" :key="project.name" :project="project" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import SyncDatabaseSchema from "@/components/SyncDatabaseSchemaV1/index.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>

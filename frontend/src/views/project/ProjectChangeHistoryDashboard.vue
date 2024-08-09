<template>
  <ProjectChangeHistoryPanel
    :database-list="databaseV1List"
    :project="project"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectChangeHistoryPanel from "@/components/ProjectChangeHistoryPanel.vue";
import { useDatabaseV1Store, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});
</script>

<template>
  <ProjectChangeHistoryPanel :database-list="databaseV1List" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectChangeHistoryPanel from "@/components/ProjectChangeHistoryPanel.vue";
import { useProjectV1Store, useDatabaseV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});
</script>

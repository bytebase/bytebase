<template>
  <ProjectDatabasesPanel :project="project" :database-list="databaseV1List" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import {
  useDatabaseV1Store,
  useCurrentUserV1,
  useAppFeature,
  useProjectByName,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List, isDatabaseV1Alterable } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const currentUser = useCurrentUserV1();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const hideInalterableDatabases = useAppFeature(
  "bb.feature.databases.hide-inalterable"
);

const databaseV1List = computed(() => {
  let list = useDatabaseV1Store().databaseListByProject(project.value.name);
  list = sortDatabaseV1List(list);
  // If embedded in iframe, only show alterable databases.
  if (hideInalterableDatabases.value) {
    list = list.filter((db) => isDatabaseV1Alterable(db, currentUser.value));
  }
  return list;
});
</script>

<template>
  <ProjectDatabasesPanel :project="project" :database-list="databaseV1List" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import {
  useDatabaseV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  useAppFeature,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List, isDatabaseV1Alterable } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const currentUser = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const hideInalterableDatabases = useAppFeature(
  "bb.feature.databases.hide-inalterable"
);

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

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

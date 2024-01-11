<template>
  <ProjectDatabasesPanel :project="project" :database-list="databaseV1List" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import {
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  usePageMode,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List, isDatabaseV1Alterable } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const currentUser = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const pageMode = usePageMode();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${project.value.name}"`,
  }))
);

const databaseV1List = computed(() => {
  let list = useDatabaseV1Store().databaseListByProject(project.value.name);
  list = sortDatabaseV1List(list);
  // In standalone mode, only show alterable databases.
  if (pageMode.value === "STANDALONE") {
    list = list.filter((db) => isDatabaseV1Alterable(db, currentUser.value));
  }
  return list;
});
</script>

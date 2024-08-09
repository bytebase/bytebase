<template>
  <ProjectDeploymentConfigPanel
    id="deployment-config"
    :project="project"
    :database-list="databaseV1List"
    :allow-edit="allowEdit"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectDeploymentConfigPanel from "@/components/ProjectDeploymentConfigPanel.vue";
import { useProjectByName } from "@/store";
import { useDatabaseV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { sortDatabaseV1List } from "@/utils";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});
</script>

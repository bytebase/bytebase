<template>
  <ProjectDeploymentConfigPanel
    v-if="isTenantProject"
    id="deployment-config"
    :project="project"
    :database-list="databaseV1List"
    :allow-edit="allowEdit"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import ProjectDeploymentConfigPanel from "@/components/ProjectDeploymentConfigPanel.vue";
import { useProjectV1Store } from "@/store";
import { useDatabaseV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";
import { sortDatabaseV1List } from "@/utils";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const isTenantProject = computed(() => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});
</script>

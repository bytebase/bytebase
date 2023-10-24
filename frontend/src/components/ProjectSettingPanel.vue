<template>
  <div class="max-w-3xl mx-auto space-y-6 mb-6">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
      <ProjectBranchProtectionRulesSettingPanel
        :project="project"
        :allow-edit="allowEdit"
      />
      <div v-if="isTenantProject" class="pt-6">
        <ProjectDeploymentConfigPanel
          id="deployment-config"
          :project="project"
          :database-list="databaseV1List"
          :allow-edit="allowEdit"
        />
      </div>
      <div class="pt-4">
        <ProjectArchiveRestoreButton :project="project" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { sortDatabaseV1List } from "../utils";
import ProjectArchiveRestoreButton from "./Project/ProjectArchiveRestoreButton.vue";
import ProjectGeneralSettingPanel from "./Project/ProjectGeneralSettingPanel.vue";

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<ComposedProject>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});

const isTenantProject = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(props.project.name);
  return sortDatabaseV1List(list);
});
</script>

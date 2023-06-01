<template>
  <template v-if="hash === 'overview'">
    <ProjectOverviewPanel id="overview" :project="projectV1" />
  </template>
  <template v-if="hash === 'databases'">
    <ProjectDeploymentConfigPanel
      v-if="isTenantProject"
      id="deployment-config"
      :project="projectV1"
      :database-list="databaseV1List"
      :allow-edit="allowEdit"
    />
    <ProjectDatabasesPanel v-else :database-list="databaseV1List" />
  </template>
  <template v-if="isDev && hash === 'database-groups'">
    <ProjectDatabaseGroupPanel :project="projectV1" />
  </template>
  <template v-if="hash === 'change-history'">
    <ProjectChangeHistoryPanel
      id="change-history"
      :database-list="databaseV1List"
    />
  </template>
  <template v-if="hash === 'slow-query'">
    <ProjectSlowQueryPanel :project="projectV1" />
  </template>
  <template v-if="hash === 'activity'">
    <ProjectActivityPanel id="activity" :project="projectV1" />
  </template>
  <template
    v-if="Number(project.id) !== DEFAULT_PROJECT_ID && hash === 'gitops'"
  >
    <ProjectVersionControlPanel
      id="gitops"
      :project="projectV1"
      :allow-edit="allowEdit"
    />
  </template>
  <template
    v-if="Number(project.id) !== DEFAULT_PROJECT_ID && hash === 'webhook'"
  >
    <ProjectWebhookPanel
      id="webhook"
      :project="projectV1"
      :allow-edit="allowEdit"
    />
  </template>
  <template
    v-if="Number(project.id) !== DEFAULT_PROJECT_ID && hash === 'setting'"
  >
    <ProjectSettingPanel
      id="setting"
      :project="projectV1"
      :allow-edit="allowEdit"
    />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";

import { DEFAULT_PROJECT_ID } from "@/types";
import { idFromSlug, sortDatabaseV1List } from "../utils";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectChangeHistoryPanel from "../components/ProjectChangeHistoryPanel.vue";
import ProjectSlowQueryPanel from "../components/ProjectSlowQueryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectDatabasesPanel from "../components/ProjectDatabasesPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import ProjectDeploymentConfigPanel from "../components/ProjectDeploymentConfigPanel.vue";
import {
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useLegacyProjectStore,
  useProjectV1Store,
} from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";
import ProjectDatabaseGroupPanel from "@/components/DatabaseGroup/ProjectDatabaseGroupPanel.vue";

const props = defineProps({
  projectWebhookSlug: {
    default: undefined,
    type: String,
  },
  projectSlug: {
    required: true,
    type: String,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const route = useRoute();
const projectStore = useLegacyProjectStore();
const projectV1Store = useProjectV1Store();

const hash = computed(() => route.hash.replace(/^#?/, ""));

const project = computed(() => {
  return projectStore.getProjectById(idFromSlug(props.projectSlug));
});
const projectV1 = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});

useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${projectV1.value.name}"`,
  }))
);

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(projectV1.value.name);
  return sortDatabaseV1List(list);
});

const isTenantProject = computed(() => {
  return projectV1.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});
</script>

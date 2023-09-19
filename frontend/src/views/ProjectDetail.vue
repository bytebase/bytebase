<template>
  <template v-if="hash === 'overview'">
    <ProjectOverviewPanel id="overview" :project="project" />
  </template>
  <template v-if="hash === 'branches'">
    <ProjectBranchesPanel id="branches" :project-id="project.uid" />
  </template>
  <template v-if="hash === 'databases'">
    <ProjectDeploymentConfigPanel
      v-if="isTenantProject"
      id="deployment-config"
      :project="project"
      :database-list="databaseV1List"
      :allow-edit="allowEdit"
    />
    <ProjectDatabasesPanel v-else :database-list="databaseV1List" />
  </template>
  <template v-if="hash === 'database-groups'">
    <ProjectDatabaseGroupPanel :project="project" />
  </template>
  <template v-if="hash === 'change-history'">
    <ProjectChangeHistoryPanel
      id="change-history"
      :database-list="databaseV1List"
    />
  </template>
  <template v-if="hash === 'slow-query'">
    <ProjectSlowQueryPanel :project="project" />
  </template>
  <template v-if="hash === 'activity'">
    <ProjectActivityPanel id="activity" :project="project" />
  </template>
  <template v-if="!isDefaultProject && hash === 'gitops'">
    <ProjectVersionControlPanel
      id="gitops"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'webhook'">
    <ProjectWebhookPanel
      id="webhook"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'members'">
    <ProjectMemberPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'setting'">
    <ProjectSettingPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import ProjectDatabaseGroupPanel from "@/components/DatabaseGroup/ProjectDatabaseGroupPanel.vue";
import ProjectBranchesPanel from "@/components/ProjectBranchesPanel.vue";
import {
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectChangeHistoryPanel from "../components/ProjectChangeHistoryPanel.vue";
import ProjectDatabasesPanel from "../components/ProjectDatabasesPanel.vue";
import ProjectDeploymentConfigPanel from "../components/ProjectDeploymentConfigPanel.vue";
import ProjectMemberPanel from "../components/ProjectMember/ProjectMemberPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import ProjectSlowQueryPanel from "../components/ProjectSlowQueryPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import { idFromSlug, sortDatabaseV1List } from "../utils";

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
const projectV1Store = useProjectV1Store();

const hash = computed(() => route.hash.replace(/^#?/, ""));

const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${project.value.name}"`,
  }))
);

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});

const isTenantProject = computed(() => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});
</script>

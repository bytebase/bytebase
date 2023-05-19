<template>
  <template v-if="hash === 'overview'">
    <ProjectOverviewPanel
      id="overview"
      :project="projectV1"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'databases'">
    <ProjectDeploymentConfigPanel
      v-if="isTenantProject"
      id="deployment-config"
      :project="projectV1"
      :database-list="databaseList"
      :allow-edit="allowEdit"
    />
    <ProjectDatabasesPanel v-else :database-list="databaseList" />
  </template>
  <template v-if="hash === 'change-history'">
    <ProjectMigrationHistoryPanel
      id="change-history"
      :database-list="databaseList"
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
      :project="project"
      :project-v1="projectV1"
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

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { cloneDeep } from "lodash-es";
import { useRoute } from "vue-router";

import { DEFAULT_PROJECT_ID } from "@/types";
import { idFromSlug, sortDatabaseListByEnvironmentV1 } from "../utils";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectSlowQueryPanel from "../components/ProjectSlowQueryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectDatabasesPanel from "../components/ProjectDatabasesPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import ProjectDeploymentConfigPanel from "../components/ProjectDeploymentConfigPanel.vue";
import {
  useDatabaseStore,
  useEnvironmentV1List,
  useLegacyProjectStore,
  useProjectV1Store,
} from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";

export default defineComponent({
  name: "ProjectDetail",
  components: {
    ProjectActivityPanel,
    ProjectMigrationHistoryPanel,
    ProjectSlowQueryPanel,
    ProjectOverviewPanel,
    ProjectVersionControlPanel,
    ProjectWebhookPanel,
    ProjectSettingPanel,
    ProjectDeploymentConfigPanel,
    ProjectDatabasesPanel,
  },
  props: {
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
  },
  setup(props) {
    const route = useRoute();
    const databaseStore = useDatabaseStore();
    const projectStore = useLegacyProjectStore();
    const projectV1Store = useProjectV1Store();

    const hash = computed(() => route.hash.replace(/^#?/, ""));

    const project = computed(() => {
      return projectStore.getProjectById(idFromSlug(props.projectSlug));
    });
    const projectV1 = computed(() => {
      return projectV1Store.getProjectByUID(
        String(idFromSlug(props.projectSlug))
      );
    });

    const environmentList = useEnvironmentV1List(false /* !showDeleted */);

    const prepareDatabaseList = () => {
      databaseStore.fetchDatabaseListByProjectId(String(project.value.id));
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      const list = cloneDeep(
        databaseStore
          .getDatabaseListByProjectId(String(project.value.id))
          .filter((db) => db.syncStatus === "OK")
      );
      return sortDatabaseListByEnvironmentV1(list, environmentList.value);
    });

    const isTenantProject = computed(() => {
      return projectV1.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
    });

    return {
      DEFAULT_PROJECT_ID,
      hash,
      project,
      projectV1,
      databaseList,
      isTenantProject,
    };
  },
});
</script>

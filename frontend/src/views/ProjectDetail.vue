<template>
  <template v-if="hash === 'overview'">
    <ProjectOverviewPanel
      id="overview"
      :project="project"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'databases'">
    <ProjectDeploymentConfigPanel
      v-if="isTenantProject"
      id="deployment-config"
      :project="project"
      :database-list="databaseList"
      :allow-edit="allowEdit"
    />
    <ProjectDatabasesPanel
      v-else
      :project="project"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'change-history'">
    <ProjectMigrationHistoryPanel
      id="change-history"
      :project="project"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'slow-query'">
    <ProjectSlowQueryPanel :project="project" />
  </template>
  <template v-if="hash === 'activity'">
    <ProjectActivityPanel id="activity" :project="project" />
  </template>
  <template v-if="hash === 'gitops'">
    <ProjectVersionControlPanel
      id="gitops"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="hash === 'webhook'">
    <ProjectWebhookPanel
      id="webhook"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="project.id !== DEFAULT_PROJECT_ID && hash === 'setting'">
    <ProjectSettingPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { cloneDeep } from "lodash-es";
import { useRoute } from "vue-router";

import { DEFAULT_PROJECT_ID } from "@/types";
import { idFromSlug, sortDatabaseList } from "../utils";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectSlowQueryPanel from "../components/ProjectSlowQueryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectDatabasesPanel from "../components/ProjectDatabasesPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import ProjectDeploymentConfigPanel from "../components/ProjectDeploymentConfigPanel.vue";
import { useDatabaseStore, useEnvironmentList, useProjectStore } from "@/store";

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
    const projectStore = useProjectStore();

    const hash = computed(() => route.hash.replace(/^#?/, ""));

    const project = computed(() => {
      return projectStore.getProjectById(idFromSlug(props.projectSlug));
    });

    const environmentList = useEnvironmentList(["NORMAL"]);

    const prepareDatabaseList = () => {
      databaseStore.fetchDatabaseListByProjectId(project.value.id);
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      const list = cloneDeep(
        databaseStore
          .getDatabaseListByProjectId(project.value.id)
          .filter((db) => db.syncStatus === "OK")
      );
      return sortDatabaseList(list, environmentList.value);
    });

    const isTenantProject = computed(() => {
      return project.value.tenantMode === "TENANT";
    });

    return {
      DEFAULT_PROJECT_ID,
      hash,
      project,
      databaseList,
      isTenantProject,
    };
  },
});
</script>

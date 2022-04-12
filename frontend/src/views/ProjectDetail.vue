<template>
  <template v-if="hash === 'overview'">
    <ProjectOverviewPanel
      id="overview"
      :project="project"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'migration-history'">
    <ProjectMigrationHistoryPanel
      id="migration-history"
      :project="project"
      :database-list="databaseList"
    />
  </template>
  <template v-if="hash === 'activity'">
    <ProjectActivityPanel id="activity" :project="project" />
  </template>
  <template v-else-if="hash === 'version-control'">
    <ProjectVersionControlPanel
      id="version-control"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-else-if="hash === 'webhook'">
    <ProjectWebhookPanel
      id="webhook"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-else-if="hash === 'setting'">
    <ProjectSettingPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-else-if="hash === 'deployment-config'">
    <ProjectDeploymentConfigPanel
      id="deployment-config"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { useStore } from "vuex";
import { idFromSlug, sortDatabaseList } from "../utils";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import ProjectDeploymentConfigPanel from "../components/ProjectDeploymentConfigPanel.vue";
import { cloneDeep } from "lodash-es";
import { useRoute } from "vue-router";
import { useDatabaseStore, useEnvironmentList, useProjectStore } from "@/store";

export default defineComponent({
  name: "ProjectDetail",
  components: {
    ProjectActivityPanel,
    ProjectMigrationHistoryPanel,
    ProjectOverviewPanel,
    ProjectVersionControlPanel,
    ProjectWebhookPanel,
    ProjectSettingPanel,
    ProjectDeploymentConfigPanel,
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
    const store = useStore();
    const databaseStore = useDatabaseStore();
    const route = useRoute();
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
        databaseStore.getDatabaseListByProjectId(project.value.id)
      );
      return sortDatabaseList(list, environmentList.value);
    });

    return {
      hash,
      project,
      databaseList,
    };
  },
});
</script>

<template>
  <template v-if="selectedTab == OVERVIEW_TAB">
    <ProjectOverviewPanel
      id="overview"
      :project="project"
      :databaseList="databaseList"
    />
  </template>
  <template v-if="selectedTab == MIGRATION_HISTORY_TAB">
    <ProjectMigrationHistoryPanel
      id="migration-history"
      :project="project"
      :databaseList="databaseList"
    />
  </template>
  <template v-if="selectedTab == ACTIVITY_TAB">
    <ProjectActivityPanel id="activity" :project="project" />
  </template>
  <template v-else-if="selectedTab == VERSION_CONTROL_TAB">
    <ProjectVersionControlPanel
      id="version-control"
      :project="project"
      :allowEdit="allowEdit"
    />
  </template>
  <template v-else-if="selectedTab == PROJECT_HOOK_TAB">
    <ProjectWebhookPanel
      id="webhook"
      :project="project"
      :allowEdit="allowEdit"
    />
  </template>
  <template v-else-if="selectedTab == SETTING_TAB">
    <ProjectSettingPanel
      id="setting"
      :project="project"
      :allowEdit="allowEdit"
    />
  </template>
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";
import { idFromSlug, sortDatabaseList } from "../utils";
import ProjectActivityPanel from "../components/ProjectActivityPanel.vue";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import { cloneDeep } from "lodash";

const OVERVIEW_TAB = 0;
const MIGRATION_HISTORY_TAB = 1;
const ACTIVITY_TAB = 2;
const VERSION_CONTROL_TAB = 3;
const PROJECT_HOOK_TAB = 4;
const SETTING_TAB = 5;

export default {
  name: "ProjectDetail",
  components: {
    ProjectActivityPanel,
    ProjectMigrationHistoryPanel,
    ProjectOverviewPanel,
    ProjectVersionControlPanel,
    ProjectWebhookPanel,
    ProjectSettingPanel,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
    selectedTab: {
      required: true,
      type: Number,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const project = computed(() => {
      return store.getters["project/projectByID"](
        idFromSlug(props.projectSlug)
      );
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    const prepareDatabaseList = () => {
      store.dispatch("database/fetchDatabaseListByProjectID", project.value.id);
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      const list = cloneDeep(
        store.getters["database/databaseListByProjectID"](project.value.id)
      );
      return sortDatabaseList(list, environmentList.value);
    });

    return {
      OVERVIEW_TAB,
      MIGRATION_HISTORY_TAB,
      ACTIVITY_TAB,
      VERSION_CONTROL_TAB,
      PROJECT_HOOK_TAB,
      SETTING_TAB,
      project,
      databaseList,
    };
  },
};
</script>

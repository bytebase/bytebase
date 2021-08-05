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
import { idFromSlug, isProjectOwner, sortDatabaseList } from "../utils";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import { cloneDeep } from "lodash";

const OVERVIEW_TAB = 0;
const MIGRATION_HISTORY_TAB = 1;
const VERSION_CONTROL_TAB = 2;
const PROJECT_HOOK_TAB = 3;
const SETTING_TAB = 4;

export default {
  name: "ProjectDetail",
  components: {
    ProjectMigrationHistoryPanel,
    ProjectOverviewPanel,
    ProjectVersionControlPanel,
    ProjectWebhookPanel,
    ProjectSettingPanel,
  },
  props: {
    selectedTab: {
      required: true,
      type: Number,
    },
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    const prepareDatabaseList = () => {
      store.dispatch("database/fetchDatabaseListByProjectId", project.value.id);
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      const list = cloneDeep(
        store.getters["database/databaseListByProjectId"](project.value.id)
      );
      return sortDatabaseList(list, environmentList.value);
    });

    // Only the project owner can edit the project general info and configure version control.
    // This means even the workspace owner won't be able to edit it.
    // On the other hand, we allow workspace owner to change project membership in case
    // project is locked somehow. See the relevant method in ProjectMemberTable for more info.
    const allowEdit = computed(() => {
      if (project.value.rowStatus == "ARCHIVED") {
        return false;
      }

      for (const member of project.value.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    return {
      OVERVIEW_TAB,
      MIGRATION_HISTORY_TAB,
      VERSION_CONTROL_TAB,
      PROJECT_HOOK_TAB,
      SETTING_TAB,
      project,
      databaseList,
      allowEdit,
    };
  },
};
</script>

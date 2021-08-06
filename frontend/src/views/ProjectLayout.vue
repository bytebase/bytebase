<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-6 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
  </h1>
  <BBTabFilter
    class="px-3 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="projectTabItemList.map((item) => item.name)"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        selectTab(index);
      }
    "
  />
  <div class="max-w-7xl mx-auto py-6 px-6">
    <template v-if="state.selectedIndex == OVERVIEW_TAB">
      <ProjectOverviewPanel
        id="overview"
        :project="project"
        :databaseList="databaseList"
      />
    </template>
    <template v-if="state.selectedIndex == MIGRATION_HISTORY_TAB">
      <ProjectMigrationHistoryPanel
        id="migration-history"
        :project="project"
        :databaseList="databaseList"
      />
    </template>
    <template v-else-if="state.selectedIndex == VERSION_CONTROL_TAB">
      <ProjectVersionControlPanel
        id="version-control"
        :project="project"
        :allowEdit="allowEdit"
      />
    </template>
    <template v-else-if="state.selectedIndex == WEBHOOK_TAB">
      <ProjectWebhookPanel
        id="webhook"
        :project="project"
        :allowEdit="allowEdit"
      />
    </template>
    <template v-else-if="state.selectedIndex == SETTING_TAB">
      <ProjectSettingPanel
        id="setting"
        :project="project"
        :allowEdit="allowEdit"
      />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watch, watchEffect } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { idFromSlug, isProjectOwner, sortDatabaseList } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import ProjectMigrationHistoryPanel from "../components/ProjectMigrationHistoryPanel.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "../components/ProjectWebhookPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";
import { cloneDeep } from "lodash";

const OVERVIEW_TAB = 0;
const MIGRATION_HISTORY_TAB = 1;
const VERSION_CONTROL_TAB = 2;
const WEBHOOK_TAB = 3;
const SETTING_TAB = 4;

type ProjectTabItem = {
  name: string;
  hash: string;
};

const projectTabItemList: ProjectTabItem[] = [
  { name: "Overview", hash: "overview" },
  { name: "Migration History", hash: "migration-history" },
  { name: "Version Control", hash: "version-control" },
  { name: "Webhooks", hash: "webhook" },
  { name: "Settings", hash: "setting" },
];

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "ProjectLayout",
  components: {
    ArchiveBanner,
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
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
    });

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

    const selectProjectTabOnHash = () => {
      if (router.currentRoute.value.hash) {
        for (let i = 0; i < projectTabItemList.length; i++) {
          if (
            projectTabItemList[i].hash ==
            router.currentRoute.value.hash.slice(1)
          ) {
            selectTab(i);
            break;
          }
        }
      } else {
        selectTab(OVERVIEW_TAB);
      }
    };

    onMounted(() => {
      selectProjectTabOnHash();
    });

    watch(
      () => router.currentRoute.value.hash,
      () => {
        if (router.currentRoute.value.name == "workspace.project.detail") {
          selectProjectTabOnHash();
        }
      }
    );

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

    const selectTab = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.project.detail",
        hash: "#" + projectTabItemList[index].hash,
      });
    };

    return {
      OVERVIEW_TAB,
      MIGRATION_HISTORY_TAB,
      VERSION_CONTROL_TAB,
      WEBHOOK_TAB,
      SETTING_TAB,
      state,
      project,
      databaseList,
      allowEdit,
      selectTab,
      projectTabItemList,
    };
  },
};
</script>

<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-4 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
  </h1>
  <BBTableTabFilter
    class="px-1 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="projectTabItemList.map((item) => item.name)"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        selectTab(index);
      }
    "
  />
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <div v-if="state.selectedIndex == OVERVIEW_TAB" class="space-y-6">
      <div class="space-y-2">
        <p class="text-lg font-medium leading-7 text-main">Databases</p>
        <DatabaseTable :mode="'PROJECT'" :databaseList="databaseList" />
      </div>
      <div class="space-y-2">
        <p class="text-lg font-medium leading-7 text-main">Issues</p>
        <IssueTable
          :mode="'PROJECT'"
          :issueSectionList="[
            {
              title: 'In progress',
              list: state.progressIssueList,
            },
            {
              title: 'Recently Closed',
              list: state.closedIssueList,
            },
          ]"
        />
      </div>
    </div>
    <template v-else-if="state.selectedIndex == VERSION_CONTROL_TAB">
      <ProjectVersionControlPanel :project="project" />
    </template>
    <template v-else-if="state.selectedIndex == SETTING_TAB">
      <div class="max-w-3xl mx-auto space-y-4">
        <div class="divide-y divide-block-border space-y-6">
          <ProjectGeneralSettingPanel :project="project" />
          <ProjectMemberPanel class="pt-4" :project="project" />
        </div>
        <template v-if="allowArchiveOrRestore">
          <template v-if="project.rowStatus == 'NORMAL'">
            <BBButtonConfirm
              :style="'ARCHIVE'"
              :buttonText="'Archive this project'"
              :okText="'Archive'"
              :confirmTitle="`Archive project '${project.name}'?`"
              :confirmDescription="'Archived project will not be shown on the normal interface. You can still restore later from the Archive page.'"
              :requireConfirm="true"
              @confirm="archiveOrRestoreProject(true)"
            />
          </template>
          <template v-else-if="project.rowStatus == 'ARCHIVED'">
            <BBButtonConfirm
              :style="'RESTORE'"
              :buttonText="'Restore this project'"
              :okText="'Restore'"
              :confirmTitle="`Restore project '${project.name}' to normal state?`"
              :confirmDescription="''"
              :requireConfirm="true"
              @confirm="archiveOrRestoreProject(false)"
            />
          </template>
        </template>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watchEffect, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { idFromSlug, isProjectOwner } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import ProjectGeneralSettingPanel from "../components/ProjectGeneralSettingPanel.vue";
import ProjectMemberPanel from "../components/ProjectMemberPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import IssueTable from "../components/IssueTable.vue";
import { ProjectPatch, Issue } from "../types";

const OVERVIEW_TAB = 0;
const VERSION_CONTROL_TAB = 1;
const SETTING_TAB = 2;

type ProjectTabItem = {
  name: string;
  hash: string;
};

const projectTabItemList: ProjectTabItem[] = [
  { name: "Overview", hash: "overview" },
  { name: "Version Control", hash: "version-control" },
  { name: "Settings", hash: "setting" },
];

interface LocalState {
  selectedIndex: number;
  progressIssueList: Issue[];
  closedIssueList: Issue[];
}

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    ProjectGeneralSettingPanel,
    ProjectMemberPanel,
    ProjectVersionControlPanel,
    IssueTable,
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

    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
      progressIssueList: [],
      closedIssueList: [],
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

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const databaseList = computed(() => {
      return store.getters["database/databaseListByProjectId"](
        project.value.id
      );
    });

    const prepareDatabaseList = () => {
      store.dispatch(
        "database/fetchDatabaseListByProjectId",
        idFromSlug(props.projectSlug)
      );
    };

    watchEffect(prepareDatabaseList);

    const prepareIssueList = () => {
      store
        .dispatch(
          "issue/fetchIssueListForProject",
          idFromSlug(props.projectSlug)
        )
        .then((issueList: Issue[]) => {
          state.progressIssueList = [];
          state.closedIssueList = [];
          for (const issue of issueList) {
            // "OPEN"
            if (issue.status === "OPEN") {
              state.progressIssueList.push(issue);
            }
            // "DONE" or "CANCELED"
            else if (issue.status === "DONE" || issue.status === "CANCELED") {
              state.closedIssueList.push(issue);
            }
          }
        });
    };

    watchEffect(prepareIssueList);

    // Only the project owner can archive/restore the project info.
    // This means even the workspace owner won't be able to edit it.
    // There seems to be no good reason that workspace owner needs to archive/restore the project.
    const allowArchiveOrRestore = computed(() => {
      for (const member of project.value.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const archiveOrRestoreProject = (archive: boolean) => {
      const projectPatch: ProjectPatch = {
        updaterId: currentUser.value.id,
        rowStatus: archive ? "ARCHIVED" : "NORMAL",
      };
      store.dispatch("project/patchProject", {
        projectId: project.value.id,
        projectPatch,
      });
    };

    const selectTab = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.project.detail",
        hash: "#" + projectTabItemList[index].hash,
      });
    };

    return {
      OVERVIEW_TAB,
      VERSION_CONTROL_TAB,
      SETTING_TAB,
      state,
      project,
      databaseList,
      allowArchiveOrRestore,
      archiveOrRestoreProject,
      selectTab,
      projectTabItemList,
    };
  },
};
</script>

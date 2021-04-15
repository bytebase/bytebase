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
    :tabList="['Overview', 'Repository', 'Settings']"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
      }
    "
  />
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <template v-if="state.selectedIndex == OVERVIEW_TAB">
      <div class="space-y-2">
        <p class="text-lg font-medium leading-7 text-main">Databases</p>
        <DatabaseTable :mode="'PROJECT'" :databaseList="databaseList" /></div
    ></template>
    <template v-else-if="state.selectedIndex == REPO_TAB"> </template>
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
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { idFromSlug, isProjectOwner } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import ProjectGeneralSettingPanel from "../components/ProjectGeneralSettingPanel.vue";
import ProjectMemberPanel from "../components/ProjectMemberPanel.vue";

const OVERVIEW_TAB = 0;
const REPO_TAB = 1;
const SETTING_TAB = 2;

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    ProjectGeneralSettingPanel,
    ProjectMemberPanel,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const databaseList = computed(() => {
      return store.getters["database/databaseListByInstanceId"](
        project.value.id
      );
    });

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
      store
        .dispatch("project/patchProject", {
          projectId: project.value.id,
          projectPatch: {
            rowStatus: archive ? "ARCHIVED" : "NORMAL",
          },
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      OVERVIEW_TAB,
      REPO_TAB,
      SETTING_TAB,
      state,
      project,
      databaseList,
      allowArchiveOrRestore,
      archiveOrRestoreProject,
    };
  },
};
</script>

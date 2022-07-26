<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <BBTooltipButton
        type="normal"
        tooltip-mode="ALWAYS"
        @click="goDefaultProject"
      >
        {{ $t("common.visit-default-project") }}
        <template #tooltip>
          <div class="whitespace-pre-wrap">
            {{ $t("quick-action.default-db-hint") }}
          </div>
        </template>
      </BBTooltipButton>
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('project.dashboard.search-bar-placeholder')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>
    <ProjectTable :project-list="filteredList(state.projectList)" />
  </div>
</template>

<script lang="ts">
import {
  useCurrentUser,
  useUIStateStore,
  useProjectStore,
  useHelpStore,
} from "@/store";
import { useRouter } from "vue-router";
import {
  watchEffect,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  defineComponent,
} from "vue";
import ProjectTable from "../components/ProjectTable.vue";
import { Project, UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
}

export default defineComponent({
  name: "ProjectDashboard",
  components: {
    ProjectTable,
  },
  setup() {
    const router = useRouter();
    const searchField = ref();
    const mountedTimer = ref();

    const uiStateStore = useUIStateStore();
    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();
    const helpStore = useHelpStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("guide.help.project")) {
        mountedTimer.value = setTimeout(() => {
          helpStore.showHelp("help.project", true);
          uiStateStore.saveIntroStateByKey({
            key: "project.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    onUnmounted(() => {
      clearTimeout(mountedTimer.value);
    });

    const prepareProjectList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        projectStore
          .fetchProjectListByUser({
            userId: currentUser.value.id,
            rowStatusList: ["NORMAL"],
          })
          .then((projectList: Project[]) => {
            state.projectList = projectList;
          });
      }
    };

    watchEffect(prepareProjectList);

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const goDefaultProject = () => {
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: DEFAULT_PROJECT_ID,
        },
      });
    };

    const filteredList = (list: Project[]) => {
      if (!state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((issue) => {
        return (
          !state.searchText ||
          issue.name.toLowerCase().includes(state.searchText.toLowerCase())
        );
      });
    };

    return {
      searchField,
      state,
      filteredList,
      changeSearchText,
      goDefaultProject,
    };
  },
});
</script>

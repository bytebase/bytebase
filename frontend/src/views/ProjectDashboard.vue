<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
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
import { useCurrentUser, useUIStateStore, useProjectStore } from "@/store";
import { watchEffect, onMounted, reactive, ref, defineComponent } from "vue";
import ProjectTable from "../components/ProjectTable.vue";
import { Project, UNKNOWN_ID } from "../types";

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
    const searchField = ref();

    const uiStateStore = useUIStateStore();
    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("project.visit")) {
        uiStateStore.saveIntroStateByKey({
          key: "project.visit",
          newState: true,
        });
      }
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
    };
  },
});
</script>

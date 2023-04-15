<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <SearchBox
        :value="state.searchText"
        :placeholder="$t('project.dashboard.search-bar-placeholder')"
        :autofocus="true"
        style="width: 12rem"
        @update:value="changeSearchText($event)"
      />
    </div>
    <ProjectTable
      :project-list="filteredList(state.projectList)"
      :left-bordered="false"
      :right-bordered="false"
    />
  </div>
</template>

<script lang="ts" setup>
import { watchEffect, onMounted, reactive } from "vue";

import { useCurrentUser, useUIStateStore, useProjectStore } from "@/store";
import { SearchBox } from "@/components/v2";
import ProjectTable from "../components/ProjectTable.vue";
import { Project, UNKNOWN_ID } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
}

const uiStateStore = useUIStateStore();
const currentUser = useCurrentUser();
const projectStore = useProjectStore();

const state = reactive<LocalState>({
  projectList: [],
  searchText: "",
});

onMounted(() => {
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
  const keyword = state.searchText.trim().toLowerCase();
  if (!keyword) {
    // Select "All"
    return list;
  }
  return list.filter((project) => {
    return (
      project.name.toLowerCase().includes(keyword) ||
      project.key.toLowerCase().includes(keyword)
    );
  });
};
</script>

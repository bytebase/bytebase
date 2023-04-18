<template>
  <div class="flex flex-col">
    <div class="py-2 flex justify-between items-center">
      <div class="flex justify-start items-center gap-x-2">
        <BBAttention
          :style="'INFO'"
          :title="$t('setting.project.description')"
        />
      </div>

      <div class="flex justify-end items-center gap-x-2">
        <SearchBox
          :value="state.searchText"
          :placeholder="$t('project.dashboard.search-bar-placeholder')"
          :autofocus="true"
          @update:value="changeSearchText($event)"
        />
      </div>
    </div>

    <div class="py-2">
      <NCheckbox v-model:checked="state.includesArchived">
        <span class="textinfolabel">
          {{ $t("setting.project.show-archived") }}
        </span>
      </NCheckbox>
    </div>

    <ProjectTable :project-list="filteredList" />

    <BBModal
      v-if="state.showCreateModal"
      class="relative overflow-hidden"
      :title="$t('quick-action.create-project')"
      @close="state.showCreateModal = false"
    >
      <ProjectCreate @dismiss="state.showCreateModal = false" />
    </BBModal>
  </div>
</template>

<script setup lang="ts">
import { watchEffect, reactive, computed } from "vue";
import { NCheckbox } from "naive-ui";

import { DEFAULT_PROJECT_ID, Project } from "../types";
import { useProjectStore } from "@/store";
import ProjectTable from "../components/ProjectTable.vue";
import ProjectCreate from "../components/ProjectCreate.vue";
import { SearchBox } from "@/components/v2";

interface LocalState {
  projectList: Project[];
  searchText: string;
  showCreateModal: boolean;
  includesArchived: boolean;
}

const projectStore = useProjectStore();

const state = reactive<LocalState>({
  projectList: [],
  searchText: "",
  showCreateModal: false,
  includesArchived: false,
});

const prepareProjectList = async () => {
  const projectList = [...(await projectStore.fetchAllProjectList())];
  // Put "Unassigned" to the first;
  const unassignedIndex = projectList.findIndex(
    (project) => project.id === DEFAULT_PROJECT_ID
  );
  if (unassignedIndex >= 0) {
    const unassignedProject = projectList[unassignedIndex];
    projectList.splice(unassignedIndex, 1);
    projectList.unshift(unassignedProject);
  }
  state.projectList = projectList;
};

watchEffect(prepareProjectList);

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

const filteredList = computed(() => {
  let list = state.projectList;
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    list = list.filter(
      (project) =>
        project.name.toLowerCase().includes(keyword) ||
        project.key.toLowerCase().includes(keyword)
    );
  }
  if (!state.includesArchived) {
    list = list.filter((project) => project.rowStatus === "NORMAL");
  }
  return list;
});
</script>

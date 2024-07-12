<template>
  <div class="flex flex-col space-y-4">
    <div class="px-4 flex justify-end">
      <SearchBox
        v-model:value="state.searchText"
        class="!max-w-full md:!max-w-[18rem]"
        :placeholder="$t('common.filter-by-name')"
        :autofocus="true"
      />
    </div>
    <div class="w-full px-4">
      <ProjectV1Table :project-list="filteredProjectList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { useUIStateStore, useProjectV1List } from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { filterProjectV1ListByKeyword } from "@/utils";

interface LocalState {
  searchText: string;
}

const state = reactive<LocalState>({
  searchText: "",
});
const { projectList } = useProjectV1List();

const filteredProjectList = computed(() => {
  const list = projectList.value.filter(
    (project) => project.name !== DEFAULT_PROJECT_NAME
  );
  return filterProjectV1ListByKeyword(list, state.searchText);
});

onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});
</script>

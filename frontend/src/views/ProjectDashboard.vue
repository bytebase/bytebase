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
    <ProjectV1Table :project-list="filteredProjectList" class="border-x-0" />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";

import {
  useUIStateStore,
  useProjectV1ListByUser,
  useCurrentUserV1,
} from "@/store";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { filterProjectV1ListByKeyword } from "@/utils";

interface LocalState {
  searchText: string;
}

const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  searchText: "",
});

const { projectList } = useProjectV1ListByUser(currentUserV1);

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

const filteredProjectList = computed(() => {
  const list = projectList.value;
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

<template>
  <div class="flex flex-col space-y-4">
    <div class="px-4 flex justify-end items-center">
      <SearchBox
        v-model:value="state.searchText"
        :placeholder="$t('common.filter-by-name')"
        :autofocus="true"
        style="width: 12rem"
      />
    </div>
    <ProjectV1Table :project-list="filteredProjectList" class="border-x-0" />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import {
  useUIStateStore,
  useProjectV1ListByCurrentUser,
  useCurrentUserV1,
} from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { filterProjectV1ListByKeyword, isMemberOfProjectV1 } from "@/utils";

interface LocalState {
  searchText: string;
}

const state = reactive<LocalState>({
  searchText: "",
});
const currentUserV1 = useCurrentUserV1();
const { projectList } = useProjectV1ListByCurrentUser();

const filteredProjectList = computed(() => {
  const list = projectList.value.filter(
    (project) =>
      project.uid != String(DEFAULT_PROJECT_ID) &&
      isMemberOfProjectV1(project.iamPolicy, currentUserV1.value)
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

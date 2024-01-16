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
    <ProjectV1Table :project-list="filteredProjectList" class="border-x-0" />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { useUIStateStore, useProjectV1ListByCurrentUser } from "@/store";
import { filterProjectV1ListByKeyword } from "@/utils";

interface LocalState {
  searchText: string;
}

const state = reactive<LocalState>({
  searchText: "",
});
const { projectList } = useProjectV1ListByCurrentUser();

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

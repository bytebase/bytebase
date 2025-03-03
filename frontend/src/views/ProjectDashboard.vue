<template>
  <div class="flex flex-col space-y-4">
    <div class="px-4 flex items-center space-x-2">
      <SearchBox
        v-model:value="state.searchText"
        style="max-width: 100%"
        :placeholder="$t('common.filter-by-name')"
        :autofocus="true"
      />
      <NButton
        v-if="hasWorkspacePermissionV2('bb.projects.create')"
        type="primary"
        @click="state.showCreateDrawer = true"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("quick-action.new-project") }}
      </NButton>
    </div>
    <PagedTable
      ref="projectPagedTable"
      session-key="bb.project-table"
      :fetch-list="fetchProjects"
      :footer-class="'mx-4'"
    >
      <template #table="{ list, loading }">
        <ProjectV1Table
          :bordered="false"
          :loading="loading"
          :project-list="list"
        />
      </template>
    </PagedTable>
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel @dismiss="state.showCreateDrawer = false" />
  </Drawer>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { onMounted, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { Drawer } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useUIStateStore, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_NAME, type ComposedProject } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
}

const state = reactive<LocalState>({
  searchText: "",
  showCreateDrawer: false,
});
const projectStore = useProjectV1Store();
const projectPagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedProject>>>();

onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});

const fetchProjects = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, projects } = await projectStore.fetchProjectList({
    showDeleted: false,
    pageToken,
    pageSize,
    query: state.searchText,
  });
  return {
    nextPageToken: nextPageToken ?? "",
    list: projects.filter((project) => project.name !== DEFAULT_PROJECT_NAME),
  };
};

watch(
  () => state.searchText,
  useDebounceFn(async () => {
    await projectPagedTable.value?.refresh();
  }, 500)
);
</script>

<template>
  <PagedTable
    ref="projectPagedTable"
    :session-key="sessionKey"
    :fetch-list="fetchProjects"
    :order-keys="['title']"
  >
    <template #table="{ list, loading, sorters, onSortersUpdate }">
      <ProjectV1Table
        v-bind="$attrs"
        :loading="loading"
        :project-list="list"
        :selected-project-names="selectedProjectNames"
        :show-selection="showSelection"
        :show-labels="showLabels"
        :keyword="filter.query"
        :sorters="sorters"
        @update:selected-project-names="updateSelectedProjectNames"
        @update:sorters="onSortersUpdate"
      />
    </template>
  </PagedTable>
</template>

<script lang="tsx" setup>
import { ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { type ProjectFilter, useProjectV1Store } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ProjectV1Table from "./ProjectV1Table.vue";

const props = withDefaults(
  defineProps<{
    filter: ProjectFilter;
    sessionKey: string;
    selectedProjectNames?: string[];
    showSelection?: boolean;
    showLabels?: boolean;
  }>(),
  {
    selectedProjectNames: () => [],
    showSelection: false,
    showLabels: true,
  }
);

const emit = defineEmits<{
  (event: "update:selected-project-names", projectNames: string[]): void;
}>();

const projectStore = useProjectV1Store();

const projectPagedTable = ref<ComponentExposed<typeof PagedTable<Project>>>();

const updateSelectedProjectNames = (projectNames: string[]) => {
  emit("update:selected-project-names", projectNames);
};

const refresh = () => {
  projectPagedTable.value?.refresh();
};

defineExpose({ refresh });

const fetchProjects = async ({
  pageToken,
  pageSize,
  orderBy,
}: {
  pageToken: string;
  pageSize: number;
  orderBy?: string;
}) => {
  const { nextPageToken, projects } = await projectStore.fetchProjectList({
    pageToken,
    pageSize,
    filter: props.filter,
    orderBy,
  });
  return {
    nextPageToken: nextPageToken ?? "",
    list: projects,
  };
};

watch(
  () => props.filter,
  () => projectPagedTable.value?.refresh(),
  { deep: true }
);
</script>

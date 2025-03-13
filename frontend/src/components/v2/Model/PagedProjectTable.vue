<template>
  <PagedTable
    ref="projectPagedTable"
    :session-key="sessionKey"
    :fetch-list="fetchProjects"
  >
    <template #table="{ list, loading }">
      <ProjectV1Table v-bind="$attrs" :loading="loading" :project-list="list" />
    </template>
  </PagedTable>
</template>

<script lang="tsx" setup>
import { ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useProjectV1Store } from "@/store";
import { type ComposedProject } from "@/types";
import ProjectV1Table from "./ProjectV1Table.vue";

const props = defineProps<{
  search: string;
  sessionKey: string;
  includeDefault: boolean;
}>();

const projectStore = useProjectV1Store();

const projectPagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedProject>>>();

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
    query: props.search,
    excludeDefault: !props.includeDefault,
  });
  return {
    nextPageToken: nextPageToken ?? "",
    list: projects,
  };
};

watch(
  () => props.search,
  () => projectPagedTable.value?.refresh()
);
</script>

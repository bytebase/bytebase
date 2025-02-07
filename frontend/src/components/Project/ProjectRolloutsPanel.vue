<template>
  <div class="w-full flex flex-col gap-y-2">
    <PagedTable
      :key="project.name"
      :session-key="`project-${project.name}-releases`"
      :fetch-list="fetchRolloutList"
    >
      <template #table="{ list, loading }">
        <RolloutDataTable
          :bordered="true"
          :loading="loading"
          :rollout-list="list"
        />
      </template>
    </PagedTable>
  </div>
</template>

<script lang="ts" setup>
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useRolloutStore } from "@/store";
import type { ComposedProject } from "@/types";
import RolloutDataTable from "../Rollout/RolloutDataTable.vue";

const props = defineProps<{
  project: ComposedProject;
}>();

const rolloutStore = useRolloutStore();

const fetchRolloutList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, rollouts } = await rolloutStore.fetchRolloutsByProject(
    props.project.name,
    { pageSize, pageToken }
  );
  return {
    nextPageToken,
    list: rollouts,
  };
};
</script>

<template>
  <div class="w-full flex flex-col gap-y-2">
    <PagedRolloutTable
      :key="project.name"
      :session-key="`project-${project.name}-releases`"
      :project="project.name"
      :page-size="50"
    >
      <template #table="{ rolloutList, loading }">
        <RolloutDataTable
          :bordered="true"
          :loading="loading"
          :rollout-list="rolloutList"
        />
      </template>
    </PagedRolloutTable>
  </div>
</template>

<script lang="ts" setup>
import type { ComposedProject } from "@/types";
import PagedRolloutTable from "../Rollout/PagedRolloutTable.vue";
import RolloutDataTable from "../Rollout/RolloutDataTable.vue";

defineProps<{
  project: ComposedProject;
}>();
</script>

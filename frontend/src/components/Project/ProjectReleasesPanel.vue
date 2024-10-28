<template>
  <div class="flex flex-col gap-y-2">
    <PagedReleaseTable
      :key="project.name"
      :session-key="`project-${project.name}-releases`"
      :project="project.name"
      :page-size="50"
    >
      <template #table="{ releaseList, loading }">
        <ReleaseDataTable
          :bordered="true"
          :loading="loading"
          :release-list="releaseList"
        />
      </template>
    </PagedReleaseTable>
  </div>
</template>

<script lang="ts" setup>
import type { ComposedProject } from "@/types";
import PagedReleaseTable from "../Release/PagedReleaseTable.vue";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

defineProps<{
  project: ComposedProject;
}>();
</script>

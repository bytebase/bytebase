<template>
  <div class="flex flex-col gap-y-2">
    <PagedReleaseTable
      :key="project.name"
      :session-key="`project-${project.name}-releases`"
      :project="project.name"
      :page-size="50"
      :show-deleted="state.showArchived"
    >
      <template #table="{ releaseList, loading }">
        <ReleaseDataTable
          :bordered="true"
          :loading="loading"
          :release-list="releaseList"
        />
      </template>
    </PagedReleaseTable>
    <p>
      <NCheckbox v-model:checked="state.showArchived">
        <span class="textinfolabel">{{ $t("release.show-archived") }}</span>
      </NCheckbox>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { reactive } from "vue";
import type { ComposedProject } from "@/types";
import PagedReleaseTable from "../Release/PagedReleaseTable.vue";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

defineProps<{
  project: ComposedProject;
}>();

interface LocalState {
  showArchived: boolean;
}

const state = reactive<LocalState>({
  showArchived: false,
});
</script>

<template>
  <div class="w-full flex flex-col gap-y-4">
    <div class="w-full flex flex-row justify-end items-center">
      <router-link :to="`/${project.name}/releases/new`">
        <NButton type="primary">
          <template #icon>
            <PlusIcon />
          </template>
          {{ $t("release.create") }}
        </NButton>
      </router-link>
    </div>
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
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import type { ComposedProject } from "@/types";
import PagedReleaseTable from "../Release/PagedReleaseTable.vue";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

defineProps<{
  project: ComposedProject;
}>();
</script>

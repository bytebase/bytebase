<template>
  <div class="w-full flex flex-col gap-y-4">
    <NAlert type="info">
      <span>{{ $t("release.usage-description") }}</span>
    </NAlert>
    <div class="w-full flex flex-row justify-end items-center">
      <!-- Only show create button in dev mode -->
      <router-link v-if="isDev" :to="`/${project.name}/releases/new`">
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
import { NAlert, NButton } from "naive-ui";
import type { ComposedProject } from "@/types";
import PagedReleaseTable from "../Release/PagedReleaseTable.vue";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

defineProps<{
  project: ComposedProject;
}>();
</script>

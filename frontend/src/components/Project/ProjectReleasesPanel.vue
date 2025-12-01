<template>
  <div class="w-full flex flex-col gap-y-4">
    <NAlert type="info">
      <span>{{ $t("release.usage-description") }}</span>
      <LearnMoreLink
        url="https://docs.bytebase.com/gitops/migration-based-workflow/release/?source=console"
        class="ml-1"
      />
    </NAlert>
    <PagedTable
      :key="project.name"
      :session-key="`project-${project.name}-releases`"
      :fetch-list="fetchReleaseList"
    >
      <template #table="{ list, loading }">
        <ReleaseDataTable
          :bordered="true"
          :loading="loading"
          :release-list="list"
        />
      </template>
    </PagedTable>
  </div>
</template>

<script lang="ts" setup>
import { NAlert } from "naive-ui";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useReleaseStore } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

const props = defineProps<{
  project: Project;
}>();

const releaseStore = useReleaseStore();

const fetchReleaseList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, releases } = await releaseStore.fetchReleasesByProject(
    props.project.name,
    { pageSize, pageToken },
    false
  );
  return {
    nextPageToken,
    list: releases,
  };
};
</script>

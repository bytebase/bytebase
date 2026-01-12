<template>
  <div class="flex flex-col gap-y-4" v-bind="$attrs">
    <PagedTable
      ref="changedlogPagedTable"
      :session-key="`bb.paged-changelog-table.${database.name}`"
      :fetch-list="fetchChangelogList"
    >
      <template #table="{ list, loading }">
        <ChangelogDataTable
          :key="`changelog-table.${database.name}`"
          :loading="loading"
          :changelogs="list"
          :show-selection="false"
        />
      </template>
    </PagedTable>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { ChangelogDataTable } from "@/components/Changelog";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useChangelogStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const changelogStore = useChangelogStore();
const changedlogPagedTable =
  ref<ComponentExposed<typeof PagedTable<Changelog>>>();

const fetchChangelogList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, changelogs } = await changelogStore.fetchChangelogList(
    {
      parent: props.database.name,
      pageSize,
      pageToken,
    }
  );
  return {
    nextPageToken,
    list: changelogs,
  };
};
</script>

<template>
  <div class="flex flex-col space-y-2" v-bind="$attrs">
    <PagedTable
      :key="pagedRevisionTableSessionKey"
      :session-key="pagedRevisionTableSessionKey"
      :fetch-list="fetchRevisionList"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :show-selection="true"
        />
      </template>
    </PagedTable>
  </div>
</template>

<script lang="ts" setup>
import { RevisionDataTable } from "@/components/Revision";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { databaseServiceClient } from "@/grpcweb";
import type { ComposedDatabase } from "@/types";
import { useDatabaseDetailContext } from "./context";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { pagedRevisionTableSessionKey } = useDatabaseDetailContext();

const fetchRevisionList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, revisions } =
    await databaseServiceClient.listRevisions({
      parent: props.database.name,
      pageSize,
      pageToken,
    });
  return {
    nextPageToken,
    list: revisions,
  };
};
</script>

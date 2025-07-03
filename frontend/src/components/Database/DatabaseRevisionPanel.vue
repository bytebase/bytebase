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
import { create } from "@bufbuild/protobuf";
import { RevisionDataTable } from "@/components/Revision";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { revisionServiceClientConnect } from "@/grpcweb";
import type { ComposedDatabase } from "@/types";
import { ListRevisionsRequestSchema } from "@/types/proto-es/v1/revision_service_pb";
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
  const request = create(ListRevisionsRequestSchema, {
    parent: props.database.name,
    pageSize,
    pageToken,
  });
  const { nextPageToken, revisions } =
    await revisionServiceClientConnect.listRevisions(request);
  return {
    nextPageToken,
    list: revisions,
  };
};
</script>

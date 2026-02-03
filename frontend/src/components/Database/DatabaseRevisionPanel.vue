<template>
  <div class="flex flex-col gap-y-2" v-bind="$attrs">
    <div class="flex justify-between items-center">
      <div></div>
      <NButton type="primary" @click="showCreateRevisionDrawer = true">
        {{ $t("common.import") }}
      </NButton>
    </div>
    <PagedTable
      ref="revisionPagedTable"
      :session-key="`bb.paged-revision-table.${database.name}`"
      :fetch-list="fetchRevisionList"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :show-selection="true"
          @delete="refreshList"
        />
      </template>
    </PagedTable>
  </div>

  <!-- Create Revision Drawer -->
  <CreateRevisionDrawer
    v-model:show="showCreateRevisionDrawer"
    :database="database.name"
    @created="handleRevisionCreated"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { RevisionDataTable } from "@/components/Revision";
import CreateRevisionDrawer from "@/components/Revision/CreateRevisionDrawer.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { revisionServiceClientConnect } from "@/connect";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { ListRevisionsRequestSchema } from "@/types/proto-es/v1/revision_service_pb";

const props = defineProps<{
  database: Database;
}>();

const revisionPagedTable = ref<ComponentExposed<typeof PagedTable<Revision>>>();
const showCreateRevisionDrawer = ref(false);

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

const handleRevisionCreated = () => {
  showCreateRevisionDrawer.value = false;
  revisionPagedTable.value?.refresh();
};

const refreshList = () => {
  revisionPagedTable.value?.refresh();
};
</script>

<template>
  <div class="flex flex-col gap-y-2" v-bind="$attrs">
    <div class="flex justify-between items-center">
      <div></div>
      <NButton type="primary" @click="showCreateRevisionDrawer = true">
        {{ $t("common.import") }}
      </NButton>
    </div>
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
          :custom-click="true"
          @row-click="(name) => (selectedRevisionName = name)"
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

  <Drawer
    :show="!!selectedRevisionName"
    @close="selectedRevisionName = undefined"
  >
    <DrawerContent
      style="width: 75vw; max-width: calc(100vw - 8rem)"
      :title="$t('common.detail')"
    >
      <RevisionDetailPanel
        v-if="selectedRevisionName"
        :database="database"
        :revision-name="selectedRevisionName"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { ref } from "vue";
import { RevisionDataTable, RevisionDetailPanel } from "@/components/Revision";
import CreateRevisionDrawer from "@/components/Revision/CreateRevisionDrawer.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { revisionServiceClientConnect } from "@/connect";
import type { ComposedDatabase } from "@/types";
import { ListRevisionsRequestSchema } from "@/types/proto-es/v1/revision_service_pb";
import { useDatabaseDetailContext } from "./context";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { pagedRevisionTableSessionKey } = useDatabaseDetailContext();
const showCreateRevisionDrawer = ref(false);
const selectedRevisionName = ref<string>();

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

  // Refresh the revision list to show the newly created revision
  // The PagedTable component will automatically refresh when we trigger it
  pagedRevisionTableSessionKey.value = `${pagedRevisionTableSessionKey.value}-refresh-${Date.now()}`;
};
</script>

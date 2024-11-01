<template>
  <div class="flex flex-col space-y-2" v-bind="$attrs">
    <PagedRevisionTable
      :key="pagedRevisionTableSessionKey"
      :database="database"
      :session-key="pagedRevisionTableSessionKey"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :show-selection="true"
        />
      </template>
    </PagedRevisionTable>
  </div>
</template>

<script lang="ts" setup>
import { PagedRevisionTable, RevisionDataTable } from "@/components/Revision";
import type { ComposedDatabase } from "@/types";
import { useDatabaseDetailContext } from "./context";

defineProps<{
  database: ComposedDatabase;
}>();

const { pagedRevisionTableSessionKey } = useDatabaseDetailContext();
</script>

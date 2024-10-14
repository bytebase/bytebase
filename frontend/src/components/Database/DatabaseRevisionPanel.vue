<template>
  <div class="flex flex-col space-y-4" v-bind="$attrs">
    <PagedRevisionTable
      :database="database"
      session-key="bb.paged-revision-table"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :custom-click="true"
          :show-selection="true"
        />
      </template>
    </PagedRevisionTable>
  </div>
</template>

<script lang="ts" setup>
import { PagedRevisionTable, RevisionDataTable } from "@/components/Revision";
import type { ComposedDatabase } from "@/types";

defineProps<{
  database: ComposedDatabase;
}>();
</script>

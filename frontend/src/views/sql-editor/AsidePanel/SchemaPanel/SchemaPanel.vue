<template>
  <template v-if="databaseMetadata">
    <DatabaseSchema
      v-show="!state.selected"
      :database="database"
      :database-metadata="databaseMetadata"
      @select-table="handleSelectTable"
    />
    <TableSchema
      v-if="state.selected"
      :database="database"
      :database-metadata="databaseMetadata"
      :schema="state.selected.schema"
      :table="state.selected.table"
      @close="state.selected = undefined"
    />
  </template>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from "vue";
import { storeToRefs } from "pinia";

import { UNKNOWN_ID } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import { useDatabaseById, useDBSchemaStore, useTabStore } from "@/store";
import DatabaseSchema from "./DatabaseSchema.vue";
import TableSchema from "./TableSchema.vue";

type LocalState = {
  selected?: { schema: SchemaMetadata; table: TableMetadata };
};

const state = reactive<LocalState>({
  selected: undefined,
});

const dbSchemaStore = useDBSchemaStore();
const { currentTab } = storeToRefs(useTabStore());
const conn = computed(() => currentTab.value.connection);

const database = useDatabaseById(computed(() => conn.value.databaseId));
const databaseMetadata = ref<DatabaseMetadata>();

const handleSelectTable = (schema: SchemaMetadata, table: TableMetadata) => {
  state.selected = { schema, table };
};

watch(
  () => database.value.id,
  async (databaseId) => {
    state.selected = undefined;
    databaseMetadata.value = undefined;
    if (databaseId !== UNKNOWN_ID) {
      databaseMetadata.value =
        await dbSchemaStore.getOrFetchDatabaseMetadataById(
          databaseId,
          /* !skipCache */ false
        );
    }
  },
  { immediate: true }
);
</script>

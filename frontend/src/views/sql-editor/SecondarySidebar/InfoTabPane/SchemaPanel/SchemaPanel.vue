<template>
  <div class="flex flex-col h-full py-0.5 gap-y-1">
    <div class="flex items-center gap-x-1 px-2 pt-1.5">
      <NInput
        v-model:value="keyword"
        size="small"
        :placeholder="$t('common.filter-by-name')"
        :clearable="true"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>

    <div class="w-full h-full relative overflow-hidden">
      <template v-if="databaseMetadata">
        <DatabaseSchema
          :database="database"
          :database-metadata="databaseMetadata"
          :header-clickable="selected !== undefined"
          @click-header="selected = undefined"
          @select-table="handleSelectTable"
        />
        <Transition name="slide-up">
          <TableSchema
            v-if="selected"
            class="absolute bottom-0 w-full h-[calc(100%-33px)] bg-white"
            :database="database"
            :schema="selected.schema"
            :table="selected.table"
            @close="selected = undefined"
          />
        </Transition>
      </template>

      <div
        v-else
        class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import { useDatabaseV1ByUID, useDBSchemaV1Store, useTabStore } from "@/store";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import DatabaseSchema from "./DatabaseSchema.vue";
import TableSchema from "./TableSchema.vue";
import { provideSchemaPanelContext } from "./context";

const { selectedDatabaseSchemaByDatabaseName } = useSQLEditorContext();

const dbSchemaStore = useDBSchemaV1Store();
const { currentTab } = storeToRefs(useTabStore());
const conn = computed(() => currentTab.value.connection);
const { keyword } = provideSchemaPanelContext();

const { database } = useDatabaseV1ByUID(computed(() => conn.value.databaseId));
const databaseMetadata = ref<DatabaseMetadata>();

const selected = computed({
  get() {
    return selectedDatabaseSchemaByDatabaseName.value.get(database.value.name);
  },
  set(selected) {
    if (!selected) {
      selectedDatabaseSchemaByDatabaseName.value.delete(database.value.name);
    } else {
      selectedDatabaseSchemaByDatabaseName.value.set(
        database.value.name,
        selected
      );
    }
  },
});

const handleSelectTable = async (
  schema: SchemaMetadata,
  table: TableMetadata
) => {
  const tableMetadata = await dbSchemaStore.getOrFetchTableMetadata({
    database: database.value.name,
    schema: schema.name,
    table: table.name,
  });
  const databaseMetadata = useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name
  );

  selected.value = {
    db: database.value,
    database: databaseMetadata,
    schema,
    table: tableMetadata,
  };
};

watch(
  () => database.value.name,
  async (name) => {
    if (!name) return;
    databaseMetadata.value = await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: name,
      skipCache: false,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
    });
  },
  { immediate: true }
);
</script>

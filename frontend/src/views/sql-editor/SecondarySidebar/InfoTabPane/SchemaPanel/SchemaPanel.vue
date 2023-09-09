<template>
  <div class="flex flex-col h-full py-0.5 gap-y-1">
    <div class="flex items-center gap-x-1 px-2 pt-1.5">
      <NInput
        v-model:value="keyword"
        size="small"
        :placeholder="$t('sql-editor.filter-by-name')"
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
          @alter-schema="emit('alter-schema', $event)"
        />
        <Transition name="slide-up" appear>
          <TableSchema
            v-if="selected"
            class="absolute bottom-0 w-full h-[calc(100%-33px)] bg-white"
            :database="database"
            :database-metadata="databaseMetadata"
            :schema="selected.schema"
            :table="selected.table"
            @close="selected = undefined"
            @alter-schema="emit('alter-schema', $event)"
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
} from "@/types/proto/v1/database_service";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import DatabaseSchema from "./DatabaseSchema.vue";
import TableSchema from "./TableSchema.vue";
import { provideSchemaPanelContext } from "./context";

const emit = defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const { selectedDatabaseSchema: selected } = useSQLEditorContext();

const dbSchemaStore = useDBSchemaV1Store();
const { currentTab } = storeToRefs(useTabStore());
const conn = computed(() => currentTab.value.connection);
const { keyword } = provideSchemaPanelContext();

const { database } = useDatabaseV1ByUID(computed(() => conn.value.databaseId));
const databaseMetadata = ref<DatabaseMetadata>();

const handleSelectTable = (schema: SchemaMetadata, table: TableMetadata) => {
  selected.value = { schema, table };
};

watch(
  () => database.value.name,
  async (name) => {
    selected.value = undefined;
    databaseMetadata.value = await dbSchemaStore.getOrFetchDatabaseMetadata(
      name,
      /* !skipCache */ false
    );
  },
  { immediate: true }
);
</script>

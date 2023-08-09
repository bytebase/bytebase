<template>
  <div class="w-full h-full relative overflow-hidden">
    <template v-if="databaseMetadata">
      <DatabaseSchema
        :database="database"
        :database-metadata="databaseMetadata"
        :header-clickable="state.selected !== undefined"
        @click-header="state.selected = undefined"
        @select-table="handleSelectTable"
        @alter-schema="emit('alter-schema', $event)"
      />
      <Transition name="slide-up" appear>
        <TableSchema
          v-if="state.selected"
          class="absolute bottom-0 w-full h-[calc(100%-41px)] bg-white"
          :database="database"
          :database-metadata="databaseMetadata"
          :schema="state.selected.schema"
          :table="state.selected.table"
          @close="state.selected = undefined"
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
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { useDatabaseV1ByUID, useDBSchemaV1Store, useTabStore } from "@/store";
import { DatabaseId } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import DatabaseSchema from "./DatabaseSchema.vue";
import TableSchema from "./TableSchema.vue";

type LocalState = {
  selected?: { schema: SchemaMetadata; table: TableMetadata };
};

const emit = defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: DatabaseId; schema: string; table: string }
  ): void;
}>();

const state = reactive<LocalState>({
  selected: undefined,
});

const dbSchemaStore = useDBSchemaV1Store();
const { currentTab } = storeToRefs(useTabStore());
const conn = computed(() => currentTab.value.connection);

const { database } = useDatabaseV1ByUID(computed(() => conn.value.databaseId));
const databaseMetadata = ref<DatabaseMetadata>();

const handleSelectTable = (schema: SchemaMetadata, table: TableMetadata) => {
  state.selected = { schema, table };
};

watch(
  () => database.value.name,
  async (name) => {
    state.selected = undefined;
    databaseMetadata.value = await dbSchemaStore.getOrFetchDatabaseMetadata(
      name,
      /* !skipCache */ false
    );
  },
  { immediate: true }
);
</script>

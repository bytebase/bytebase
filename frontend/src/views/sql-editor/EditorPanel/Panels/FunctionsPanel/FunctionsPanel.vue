<template>
  <div v-if="metadata" class="px-2 py-2 h-full overflow-hidden flex flex-col">
    <div
      v-if="showToolbar"
      class="pb-2 w-full flex flex-row justify-between items-center"
    >
      <template v-if="!metadata.func">
        <div
          v-if="showSchemaSelect"
          class="flex flex-row justify-start items-center text-sm gap-x-2"
        >
          <span>Schema:</span>
          <NSelect
            v-model:value="selectedSchemaName"
            :options="schemaSelectOptions"
            class="min-w-[8rem]"
          />
        </div>
      </template>
      <template v-else>
        <NButton size="small" @click="deselectFunction">
          <ArrowLeftIcon class="w-4 h-4" />
        </NButton>
      </template>
    </div>

    <template v-if="metadata.schema">
      <FunctionsTable
        v-if="!metadata.func"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :funcs="metadata.schema.functions"
        @click="handleSelectFunction"
      />
      <MonacoEditor
        v-if="metadata.func"
        :content="metadata.func.definition"
        :readonly="true"
        class="border w-full rounded flex-1 relative"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "lucide-vue-next";
import { NButton, NSelect } from "naive-ui";
import { computed, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSelectSchema } from "../../common";
import FunctionsTable from "./FunctionsTable.vue";

const {
  selectedSchemaName,
  options: schemaSelectOptions,
  showSchemaSelect,
} = useSelectSchema();
const { database } = useConnectionOfCurrentSQLEditorTab();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const metadata = ref<{
  database: DatabaseMetadata;
  schema?: SchemaMetadata;
  func?: FunctionMetadata;
}>();

const showToolbar = computed(() => {
  if (metadata.value?.func) {
    // show function detail and back button
    return true;
  }

  if (showSchemaSelect.value) return true;
  return false;
});

const handleSelectFunction = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}) => {
  metadata.value = selected;
};

const deselectFunction = () => {
  if (!metadata.value) return;
  metadata.value.func = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      func: undefined,
    };
  },
  { immediate: true }
);
</script>

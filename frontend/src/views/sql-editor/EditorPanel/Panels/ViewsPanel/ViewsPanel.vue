<template>
  <div v-if="metadata" class="px-2 py-2 h-full overflow-hidden flex flex-col">
    <div
      v-if="showToolbar"
      class="pb-2 w-full flex flex-row justify-between items-center"
    >
      <template v-if="!metadata.view">
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
        <NButton size="small" @click="deselectView">
          <ArrowLeftIcon class="w-4 h-4" />
        </NButton>
      </template>
    </div>

    <template v-if="metadata.schema">
      <ViewsTable
        v-if="!metadata.view"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :views="metadata.schema.views"
        @click="handleSelectView"
      />
      <MonacoEditor
        v-if="metadata.view"
        :content="metadata.view.definition"
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
  ViewMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSelectSchema } from "../../common";
import ViewsTable from "./ViewsTable.vue";

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
  view?: ViewMetadata;
}>();

const showToolbar = computed(() => {
  if (metadata.value?.view) {
    // show view detail and back button
    return true;
  }

  if (showSchemaSelect.value) return true;
  return false;
});

const handleSelectView = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}) => {
  metadata.value = selected;
};

const deselectView = () => {
  if (!metadata.value) return;
  metadata.value.view = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      view: undefined,
    };
  },
  { immediate: true }
);
</script>
